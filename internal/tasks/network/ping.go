package network

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/iceymoss/go-task/internal/core"
	"github.com/iceymoss/go-task/internal/tasks/base_task"
	"github.com/iceymoss/go-task/pkg/constants"
)

const NetworkHttpTaskName = "network:http:ping"

// PingTask 结构体
type PingTask struct {
	base_task.BaseTask
}

func NewPingTask() core.Task {
	return &PingTask{
		BaseTask: base_task.BaseTask{
			Name:        NetworkHttpTaskName,
			DefaultCron: "@every 1m",
			DefaultParams: map[string]any{
				"url":     "https://www.google.com",
				"timeout": 5,
			},
			TaskType: constants.TaskTypeSYSTEM,
		},
	}
}

func (t *PingTask) Run(ctx context.Context, params map[string]any) error {
	// 1. 即使是自动任务，也可以读取 Params，因为我们注册时传进去了
	url, _ := params["url"].(string)

	log.Printf("📡 [Ping] Pinging %s ...", url)

	// ... (简单的 Ping 逻辑)
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Head(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("status code %d", resp.StatusCode)
	}

	log.Printf("✅ [Ping] Success: %s", url)
	return nil
}
