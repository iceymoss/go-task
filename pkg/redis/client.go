package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client Redis客户端包装器
type Client struct {
	client *redis.Client
}

// NewClient 创建Redis客户端
func NewClient(addr, password string, db int) *Client {
	return &Client{
		client: redis.NewClient(&redis.Options{
			Addr:         addr,
			Password:     password,
			DB:           db,
			PoolSize:     10,
			MinIdleConns: 5,
			MaxRetries:   3,
			IdleTimeout:  5 * time.Minute,
		}),
	}
}

// Close 关闭连接
func (c *Client) Close() error {
	return c.client.Close()
}

// Ping 测试连接
func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// ============================================
// String 操作
// ============================================

// Set 设置键值
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.client.Set(ctx, key, value, expiration).Err()
}

// Get 获取值
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

// GetJSON 获取JSON值
func (c *Client) GetJSON(ctx context.Context, key string, out interface{}) error {
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), out)
}

// SetJSON 设置JSON值
func (c *Client) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, expiration).Err()
}

// Del 删除键
func (c *Client) Del(ctx context.Context, keys ...string) error {
	return c.client.Del(ctx, keys...).Err()
}

// Exists 检查键是否存在
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	return c.client.Exists(ctx, key).Result()
}

// Expire 设置过期时间
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.client.Expire(ctx, key, expiration).Err()
}

// TTL 获取剩余过期时间
func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, key).Result()
}

// ============================================
// Hash 操作
// ============================================

// HSet 设置Hash字段
func (c *Client) HSet(ctx context.Context, key, field string, value interface{}) error {
	return c.client.HSet(ctx, key, field, value).Err()
}

// HGet 获取Hash字段
func (c *Client) HGet(ctx context.Context, key, field string) (string, error) {
	return c.client.HGet(ctx, key, field).Result()
}

// HGetAll 获取所有Hash字段
func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return c.client.HGetAll(ctx, key).Result()
}

// HDel 删除Hash字段
func (c *Client) HDel(ctx context.Context, key string, fields ...string) error {
	return c.client.HDel(ctx, key, fields...).Err()
}

// HExists 检查Hash字段是否存在
func (c *Client) HExists(ctx context.Context, key, field string) (bool, error) {
	return c.client.HExists(ctx, key, field).Result()
}

// HIncrBy Hash字段增量
func (c *Client) HIncrBy(ctx context.Context, key, field string, incr int64) (int64, error) {
	return c.client.HIncrBy(ctx, key, field, incr).Result()
}

// ============================================
// List 操作
// ============================================

// LPush 从左侧推入列表
func (c *Client) LPush(ctx context.Context, key string, values ...interface{}) error {
	return c.client.LPush(ctx, key, values...).Err()
}

// RPush 从右侧推入列表
func (c *Client) RPush(ctx context.Context, key string, values ...interface{}) error {
	return c.client.RPush(ctx, key, values...).Err()
}

// LPop 从左侧弹出列表
func (c *Client) LPop(ctx context.Context, key string) (string, error) {
	return c.client.LPop(ctx, key).Result()
}

// RPop 从右侧弹出列表
func (c *Client) RPop(ctx context.Context, key string) (string, error) {
	return c.client.RPop(ctx, key).Result()
}

// LLen 获取列表长度
func (c *Client) LLen(ctx context.Context, key string) (int64, error) {
	return c.client.LLen(ctx, key).Result()
}

// LRange 获取列表范围内的元素
func (c *Client) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.client.LRange(ctx, key, start, stop).Result()
}

// ============================================
// Set 操作
// ============================================

// SAdd 添加到集合
func (c *Client) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return c.client.SAdd(ctx, key, members...).Err()
}

// SRem 从集合移除
func (c *Client) SRem(ctx context.Context, key string, members ...interface{}) error {
	return c.client.SRem(ctx, key, members...).Err()
}

// SMembers 获取所有成员
func (c *Client) SMembers(ctx context.Context, key string) ([]string, error) {
	return c.client.SMembers(ctx, key).Result()
}

// SIsMember 检查是否是成员
func (c *Client) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return c.client.SIsMember(ctx, key, member).Result()
}

// SCard 获取集合大小
func (c *Client) SCard(ctx context.Context, key string) (int64, error) {
	return c.client.SCard(ctx, key).Result()
}

// ============================================
// Sorted Set 操作
// ============================================

// ZAdd 添加到有序集合
func (c *Client) ZAdd(ctx context.Context, key string, score float64, member interface{}) error {
	return c.client.ZAdd(ctx, key, redis.Z{
		Score:  score,
		Member: member,
	}).Err()
}

// ZRem 从有序集合移除
func (c *Client) ZRem(ctx context.Context, key string, members ...interface{}) error {
	return c.client.ZRem(ctx, key, members...).Err()
}

// ZRange 按分数范围获取
func (c *Client) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.client.ZRange(ctx, key, start, stop).Result()
}

// ZRangeByScore 按分数范围获取
func (c *Client) ZRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) ([]string, error) {
	return c.client.ZRangeByScore(ctx, key, opt).Result()
}

// ZPopMin 弹出最小分数的成员
func (c *Client) ZPopMin(ctx context.Context, key string) (*redis.Z, error) {
	return c.client.ZPopMin(ctx, key).Result()
}

// ZCard 获取有序集合大小
func (c *Client) ZCard(ctx context.Context, key string) (int64, error) {
	return c.client.ZCard(ctx, key).Result()
}

// ZScore 获取成员分数
func (c *Client) ZScore(ctx context.Context, key string, member interface{}) (float64, error) {
	return c.client.ZScore(ctx, key, member).Result()
}

