package objects

import (
	"gorm.io/gorm"
	"time"
)

// SysJob 对应 sys_jobs 表
type SysJob struct {
	ID             uint   `gorm:"primarykey"`
	Name           string `gorm:"uniqueIndex;size:128"` // 任务名称
	CronExpr       string `gorm:"size:64"`
	ServiceHandler string `gorm:"size:128"`  // 关联代码里的 RegisterAuto Key
	Params         string `gorm:"type:json"` // 存储 JSON 字符串
	Status         int    `gorm:"default:1"` // 1 Enable, 0 Disable
	NextRunTime    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

func (SysJob) TableName() string {
	return "sys_jobs"
}
