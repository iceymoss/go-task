package models

import (
	"time"
)

// WorkflowNodeExecution 工作流节点执行记录模型
type WorkflowNodeExecution struct {
	ID             uint   `gorm:"primaryKey" json:"id"`
	WorkflowExecID string `gorm:"index:idx_workflow_exec;size:64;not null" json:"workflow_exec_id"` // 工作流执行ID
	NodeID         string `gorm:"index:idx_node;size:100;not null" json:"node_id"`                  // 节点ID
	JobID          *uint  `gorm:"index:idx_job" json:"job_id,omitempty"`                            // 关联任务ID
	JobName        string `gorm:"size:100" json:"job_name,omitempty"`                               // 任务名称

	// 执行状态
	Status string `gorm:"index:idx_status;size:20;not null" json:"status"` // pending, running, success, failed, skipped, timeout

	// 时间信息
	ScheduledAt time.Time  `gorm:"not null" json:"scheduled_at"` // 计划执行时间
	StartedAt   *time.Time `json:"started_at,omitempty"`         // 开始时间
	FinishedAt  *time.Time `json:"finished_at,omitempty"`        // 完成时间
	DurationMs  *int64     `json:"duration_ms,omitempty"`        // 执行时长

	// 重试信息
	RetryCount int `gorm:"default:0" json:"retry_count"` // 重试次数

	// 执行结果
	InputParams  string `gorm:"type:json" json:"input_params,omitempty"`  // 输入参数
	OutputData   string `gorm:"type:json" json:"output_data,omitempty"`   // 输出数据
	Output       string `gorm:"type:text" json:"output,omitempty"`        // 标准输出
	ErrorMessage string `gorm:"type:text" json:"error_message,omitempty"` // 错误消息

	// 时间戳
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (WorkflowNodeExecution) TableName() string {
	return "sys_workflow_node_executions"
}
