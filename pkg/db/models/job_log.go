package models

import (
	"time"
)

// JobLog 执行日志模型
type JobLog struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	ExecutionID string `gorm:"index:idx_execution;size:64;not null" json:"execution_id"` // 执行ID
	JobID       uint   `gorm:"index:idx_job;not null" json:"job_id"`                     // 任务ID
	JobName     string `gorm:"size:100;not null" json:"job_name"`                        // 任务名称

	// 日志信息
	LogLevel string  `gorm:"index:idx_level;size:20;not null" json:"log_level"` // debug, info, warning, error
	Message  string  `gorm:"type:text;not null" json:"message"`                 // 日志消息
	Fields   *string `gorm:"type:json" json:"fields,omitempty"`                 // 结构化字段
	Status   uint    `gorm:"size:20" json:"status,omitempty"`                   // 1成功 0失败

	// 时间信息
	Timestamp time.Time `gorm:"index" json:"timestamp"` // 日志时间戳
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (JobLog) TableName() string {
	return "sys_job_logs"
}
