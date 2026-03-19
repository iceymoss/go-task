package models

import (
	"time"
)

// JobVersion 任务版本模型
type JobVersion struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	JobID     uint      `gorm:"index:idx_job_id;not null" json:"job_id"` // 任务ID
	Version   string    `gorm:"size:20;not null" json:"version"`         // 版本号
	Config    string    `gorm:"type:json;not null" json:"config"`        // 任务配置快照
	ChangeLog string    `gorm:"type:text" json:"change_log"`             // 变更说明
	CreatedBy *uint     `json:"created_by,omitempty"`                    // 创建人
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (JobVersion) TableName() string {
	return "sys_job_versions"
}
