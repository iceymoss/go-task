package engine

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/iceymoss/go-task/pkg/logger"
	"go.uber.org/zap"
)

// RetryPolicy 重试策略
type RetryPolicy struct {
	MaxAttempts        int           // 最大重试次数
	InitialDelay       time.Duration // 初始延迟
	MaxDelay          time.Duration // 最大延迟
	BackoffMultiplier float64       // 退避乘数
	JitterEnabled      bool          // 是否启用随机抖动
	RetryableErrors   []error       // 可重试的错误列表
}

// DefaultRetryPolicy 默认重试策略
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts:        3,
		InitialDelay:       1 * time.Second,
		MaxDelay:          30 * time.Second,
		BackoffMultiplier: 2.0,
		JitterEnabled:      true,
		RetryableErrors:   nil,
	}
}

// RetryManager 重试管理器
type RetryManager struct {
	policies map[string]*RetryPolicy // 任务名 -> 重试策略
	mu       sync.RWMutex
}

// NewRetryManager 创建重试管理器
func NewRetryManager() *RetryManager {
	return &RetryManager{
		policies: make(map[string]*RetryPolicy),
	}
}

// SetPolicy 设置重试策略
func (rm *RetryManager) SetPolicy(taskName string, policy *RetryPolicy) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.policies[taskName] = policy

	logger.Info("📋 [Retry] Set retry policy",
		zap.String("task", taskName),
		zap.Int("max_attempts", policy.MaxAttempts),
		zap.Duration("initial_delay", policy.InitialDelay),
		zap.Duration("max_delay", policy.MaxDelay),
	)
}

// GetPolicy 获取重试策略
func (rm *RetryManager) GetPolicy(taskName string) (*RetryPolicy, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	policy, exists := rm.policies[taskName]
	return policy, exists
}

// ShouldRetry 判断是否应该重试
func (rm *RetryManager) ShouldRetry(taskName string, attempt int, err error) bool {
	policy, exists := rm.GetPolicy(taskName)
	if !exists {
		// 没有配置策略，使用默认策略
		policy = DefaultRetryPolicy()
	}

	if attempt >= policy.MaxAttempts {
		return false
	}

	// 如果配置了可重试的错误列表，检查错误是否在列表中
	if len(policy.RetryableErrors) > 0 {
		for _, retryableErr := range policy.RetryableErrors {
			if err == retryableErr {
				return true
			}
		}
		return false
	}

	return true
}

// CalculateDelay 计算重试延迟
func (rm *RetryManager) CalculateDelay(taskName string, attempt int) time.Duration {
	policy, exists := rm.GetPolicy(taskName)
	if !exists {
		policy = DefaultRetryPolicy()
	}

	// 指数退避
	delay := time.Duration(float64(policy.InitialDelay) * math.Pow(policy.BackoffMultiplier, float64(attempt-1)))

	// 限制最大延迟
	if delay > policy.MaxDelay {
		delay = policy.MaxDelay
	}

	// 添加随机抖动（避免惊群效应）
	if policy.JitterEnabled {
		jitter := time.Duration(rand.Float64() * float64(delay) * 0.5) // 50%的随机抖动
		delay = delay - jitter
		if delay < 0 {
			delay = 0
		}
	}

	return delay
}

// ExecuteWithRetry 带重试的执行
func (rm *RetryManager) ExecuteWithRetry(taskName string, ctx context.Context, executeFunc func(context.Context) error) error {
	var lastErr error

	for attempt := 1; ; attempt++ {
		err := executeFunc(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		// 检查是否应该重试
		if !rm.ShouldRetry(taskName, attempt, err) {
			break
		}

		// 计算延迟
		delay := rm.CalculateDelay(taskName, attempt)

		logger.Warn("🔄 [Retry] Task failed, will retry",
			zap.String("task", taskName),
			zap.Int("attempt", attempt),
			zap.Duration("delay", delay),
			zap.Error(err),
		)

		// 发射重试事件
		if eventManager := GetGlobalEventManager(); eventManager != nil {
			eventManager.Emit(&Event{
				Type:      EventTypeJobRetry,
				TaskName:  taskName,
				TimeStamp: time.Now(),
				Context:   ctx,
				Error:     err,
				Data: map[string]any{
					"attempt":     attempt,
					"retry_delay": delay,
				},
			})
		}

		// 等待
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return fmt.Errorf("task failed after %d attempts, last error: %w", rm.getMaxAttempts(taskName), lastErr)
}

// getMaxAttempts 获取最大重试次数
func (rm *RetryManager) getMaxAttempts(taskName string) int {
	policy, exists := rm.GetPolicy(taskName)
	if !exists {
		policy = DefaultRetryPolicy()
	}
	return policy.MaxAttempts
}

// ==================== 预定义的重试策略 ====================

// ExponentialBackoffPolicy 指数退避策略
func ExponentialBackoffPolicy(maxAttempts int, initialDelay time.Duration) *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts:        maxAttempts,
		InitialDelay:       initialDelay,
		MaxDelay:          30 * time.Minute,
		BackoffMultiplier: 2.0,
		JitterEnabled:      true,
	}
}

// LinearBackoffPolicy 线性退避策略
func LinearBackoffPolicy(maxAttempts int, delay time.Duration) *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts:        maxAttempts,
		InitialDelay:       delay,
		MaxDelay:          time.Duration(maxAttempts) * delay,
		BackoffMultiplier: 1.0,
		JitterEnabled:      true,
	}
}

// FixedDelayPolicy 固定延迟策略
func FixedDelayPolicy(maxAttempts int, delay time.Duration) *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts:        maxAttempts,
		InitialDelay:       delay,
		MaxDelay:          delay,
		BackoffMultiplier: 1.0,
		JitterEnabled:      false,
	}
}

// NoRetryPolicy 不重试策略
func NoRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts:        1,
		InitialDelay:       0,
		MaxDelay:          0,
		BackoffMultiplier: 1.0,
		JitterEnabled:      false,
	}
}

// ==================== 重试装饰器 ====================

// RetryWithPolicy 使用指定策略的重试装饰器
func RetryWithPolicy(retryManager *RetryManager, taskName string) JobWrapper {
	return func(next JobFunc) JobFunc {
		return func(ctx context.Context) error {
			return retryManager.ExecuteWithRetry(taskName, ctx, next)
		}
	}
}

// ==================== 全局实例 ====================

var (
	globalEventManager *EventManager
	eventManagerOnce sync.Once
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
