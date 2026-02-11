package core

import "context"

// TaskCreator 定义任务构造函数签名
type TaskCreator func() Task

// Task 任务接口
type Task interface {
	// Run 执行任务逻辑
	// params 是从配置文件传入的动态参数
	Run(ctx context.Context, params map[string]any) error

	// Identifier 返回任务唯一标识 (用于日志)
	Identifier() string
}
