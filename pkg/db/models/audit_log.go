package models

import (
	"time"
)

// AuditLog 审计日志模型
type AuditLog struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	UserID     *uint  `gorm:"index:idx_user" json:"user_id,omitempty"`              // 用户ID
	Username   string `gorm:"index:idx_username;size:100" json:"username"`          // 用户名
	Action     string `gorm:"index:idx_action;size:50;not null" json:"action"`      // 操作类型
	Resource   string `gorm:"index:idx_resource;size:100;not null" json:"resource"` // 资源类型
	ResourceID string `gorm:"size:100" json:"resource_id,omitempty"`                // 资源ID

	// 操作详情
	Details  string `gorm:"type:json" json:"details,omitempty"`   // 操作详情(JSON)
	OldValue string `gorm:"type:text" json:"old_value,omitempty"` // 旧值
	NewValue string `gorm:"type:text" json:"new_value,omitempty"` // 新值

	// 请求信息
	Method    string `gorm:"size:10" json:"method,omitempty"`      // HTTP方法
	Path      string `gorm:"size:500" json:"path,omitempty"`       // 请求路径
	IP        string `gorm:"size:50" json:"ip"`                    // IP地址
	UserAgent string `gorm:"size:500" json:"user_agent,omitempty"` // User-Agent
	RequestID string `gorm:"size:64" json:"request_id,omitempty"`  // 请求ID

	// 结果
	Status    string `gorm:"index:idx_status;size:20;not null" json:"status"` // success, failed
	ErrorCode string `gorm:"size:20" json:"error_code,omitempty"`             // 错误码

	// 时间戳
	CreatedAt time.Time `gorm:"index:idx_created;not null" json:"created_at"`
}

// TableName 指定表名
func (AuditLog) TableName() string {
	return "sys_audit_logs"
}
