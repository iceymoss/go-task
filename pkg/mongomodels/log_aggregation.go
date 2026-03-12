package mongomodels

import (
	"time"
)

// LogAggregation 日志聚合模型
type LogAggregation struct {
	ID          string                 `bson:"_id,omitempty"`        // MongoDB ObjectID
	JobID       uint                   `bson:"job_id"`                // 任务ID
	JobName     string                 `bson:"job_name"`              // 任务名称
	ExecutionID string                 `bson:"execution_id"`          // 执行ID

	// 聚合维度
	WindowStart time.Time              `bson:"window_start"`          // 窗口开始时间
	WindowEnd   time.Time              `bson:"window_end"`            // 窗口结束时间
	Granularity string                 `bson:"granularity"`           // 粒度: minute, hour, day

	// 日志统计
	TotalLogs   int64                  `bson:"total_logs"`            // 总日志数
	LogLevels   map[string]int64       `bson:"log_levels"`            // 各级别日志数
	ErrorCount  int64                  `bson:"error_count"`           // 错误日志数
	WarnCount   int64                  `bson:"warn_count"`            // 警告日志数

	// 采样日志
	SampleLogs  []SampleLogEntry       `bson:"sample_logs"`           // 采样日志

	// 创建时间
	CreatedAt   time.Time              `bson:"created_at"`
}

// SampleLogEntry 采样日志条目
type SampleLogEntry struct {
	Timestamp time.Time `bson:"timestamp"`
	Level     string    `bson:"level"`
	Message   string    `bson:"message"`
}

// CollectionName 集合名称
func (LogAggregation) CollectionName() string {
	return "log_aggregations"
}