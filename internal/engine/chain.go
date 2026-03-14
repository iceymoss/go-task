package engine

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"sync/atomic"
	"time"

	"github.com/iceymoss/go-task/pkg/logger"

	"go.uber.org/zap"
)

// JobFunc 任务函数类型
type JobFunc func(ctx context.Context) error

// JobWrapper 任务包装器类型
type JobWrapper func(JobFunc) JobFunc

// Chain 任务链，用于组合多个JobWrapper
type Chain []JobWrapper

// Then 添加包装器到链中
func (c Chain) Then(wrappers ...JobWrapper) Chain {
	return append(c, wrappers...)
}

// Apply 应用所有包装器到任务函数
func (c Chain) Apply(job JobFunc) JobFunc {
	// Chain中每一个元素都是一个返回JobWrapper类型的函数
	// 所以这里就是将job这个函数增加chain中每一个函数的能力
	for i := len(c) - 1; i >= 0; i-- {
		job = c[i](job)
	}
	return job
}

// ==================== 内置包装器 ====================

// Recover 恢复panic，记录日志
func Recover(logger *zap.Logger) JobWrapper {
	return func(next JobFunc) JobFunc {
		return func(ctx context.Context) error {
			defer func() {
				if r := recover(); r != nil {
					stack := debug.Stack()
					logger.Error("❌ [JobWrapper] Panic recovered",
						zap.Any("panic", r),
						zap.String("stack", string(stack)),
					)
				}
			}()
			return next(ctx)
		}
	}
}

// DelayIfStillRunning 如果任务正在运行，则延迟执行
func DelayIfStillRunning(logger *zap.Logger) JobWrapper {
	var running int32
	return func(next JobFunc) JobFunc {
		return func(ctx context.Context) error {
			if !atomic.CompareAndSwapInt32(&running, 0, 1) {
				logger.Info("⏳ [JobWrapper] Job is still running, delaying...")
				return fmt.Errorf("job is still running")
			}
			defer atomic.StoreInt32(&running, 0)
			return next(ctx)
		}
	}
}

// SkipIfStillRunning 如果任务正在运行，则跳过执行
func SkipIfStillRunning(logger *zap.Logger) JobWrapper {
	var running int32
	return func(next JobFunc) JobFunc {
		return func(ctx context.Context) error {
			if !atomic.CompareAndSwapInt32(&running, 0, 1) {
				logger.Info("⏭️ [JobWrapper] Job is still running, skipping...")
				return nil
			}
			defer atomic.StoreInt32(&running, 0)
			return next(ctx)
		}
	}
}

// Timeout 设置任务超时时间的包装器
func Timeout(timeout time.Duration) JobWrapper {
	return func(next JobFunc) JobFunc {
		return func(ctx context.Context) error {
			// 基于父 ctx 派生出一个带有超时的 context
			timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
			// defer cancel() 极其重要：无论任务是成功完成还是超时，都要及时释放 Context 占用的底层资源
			defer cancel()

			// 创建一个接收错误的通道
			// 容量必须设为 1
			// 如果设为无缓冲通道 (make(chan error))，当触发超时退出后，如果实际任务在未来的某一天跑完了，
			// 它往 done 里写数据时会因为没有人接收而永远阻塞，导致 Goroutine 永久泄漏！
			done := make(chan error, 1)

			// 开启一个后台协程去真正执行任务
			go func() {
				// 防止子协程崩溃带走整个系统
				defer func() {
					if r := recover(); r != nil {
						done <- fmt.Errorf("panic in timeout goroutine: %v\n%s", r, debug.Stack())
					}
				}()
				done <- next(timeoutCtx)
			}()

			// 使用 select 监听谁先到来
			select {
			case err := <-done:
				// 任务在超时时间内顺利（或报错）跑完了
				return err
			case <-timeoutCtx.Done():
				// 时间到了，任务还没跑完。timeoutCtx.Done() 的通道会收到关闭信号
				return fmt.Errorf("job timed out after %v", timeout)
			}
		}
	}
}

