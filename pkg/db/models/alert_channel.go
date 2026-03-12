package models

import (
	"time"
)

// AlertChannel 告警通知渠道模型
type AlertChannel struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"not null;size:100" json:"name"`                       // 渠道名称
	ChannelType string `gorm:"index:idx_type;not null;size:20" json:"channel_type"` // email, sms, dingtalk, wechat, feishu, slack, webhook, telegram
	Config      string `gorm:"type:json;not null" json:"config"`                    // 渠道配置(加密)
	Priority    int    `gorm:"default:0" json:"priority"`                           // 优先级
	Enable      bool   `gorm:"index:idx_enable;default:true" json:"enable"`         // 是否启用

	// 时间戳
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName 指定表名
func (AlertChannel) TableName() string {
	return "sys_alert_channels"
}
