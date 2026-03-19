package models

import (
	"time"
)

// TaskTemplate 任务模板模型
type TaskTemplate struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	TemplateID  string `gorm:"uniqueIndex;size:64;not null" json:"template_id"` // 模板ID
	Name        string `gorm:"not null;size:200" json:"name"`                   // 模板名称
	DisplayName string `gorm:"not null;size:200" json:"display_name"`           // 显示名称
	Description string `gorm:"type:text" json:"description"`                    // 描述
	Category    string `gorm:"index:idx_category;size:50" json:"category"`      // 分类: data_sync, data_clean, etc
	TaskType    string `gorm:"not null;size:50" json:"task_type"`               // 任务类型

	// 模板定义
	Template      string `gorm:"type:json;not null" json:"template"`        // 模板内容
	ParamsSchema  string `gorm:"type:json;not null" json:"params_schema"`   // 参数Schema
	DefaultParams string `gorm:"type:json" json:"default_params,omitempty"` // 默认参数

	// 版本信息
	Version      string `gorm:"size:20;default:'1.0.0'" json:"version"`           // 版本号
	BaseTemplate *uint  `gorm:"index:idx_base" json:"base_template_id,omitempty"` // 父模板ID

	// 访问控制
	IsPublic  bool  `gorm:"index:idx_public;default:true" json:"is_public"` // 是否公开
	CreatedBy *uint `json:"created_by,omitempty"`                           // 创建人

	// 时间戳
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName 指定表名
func (TaskTemplate) TableName() string {
	return "sys_task_templates"
}
