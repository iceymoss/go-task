package models

import (
	"time"
)

// Job 任务模型
type Job struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"uniqueIndex;not null;size:100"` // 任务唯一标识
	DisplayName string `gorm:"not null;size:200"`             // 显示名称
	Type        string `gorm:"not null;size:50"`              // 任务类型: shell, http, email, sql, custom
	CronExpr    string `gorm:"not null;size:100"`             // Cron 表达式
	Enable      bool   `gorm:"default:true"`                  // 是否启用
	Source      string `gorm:"default:'web';size:20"`         // 来源: system, yaml, web

	// 任务参数（JSON 格式存储）
	Params string `gorm:"type:text"` // JSON 字符串存储参数

	// 依赖关系
	Dependencies string `gorm:"type:text"` // JSON 数组: ["task1", "task2"]

	// 优先级和配置
	Priority   int `gorm:"default:0"`    // 优先级
	Timeout    int `gorm:"default:3600"` // 超时时间(秒)
	MaxRetries int `gorm:"default:3"`    // 最大重试次数

	// 模板相关
	IsTemplate bool  `gorm:"default:false"` // 是否为模板
	TemplateID *uint // 模板ID

	// 元数据
	Description string `gorm:"type:text"` // 任务描述
	Tags        string `gorm:"type:text"` // 标签（JSON 数组）

	// 时间戳
	CreatedAt time.Time
	UpdatedAt time.Time
	LastRunAt *time.Time
	DeletedAt *time.Time `gorm:"index"`
}

// TableName 指定表名
func (Job) TableName() string {
	return "sys_jobs"
}
