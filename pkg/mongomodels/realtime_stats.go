package mongomodels

import (
	"time"
)

// RealtimeStats 实时统计模型
type RealtimeStats struct {
	ID          string                 `bson:"_id,omitempty"`        // MongoDB ObjectID
	StatType    string                 `bson:"stat_type"`             // 统计类型: job, workflow, worker, system
	StatKey     string                 `bson:"stat_key"`              // 统计键: job_id, worker_id, etc.
	StatName    string                 `bson:"stat_name"`             // 统计名称

	// 时间窗口
	WindowStart time.Time              `bson:"window_start"`          // 窗口开始
	WindowEnd   time.Time              `bson:"window_end"`            // 窗口结束
	Granularity string                 `bson:"granularity"`           // 粒度: minute, hour

	// 统计指标
	Metrics     map[string]interface{} `bson:"metrics"`               // 动态指标
	Series      []StatSeriesEntry      `bson:"series"`                // 时间序列数据

	// 标签
	Tags        map[string]string      `bson:"tags"`                  // 标签

	// 创建/更新时间
	UpdatedAt   time.Time              `bson:"updated_at"`
	CreatedAt   time.Time              `bson:"created_at"`
}

// StatSeriesEntry 时间序列条目
type StatSeriesEntry struct {
	Timestamp time.Time              `bson:"timestamp"`
	Values    map[string]interface{} `bson:"values"`
}

// CollectionName 集合名称
func (RealtimeStats) CollectionName() string {
	return "realtime_stats"
}