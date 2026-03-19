package core

import (
	"context"

	"go.uber.org/zap"
)

// TaskMetadata 任务元数据
type TaskMetadata struct {
	// 基本信息
	Name        string `json:"name"`         // 任务名称（必须）
	DisplayName string `json:"display_name"` // 显示名称
	Description string `json:"description"`  // 描述

	// 分类信息
	Category string   `json:"category"` // 任务分类: ops, data, ai, notification, workflow
	Tags     []string `json:"tags"`     // 标签

	// 作者信息
	Author  string `json:"author"`  // 作者
	Version string `json:"version"` // 版本号

	Type string `json:"type"`

	// 参数Schema（用于UI自动生成表单）
	ParamSchema ParamSchema `json:"param_schema"`
}

// ParamSchema 参数Schema定义（参考JSON Schema）
type ParamSchema struct {
	Type        string                 `json:"type"`                 // 类型: string, integer, boolean, object, array
	Title       string                 `json:"title"`                // 字段标题
	Description string                 `json:"description"`          // 字段描述
	Default     interface{}            `json:"default"`              // 默认值
	Required    bool                   `json:"required"`             // 是否必填
	Minimum     *float64               `json:"minimum,omitempty"`    // 最小值（数字类型）
	Maximum     *float64               `json:"maximum,omitempty"`    // 最大值（数字类型）
	MinLength   *int                   `json:"minLength,omitempty"`  // 最小长度（字符串类型）
	MaxLength   *int                   `json:"maxLength,omitempty"`  // 最大长度（字符串类型）
	Pattern     string                 `json:"pattern,omitempty"`    // 正则表达式
	Enum        []interface{}          `json:"enum,omitempty"`       // 枚举值
	Properties  map[string]ParamSchema `json:"properties,omitempty"` // 子属性（对象类型）
	Items       *ParamSchema           `json:"items,omitempty"`      // 数组元素类型
}

// TaskProgress 任务进度（用于长时间运行的任务）
type TaskProgress struct {
	Current int    `json:"current"` // 当前进度
	Total   int    `json:"total"`   // 总进度
	Message string `json:"message"` // 进度消息
}

// TaskContext 任务上下文
type TaskContext struct {
	context.Context

	// 任务标识
	TaskID      string // 任务ID（来自 sys_jobs.id）
	TaskName    string // 任务名称（来自 sys_jobs.name）
	ExecutionID string // 执行ID（WUID）

	// 进度回调
	OnProgress func(TaskProgress) // 进度更新回调

	// 日志记录器
	Logger *zap.Logger // 结构化日志记录器

	// Worker信息
	WorkerID string // 执行Worker ID
	Hostname string // 主机名

	// 扩展字段
	Metadata map[string]interface{} // 元数据
}

// Tasker 增强的任务接口
type Tasker interface {
	// Run 执行任务逻辑（核心方法）
	// ctx: 任务上下文，包含TaskID、ExecutionID、Logger等
	// params: 任务参数（从配置传入）
	Run(ctx *TaskContext, params map[string]any) error

	// Identifier 返回任务唯一标识（用于日志和注册）
	Identifier() string

	// Metadata 返回任务元数据（用于UI展示和参数验证）
	Metadata() TaskMetadata

	// ValidateParams 验证参数（可选实现，默认实现使用 ParamSchema）
	ValidateParams(params map[string]any) error

	// BeforeRun 任务执行前的钩子（可选实现）
	// 返回error表示拒绝执行
	BeforeRun(ctx *TaskContext, params map[string]any) error

	// AfterRun 任务执行后的钩子（可选实现）
	// 无论成功或失败都会调用
	AfterRun(ctx *TaskContext, params map[string]any, err error) error
}

// BaseTask 基础任务实现（提供默认方法）
type BaseTask struct {
	metadata TaskMetadata
}

// Metadata 返回任务元数据
func (b *BaseTask) Metadata() TaskMetadata {
	return b.metadata
}

// ValidateParams 默认参数验证实现
// 使用 ParamSchema 进行验证
func (b *BaseTask) ValidateParams(params map[string]any) error {
	return validateParamsWithSchema(b.metadata.ParamSchema, params)
}

// BeforeRun 默认前置钩子（空实现）
func (b *BaseTask) BeforeRun(ctx *TaskContext, params map[string]any) error {
	return nil
}

// AfterRun 默认后置钩子（空实现）
func (b *BaseTask) AfterRun(ctx *TaskContext, params map[string]any, err error) error {
	return nil
}

// validateParamsWithSchema 使用Schema验证参数
func validateParamsWithSchema(schema ParamSchema, params map[string]any) error {
	// TODO: 实现完整的JSON Schema验证逻辑
	// 这里先做简单实现

	// 检查必填参数
	if schema.Required {
		_, exists := params[schema.Type]
		if !exists && len(params) == 0 {
			return &ValidationError{
				Field:   "root",
				Message: "参数不能为空",
			}
		}
	}

	// 类型检查
	for key, value := range params {
		switch schema.Type {
		case "string":
			if _, ok := value.(string); !ok {
				return &ValidationError{
					Field:   key,
					Message: "参数必须是字符串类型",
				}
			}
		case "integer":
			if _, ok := value.(int); !ok {
				if _, ok := value.(float64); !ok {
					return &ValidationError{
						Field:   key,
						Message: "参数必须是整数类型",
					}
				}
			}
		case "boolean":
			if _, ok := value.(bool); !ok {
				return &ValidationError{
					Field:   key,
					Message: "参数必须是布尔类型",
				}
			}
		case "object":
			if _, ok := value.(map[string]interface{}); !ok {
				return &ValidationError{
					Field:   key,
					Message: "参数必须是对象类型",
				}
			}
		case "array":
			if _, ok := value.([]interface{}); !ok {
				return &ValidationError{
					Field:   key,
					Message: "参数必须是数组类型",
				}
			}
		}
	}

	return nil
}

// ValidationError 参数验证错误
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return "参数验证失败 [" + e.Field + "]: " + e.Message
}

// CompositeTask 组合任务（工作流任务的包装器）
type CompositeTask struct {
	BaseTask
	workflowID   string         // 工作流ID
	dag          interface{}    // DAG定义
	globalParams map[string]any // 全局参数
}

// NewCompositeTask 创建组合任务
func NewCompositeTask(workflowID string, dag interface{}, globalParams map[string]any) *CompositeTask {
	return &CompositeTask{
		BaseTask: BaseTask{
			metadata: TaskMetadata{
				Name:     "workflow_" + workflowID,
				Type:     "composite",
				Category: "workflow",
				ParamSchema: ParamSchema{
					Type:  "object",
					Title: "组合任务参数",
				},
			},
		},
		workflowID:   workflowID,
		dag:          dag,
		globalParams: globalParams,
	}
}

// Identifier 返回标识
func (c *CompositeTask) Identifier() string {
	return "workflow:" + c.workflowID
}

// Run 执行工作流
func (c *CompositeTask) Run(ctx *TaskContext, params map[string]any) error {
	// TODO: 调用工作流执行引擎
	ctx.Logger.Info("执行组合任务",
		zap.String("workflow_id", c.workflowID),
		zap.Any("params", params),
	)
	return nil
}
