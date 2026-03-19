package models

import (
	"time"
)

// Workflow 工作流定义模型
type Workflow struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	WorkflowID  string `gorm:"uniqueIndex;size:64;not null" json:"workflow_id"` // 工作流ID
	Name        string `gorm:"not null;size:200" json:"name"`                   // 工作流名称
	Description string `gorm:"type:text" json:"description"`                    // 描述
	Version     string `gorm:"size:20;default:'1.0.0'" json:"version"`          // 版本号

	// 工作流定义
	DAG            string `gorm:"type:json;not null" json:"dag"`              // DAG定义(节点和边)
	GlobalParams   string `gorm:"type:json" json:"global_params,omitempty"`   // 全局参数
	ScheduleConfig string `gorm:"type:json" json:"schedule_config,omitempty"` // 调度配置

	// 失败策略
	FailureStrategy string `gorm:"size:20;default:'fail_fast'" json:"failure_strategy"` // fail_fast, continue, retry_all

	// 状态
	Enable bool   `gorm:"index:idx_enable;default:true" json:"enable"` // 是否启用
	Status string `gorm:"size:20;default:'active'" json:"status"`      // active, paused, archived

	// 元数据
	Tags      string `gorm:"type:json" json:"tags,omitempty"` // 标签
	Author    string `gorm:"size:100" json:"author"`          // 作者
	CreatedBy *uint  `json:"created_by,omitempty"`            // 创建人

	// 时间戳
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName 指定表名
func (Workflow) TableName() string {
	return "sys_workflows"
}
