package mongomodels

import (
	"time"
)

// ReportData 报告数据模型
type ReportData struct {
	ID          string                 `bson:"_id,omitempty"`        // MongoDB ObjectID
	ReportID    string                 `bson:"report_id"`             // 报告ID
	ReportName  string                 `bson:"report_name"`           // 报告名称
	ReportType  string                 `bson:"report_type"`           // 报告类型: daily, weekly, monthly, custom

	// 时间范围
	StartTime   time.Time              `bson:"start_time"`            // 开始时间
	EndTime     time.Time              `bson:"end_time"`              // 结束时间
	GeneratedAt time.Time              `bson:"generated_at"`          // 生成时间

	// 报告内容
	Summary     map[string]interface{} `bson:"summary"`               // 摘要统计
	Sections    []ReportSection        `bson:"sections"`              // 报告章节
	Charts      []ChartConfig          `bson:"charts"`                // 图表配置
	Metrics     map[string]interface{} `bson:"metrics"`               // 详细指标

	// 配置
	Config      map[string]interface{} `bson:"config"`                 // 报告配置

	// 元数据
	CreatedBy   string                 `bson:"created_by"`            // 创建人
	CreatedAt   time.Time              `bson:"created_at"`
	UpdatedAt   time.Time              `bson:"updated_at"`
}

// ReportSection 报告章节
type ReportSection struct {
	Title   string                 `bson:"title"`
	Content string                 `bson:"content"`
	Order   int                    `bson:"order"`
	Data    map[string]interface{} `bson:"data"`
}

// ChartConfig 图表配置
type ChartConfig struct {
	ChartType string                 `bson:"chart_type"` // line, bar, pie, table
	Title     string                 `bson:"title"`
	Config    map[string]interface{} `bson:"config"`
	Data      map[string]interface{} `bson:"data"`
}

// CollectionName 集合名称
func (ReportData) CollectionName() string {
	return "report_data"
}