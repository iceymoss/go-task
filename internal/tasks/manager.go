package tasks

import (
	"github.com/iceymoss/go-task/internal/conf"
	"github.com/iceymoss/go-task/internal/core"
	"github.com/iceymoss/go-task/internal/engine"
	"github.com/iceymoss/go-task/internal/tasks/ai"
	"github.com/iceymoss/go-task/internal/tasks/email"
	"github.com/iceymoss/go-task/internal/tasks/network"
	"github.com/iceymoss/go-task/internal/tasks/shell"
	"github.com/iceymoss/go-task/internal/tasks/sql"
	"github.com/iceymoss/go-task/pkg/constants"
	"github.com/iceymoss/go-task/pkg/logger"

	"go.uber.org/zap"
)

// LoadAllTasks 统一装配：负责将任务注册到菜单，并交给调度器运行
func LoadAllTasks(registry *engine.TaskRegistry, scheduler *engine.Scheduler, cfg *conf.Config) {
	var allCreators []core.TaskCreator
	allCreators = append(allCreators, ai.Creators()...)
	allCreators = append(allCreators, email.Creators()...)
	allCreators = append(allCreators, network.Creators()...)
	allCreators = append(allCreators, sql.Creators()...)
	allCreators = append(allCreators, shell.Creators()...)

	for _, creator := range allCreators {
		task := creator()
		name := task.Identifier()

		// 将所有任务执行逻辑都注册到任务注册中心
		registry.Register(name, creator)

		// 如果是系统任务，直接将对应的系统任务参数添加到调度器中
		if task.GetTaskType() == constants.TaskTypeSYSTEM {
			err := scheduler.AddJob(
				task.GetDefaultCron(),
				name,
				name,
				task.GetDefaultParams(),
				string(task.GetTaskType()),
			)
			if err != nil {
				logger.Logger.Error("add system job failed", zap.String("task_name", name), zap.Error(err))
			}
		}
	}

	// 处理外部 YAML 的覆盖配置
	for _, job := range cfg.Jobs {
		if !job.Enable {
			continue
		}

		cronExpr := job.Cron
		if cronExpr == "" { // 如果 YAML 没配时间，读任务内置时间
			if creator, err := registry.Get(job.Name); err == nil {
				cronExpr = creator().GetDefaultCron()
				if cronExpr == "" {
					logger.Error("task has no cron", zap.String("task_name", job.Name))
					continue
				}
			}
		}

		err := scheduler.AddJob(cronExpr, job.Name, job.Name, job.Params, string(constants.TaskTypeYAML))
		if err != nil {
			logger.Logger.Error("add config job failed", zap.String("task_name", job.Name), zap.Error(err))
		}
	}

}
