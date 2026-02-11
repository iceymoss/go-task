package objects

import "time"

// SysJobLog 对应 sys_job_logs 表
type SysJobLog struct {
	ID          uint   `gorm:"primarykey"`
	JobName     string `gorm:"index;size:128"`
	HandlerName string `gorm:"size:128"`
	Status      int    // 0 Running, 1 Success, 2 Failed
	ErrorMsg    string `gorm:"type:text"`
	DurationMs  int64
	StartTime   time.Time
	EndTime     *time.Time
}

func (s SysJobLog) TableName() string {
	return "sys_job_logs"
}
