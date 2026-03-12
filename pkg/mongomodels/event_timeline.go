package mongomodels

import (
	"time"
)

// EventTimeline 事件时间线模型
type EventTimeline struct {
	ID          string                 `bson:"_id,omitempty"`        // MongoDB ObjectID
	EventID     string                 `bson:"event_id"`             // 事件ID
	EventType   string                 `bson:"event_type"`           // 事件类型: job_created, job_updated, execution_started, execution_completed, alert_triggered, etc.

	// 关联信息
	ResourceType string                `bson:"resource_type"`         // 资源类型: job, workflow, execution, alert
	ResourceID   string                `bson:"resource_id"`           // 资源ID
	ResourceName string                `bson:"resource_name"`         // 资源名称

	// 事件内容
	Title       string                 `bson:"title"`                 // 事件标题
	Description string                 `bson:"description"`           // 事件描述
	Details     map[string]interface{} `bson:"details"`              // 详细信息
	Changes     map[string]interface{} `bson:"changes"`               // 变更内容

	// 上下文
	UserID      string                 `bson:"user_id"`              // 操作用户ID
	Username    string                 `bson:"username"`             // 操作用户名
	IP          string                 `bson:"ip"`                   // IP地址
	UserAgent   string                 `bson:"user_agent"`           // User-Agent
	RequestID   string                 `bson:"request_id"`           // 请求ID

	// 标签
	Tags        []string               `bson:"tags"`                 // 标签
	Severity    string                 `bson:"severity"`             // 严重程度: info, warning, error, critical

	// 时间信息
	Timestamp   time.Time              `bson:"timestamp"`            // 事件时间戳
	CreatedAt   time.Time              `bson:"created_at"`
}

// CollectionName 集合名称
func (EventTimeline) CollectionName() string {
	return "event_timelines"
}