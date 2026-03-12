package models

import (
	"time"
)

// ParamTemplate 参数模板模型
type ParamTemplate struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	Name          string     `gorm:"uniqueIndex;size:100;not null" json:"name"`             // 模板名称
	DisplayName   string     `gorm:"not null;size:200" json:"display_name"`                 // 显示名称
	Description   string     `gorm:"type:text" json:"description"`                          // 描述
	TaskType      string     `gorm:"index:idx_task_type;not null;size:50" json:"task_type"` // 适用任务类型
	ParamsSchema  string     `gorm:"type:json;not null" json:"params_schema"`               // 参数Schema定义
	DefaultParams string     `gorm:"type:json" json:"default_params"`                       // 默认参数值
	IsPublic      bool       `gorm:"default:true" json:"is_public"`                         // 是否公开
	CreatedBy     *uint      `json:"created_by,omitempty"`                                  // 创建人
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName 指定表名
func (ParamTemplate) TableName() string {
	return "sys_param_templates"
}
