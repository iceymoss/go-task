package models

import (
	"time"
)

// WorkflowExecution 工作流执行记录模型
type WorkflowExecution struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	ExecutionID  string `gorm:"uniqueIndex;size:64;not null" json:"execution_id"`       // 执行ID
	WorkflowID   string `gorm:"index:idx_workflow;size:64;not null" json:"workflow_id"` // 工作流ID
	WorkflowName string `gorm:"not null;size:200" json:"workflow_name"`                 // 工作流名称

	// 执行状态
	Status string `gorm:"index:idx_status;size:20;not null" json:"status"` // pending, running, success, failed, cancelled, partial_success

	// 节点状态
	NodeStatus string `gorm:"type:json;not null" json:"node_status"` // 节点执行状态

	// 时间信息
	ScheduledAt time.Time  `gorm:"index:idx_scheduled;not null" json:"scheduled_at"` // 计划执行时间
	StartedAt   *time.Time `json:"started_at,omitempty"`                             // 开始时间
	FinishedAt  *time.Time `json:"finished_at,omitempty"`                            // 完成时间
	DurationMs  *int64     `json:"duration_ms,omitempty"`                            // 执行时长

	// 触发信息
	TriggerType   string `gorm:"size:20" json:"trigger_type,omitempty"`   // 触发类型
	TriggerSource string `gorm:"size:50" json:"trigger_source,omitempty"` // 触发来源

	// 错误信息
	ErrorMessage string `gorm:"type:text" json:"error_message,omitempty"` // 错误消息
	FailedNodes  string `gorm:"type:json" json:"failed_nodes,omitempty"`  // 失败的节点列表

	// 扩展字段
	Metadata string `gorm:"type:json" json:"metadata,omitempty"` // 元数据
	Tags     string `gorm:"type:json" json:"tags,omitempty"`     // 标签

	// 时间戳
	CreatedAt time.Time `gorm:"index:idx_created" json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (WorkflowExecution) TableName() string {
	return "sys_workflow_executions"
}
