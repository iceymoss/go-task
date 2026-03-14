package engine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
)

const DefaultLeaderKeyTTL = 15

// LeaderElector 抽象的选主接口
type LeaderElector interface {
	// Start 启动选主。在这里接收 Scheduler 传来的回调函数
	Start(ctx context.Context, onStartedLeading func(), onStoppedLeading func()) error
	Stop(ctx context.Context) error
	IsLeader() bool
}

// RedisLeaderElector 基于 Redis 的简单选主实现
// 使用一个带 TTL 的 key 做 Leader 锁，value 为实例 ID。
type RedisLeaderElector struct {
	client        *redis.Client
	key           string
	id            string
	ttl           time.Duration
	renewInterval time.Duration

	logger Logger

	isLeader int32
	mu       sync.RWMutex
	started  bool
}

func NewRedisLeaderElector(client *redis.Client, key string, ttl, renewInterval time.Duration, logger Logger) *RedisLeaderElector {
	if ttl <= 0 {
		ttl = DefaultLeaderKeyTTL * time.Second
	}
	if renewInterval <= 0 || renewInterval >= ttl {
		renewInterval = ttl / 2
	}

	return &RedisLeaderElector{
		client:        client,
		key:           key,
		id:            defaultInstanceID(),
		ttl:           ttl,
		renewInterval: renewInterval,
		logger:        logger,
	}
}

// defaultInstanceID 生成当前实例的标识
func defaultInstanceID() string {
	host, err := os.Hostname()
	if err != nil {
		host = "unknown-host"
	}
	return fmt.Sprintf("%s-%d", host, time.Now().UnixNano())
}

// Start 启动选主循环（非阻塞）
func (r *RedisLeaderElector) Start(ctx context.Context, onStarted func(), onStopped func()) error {
	r.mu.Lock()
	if r.started {
		r.mu.Unlock()
		return nil
	}
	r.started = true
	r.mu.Unlock()

	// 启动后台协程，把回调函数带进去
	go r.loop(ctx, onStarted, onStopped)
	return nil
}

// loop 主循环：尝试抢占锁、续约、检测是否失去 Leader
func (r *RedisLeaderElector) loop(ctx context.Context, onStarted func(), onStopped func()) {
	// 续约间隔
	ticker := time.NewTicker(r.renewInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if r.IsLeader() {
				_ = r.releaseLock(context.Background())
				r.setLeader(false)
				if onStopped != nil {
					onStopped() // 触发停止回调
				}
			}
			return
		case <-ticker.C:
			if r.IsLeader() {
				if err := r.renewLock(ctx); err != nil {
					r.logger.Info("⚠️ [LeaderElector] renew lock failed", err)
					r.setLeader(false)
					if onStopped != nil {
						onStopped() // 失去锁，触发停止回调
					}
				}
			} else {
				ok, err := r.acquireLock(ctx)
				if err != nil {
					continue
				}
				if ok {
					r.setLeader(true)
					r.logger.Info("👑 [LeaderElector] became leader", "id", r.id)
					if onStarted != nil {
						onStarted() // 抢到锁，触发启动回调
					}
				}
			}
		}
	}
}

// acquireLock 使用 SETNX + TTL 抢占锁
func (r *RedisLeaderElector) acquireLock(ctx context.Context) (bool, error) {
	ok, err := r.client.SetNX(ctx, r.key, r.id, r.ttl).Result()
	return ok, err
}

// renewLock 续约锁（仅在自己仍然是锁持有者时）
func (r *RedisLeaderElector) renewLock(ctx context.Context) error {
	val, err := r.client.Get(ctx, r.key).Result()
	if errors.Is(err, redis.Nil) {
		// key 不存在，锁已丢失
		return fmt.Errorf("lock key missing")
	}
	if err != nil {
		return err
	}
	if val != r.id {
		// 已经不是自己持有
		return fmt.Errorf("lock owned by another instance")
	}

	// 续约 TTL
	_, err = r.client.Expire(ctx, r.key, r.ttl).Result()
	return err
}

// releaseLock 释放锁（仅在自己仍是持有者时）
func (r *RedisLeaderElector) releaseLock(ctx context.Context) error {
	val, err := r.client.Get(ctx, r.key).Result()
	if errors.Is(err, redis.Nil) {
		return nil
	}
	if err != nil {
		return err
	}
	if val != r.id {
		return nil
	}
	_, err = r.client.Del(ctx, r.key).Result()
	return err
}

// Stop 停止选主（通过取消 ctx 实现）
func (r *RedisLeaderElector) Stop(ctx context.Context) error {
	// 实际停止由外部 ctx 控制，这里只做一次性释放锁的兜底
	return r.releaseLock(ctx)
}

// IsLeader 是否为 Leader
func (r *RedisLeaderElector) IsLeader() bool {
	return atomic.LoadInt32(&r.isLeader) == 1
}

func (r *RedisLeaderElector) setLeader(v bool) {
	if v {
		atomic.StoreInt32(&r.isLeader, 1)
	} else {
		atomic.StoreInt32(&r.isLeader, 0)
	}
}
