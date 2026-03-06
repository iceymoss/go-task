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

type Scheduler struct {
	cron              *cron.Cron
	Stats             *StatManager
	DependencyManager *DependencyManager
	EventManager      *EventManager
	RetryManager      *RetryManager
	registered        map[string]struct {
		task   core.Task
		params map[string]any
		chain  Chain
	}
	mu                sync.RWMutex
}

func NewScheduler() *Scheduler {
	scheduler := &Scheduler{
		cron:              cron.New(cron.WithSeconds()),
		Stats:             NewStatManager(),
		DependencyManager: NewDependencyManager(),
		EventManager:      NewEventManager(),
		RetryManager:      NewRetryManager(),
		registered: make(map[string]struct {
			task   core.Task
			params map[string]any
			chain  Chain
		}),
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

	return scheduler
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
	s.registered[uniqueJobName] = struct {
		task   core.Task
		params map[string]any
	}{taskInstance, params}

	// 3. 包装执行逻辑
	wrapper := func() {
		s.runTaskWithStats(uniqueJobName, taskInstance, params)
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
func (s *Scheduler) runTaskWithStats(name string, task core.Task, params map[string]any) {
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

	err := task.Run(ctx, params)

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
		})
	}
}

// ManualRun 手动触发
func (s *Scheduler) ManualRun(uniqueJobName string) error {
	reg, ok := s.registered[uniqueJobName]
	if !ok {
		return fmt.Errorf("job not found")
	}
	go s.runTaskWithStats(uniqueJobName, reg.task, reg.params)
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

func (s *Scheduler) Start() {
	s.cron.Start()
}
func (s *Scheduler) Stop() {
	s.cron.Stop()
}
