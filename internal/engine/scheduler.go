package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/iceymoss/go-task/internal/core"
	"github.com/iceymoss/go-task/pkg/logger"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

const (
	defaultTaskTimeout = 2 * time.Hour
)

type JobDefinition struct {
	creator  core.TaskCreator // 任务实现
	params   map[string]any   // 任务参数
	chain    Chain            // 任务链, 可以加入日志，重试，限流，日志，指标，历史记录等操作
	priority int              // 任务优先级
	timeout  time.Duration    // 任务超时时间
}

type Scheduler struct {
	cron              *cron.Cron               // 任务调度器
	Stats             *StatManager             // 任务状态管理器
	DependencyManager *DependencyManager       // 任务依赖管理器
	EventManager      *EventManager            // 事件管理器
	RetryManager      *RetryManager            // 重试管理器
	TaskQueue         *TaskQueue               // 任务队列（可选，支持优先级和限流）
	logger            Logger                   // 日志管理器
	leaderElector     LeaderElector            // 选主器（可选，支持分布式部署）
	leaderCancel      context.CancelFunc       // 选主停止函数
	registry          *TaskRegistry            // 调度器持有一个菜单(注册表)
	jobDefinition     map[string]JobDefinition // 存放具体的任务订单
	mu                sync.RWMutex             // 保护 registered 和任务状态的并发访问
}

func NewScheduler(registry *TaskRegistry, opts ...Option) *Scheduler {
	scheduler := &Scheduler{
		cron:              cron.New(cron.WithSeconds()),
		Stats:             NewStatManager(),
		EventManager:      NewEventManager(NewDefaultLogger()),
		DependencyManager: NewDependencyManager(NewDefaultLogger()),
		jobDefinition:     make(map[string]JobDefinition),
		registry:          registry,
	}

	scheduler.RetryManager = NewRetryManager(scheduler.EventManager)

	// 初始化任务队列（默认 10 个 worker）
	scheduler.TaskQueue = NewTaskQueue(scheduler.runTaskWithStats, defaultWorkerNum)

	// 事件监听 (日志、指标、依赖控制)
	scheduler.EventManager.OnFunc(EventTypeBeforeJob, LoggingEventHandler(scheduler.logger))
	scheduler.EventManager.OnFunc(EventTypeAfterJob, LoggingEventHandler(scheduler.logger))
	scheduler.EventManager.OnFunc(EventTypeJobError, LoggingEventHandler(scheduler.logger))
	scheduler.EventManager.OnFunc(EventTypeJobPanic, LoggingEventHandler(scheduler.logger))

	scheduler.EventManager.OnFunc(EventTypeAfterJob, MetricsEventHandler(scheduler.logger))
	scheduler.EventManager.OnFunc(EventTypeJobError, MetricsEventHandler(scheduler.logger))

	scheduler.EventManager.OnFunc(EventTypeAfterJob, DependencyEventHandler(scheduler.DependencyManager, scheduler.EventManager))
	scheduler.EventManager.OnFunc(EventTypeJobError, DependencyEventHandler(scheduler.DependencyManager, scheduler.EventManager))
	scheduler.EventManager.OnFunc(EventTypeDependencyMet, DependencyMetEventHandler(scheduler))

	scheduler.EventManager.OnFunc(EventTypeJobSkipped, LoggingEventHandler(scheduler.logger))
	scheduler.EventManager.OnFunc(EventTypeJobRetry, LoggingEventHandler(scheduler.logger))

	// 应用外部传入的 Option (可以覆盖上面的默认值)
	for _, opt := range opts {
		opt(scheduler)
	}

	return scheduler
}

// Option 定义了调度器的配置选项
type Option func(*Scheduler)

// WithWorkerNum 配置任务队列的并发 worker 数量
func WithWorkerNum(num int) Option {
	return func(s *Scheduler) {
		s.TaskQueue = NewTaskQueue(s.runTaskWithStats, num)
	}
}

// WithEventManager 允许外部注入一个完全自定义的事件管理器
func WithEventManager(em *EventManager) Option {
	return func(s *Scheduler) {
		s.EventManager = em
	}
}

