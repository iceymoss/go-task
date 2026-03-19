package models

import (
	"time"
)

// Role 角色模型
type Role struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Name        string     `gorm:"uniqueIndex;size:50;not null" json:"name"` // 角色名: admin, operator, viewer
	DisplayName string     `gorm:"not null;size:100" json:"display_name"`    // 显示名称
	Description string     `gorm:"type:text" json:"description"`             // 描述
	Permissions string     `gorm:"type:json;not null" json:"permissions"`    // 权限列表: ["job:create", "job:delete"]
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName 指定表名
func (Role) TableName() string {
	return "roles"
}
