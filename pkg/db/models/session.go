package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Session 会话模型
type Session struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Token     string    `gorm:"uniqueIndex;size:500;not null" json:"-"`
	ExpiresAt time.Time `gorm:"index;not null" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	User      User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// BeforeCreate GORM钩子：创建前自动生成UUID
func (s *Session) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}

// IsExpired 检查会话是否过期
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// TableName 指定表名
func (Session) TableName() string {
	return "sessions"
}