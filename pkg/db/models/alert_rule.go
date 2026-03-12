package models

import (
	"time"
)

// AlertRule 告警规则模型
type AlertRule struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"not null;size:100" json:"name"`             // 规则名称
	Description string `gorm:"type:text" json:"description"`              // 描述
	JobID       *uint  `gorm:"index:idx_job" json:"job_id,omitempty"`     // 关联任务ID(为空表示全局规则)
	GroupID     *uint  `gorm:"index:idx_group" json:"group_id,omitempty"` // 关联分组ID(为空表示全局)

	// 告警条件
	AlertType         string `gorm:"index:idx_alert_type;size:20;not null" json:"alert_type"` // failure, timeout, retry_exceeded, success, duration_exceeded
	Condition         string `gorm:"size:20;not null" json:"condition"`                       // immediate, count_threshold, rate_threshold
	ThresholdCount    int    `gorm:"default:1" json:"threshold_count"`                        // 阈值(次数)
	ThresholdTime     int    `gorm:"default:300" json:"threshold_time"`                       // 时间窗口(秒)
	ThresholdDuration *int   `json:"threshold_duration,omitempty"`                            // 时长阈值(毫秒)

	// 告警级别
	AlertLevel string `gorm:"size:20;default:'warning'" json:"alert_level"` // info, warning, error, critical

	// 静默规则
	SilenceEnabled bool       `gorm:"default:false" json:"silence_enabled"` // 是否启用静默
	SilenceStart   *time.Time `json:"silence_start,omitempty"`              // 静默开始时间
	SilenceEnd     *time.Time `json:"silence_end,omitempty"`                // 静默结束时间

	// 状态
	Enable bool `gorm:"index:idx_enable;default:true" json:"enable"` // 是否启用

	// 时间戳
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName 指定表名
func (AlertRule) TableName() string {
	return "sys_alert_rules"
}
