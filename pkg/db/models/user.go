package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:50;not null" json:"username"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	Email        string    `gorm:"size:100" json:"email"`
	Role         string    `gorm:"size:20;default:'user'" json:"role"` // admin, user
	IsActive     bool      `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	Sessions     []Session `gorm:"foreignKey:UserID" json:"sessions,omitempty"`
}

// SetPassword 设置密码（自动加密）
func (u *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hashedPassword)
	return nil
}

// CheckPassword 验证密码
func (u *User) CheckPassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
}

// BeforeCreate GORM钩子：创建前
func (u *User) BeforeCreate(tx *gorm.DB) error {
	return nil
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}