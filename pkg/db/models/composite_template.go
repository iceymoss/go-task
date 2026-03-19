package models

import (
	"time"
)

// CompositeTemplate 复合模板模型（组合多个模板）
type CompositeTemplate struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	TemplateID  string `gorm:"uniqueIndex;size:64;not null" json:"template_id"` // 模板ID
	Name        string `gorm:"not null;size:200" json:"name"`                   // 模板名称
	DisplayName string `gorm:"not null;size:200" json:"display_name"`           // 显示名称
	Description string `gorm:"type:text" json:"description"`                    // 描述

	// 模板组合
	Components   string `gorm:"type:json;not null" json:"components"`    // 组件列表: [{"type": "task", "id": "xxx"}, {"type": "workflow", "id": "yyy"}]
	Dependencies string `gorm:"type:json" json:"dependencies,omitempty"` // 依赖关系

	// 参数配置
	GlobalParams  string `gorm:"type:json" json:"global_params,omitempty"`  // 全局参数
	ParamsSchema  string `gorm:"type:json" json:"params_schema,omitempty"`  // 参数Schema
	DefaultParams string `gorm:"type:json" json:"default_params,omitempty"` // 默认参数

	// 版本信息
	Version string `gorm:"size:20;default:'1.0.0'" json:"version"` // 版本号

	// 访问控制
	IsPublic  bool  `gorm:"index:idx_public;default:true" json:"is_public"` // 是否公开
	CreatedBy *uint `json:"created_by,omitempty"`                           // 创建人

	// 时间戳
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName 指定表名
func (CompositeTemplate) TableName() string {
	return "sys_composite_templates"
}
