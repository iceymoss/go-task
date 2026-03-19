package models

import (
	"time"
)

// JobExecution 任务执行记录模型
type JobExecution struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	ExecutionID string `gorm:"uniqueIndex;size:64;not null" json:"execution_id"`     // 执行唯一标识(WUID)
	JobID       uint   `gorm:"index:idx_job;not null" json:"job_id"`                 // 任务ID
	JobName     string `gorm:"index:idx_job_name;size:100;not null" json:"job_name"` // 任务名称
	WorkerID    string `gorm:"size:100" json:"worker_id,omitempty"`                  // 执行Worker ID

	// 执行状态
	Status string `gorm:"index:idx_status;size:20;not null" json:"status"` // pending, running, success, failed, timeout, cancelled

	// 时间信息
	ScheduledAt time.Time  `gorm:"index:idx_scheduled;not null" json:"scheduled_at"` // 计划执行时间
	StartedAt   *time.Time `gorm:"index:idx_started" json:"started_at,omitempty"`    // 实际开始时间
	FinishedAt  *time.Time `json:"finished_at,omitempty"`                            // 完成时间
	DurationMs  *int64     `json:"duration_ms,omitempty"`                            // 执行时长(毫秒)

	// 重试信息
	RetryCount  int    `gorm:"default:0" json:"retry_count"`            // 重试次数
	MaxRetries  int    `gorm:"default:3" json:"max_retries"`            // 最大重试次数
	RetryReason string `gorm:"type:text" json:"retry_reason,omitempty"` // 重试原因

	// 依赖信息
	Dependencies     string `gorm:"type:json" json:"dependencies,omitempty"`      // 依赖任务列表
	DependencyStatus string `gorm:"type:json" json:"dependency_status,omitempty"` // 依赖任务执行状态

	// 执行结果
	Result       string `gorm:"type:json" json:"result,omitempty"`        // 执行结果数据
	Output       string `gorm:"type:text" json:"output,omitempty"`        // 标准输出
	ErrorMessage string `gorm:"type:text" json:"error_message,omitempty"` // 错误消息
	ErrorStack   string `gorm:"type:text" json:"error_stack,omitempty"`   // 错误堆栈

	// 触发信息
	TriggerType   string `gorm:"size:20" json:"trigger_type,omitempty"`   // 触发类型
	TriggerSource string `gorm:"size:50" json:"trigger_source,omitempty"` // 触发来源: cron, manual, webhook, api

	// 扩展字段
	Metadata string `gorm:"type:json" json:"metadata,omitempty"` // 元数据(标签、环境等)
	Tags     string `gorm:"type:json" json:"tags,omitempty"`     // 标签

	// 时间戳
	CreatedAt time.Time `gorm:"index:idx_created" json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (JobExecution) TableName() string {
	return "sys_job_executions"
}
