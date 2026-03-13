package engine

import (
	"context"
	"sync"
	"time"

	"github.com/iceymoss/go-task/pkg/logger"

	"go.uber.org/zap"
)

// ==================== 全局事件实例 ====================

var (
	globalEventManager *EventManager
	eventManagerOnce   sync.Once
)

// SetGlobalEventManager 设置全局事件管理器
func SetGlobalEventManager(em *EventManager) {
	eventManagerOnce.Do(func() {
		globalEventManager = em
	})
}

// GetGlobalEventManager 获取全局事件管理器
func GetGlobalEventManager() *EventManager {
	return globalEventManager
}

// EventType 事件类型
type EventType string

const (
	EventTypeBeforeJob     EventType = "before_job"     // 任务开始前
	EventTypeAfterJob      EventType = "after_job"      // 任务完成后
	EventTypeJobError      EventType = "job_error"      // 任务出错
	EventTypeJobPanic      EventType = "job_panic"      // 任务panic
	EventTypeJobSkipped    EventType = "job_skipped"    // 任务被跳过
	EventTypeJobRetry      EventType = "job_retry"      // 任务重试
	EventTypeDependencyMet EventType = "dependency_met" // 依赖满足
)

// Event 任务事件
type Event struct {
	Type      EventType       // 事件类型
	TaskName  string          // 任务名称
	TimeStamp time.Time       // 时间戳
	Context   context.Context // 上下文
	Error     error           // 错误信息
	Data      map[string]any  // 附加数据
}

// EventHandler 事件处理器接口
type EventHandler interface {
	Handle(event *Event)
}

// EventHandlerFunc 函数类型的事件处理器
type EventHandlerFunc func(event *Event)

func (f EventHandlerFunc) Handle(event *Event) {
	f(event)
}

// EventManager 事件管理器
type EventManager struct {
	handlers map[EventType][]EventHandler // 事件类型 -> 处理器列表
	mu       sync.RWMutex                 // 保护 handlers 的并发访问
}

// NewEventManager 创建事件管理器
func NewEventManager() *EventManager {
	return &EventManager{
		handlers: make(map[EventType][]EventHandler),
	}
}

// On 注册事件处理器
func (em *EventManager) On(eventType EventType, handler EventHandler) {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.handlers[eventType] = append(em.handlers[eventType], handler)
	logger.Info("📡 [EventManager] Registered event handler",
		zap.String("event_type", string(eventType)),
	)
}

// OnFunc 注册函数类型的事件处理器
func (em *EventManager) OnFunc(eventType EventType, handlerFunc func(event *Event)) {
	em.On(eventType, EventHandlerFunc(handlerFunc))
}

// Emit 发射事件
func (em *EventManager) Emit(event *Event) {
	em.mu.RLock()
	handlers := em.handlers[event.Type]
	em.mu.RUnlock()

	// 在新的goroutine中执行所有处理器，避免阻塞
	for _, handler := range handlers {
		go func(h EventHandler) {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("❌ [EventManager] Panic in event handler",
						zap.Any("panic", r),
						zap.String("event_type", string(event.Type)),
						zap.String("task_name", event.TaskName),
					)
				}
			}()
			h.Handle(event)
		}(handler)
	}

	logger.Debug("📡 [EventManager] Emitted event",
		zap.String("event_type", string(event.Type)),
		zap.String("task_name", event.TaskName),
	)
}

// Remove 移除指定事件类型的所有处理器
func (em *EventManager) Remove(eventType EventType) {
	em.mu.Lock()
	defer em.mu.Unlock()

	delete(em.handlers, eventType)
}

// ==================== 预定义的事件处理器 ====================

// LoggingEventHandler 记录事件日志
func LoggingEventHandler() EventHandlerFunc {
	return func(event *Event) {
		fields := []zap.Field{
			zap.String("event_type", string(event.Type)),
			zap.String("task_name", event.TaskName),
			zap.Time("timestamp", event.TimeStamp),
		}

		if event.Error != nil {
			fields = append(fields, zap.Error(event.Error))
		}

		switch event.Type {
		case EventTypeBeforeJob:
			logger.Info("🚀 [Event] Job starting", fields...)
		case EventTypeAfterJob:
			logger.Info("✅ [Event] Job completed", fields...)
		case EventTypeJobError:
			logger.Error("❌ [Event] Job failed", fields...)
		case EventTypeJobPanic:
			logger.Error("💥 [Event] Job panicked", fields...)
		case EventTypeJobSkipped:
			logger.Info("⏭️ [Event] Job skipped", fields...)
		case EventTypeJobRetry:
			logger.Warn("🔄 [Event] Job retrying", fields...)
		case EventTypeDependencyMet:
			logger.Info("✅ [Event] Dependencies met", fields...)
		}
	}
}

