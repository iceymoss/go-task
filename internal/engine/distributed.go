package engine

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/iceymoss/go-task/pkg/db"
)

// LeaderElector 抽象的选主接口
type LeaderElector interface {
	// Start 启动选主循环（非阻塞或长时间阻塞均可，由实现决定）
	Start(ctx context.Context) error
	// Stop 停止选主循环
	Stop(ctx context.Context) error
	// IsLeader 当前实例是否为 Leader
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

	onStartedLeading func()
	onStoppedLeading func()

	isLeader int32

	mu      sync.RWMutex
	started bool
}

// NewRedisLeaderElector 创建 Redis 选主器
func NewRedisLeaderElector(
	client *redis.Client,
	key string,
	ttl, renewInterval time.Duration,
	onStartedLeading func(),
	onStoppedLeading func(),
) *RedisLeaderElector {
	if ttl <= 0 {
		ttl = 15 * time.Second
	}
	if renewInterval <= 0 || renewInterval >= ttl {
		renewInterval = ttl / 2
	}

	id := defaultInstanceID()

	return &RedisLeaderElector{
		client:           client,
		key:              key,
		id:               id,
		ttl:              ttl,
		renewInterval:    renewInterval,
		onStartedLeading: onStartedLeading,
		onStoppedLeading: onStoppedLeading,
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
func (r *RedisLeaderElector) Start(ctx context.Context) error {
	r.mu.Lock()
	if r.started {
		r.mu.Unlock()
		return nil
	}
	r.started = true
	r.mu.Unlock()

	go r.loop(ctx)
	return nil
}

// loop 主循环：尝试抢占锁、续约、检测是否失去 Leader
func (r *RedisLeaderElector) loop(ctx context.Context) {
	ticker := time.NewTicker(r.renewInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// 退出前如果是 Leader，尝试释放锁
			if r.IsLeader() {
				if err := r.releaseLock(context.Background()); err != nil {
					log.Printf("⚠️ [LeaderElector] release lock failed: %v", err)
				}
				r.setLeader(false)
			}
			return
		case <-ticker.C:
			if r.IsLeader() {
				// 已是 Leader，尝试续约
				if err := r.renewLock(ctx); err != nil {
					log.Printf("⚠️ [LeaderElector] renew lock failed: %v", err)
					// 续约失败时，下一轮会尝试重新抢占
					r.setLeader(false)
					if r.onStoppedLeading != nil {
						r.onStoppedLeading()
					}
				}
			} else {
				// 非 Leader，尝试抢占
				ok, err := r.acquireLock(ctx)
				if err != nil {
					log.Printf("⚠️ [LeaderElector] acquire lock error: %v", err)
					continue
				}
				if ok {
					r.setLeader(true)
					log.Printf("👑 [LeaderElector] became leader, id=%s", r.id)
					if r.onStartedLeading != nil {
						r.onStartedLeading()
					}
				}
			}
		}
	}
}

// acquireLock 使用 SETNX + TTL 抢占锁
func (r *RedisLeaderElector) acquireLock(ctx context.Context) (bool, error) {
	if r.client == nil {
		r.client = db.GetRedisConn()
	}
	ok, err := r.client.SetNX(ctx, r.key, r.id, r.ttl).Result()
	return ok, err
}

// renewLock 续约锁（仅在自己仍然是锁持有者时）
func (r *RedisLeaderElector) renewLock(ctx context.Context) error {
	if r.client == nil {
		r.client = db.GetRedisConn()
	}

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
	if r.client == nil {
		r.client = db.GetRedisConn()
	}

	val, err := r.client.Get(ctx, r.key).Result()
	if err == redis.Nil {
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

// EnableRedisLeaderElection 为调度器启用基于 Redis 的分布式选主
// key 示例: "go-task:scheduler:leader"
func (s *Scheduler) EnableRedisLeaderElection(key string, ttl, renewInterval time.Duration) {
	client := db.GetRedisConn()

	onStarted := func() {
		log.Printf("👑 [Scheduler] This instance became leader, starting cron")
		s.cron.Start()
	}
	onStopped := func() {
		log.Printf("👋 [Scheduler] Lost leadership, stopping cron")
		s.cron.Stop()
	}

	s.leaderElector = NewRedisLeaderElector(client, key, ttl, renewInterval, onStarted, onStopped)
}