// WithCronOptions 允许用户自定义底层 cron 的行为 (例如更换时区)
func WithCronOptions(opts ...cron.Option) Option {
	return func(s *Scheduler) {
		s.cron = cron.New(opts...)
	}
}

// WithHistoryStorage 允许用户注入自定义的历史记录存储器
func WithHistoryStorage(storage HistoryStorage) Option {
	return func(s *Scheduler) {
		s.EventManager.OnFunc(EventTypeAfterJob, NewHistoryEventHandler(storage))
		s.EventManager.OnFunc(EventTypeJobError, NewHistoryEventHandler(storage))
	}
}

// WithLeaderElector 注入分布式选主器
func WithLeaderElector(elector LeaderElector) Option {
	return func(s *Scheduler) {
		s.leaderElector = elector
	}
}

// WithLogger 外部注入自定义的日志实现
func WithLogger(l Logger) Option {
	return func(s *Scheduler) {
		if l != nil {
			s.logger = l
		}
	}
}

// buildDefaultChain 为任务构建默认的执行链：
// Logging + Metrics + RetryWithPolicy + Timeout
func (s *Scheduler) buildDefaultChain(taskName string) Chain {
	return Chain{}.
		Then(
			Logging(taskName),
			Metrics(taskName),
			RetryWithPolicy(s.RetryManager, taskName),
			Timeout(defaultTaskTimeout),
		)
}

// AddJob 向调度内核中动态注册并启动一个具体的任务实例 (Job Instance)。
// 该方法支持“一模多跑”的高级特性：允许基于同一个底层任务模板 (taskName)，
// 注入不同的动态参数和频率，生成多个互相隔离的运行实例 (uniqueJobName)。
//
// 参数说明:
//   - cronExpr:      任务的执行频率，标准 Cron 表达式 (如 "@every 1m" 或 "0 * * * *")。
//   - taskName:      任务模板名 (Template Name)，必须是已在 TaskRegistry 中注册的标识 (如 "sys:google_ping")，内核据此寻找执行逻辑。
//   - uniqueJobName: 任务实例的全系统唯一标识 (Instance ID)，如 "job_ping_baidu"。内核依据此 ID 进行并发隔离、依赖拓扑构建、状态追踪及手动触发。
//   - params:        该实例的专属运行时参数。执行前会与任务模板自带的 DefaultParams 发生合并与覆盖。
//   - source:        任务来源标记 (如 "SYSTEM", "YAML", "API")，仅用于控制台展示、日志追踪与运维审计。
//
// 返回值:
//   - error: 当传入的 taskName 在注册表中不存在，或 cronExpr 语法解析失败时，将拒绝挂载并返回错误。
func (s *Scheduler) AddJob(cronExpr, taskName string, uniqueJobName string, params map[string]any, source string) error {
	// 获取任务实现
	creator, err := s.registry.Get(taskName)
	if err != nil {
		return err
	}

	// 初始化状态
	s.Stats.Set(uniqueJobName, &JobStats{
		Name:       uniqueJobName,
		CronExpr:   cronExpr,
		Status:     Idle,
		LastResult: LastResultPending,
		Source:     source,
	})

	// 保存引用以便手动触发
	s.mu.Lock()
	s.jobDefinition[uniqueJobName] = JobDefinition{
		creator:  creator,
		params:   params,
		chain:    s.buildDefaultChain(uniqueJobName),
		priority: 0,
	}
	s.mu.Unlock()

	// 包装执行逻辑
	wrapper := func() {
		s.Dispatch(uniqueJobName)
	}

	// 加入 Cron 底层任务调度器中，负责任务调度
	entryID, err := s.cron.AddFunc(cronExpr, wrapper)
	if err == nil {
		stat, ok := s.Stats.Get(uniqueJobName)
		if !ok {
			return fmt.Errorf("⚠️ [Schedule] Job not jobDefinition: %s", uniqueJobName)
		}
		stat.RawNext = s.cron.Entry(entryID).Next
		stat.NextRunTime = stat.RawNext.Format("2006-01-02 15:04:05")
	}
	return err
}

