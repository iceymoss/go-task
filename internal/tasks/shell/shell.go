package shell

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/iceymoss/go-task/internal/core"
	"github.com/iceymoss/go-task/pkg/logger"
	"go.uber.org/zap"
)

const TaskName = "shell"

// ShellTask Shell 命令任务
type ShellTask struct{}

func init() {
	// 注册到任务管理器（虽然主要是通过工厂使用，但也支持直接注册）
	// tasks.Register(TaskName, NewShellTask)
}

func NewShellTask() core.Task {
	return &ShellTask{}
}

func (t *ShellTask) Identifier() string {
	return TaskName
}

// ShellParams 参数结构
type ShellParams struct {
	Command    string   `json:"command" binding:"required"` // 要执行的命令
	WorkingDir string   `json:"working_dir"`                // 工作目录
	Env        []string `json:"env"`                        // 环境变量
	Timeout    int      `json:"timeout"`                    // 超时时间（秒）
}

func (t *ShellTask) Run(ctx context.Context, params map[string]any) error {
	// 解析参数
	p := parseParams(params)
	if p.Command == "" {
		return fmt.Errorf("command is required")
	}

	logger.Info("🚀 [ShellTask] Starting command",
		zap.String("command", p.Command),
		zap.String("working_dir", p.WorkingDir),
	)

	// 准备命令
	// 支持多平台：Linux/Mac 使用 sh，Windows 使用 cmd
	var cmd *exec.Cmd
	if strings.Contains(strings.ToLower(p.Command), "||") || strings.Contains(strings.ToLower(p.Command), "&&") {
		// 如果包含管道或逻辑运算符，使用 sh -c
		cmd = exec.CommandContext(ctx, "sh", "-c", p.Command)
	} else {
		// 简单命令，直接执行
		parts := strings.Fields(p.Command)
		if len(parts) > 0 {
			cmd = exec.CommandContext(ctx, parts[0], parts[1:]...)
		} else {
			return fmt.Errorf("invalid command")
		}
	}

	// 设置工作目录
	if p.WorkingDir != "" {
		cmd.Dir = p.WorkingDir
	}

	// 设置环境变量
	if len(p.Env) > 0 {
		cmd.Env = append(cmd.Env, p.Env...)
	}

	// 执行命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("❌ [ShellTask] Command failed",
			zap.String("command", p.Command),
			zap.Error(err),
			zap.String("output", string(output)),
		)
		return fmt.Errorf("command failed: %w, output: %s", err, string(output))
	}

	logger.Info("✅ [ShellTask] Command completed successfully",
		zap.String("command", p.Command),
		zap.String("output", string(output)),
	)

	return nil
}

func parseParams(params map[string]any) ShellParams {
	p := ShellParams{
		Timeout: 300, // 默认5分钟
	}

	if v, ok := params["command"].(string); ok {
		p.Command = v
	}
	if v, ok := params["working_dir"].(string); ok {
		p.WorkingDir = v
	}
	if v, ok := params["env"].([]string); ok {
		p.Env = v
	}
	if v, ok := params["timeout"].(float64); ok {
		p.Timeout = int(v)
	}

	return p
}
