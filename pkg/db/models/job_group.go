package models

import (
	"time"
)

// JobGroup 任务分组模型
type JobGroup struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Name        string     `gorm:"uniqueIndex;size:100;not null" json:"name"`   // 分组标识
	DisplayName string     `gorm:"not null;size:200" json:"display_name"`       // 显示名称
	Description string     `gorm:"type:text" json:"description"`                // 描述
	ParentID    *uint      `gorm:"index:idx_parent" json:"parent_id,omitempty"` // 父分组ID
	Level       int        `gorm:"default:1" json:"level"`                      // 层级
	Path        string     `gorm:"size:500" json:"path"`                        // 路径: /1/3/5
	Sort        int        `gorm:"default:0" json:"sort"`                       // 排序
	Icon        string     `gorm:"size:50" json:"icon"`                         // 图标
	Color       string     `gorm:"size:20" json:"color"`                        // 颜色
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName 指定表名
func (JobGroup) TableName() string {
	return "sys_job_groups"
}
