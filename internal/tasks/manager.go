package tasks

import (
	"fmt"
	"log"
	"sync"

	"github.com/iceymoss/go-task/internal/core"
	"github.com/iceymoss/go-task/pkg/constants"
)

type Scheduler interface {
	AddJob(cronExpr, taskName, uniqueJobName string, params map[string]any, source string) error
}

func ApplyAutoJobs(sched Scheduler) {
	mu.RLock()
	defer mu.RUnlock()

	fmt.Println("自动任务列表：", autoJobs)
	for _, job := range autoJobs {
		fmt.Println("注册自动任务中：", job.Name)
		// 调用调度器添加任务
		err := sched.AddJob(job.Cron, job.Name, job.Name, job.Params, string(constants.TaskTypeSYSTEM))
		if err != nil {
			log.Printf("❌ [AutoLoad] Failed to load %s: %v", job.Name, err)
		} else {
			log.Printf("✅ [AutoLoad] Loaded: %s [%s]", job.Name, job.Cron)
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

func GetTask(name string) (core.Task, error) {
	mu.RLock()
	defer mu.RUnlock()
	creator, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("task implementation '%s' not found", name)
	}
	return creator(), nil
}