// MetricsEventHandler 记录事件指标（用于后续集成Prometheus）
func MetricsEventHandler() EventHandlerFunc {
	return func(event *Event) {
		// TODO: 这里可以集成Prometheus指标
		// switch event.Type {
		// case EventTypeBeforeJob:
		//     taskStarted.WithLabelValues(event.TaskName).Inc()
		// case EventTypeAfterJob:
		//     taskCompleted.WithLabelValues(event.TaskName).Inc()
		// case EventTypeJobError:
		//     taskFailed.WithLabelValues(event.TaskName).Inc()
		// }

		logger.Debug("📊 [Event] Metrics recorded",
			zap.String("event_type", string(event.Type)),
			zap.String("task_name", event.TaskName),
		)
	}
}

// AlertConfig 发送告警通知
type AlertConfig struct {
	Enabled    bool
	OnErrors   bool
	OnRetries  bool
	MaxRetries int
}

func NewAlertEventHandler(config AlertConfig) EventHandlerFunc {
	return func(event *Event) {
		if !config.Enabled {
			return
		}

		shouldAlert := false
		var reason string

		switch event.Type {
		case EventTypeJobError:
			if config.OnErrors {
				shouldAlert = true
				reason = "job failed"
			}
		case EventTypeJobPanic:
			if config.OnErrors {
				shouldAlert = true
				reason = "job panicked"
			}
		case EventTypeJobRetry:
			if config.OnRetries {
				// 检查重试次数
				if retryCount, ok := event.Data["retry_count"].(int); ok {
					if retryCount >= config.MaxRetries {
						shouldAlert = true
						reason = "job exceeded max retries"
					}
				}
			}
		}

		if shouldAlert {
			logger.Warn("🚨 [Event] Alert triggered",
				zap.String("task_name", event.TaskName),
				zap.String("reason", reason),
				zap.Time("timestamp", event.TimeStamp),
				zap.Error(event.Error),
			)

			// TODO: 这里可以集成实际的告警系统
			// - 发送邮件
			// - 发送短信
			// - 发送到Slack/DingTalk/企业微信
			// - 集成PagerDuty
		}
	}
}

// HistoryStorage 记录任务历史
type HistoryStorage interface {
	SaveEvent(event *Event) error
}

func NewHistoryEventHandler(storage HistoryStorage) EventHandlerFunc {
	return func(event *Event) {
		if err := storage.SaveEvent(event); err != nil {
			logger.Error("❌ [Event] Failed to save event to history",
				zap.Error(err),
				zap.String("event_type", string(event.Type)),
				zap.String("task_name", event.TaskName),
			)
		}
	}
}

// WebhookConfig 发送Webhook通知
type WebhookConfig struct {
	Enabled bool
	URLs    []string
	Secret  string
}

func NewWebhookEventHandler(config WebhookConfig) EventHandlerFunc {
	return func(event *Event) {
		if !config.Enabled || len(config.URLs) == 0 {
			return
		}

		// TODO: 实现Webhook发送逻辑
		// - 准备payload
		// - 计算签名（如果有secret）
		// - 发送HTTP POST请求到所有URL
		// - 处理重试

		logger.Debug("🌐 [Event] Webhook would be sent",
			zap.String("event_type", string(event.Type)),
			zap.String("task_name", event.TaskName),
			zap.Int("url_count", len(config.URLs)),
		)
	}
}

// DependencyEventHandler 依赖事件处理器
func DependencyEventHandler(dependencyManager *DependencyManager) EventHandlerFunc {
	return func(event *Event) {
		if event.Type == EventTypeAfterJob || event.Type == EventTypeJobError {
			// 更新依赖状态
			success := event.Error == nil
			dependencyManager.UpdateTaskStatus(event.TaskName, success, event.Error)

			// 检查是否有等待此任务的任务
			dependents := dependencyManager.GetDependentTasks(event.TaskName)
			for _, dep := range dependents {
				satisfied, _ := dependencyManager.CheckDependencies(dep)
				if satisfied {
					// 发送依赖满足事件
					dependencyEvent := &Event{
						Type:      EventTypeDependencyMet,
						TaskName:  dep,
						TimeStamp: time.Now(),
						Data: map[string]any{
							"dependency_name":    event.TaskName,
							"dependency_success": success,
						},
					}

					// 通过全局 EventManager 发射事件
					if em := GetGlobalEventManager(); em != nil {
						em.Emit(dependencyEvent)
					}

					logger.Info("✅ [Event] Dependency satisfied",
						zap.String("task", dep),
						zap.String("dependency", event.TaskName),
					)
				}
			}
		}
	}
}
