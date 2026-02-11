package engine

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/iceymoss/go-task/internal/core"
	"github.com/iceymoss/go-task/internal/tasks"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron       *cron.Cron
	Stats      *StatManager
	registered map[string]struct {
		task   core.Task
		params map[string]any
	}
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		cron:  cron.New(cron.WithSeconds()),
		Stats: NewStatManager(),
		registered: make(map[string]struct {
			task   core.Task
			params map[string]any
		}),
	}
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

	// 更新开始状态
	stat.Status = "Running"
	stat.LastRunTime = time.Now().Format("2006-01-02 15:04:05")
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
		log.Printf("❌ [Schedule] Job failed: %s, err: %v", name, err)
	} else {
		stat.LastResult = "Success"
		stat.Status = "Idle"
		log.Printf("✅ [Schedule] Job finished: %s", name)
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

func (s *Scheduler) Start() {
	s.cron.Start()
}
func (s *Scheduler) Stop() {
	s.cron.Stop()
}
