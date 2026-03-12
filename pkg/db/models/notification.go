package models

import (
	"time"
)

// Notification 通知模型
type Notification struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	ReceiverID *uint  `gorm:"index:idx_receiver;not null" json:"receiver_id"` // 接收人ID
	Receiver   string `gorm:"not null;size:100" json:"receiver"`              // 接收人名称

	// 通知内容
	Type    string `gorm:"index:idx_type;size:50;not null" json:"type"` // 通知类型
	Subject string `gorm:"size:200" json:"subject,omitempty"`           // 标题
	Content string `gorm:"type:text" json:"content"`                    // 内容
	Data    string `gorm:"type:json" json:"data,omitempty"`             // 附加数据

	// 关联信息
	RelatedID   string `gorm:"size:100" json:"related_id,omitempty"`  // 关联ID
	RelatedType string `gorm:"size:50" json:"related_type,omitempty"` // 关联类型

	// 状态
	Status string     `gorm:"index:idx_status;size:20;default:'unread'" json:"status"` // unread, read, archived
	ReadAt *time.Time `json:"read_at,omitempty"`                                       // 阅读时间

	// 优先级
	Priority string `gorm:"size:20;default:'info'" json:"priority"` // info, warning, error

	// 有效期
	ExpiresAt *time.Time `json:"expires_at,omitempty"` // 过期时间

	// 时间戳
	CreatedAt time.Time `gorm:"index:idx_created;not null" json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Notification) TableName() string {
	return "sys_notifications"
}
