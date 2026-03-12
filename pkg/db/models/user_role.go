package models

import (
	"time"
)

// UserRole 用户角色关联模型
type UserRole struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index:idx_user_id;not null" json:"user_id"`
	RoleID    uint      `gorm:"index:idx_role_id;not null" json:"role_id"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (UserRole) TableName() string {
	return "user_roles"
}
