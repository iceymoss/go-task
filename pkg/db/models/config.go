package models

import (
	"time"
)

// Config 系统配置模型
type Config struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Key         string `gorm:"uniqueIndex;size:100;not null" json:"key"` // 配置键
	Value       string `gorm:"type:text" json:"value"`                   // 配置值
	Type        string `gorm:"size:20;not null" json:"type"`             // 配置类型: string, number, boolean, json
	Group       string `gorm:"index:idx_group;size:50" json:"group"`     // 配置分组
	Name        string `gorm:"size:100" json:"name"`                     // 配置名称
	Description string `gorm:"type:text" json:"description"`             // 描述

	// 验证规则
	Required   bool   `gorm:"default:false" json:"required"`         // 是否必填
	Validation string `gorm:"type:json" json:"validation,omitempty"` // 验证规则

	// 安全设置
	Sensitive bool `gorm:"default:false;index:idx_sensitive" json:"sensitive"` // 是否敏感字段
	Editable  bool `gorm:"default:true" json:"editable"`                       // 是否可编辑

	// 元数据
	UpdatedBy *uint     `json:"updated_by,omitempty"` // 更新人
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (Config) TableName() string {
	return "sys_configs"
}
