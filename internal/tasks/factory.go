package tasks

import (
	"fmt"

	"github.com/iceymoss/go-task/internal/core"
	"github.com/iceymoss/go-task/internal/tasks/email"
	"github.com/iceymoss/go-task/internal/tasks/http"
	"github.com/iceymoss/go-task/internal/tasks/shell"
	"github.com/iceymoss/go-task/internal/tasks/sql"
)

// GetTaskByType 根据任务类型创建任务实例
func GetTaskByType(taskType string) (core.Task, error) {
	switch taskType {
	case "shell":
		return shell.NewShellTask(), nil
	case "http":
		return http.NewHttpTask(), nil
	case "email":
		return email.NewEmailTask(), nil
	case "sql":
		return sql.NewSqlTask(), nil
	case "custom":
		// 自定义任务需要提前注册
		return nil, fmt.Errorf("custom task must be registered via Register()")
	default:
		return nil, fmt.Errorf("unknown task type: %s", taskType)
	}
}
