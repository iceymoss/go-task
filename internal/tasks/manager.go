package tasks

import (
	"fmt"
	"sync"

	"github.com/iceymoss/go-task/internal/conf"
	"github.com/iceymoss/go-task/internal/core"
	"github.com/iceymoss/go-task/pkg/constants"
	"github.com/iceymoss/go-task/pkg/logger"

	"go.uber.org/zap"
)

// Scheduler 定义了调度器接口，供 ApplyAutoJobs 调用
type Scheduler interface {
	AddJob(cronExpr, taskName, uniqueJobName string, params map[string]any, source string) error
}

// ApplyJobs 将自动任务注册到调度器中
func ApplyJobs(schema Scheduler, cfg *conf.Config) {
	mu.RLock()
	defer mu.RUnlock()

	// 遍历自动任务列表
	for _, job := range autoJobs {
		fmt.Println("注册自动任务中：", job.Name)
		// 调用调度器添加任务
		err := schema.AddJob(job.Cron, job.Name, job.Name, job.Params, string(constants.TaskTypeSYSTEM))
		if err != nil {
			logger.Info("❌ [AutoLoad] Failed to load", zap.Any("jobname", job.Name), zap.Error(err))
		} else {
			logger.Info("✅ [AutoLoad] Loaded", zap.Any("jobname", job.Name))
		}
	}

	// 注册所有配置型任务
	for _, job := range cfg.Jobs {
		if !job.Enable {
			continue
		}
		err := schema.AddJob(job.Cron, job.Name, job.Name, job.Params, string(constants.TaskTypeYAML))
		if err != nil {
			logger.Info("❌ [AutoLoad] Failed to load", zap.Any("jobname", job.Name), zap.Error(err))
		} else {
			logger.Info("✅ [AutoLoad] Loaded", zap.Any("jobname", job.Name), zap.Any("cron", job.Cron))
		}
	}
}

// AutoJob 定义一个“自启动任务”的结构
type AutoJob struct {
	Name    string           // 任务唯一标识
	Cron    string           // Cron 表达式
	Creator core.TaskCreator // 构造函数
	Params  map[string]any   // 默认参数
}

var (
	registry = make(map[string]core.TaskCreator) // 普通任务注册（供 Config 调用）
	autoJobs = make([]*AutoJob, 0)               // 自动任务列表（供代码直接启动）
	mu       sync.RWMutex
)

// Register 保持不变，供 Config 使用
func Register(name string, creator core.TaskCreator) {
	mu.Lock()
	defer mu.Unlock()
	registry[name] = creator
}

// RegisterAuto 注册并自动启动 开发者只需要在自己的 task 文件里调这个，就能把“逻辑+配置”一站式搞定
func RegisterAuto(name string, cron string, creator core.TaskCreator, defaultParams map[string]any) {
	mu.Lock()
	defer mu.Unlock()

	// 1. 先注册到普通池子（这样 Web 界面也能手动触发）
	registry[name] = creator

	// 2. 加入自动启动列表
	autoJobs = append(autoJobs, &AutoJob{
		Name:    name,
		Cron:    cron,
		Creator: creator,
		Params:  defaultParams,
	})
}

// GetTaskCreator 获取任务的构造函数
func GetTaskCreator(name string) (core.TaskCreator, error) {
	mu.RLock()
	defer mu.RUnlock()
	creator, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("task implementation '%s' not found", name)
	}
	return creator, nil
}

func GetTask(name string) (core.Task, error) {
	mu.RLock()
	defer mu.RUnlock()
	creator, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("task implementation '%s' not found", name)
	}
	return creator(), nil
}
