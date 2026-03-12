package models

import (
	"time"
)

// AlertHistory 告警历史模型
type AlertHistory struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	AlertID     string `gorm:"uniqueIndex;size:64;not null" json:"alert_id"`              // 告警ID
	RuleID      *uint  `gorm:"index:idx_rule" json:"rule_id,omitempty"`                   // 规则ID
	JobID       *uint  `gorm:"index:idx_job" json:"job_id,omitempty"`                     // 任务ID
	ExecutionID string `gorm:"index:idx_execution;size:64" json:"execution_id,omitempty"` // 执行ID

	// 告警信息
	AlertType  string `gorm:"size:20;not null" json:"alert_type"`           // 告警类型
	AlertLevel string `gorm:"size:20;default:'warning'" json:"alert_level"` // 告警级别
	Title      string `gorm:"size:200" json:"title,omitempty"`              // 告警标题
	Message    string `gorm:"type:text" json:"message"`                     // 告警消息
	Details    string `gorm:"type:json" json:"details,omitempty"`           // 详细信息

	// 发送状态
	Status         string `gorm:"index:idx_status;size:20;default:'pending'" json:"status"` // pending, sending, sent, failed, cancelled
	Channels       string `gorm:"type:json" json:"channels,omitempty"`                      // 发送的渠道
	FailedChannels string `gorm:"type:json" json:"failed_channels,omitempty"`               // 发送失败的渠道

	// 时间信息
	TriggeredAt time.Time  `gorm:"index:idx_triggered;not null" json:"triggered_at"` // 触发时间
	SentAt      *time.Time `json:"sent_at,omitempty"`                                // 发送时间
	CreatedAt   time.Time  `json:"created_at"`
}

// TableName 指定表名
func (AlertHistory) TableName() string {
	return "sys_alert_history"
}