// runTaskWithStats 执行并记录状态
func (s *Scheduler) runTaskWithStats(name string) {
	// 读取注册信息
	s.mu.RLock()
	reg, ok := s.jobDefinition[name]
	s.mu.RUnlock()
	if !ok {
		logger.Info("⚠️ [Schedule] Job not jobDefinition", zap.Any("name", name))
		return
	}

	task := reg.creator()
	params := reg.params
	chain := reg.chain
	timeout := reg.timeout
	if timeout <= 0 {
		timeout = defaultTaskTimeout
	}

	stat, ok := s.Stats.Get(name)
	if !ok {
		logger.Info("⚠️ [Schedule] Job not jobDefinition", zap.Any("name", name))
		return
	}
	ctx := context.Background()

	// 发射任务开始事件
	s.EventManager.Emit(&Event{
		Type:      EventTypeBeforeJob,
		TaskName:  name,
		TimeStamp: time.Now(),
		Context:   ctx,
	})

	// 更新为运行状态
	stat.Status = Running
	stat.RunCount++

	logger.Info("🚀 [Schedule] Starting job", zap.Any("name", name))

	// 执行 (带超时控制)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	startTime := time.Now()

	// 包装为 JobFunc
	jobFunc := func(c context.Context) error {
		return task.Run(c, params)
	}

	// 应用任务链（含重试、日志、指标等）
	if chain != nil && len(chain) > 0 {
		jobFunc = chain.Apply(jobFunc)
	}

	err := jobFunc(ctx)
	durationMs := time.Since(startTime).Milliseconds()

	// 更新结束状态
	if err != nil {
		stat.LastResult = fmt.Sprintf(LastResultError, err)
		stat.Status = Error
		s.DependencyManager.UpdateTaskStatus(name, false, err)
		logger.Info(fmt.Sprintf("❌ [Schedule] Job failed: %s, err: %v", name, err))

		s.EventManager.Emit(&Event{
			Type:      EventTypeJobError,
			TaskName:  name,
			TimeStamp: time.Now(),
			Context:   ctx,
			Error:     err,
			Data: map[string]any{
				"duration_ms": durationMs,
				"start_time":  startTime,
			},
		})
	} else {
		stat.LastResult = LastResultSuccess
		stat.Status = Idle
		s.DependencyManager.UpdateTaskStatus(name, true, nil)
		logger.Info("✅ [Schedule] Job finished: %s", zap.Any("name", name))

		s.EventManager.Emit(&Event{
			Type:      EventTypeAfterJob,
			TaskName:  name,
			TimeStamp: time.Now(),
			Context:   ctx,
			Data: map[string]any{
				"duration_ms": durationMs,
				"start_time":  startTime,
			},
		})
	}
}

// ManualRun 手动触发
func (s *Scheduler) ManualRun(uniqueJobName string) error {
	reg, ok := s.jobDefinition[uniqueJobName]
	if !ok {
		return fmt.Errorf("job not found")
	}
	if s.TaskQueue != nil {
		if err := s.TaskQueue.Enqueue(uniqueJobName, reg.priority); err != nil {
			return err
		}
		return nil
	}
	go s.runTaskWithStats(uniqueJobName)
	return nil
}

// AddJobWithDependency 添加带依赖的任务
func (s *Scheduler) AddJobWithDependency(cronExpr, taskName string, uniqueJobName string, params map[string]any, source string, dependencyRule *DependencyRule) error {
	// 先添加依赖规则
	if dependencyRule != nil {
		if err := s.DependencyManager.AddDependency(dependencyRule); err != nil {
			return fmt.Errorf("failed to add dependency: %w", err)
		}
	}

	// 添加任务
	return s.AddJob(cronExpr, taskName, uniqueJobName, params, source)
}

// GetDependencyChain 获取任务的依赖链
func (s *Scheduler) GetDependencyChain(taskName string) ([]string, error) {
	return s.DependencyManager.GetDependencyChain(taskName)
}

// GetDependentTasks 获取依赖于指定任务的所有任务
func (s *Scheduler) GetDependentTasks(taskName string) []string {
	return s.DependencyManager.GetDependentTasks(taskName)
}

