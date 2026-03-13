package engine

import (
	"context"
	"time"

	"github.com/iceymoss/go-task/pkg/db"
	"github.com/iceymoss/go-task/pkg/db/objects"
	"github.com/iceymoss/go-task/pkg/logger"

	"go.uber.org/zap"
)

// GormHistoryStorage 基于 GORM 的任务历史存储实现
type GormHistoryStorage struct {
}

// NewGormHistoryStorage 创建历史存储
func NewGormHistoryStorage() *GormHistoryStorage {
	// 自动迁移表结构（只在首次运行时）
	conn := db.GetMysqlConn(db.MYSQL_DB_GO_TASK)
	if err := conn.AutoMigrate(&objects.SysJobLog{}); err != nil {
		logger.Error("❌ [History] AutoMigrate failed", zap.Error(err))
	}
	return &GormHistoryStorage{}
}

// SaveEvent 根据事件持久化任务历史
func (g *GormHistoryStorage) SaveEvent(event *Event) error {
	// 只处理完成或失败的事件
	if event.Type != EventTypeAfterJob && event.Type != EventTypeJobError {
		return nil
	}

	conn := db.GetMysqlConn(db.MYSQL_DB_GO_TASK)

	status := 1 // Success
	if event.Type == EventTypeJobError {
		status = 2
	}

	var durationMs int64
	var startTime time.Time
	var endTime = event.TimeStamp

	if event.Data != nil {
		if v, ok := event.Data["duration_ms"].(int64); ok {
			durationMs = v
		}
		if v, ok := event.Data["start_time"].(time.Time); ok {
			startTime = v
		}
	}

	if startTime.IsZero() {
		// 如果没有提供开始时间，则使用结束时间减去 duration 估算
		if durationMs > 0 {
			startTime = endTime.Add(-time.Duration(durationMs) * time.Millisecond)
		} else {
			startTime = endTime
		}
	}

	logEntry := &objects.SysJobLog{
		JobName:     event.TaskName,
		HandlerName: event.TaskName,
		Status:      status,
		ErrorMsg:    "",
		DurationMs:  durationMs,
		StartTime:   startTime,
		EndTime:     &endTime,
	}
	if event.Error != nil {
		logEntry.ErrorMsg = event.Error.Error()
	}

	if err := conn.WithContext(context.Background()).Create(logEntry).Error; err != nil {
		return err
	}
	return nil
}