// Logging 记录任务执行日志
func Logging(taskName string) JobWrapper {
	return func(next JobFunc) JobFunc {
		return func(ctx context.Context) error {
			startTime := time.Now()
			logger.Info("🚀 [JobWrapper] Job started",
				zap.String("task", taskName),
				zap.Time("start_time", startTime),
			)

			err := next(ctx)

			duration := time.Since(startTime)
			if err != nil {
				logger.Error("❌ [JobWrapper] Job failed",
					zap.String("task", taskName),
					zap.Duration("duration", duration),
					zap.Error(err),
				)
			} else {
				logger.Info("✅ [JobWrapper] Job completed successfully",
					zap.String("task", taskName),
					zap.Duration("duration", duration),
				)
			}

			return err
		}
	}
}

// Metrics 记录任务执行指标（用于后续集成Prometheus）
func Metrics(taskName string) JobWrapper {
	return func(next JobFunc) JobFunc {
		return func(ctx context.Context) error {
			startTime := time.Now()

			err := next(ctx)

			duration := time.Since(startTime)

			// TODO: 这里可以集成Prometheus指标
			// taskDuration.WithLabelValues(taskName).Observe(duration.Seconds())
			// if err != nil {
			//     taskErrors.WithLabelValues(taskName, err.Error()).Inc()
			// }

			// 暂时记录到日志
			if err != nil {
				log.Printf("📊 [Metrics] Task %s failed after %v: %v", taskName, duration, err)
			} else {
				log.Printf("📊 [Metrics] Task %s completed in %v", taskName, duration)
			}

			return err
		}
	}
}

// RateLimiter 限流包装器
type RateLimiter interface {
	Allow() bool
	Release()
}

// SimpleRateLimiter 简单的令牌桶限流器
type SimpleRateLimiter struct {
	maxConcurrent int32
	current       int32
}

// NewSimpleRateLimiter 创建简单限流器
func NewSimpleRateLimiter(maxConcurrent int) *SimpleRateLimiter {
	return &SimpleRateLimiter{
		maxConcurrent: int32(maxConcurrent),
		current:       0,
	}
}

func (r *SimpleRateLimiter) Allow() bool {
	// 先加 1
	newVal := atomic.AddInt32(&r.current, 1)
	if newVal <= r.maxConcurrent {
		return true
	}
	// 如果超过了限制，必须减回去！
	atomic.AddInt32(&r.current, -1)
	return false
}

func (r *SimpleRateLimiter) Release() {
	atomic.AddInt32(&r.current, -1)
}

// RateLimit 限流包装器
func RateLimit(limiter RateLimiter) JobWrapper {
	return func(next JobFunc) JobFunc {
		return func(ctx context.Context) error {
			if !limiter.Allow() {
				return fmt.Errorf("rate limit exceeded")
			}

			defer limiter.Release()

			return next(ctx)
		}
	}
}

// CircuitBreakerPolicy 熔断器策略接口
type CircuitBreakerPolicy interface {
	Allow() bool
	RecordSuccess()
	RecordFailure()
}

// CircuitBreaker 熔断器包装器
func CircuitBreaker(breaker CircuitBreakerPolicy) JobWrapper {
	return func(next JobFunc) JobFunc {
		return func(ctx context.Context) error {
			if !breaker.Allow() {
				return fmt.Errorf("circuit breaker is open")
			}

			err := next(ctx)
			if err != nil {
				breaker.RecordFailure()
			} else {
				breaker.RecordSuccess()
			}

			return err
		}
	}
}

// Conditional 条件执行包装器
func Conditional(predicate func() bool) JobWrapper {
	return func(next JobFunc) JobFunc {
		return func(ctx context.Context) error {
			if !predicate() {
				logger.Info("⏭️ [JobWrapper] Job skipped due to condition check")
				return nil
			}
			return next(ctx)
		}
	}
}

// Validate 验证包装器，在任务执行前进行验证
func Validate(validateFunc func(ctx context.Context) error) JobWrapper {
	return func(next JobFunc) JobFunc {
		return func(ctx context.Context) error {
			if err := validateFunc(ctx); err != nil {
				return fmt.Errorf("validation failed: %w", err)
			}
			return next(ctx)
		}
	}
}

// Cleanup 清理包装器，无论任务成功或失败都会执行
func Cleanup(cleanupFunc func()) JobWrapper {
	return func(next JobFunc) JobFunc {
		return func(ctx context.Context) error {
			err := next(ctx)
			cleanupFunc()
			return err
		}
	}
}