// Dispatch 尝试分发任务：如果依赖未满足则挂起(标记为Waiting)，否则真正入队
func (s *Scheduler) Dispatch(name string) {
	// 统一通过队列执行，便于限流和优先级控制
	// 加入任务队列后，TaskQueue初始化时开启的worker会自动从队列中获取任务进行处理

	s.mu.RLock()
	reg, ok := s.jobDefinition[name]
	s.mu.RUnlock()
	if !ok {
		return
	}

	stat, ok := s.Stats.Get(name)
	if !ok {
		logger.Info("⚠️ [Schedule] Job not jobDefinition", zap.Any("name", name))
		return
	}

	// 无阻塞检查依赖状态
	satisfied, err := s.DependencyManager.CheckDependencies(name)
	if err != nil {
		stat.Status = Error
		stat.LastResult = fmt.Sprintf(LastResultDependencyCheck, err)
		logger.Info("❌ [Dispatcher] Job dependency check failed", zap.Any("name", name), zap.Error(err))
		return
	}

	// 依赖未满足，仅仅标记为挂起等待,避免不占用 Worker 协程
	if !satisfied {
		stat.Status = Waiting
		logger.Info("⏳ [Dispatcher] Job triggered but waiting for upstream dependencies...", zap.Any("name", name))
		return
	}

	// 依赖已完全满足，推入真实执行队列
	stat.Status = Queued
	if s.TaskQueue != nil {
		if err := s.TaskQueue.Enqueue(name, reg.priority); err != nil {
			logger.Info("⚠️ [Dispatcher] Enqueue job failed", zap.Any("name", name), zap.Error(err))
		}
	} else {
		go s.runTaskWithStats(name)
	}
}

// RegisterEventHandler 注册事件处理器
func (s *Scheduler) RegisterEventHandler(eventType EventType, handler EventHandler) {
	s.EventManager.On(eventType, handler)
}

// EmitEvent 发射事件
func (s *Scheduler) EmitEvent(event *Event) {
	s.EventManager.Emit(event)
}

// SetRetryPolicy 为指定任务配置重试策略
func (s *Scheduler) SetRetryPolicy(taskName string, policy *RetryPolicy) {
	s.RetryManager.SetPolicy(taskName, policy)
}

func (s *Scheduler) Start() {
	// 如果没有配置 Leader 选举，则保持单机行为：直接启动 cron
	if s.leaderElector == nil {
		s.cron.Start()
		return
	}

	// 定义抢到 Leader 和失去 Leader 时的动作
	onStarted := func() {
		logger.Info("👑 [Scheduler] This instance became leader, starting cron")
		s.cron.Start()
	}
	onStopped := func() {
		logger.Info("👋 [Scheduler] Lost leadership, stopping cron")
		s.cron.Stop()
	}

	// 有 Leader 选举时，由 LeaderElector 决定什么时候启动/停止 cron
	ctx, cancel := context.WithCancel(context.Background())
	s.leaderCancel = cancel

	err := s.leaderElector.Start(ctx, onStarted, onStopped)
	if err != nil {
		// 绝对不能 fallback 到 s.cron.Start() 如果启动多实例，会导致重复执行
		// 应该直接 Fatal，让程序起不来，引起运维注意，防止脑裂。
		logger.Fatal("[Scheduler] Fatal error: Leader election failed to start", zap.Error(err))
	}
}
func (s *Scheduler) Stop() {
	if s.leaderCancel != nil {
		s.leaderCancel()
	}
	if s.leaderElector != nil {
		_ = s.leaderElector.Stop(context.Background())
	}

	s.EventManager.Stop()

	s.cron.Stop()
	if s.TaskQueue != nil {
		s.TaskQueue.Stop()
	}
}

// SetPriority 为任务设置优先级（数值越大优先级越高）
func (s *Scheduler) SetPriority(taskName string, priority int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	reg, ok := s.jobDefinition[taskName]
	if !ok {
		return
	}
	reg.priority = priority
	s.jobDefinition[taskName] = reg
}
