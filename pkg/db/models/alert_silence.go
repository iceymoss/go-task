package models

import (
	"time"
)

// AlertSilence 告警静默模型
type AlertSilence struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	Name    string `gorm:"not null;size:100" json:"name"` // 静默规则名称
	Comment string `gorm:"type:text" json:"comment"`      // 说明

	// 匹配规则
	JobID      *uint  `gorm:"index:idx_job" json:"job_id,omitempty"` // 任务ID
	JobName    string `gorm:"size:100" json:"job_name,omitempty"`    // 任务名称
	AlertType  string `gorm:"size:20" json:"alert_type,omitempty"`   // 告警类型
	AlertLevel string `gorm:"size:20" json:"alert_level,omitempty"`  // 告警级别
	Matchers   string `gorm:"type:json" json:"matchers,omitempty"`   // 匹配器: [{"type": "job", "value": "backup"}]

	// 时间范围
	StartTime time.Time `gorm:"not null" json:"start_time"` // 开始时间
	EndTime   time.Time `gorm:"not null" json:"end_time"`   // 结束时间

	// 状态
	Status    string    `gorm:"index:idx_status;size:20;default:'active'" json:"status"` // active, expired, cancelled
	CreatedBy *uint     `json:"created_by,omitempty"`                                    // 创建人
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (AlertSilence) TableName() string {
	return "sys_alert_silences"
}
