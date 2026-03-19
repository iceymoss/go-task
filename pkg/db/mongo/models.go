package mongo

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ============================================
// 1. 任务执行日志集合
// ============================================

// JobExecutionLog 任务执行日志
type JobExecutionLog struct {
	ID          primitive.ObjectID `bson:"_id"`
	ExecutionID string            `bson:"execution_id"`
	JobID       uint              `bson:"job_id"`
	JobName     string            `bson:"job_name"`
	JobType     string            `bson:"job_type"`
	WorkerID    string            `bson:"worker_id"`

	// 执行状态
	Status string `bson:"status"` // success, failed, timeout, cancelled

	// 时间信息
	ScheduledAt time.Time  `bson:"scheduled_at"`
	StartedAt   *time.Time `bson:"started_at,omitempty"`
	FinishedAt  *time.Time `bson:"finished_at,omitempty"`
	DurationMs  int64      `bson:"duration_ms,omitempty"`

	// 重试信息
	RetryCount    int     `bson:"retry_count"`
	RetryHistory  []Retry `bson:"retry_history,omitempty"`

	// 执行日志（核心）
	Logs []LogEntry `bson:"logs"`

	// 输出
	Stdout string `bson:"stdout,omitempty"`
	Stderr string `bson:"stderr,omitempty"`

	// 进度信息
	Progress *Progress `bson:"progress,omitempty"`

	// 执行环境
	Environment *Environment `bson:"environment,omitempty"`

	// 标签和元数据
	Tags    []string                 `bson:"tags,omitempty"`
	Metadata map[string]interface{}    `bson:"metadata,omitempty"`

	// 时间戳
	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
}

// Retry 重试记录
type Retry struct {
	Attempt     int       `bson:"attempt"`
	Reason      string    `bson:"reason"`
	StartedAt   time.Time `bson:"started_at"`
	FinishedAt  time.Time `bson:"finished_at"`
	Error       string    `bson:"error,omitempty"`
}

// LogEntry 日志条目
type LogEntry struct {
	Timestamp time.Time              `bson:"timestamp"`
	Level     string                 `bson:"level"` // debug, info, warning, error
	Message   string                 `bson:"message"`
	Fields    map[string]interface{} `bson:"fields,omitempty"`
}

// Progress 进度信息
type Progress struct {
	Current int    `bson:"current"`
	Total   int    `bson:"total"`
	Message string `bson:"message"`
}

// Environment 执行环境
type Environment struct {
	Hostname      string `bson:"hostname"`
	OS            string `bson:"os"`
	GoVersion     string `bson:"go_version"`
	WorkerVersion string `bson:"worker_version"`
	CPUCount      int    `bson:"cpu_count"`
	MemoryMB      int    `bson:"memory_mb"`
}

// ============================================
// 2. 系统事件集合
// ============================================

// SystemEvent 系统事件
type SystemEvent struct {
	ID          primitive.ObjectID `bson:"_id"`
	EventType   string            `bson:"event_type"`   // job_started, job_completed, job_failed, job_retry, alert_sent, worker_joined, worker_left
	EventLevel  string            `bson:"event_level"`  // debug, info, warning, error, critical

	// 关联信息
	JobID       *uint    `bson:"job_id,omitempty"`
	JobName     *string  `bson:"job_name,omitempty"`
	ExecutionID *string  `bson:"execution_id,omitempty"`
	WorkerID    *string  `bson:"worker_id,omitempty"`

	// 事件数据
	Data        map[string]interface{} `bson:"data,omitempty"`

	// 上下文
	Context     map[string]interface{} `bson:"context,omitempty"`

	// 时间戳
	Timestamp   time.Time `bson:"timestamp"`
	CreatedAt   time.Time `bson:"created_at"`
}

// ============================================
// 3. 性能指标集合
// ============================================

// PerformanceMetric 性能指标
type PerformanceMetric struct {
	ID          primitive.ObjectID `bson:"_id"`
	MetricType  string            `bson:"metric_type"`  // task_duration, task_success_rate, worker_load, queue_length, system_cpu, system_memory
	MetricName  string            `bson:"metric_name"`

	// 指标值
	Value       float64 `bson:"value"`
	Unit        string  `bson:"unit"`

	// 维度
	Dimensions  map[string]interface{} `bson:"dimensions"`

	// 聚合信息
	Aggregation *Aggregation `bson:"aggregation,omitempty"`

	// 时间戳
	Timestamp   time.Time `bson:"timestamp"`
	CreatedAt   time.Time `bson:"created_at"`
}

// Aggregation 聚合信息
type Aggregation struct {
	Avg    float64 `bson:"avg"`
	Min    float64 `bson:"min"`
	Max    float64 `bson:"max"`
	P50    float64 `bson:"p50"`
	P95    float64 `bson:"p95"`
	P99    float64 `bson:"p99"`
	Count  int     `bson:"count"`
	Sum    float64 `bson:"sum"`
}

// ============================================
// 4. 工作流快照集合
// ============================================

