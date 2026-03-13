package engine

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/iceymoss/go-task/internal/core"
	"github.com/iceymoss/go-task/internal/tasks"

	"github.com/robfig/cron/v3"
)

type Register struct {
	task     core.Task      // 任务实现
	params   map[string]any // 任务参数
	chain    Chain          // 任务链, 可以加入日志，重试，限流，日志，指标，历史记录等操作
	priority int            // 任务优先级
}

type Scheduler struct {
	cron              *cron.Cron          // 任务调度器
	Stats             *StatManager        // 任务状态管理器
	DependencyManager *DependencyManager  // 任务依赖管理器
	EventManager      *EventManager       // 事件管理器
	RetryManager      *RetryManager       // 重试管理器
	TaskQueue         *TaskQueue          // 任务队列（可选，支持优先级和限流）
	leaderElector     LeaderElector       // 选主器（可选，支持分布式部署）
	leaderCancel      context.CancelFunc  // 选主停止函数
	registered        map[string]Register // 已注册的任务信息，key 是 uniqueJobName
	mu                sync.RWMutex        // 保护 registered 和任务状态的并发访问
}

func NewScheduler() *Scheduler {
	scheduler := &Scheduler{
		cron:              cron.New(cron.WithSeconds()),
		Stats:             NewStatManager(),
		DependencyManager: NewDependencyManager(),
		EventManager:      NewEventManager(),
		RetryManager:      NewRetryManager(),
		registered:        make(map[string]Register),
	}

	// 设置全局事件管理器
	SetGlobalEventManager(scheduler.EventManager)

	// 注册默认事件处理器
	scheduler.EventManager.OnFunc(EventTypeBeforeJob, LoggingEventHandler())
	scheduler.EventManager.OnFunc(EventTypeAfterJob, LoggingEventHandler())
	scheduler.EventManager.OnFunc(EventTypeJobError, LoggingEventHandler())
	scheduler.EventManager.OnFunc(EventTypeJobPanic, LoggingEventHandler())
	scheduler.EventManager.OnFunc(EventTypeJobSkipped, LoggingEventHandler())
	scheduler.EventManager.OnFunc(EventTypeJobRetry, LoggingEventHandler())

	scheduler.EventManager.OnFunc(EventTypeAfterJob, MetricsEventHandler())
	scheduler.EventManager.OnFunc(EventTypeJobError, MetricsEventHandler())

	// 任务历史记录
	historyStorage := NewGormHistoryStorage()
	scheduler.EventManager.OnFunc(EventTypeAfterJob, NewHistoryEventHandler(historyStorage))
	scheduler.EventManager.OnFunc(EventTypeJobError, NewHistoryEventHandler(historyStorage))

	// 初始化任务队列（默认 10 个 worker）
	scheduler.TaskQueue = NewTaskQueue(scheduler, 10)

	return scheduler
}

// buildDefaultChain 为任务构建默认的执行链：
// Logging + Metrics + RetryWithPolicy
func (s *Scheduler) buildDefaultChain(taskName string) Chain {
	return Chain{}.
		Then(
			Logging(taskName),
			Metrics(taskName),
			RetryWithPolicy(s.RetryManager, taskName),
		)
}

// AddJob 添加任务
func (s *Scheduler) AddJob(cronExpr, taskName, uniqueJobName string, params map[string]any, source string) error {
	// 1. 获取任务实现
	taskInstance, err := tasks.GetTask(taskName)
	if err != nil {
		return err
	}

	// 2. 初始化状态
	s.Stats.Set(uniqueJobName, &JobStats{
		Name:       uniqueJobName,
		CronExpr:   cronExpr,
		Status:     "Idle",
		LastResult: "Pending",
		Source:     source,
	})

	// 保存引用以便手动触发
	s.registered[uniqueJobName] = Register{
		task:     taskInstance,
		params:   params,
		chain:    s.buildDefaultChain(uniqueJobName),
		priority: 0,
	}

	// 3. 包装执行逻辑
	wrapper := func() {
		// 统一通过队列执行，便于限流和优先级控制
		if s.TaskQueue != nil {
			s.mu.RLock()
			reg := s.registered[uniqueJobName]
			s.mu.RUnlock()
			if err := s.TaskQueue.Enqueue(uniqueJobName, reg.priority); err != nil {
				log.Printf("⚠️ [Schedule] Enqueue job failed: %s, err: %v", uniqueJobName, err)
			}
		} else {
			s.runTaskWithStats(uniqueJobName)
		}
	}

	// 4. 加入 Cron
	entryID, err := s.cron.AddFunc(cronExpr, wrapper)
	if err == nil {
		stat := s.Stats.Get(uniqueJobName)
		stat.rawNext = s.cron.Entry(entryID).Next
		stat.NextRunTime = stat.rawNext.Format("2006-01-02 15:04:05")
	}
	return err
}

