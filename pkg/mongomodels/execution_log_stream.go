package mongomodels

import (
	"time"
)

// ExecutionLogStream 执行日志流模型
type ExecutionLogStream struct {
	ID          string             `bson:"_id,omitempty"`     // MongoDB ObjectID
	ExecutionID string             `bson:"execution_id"`      // 执行ID
	JobID       uint               `bson:"job_id"`            // 任务ID
	JobName     string             `bson:"job_name"`          // 任务名称

	// 日志信息
	LogLevel    string             `bson:"log_level"`         // 日志级别
	Message     string             `bson:"message"`           // 日志消息
	Fields      map[string]interface{} `bson:"fields"`       // 结构化字段

	// 时间信息
	Timestamp   time.Time          `bson:"timestamp"`         // 日志时间戳
	CreatedAt   time.Time          `bson:"created_at"`
}

// CollectionName 集合名称
func (ExecutionLogStream) CollectionName() string {
	return "execution_log_streams"
}