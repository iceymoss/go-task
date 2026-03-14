package base_task

import (
	"github.com/iceymoss/go-task/pkg/constants"
)

// BaseTask 提供所有任务共用的基础能力
type BaseTask struct {
	Name          string
	DefaultCron   string
	DefaultParams map[string]any
	TaskType      constants.TaskType
}

func (b *BaseTask) Identifier() string               { return b.Name }
func (b *BaseTask) GetDefaultCron() string           { return b.DefaultCron }
func (b *BaseTask) GetDefaultParams() map[string]any { return b.DefaultParams }
func (b *BaseTask) GetTaskType() constants.TaskType {
	return b.TaskType
}