// WorkflowSnapshot 工作流快照
type WorkflowSnapshot struct {
	ID          primitive.ObjectID `bson:"_id"`
	WorkflowID  string            `bson:"workflow_id"`
	WorkflowName string            `bson:"workflow_name"`
	ExecutionID string            `bson:"execution_id"`

	// DAG 快照
	DAG interface{} `bson:"dag"` // 完整的DAG结构

	// 节点执行状态快照
	NodeSnapshots []NodeSnapshot `bson:"node_snapshots,omitempty"`

	// 可视化数据
	Visualization *Visualization `bson:"visualization,omitempty"`

	// 时间戳
	CreatedAt time.Time `bson:"created_at"`
}

// NodeSnapshot 节点快照
type NodeSnapshot struct {
	NodeID      string     `bson:"node_id"`
	Status      string     `bson:"status"`
	StartedAt   *time.Time `bson:"started_at,omitempty"`
	FinishedAt  *time.Time `bson:"finished_at,omitempty"`
	DurationMs  int64      `bson:"duration_ms,omitempty"`
	Error       string     `bson:"error,omitempty"`
}

// Visualization 可视化数据
type Visualization struct {
	Layout    string                     `bson:"layout"` // lr, tb, radial
	Positions map[string]Position         `bson:"positions,omitempty"`
}

// Position 位置
type Position struct {
	X int `bson:"x"`
	Y int `bson:"y"`
}

// ============================================
// 5. 日志归档集合
// ============================================

// LogArchive 日志归档
type LogArchive struct {
	ID           primitive.ObjectID `bson:"_id"`
	ArchiveID    string            `bson:"archive_id"`
	ArchiveDate  string            `bson:"archive_date"` // YYYY-MM-DD
	JobIDs       []uint            `bson:"job_ids,omitempty"`

	// 统计信息
	Stats *ArchiveStats `bson:"stats,omitempty"`

	// 归档文件
	Files []ArchiveFile `bson:"files"`

	// 时间戳
	CreatedAt time.Time `bson:"created_at"`
}

// ArchiveStats 归档统计
type ArchiveStats struct {
	TotalExecutions int     `bson:"total_executions"`
	SuccessCount    int     `bson:"success_count"`
	FailedCount     int     `bson:"failed_count"`
	TotalDurationMs int64   `bson:"total_duration_ms"`
}

// ArchiveFile 归档文件
type ArchiveFile struct {
	Type     string `bson:"type"` // logs, metrics
	Path     string `bson:"path"`
	Size     int64  `bson:"size"`
	Checksum string `bson:"checksum"`
}

// ============================================
// 集合名称常量
// ============================================

const (
	CollectionJobExecutionLogs   = "job_execution_logs"
	CollectionSystemEvents     = "system_events"
	CollectionPerformanceMetrics = "performance_metrics"
	CollectionWorkflowSnapshots = "workflow_snapshots"
	CollectionLogArchives       = "log_archives"
)

// ============================================
// 索引定义
// ============================================

// JobExecutionLogIndexes 执行日志索引
var JobExecutionLogIndexes = []Index{
	{
		Keys:    map[string]int{"execution_id": 1},
		Unique:  true,
	},
	{
		Keys:    map[string]int{"job_id": -1, "created_at": -1},
	},
	{
		Keys:    map[string]int{"status": -1, "created_at": -1},
	},
	{
		Keys:    map[string]int{"started_at": -1},
	},
	{
		Keys:    map[string]int{"logs.timestamp": 1},
	},
	{
		Keys:    map[string]int{"tags": 1},
	},
	{
		Keys:    map[string]int{"metadata.trace_id": 1},
	},
}

// SystemEventIndexes 系统事件索引
var SystemEventIndexes = []Index{
	{
		Keys: map[string]int{"event_type": 1, "timestamp": -1},
	},
	{
		Keys: map[string]int{"event_level": 1, "timestamp": -1},
	},
	{
		Keys: map[string]int{"job_id": 1, "timestamp": -1},
	},
	{
		Keys: map[string]int{"execution_id": 1},
	},
	{
		Keys: map[string]int{"timestamp": -1},
	},
}

// PerformanceMetricIndexes 性能指标索引
var PerformanceMetricIndexes = []Index{
	{
		Keys: map[string]int{"metric_type": 1, "metric_name": 1, "timestamp": -1},
	},
	{
		Keys: map[string]int{"dimensions.job_id": 1, "timestamp": -1},
	},
	{
		Keys: map[string]int{"timestamp": -1},
	},
}

// WorkflowSnapshotIndexes 工作流快照索引
var WorkflowSnapshotIndexes = []Index{
	{
		Keys:   map[string]int{"workflow_id": 1, "execution_id": 1},
		Unique: true,
	},
	{
		Keys: map[string]int{"execution_id": 1},
	},
}

// LogArchiveIndexes 日志归档索引
var LogArchiveIndexes = []Index{
	{
		Keys: map[string]int{"archive_date": 1},
	},
}

// Index 索引定义
type Index struct {
	Keys    map[string]int
	Unique  bool
	Options map[string]interface{}
}