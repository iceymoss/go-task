package sql

import (
	"github.com/iceymoss/go-task/internal/core"
)

// Creators 暴露ai块下的所有任务工厂
func Creators() []core.TaskCreator {
	return []core.TaskCreator{
		NewSqlTask,
	}
}
