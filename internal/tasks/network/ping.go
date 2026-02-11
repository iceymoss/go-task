package network

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/iceymoss/go-task/internal/core"
	"github.com/iceymoss/go-task/internal/tasks"
)

// PingTask 结构体
type PingTask struct{}

// init 【核心所在】
// 只要这个包被 import，这个 init 就会执行，任务就会自动挂载
func init() {
	// 定义默认参数
	defaultParams := map[string]any{
		"url":     "https://www.google.com",
		"timeout": 5,
	}

	// 逻辑注册 + 时间配置 + 参数定义
	// 这里的 "sys:google_ping" 是任务名， "@every 1m" 是时间
	tasks.RegisterAuto("sys:google_ping", "@every 1m", NewPingTask, defaultParams)
}

func NewPingTask() core.Task {
	return &PingTask{}
}

func (t *PingTask) Identifier() string {
	return "sys:google_ping"
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