// runTaskWithStats 执行并记录状态
func (s *Scheduler) runTaskWithStats(name string) {
	// 读取注册信息
	s.mu.RLock()
	reg, ok := s.registered[name]
	s.mu.RUnlock()
	if !ok {
		log.Printf("⚠️ [Schedule] Job not registered: %s", name)
		return
	}

	task := reg.task
	params := reg.params
	chain := reg.chain

	stat := s.Stats.Get(name)
	ctx := context.Background()

	// 发射任务开始事件
	s.EventManager.Emit(&Event{
		Type:      EventTypeBeforeJob,
		TaskName:  name,
		TimeStamp: time.Now(),
		Context:   ctx,
	})

	// 更新开始状态
	stat.Status = "Waiting"
	stat.LastRunTime = time.Now().Format("2006-01-02 15:04:05")
	log.Printf("⏳ [Schedule] Job waiting for dependencies: %s", name)

	// 检查依赖关系
	if err := s.DependencyManager.WaitForDependencies(name); err != nil {
		stat.LastResult = fmt.Sprintf("Dependency error: %v", err)
		stat.Status = "Error"
		s.DependencyManager.UpdateTaskStatus(name, false, err)
		log.Printf("❌ [Schedule] Job dependency check failed: %s, err: %v", name, err)

		s.EventManager.Emit(&Event{
			Type:      EventTypeJobError,
			TaskName:  name,
			TimeStamp: time.Now(),
			Context:   ctx,
			Error:     err,
		})
		return
	}

	// 更新为运行状态
	stat.Status = "Running"
	stat.RunCount++

	log.Printf("🚀 [Schedule] Starting job: %s", name)

	// 执行 (带超时控制)
	ctx, cancel := context.WithTimeout(context.Background(), 65*time.Minute) // 考虑到有休眠，时间给长一点
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
		stat.LastResult = fmt.Sprintf("Error: %v", err)
		stat.Status = "Error"
		s.DependencyManager.UpdateTaskStatus(name, false, err)
		log.Printf("❌ [Schedule] Job failed: %s, err: %v", name, err)

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
		stat.LastResult = "Success"
		stat.Status = "Idle"
		s.DependencyManager.UpdateTaskStatus(name, true, nil)
		log.Printf("✅ [Schedule] Job finished: %s", name)

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
	reg, ok := s.registered[uniqueJobName]
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
func (s *Scheduler) AddJobWithDependency(cronExpr, taskName, uniqueJobName string, params map[string]any, source string, dependencyRule *DependencyRule) error {
	// 1. 先添加依赖规则
	if dependencyRule != nil {
		if err := s.DependencyManager.AddDependency(dependencyRule); err != nil {
			return fmt.Errorf("failed to add dependency: %w", err)
		}
	}

	// 2. 添加任务
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

	// 有 Leader 选举时，由 LeaderElector 决定什么时候启动/停止 cron
	ctx, cancel := context.WithCancel(context.Background())
	s.leaderCancel = cancel

	// 直接调用 Start
	err := s.leaderElector.Start(ctx)
	if err != nil {
		// 绝对不能 fallback 到 s.cron.Start() 如果启动多实例，会导致重复执行
		// 应该直接 Fatal，让程序起不来，引起运维注意，防止脑裂。
		log.Fatalf("[Scheduler] Fatal error: Leader election failed to start. Aborting to prevent split-brain. Error: %v", err)
	}
}
func (s *Scheduler) Stop() {
	if s.leaderCancel != nil {
		s.leaderCancel()
	}
	if s.leaderElector != nil {
		_ = s.leaderElector.Stop(context.Background())
	}

	s.cron.Stop()
	if s.TaskQueue != nil {
		s.TaskQueue.Stop()
	}
}

// SetPriority 为任务设置优先级（数值越大优先级越高）
func (s *Scheduler) SetPriority(taskName string, priority int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	reg, ok := s.registered[taskName]
	if !ok {
		return
	}
	reg.priority = priority
	s.registered[taskName] = reg
}