// ============================================
// 分布式锁
// ============================================

// TryLock 尝试获取分布式锁
func (c *Client) TryLock(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	ok, err := c.client.SetNX(ctx, key, "locked", expiration).Result()
	return ok, err
}

// Unlock 释放分布式锁
func (c *Client) Unlock(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// ============================================
// 任务队列操作
// ============================================

// EnqueueTask 将任务加入队列
func (c *Client) EnqueueTask(ctx context.Context, priority string, executionID string, timestamp int64) error {
	queueKey := KeyTaskQueue(priority)
	score := float64(timestamp)

	// 根据优先级设置分数
	switch priority {
	case PriorityHigh:
		score = float64(timestamp)
	case PriorityNormal:
		score = float64(timestamp) + 100000
	case PriorityLow:
		score = float64(timestamp) + 1000000
	}

	return c.ZAdd(ctx, queueKey, score, executionID)
}

// DequeueTask 从队列中取出任务（取出最早的任务）
func (c *Client) DequeueTask(ctx context.Context, priority string) (string, error) {
	queueKey := KeyTaskQueue(priority)
	result, err := c.ZPopMin(ctx, queueKey)
	if err != nil {
		return "", err
	}
	if result == nil {
		return "", nil
	}
	return result.Member.(string), nil
}

// GetQueueLength 获取队列长度
func (c *Client) GetQueueLength(ctx context.Context, priority string) (int64, error) {
	queueKey := KeyTaskQueue(priority)
	return c.ZCard(ctx, queueKey)
}

// ============================================
// 计数器操作
// ============================================

// Incr 计数器加1
func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	return c.client.Incr(ctx, key).Result()
}

// IncrBy 计数器增加指定值
func (c *Client) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return c.client.IncrBy(ctx, key, value).Result()
}

// Decr 计数器减1
func (c *Client) Decr(ctx context.Context, key string) (int64, error) {
	return c.client.Decr(ctx, key).Result()
}

// DecrBy 计数器减去指定值
func (c *Client) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	return c.client.DecrBy(ctx, key, value).Result()
}

// ============================================
// 批量操作
// ============================================

// MGet 批量获取多个键
func (c *Client) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	return c.client.MGet(ctx, keys...).Result()
}

// MSet 批量设置多个键
func (c *Client) MSet(ctx context.Context, values ...interface{}) error {
	return c.client.MSet(ctx, values...).Err()
}

// Pipeline 管道操作
func (c *Client) Pipeline(ctx context.Context, fn func(pipe redis.Pipeliner) error) error {
	pipe := c.client.Pipeline()
	if err := fn(pipe); err != nil {
		return err
	}
	_, err := pipe.Exec(ctx)
	return err
}

// ============================================
// 事务操作
// ============================================

// TxPipeline 事务管道
func (c *Client) TxPipeline(ctx context.Context, fn func(pipe redis.Pipeliner) error) ([]redis.Cmder, error) {
	pipe := c.client.TxPipeline()
	if err := fn(pipe); err != nil {
		return nil, err
	}
	return pipe.Exec(ctx)
}

// ============================================
// 搜索与扫描
// ============================================

// Scan 扫描键
func (c *Client) Scan(ctx context.Context, cursor uint64, match string, count int64) ([]string, uint64, error) {
	return c.client.Scan(ctx, cursor, match, count).Result()
}

// Keys 获取所有匹配的键（慎用，生产环境不推荐）
func (c *Client) Keys(ctx context.Context, pattern string) ([]string, error) {
	return c.client.Keys(ctx, pattern).Result()
}

// ============================================
// 辅助方法
// ============================================

// GetOrCreate 获取键，如果不存在则创建
func (c *Client) GetOrCreate(ctx context.Context, key string, defaultValue interface{}, expiration time.Duration) (string, error) {
	val, err := c.Get(ctx, key)
	if err == redis.Nil {
		// 键不存在，创建
		if err := c.Set(ctx, key, defaultValue, expiration); err != nil {
			return "", err
		}
		return fmt.Sprintf("%v", defaultValue), nil
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

// SetWithNX 设置键（仅在不存在时）
func (c *Client) SetWithNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return c.client.SetNX(ctx, key, value, expiration).Result()
}

// GetOrLoad 获取键，如果不存在则从加载函数加载
func (c *Client) GetOrLoad(ctx context.Context, key string, loadFn func() (interface{}, error), expiration time.Duration) (string, error) {
	val, err := c.Get(ctx, key)
	if err == redis.Nil {
		// 键不存在，加载
		data, err := loadFn()
		if err != nil {
			return "", err
		}
		if err := c.Set(ctx, key, data, expiration); err != nil {
			return "", err
		}
		return fmt.Sprintf("%v", data), nil
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

// ============================================
// 统计操作
// ============================================

// IncrementStats 增加统计计数
func (c *Client) IncrementStats(ctx context.Context, key string, field string, value int64) error {
	return c.HIncrBy(ctx, key, field, value)
}

// GetStatsField 获取统计字段
func (c *Client) GetStatsField(ctx context.Context, key, field string) (int64, error) {
	val, err := c.HGet(ctx, key, field)
	if err != nil {
		return 0, err
	}
	var result int64
	_, err = fmt.Sscanf(val, "%d", &result)
	return result, err
}

// GetStatsAll 获取所有统计数据
func (c *Client) GetStatsAll(ctx context.Context, key string) (map[string]string, error) {
	return c.HGetAll(ctx, key)
}

// ResetStats 重置统计
func (c *Client) ResetStats(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}
