package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/iceymoss/go-task/internal/engine"
	"github.com/iceymoss/go-task/pkg/db"
	"github.com/iceymoss/go-task/pkg/db/models"
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
	if err := conn.AutoMigrate(&models.JobLog{}); err != nil {
		logger.Error("❌ [History] AutoMigrate failed", zap.Error(err))
	}
	return &GormHistoryStorage{}
}

// SaveEvent 根据事件持久化任务历史
func (g *GormHistoryStorage) SaveEvent(event *engine.Event) error {
	// 只处理完成或失败的事件
	if event.Type != engine.EventTypeAfterJob && event.Type != engine.EventTypeJobError {
		return nil
	}

	conn := db.GetMysqlConn(db.MYSQL_DB_GO_TASK)

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

	msg, err := json.Marshal(event.Data)
	if err != nil {
		return err
	}

	logEntry := &models.JobLog{
		ExecutionID: fmt.Sprintf("%s-%d", event.TaskName, event.TimeStamp.Unix()),
		JobID:       uint(event.TimeStamp.Unix()),
		JobName:     event.TaskName,
		LogLevel:    string(event.Type),
		Message:     string(msg),
		Timestamp:   event.TimeStamp,
		CreatedAt:   time.Now(),
	}

	if event.Error != nil {
		errMsg, err := json.Marshal(map[string]string{"error": event.Error.Error()})
		if err != nil {
			return err
		}
		errMsgStr := string(errMsg)
		logEntry.Fields = &errMsgStr
	}

	if err := conn.WithContext(context.Background()).Create(logEntry).Error; err != nil {
		return err
	}
	return nil
}
