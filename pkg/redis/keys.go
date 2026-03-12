package redis

import "fmt"

// ============================================
// Redis Key 前缀常量
// ============================================

const (
	// 前缀
	PrefixScheduler = "scheduler:"
	PrefixTask      = "task:"
	PrefixWorker    = "worker:"
	PrefixAlert     = "alert:"
	PrefixCache     = "cache:"
	PrefixSession   = "session:"
	PrefixJWT       = "jwt:"
	PrefixWorkflow  = "workflow:"
	PrefixCron      = "cron:"
	PrefixDelayed   = "delayed:"
)

// ============================================
// 1. 分布式协调 Keys
// ============================================

// Leader选举
func KeyLeaderLock() string {
	return PrefixScheduler + "leader:lock"
}

func KeyLeaderVersion() string {
	return PrefixScheduler + "leader:version"
}

// Worker注册
func KeyWorkerInfo(workerID string) string {
	return PrefixWorker + workerID + ":info"
}

func KeyWorkerHeartbeat(workerID string) string {
	return PrefixWorker + workerID + ":heartbeat"
}

func KeyWorkersRegistered() string {
	return PrefixWorker + "registered"
}

// 集群信息
func KeyClusterInfo() string {
	return PrefixScheduler + "cluster:info"
}

// ============================================
// 2. 任务调度 Keys
// ============================================

// 任务队列
func KeyTaskQueue(priority string) string {
	// priority: high, normal, low
	return PrefixTask + "queue:" + priority
}

// 任务执行锁
func KeyTaskLock(jobID uint, executionID string) string {
	return PrefixTask + fmt.Sprintf("lock:%d:%s", jobID, executionID)
}

func KeyTaskLockRetry(executionID string) string {
	return PrefixTask + "lock:retry:" + executionID
}

// 任务实时状态
func KeyTaskStatus(jobID uint) string {
	return PrefixTask + fmt.Sprintf("status:%d", jobID)
}

// 下次执行时间（用于触发）
func KeyTaskSchedule(jobID uint) string {
	return PrefixTask + fmt.Sprintf("schedule:%d", jobID)
}

// ============================================
// 3. 限流与并发控制 Keys
// ============================================

// 限流
func KeyTaskRateLimit(jobID uint) string {
	return PrefixTask + fmt.Sprintf("ratelimit:%d", jobID)
}

// 并发控制
func KeyTaskConcurrent(jobID uint) string {
	return PrefixTask + fmt.Sprintf("concurrent:%d", jobID)
}

func KeyTaskConcurrentMax(jobID uint) string {
	return PrefixTask + fmt.Sprintf("concurrent:max:%d", jobID)
}

// 熔断器
func KeyTaskCircuit(jobID uint) string {
	return PrefixTask + fmt.Sprintf("circuit:%d", jobID)
}

// ============================================
// 4. 告警聚合 Keys
// ============================================

// 告警队列
func KeyAlertQueue(jobID uint) string {
	return PrefixAlert + fmt.Sprintf("queue:%d", jobID)
}

func KeyAlertQueueGlobal() string {
	return PrefixAlert + "queue:global"
}

// 告警聚合
func KeyAlertAggregate(jobID uint, window string) string {
	return PrefixAlert + fmt.Sprintf("aggregate:%d:%s", jobID, window)
}

// 告警限流
func KeyAlertRateLimit(jobID uint) string {
	return PrefixAlert + fmt.Sprintf("ratelimit:%d", jobID)
}

// ============================================
// 5. 缓存 Keys
// ============================================

// 任务配置缓存
func KeyJobConfig(jobID uint) string {
	return PrefixCache + fmt.Sprintf("job:config:%d", jobID)
}

// 执行记录缓存
func KeyExecution(executionID string) string {
	return PrefixCache + "execution:" + executionID
}

// 统计数据缓存
func KeyStatsJobHourly(jobID uint, date string, hour int) string {
	return PrefixCache + fmt.Sprintf("stats:job:%d:hourly:%s:%02d", jobID, date, hour)
}

func KeyStatsGlobalHourly(date string, hour int) string {
	return PrefixCache + fmt.Sprintf("stats:global:hourly:%s:%02d", date, hour)
}

// 系统配置缓存
func KeyConfig(key string) string {
	return PrefixCache + "config:" + key
}

// ============================================
// 6. 会话与认证 Keys
// ============================================

// JWT Token黑名单
func KeyJWTBlacklist(tokenHash string) string {
	return PrefixJWT + "blacklist:" + tokenHash
}

// 用户会话
func KeySession(sessionID string) string {
	return PrefixSession + sessionID
}

// 登录限流
func KeyLoginRateLimit(ip string) string {
	return PrefixSession + "login:ratelimit:" + ip
}

// ============================================
// 7. 工作流 Keys
// ============================================

// 工作流状态
func KeyWorkflowExecution(executionID string) string {
	return PrefixWorkflow + "execution:" + executionID
}

// 工作流节点锁
func KeyWorkflowNodeLock(executionID string, nodeID string) string {
	return PrefixWorkflow + fmt.Sprintf("node:lock:%s:%s", executionID, nodeID)
}

// 工作流依赖
func KeyWorkflowDependencies(executionID string) string {
	return PrefixWorkflow + "dependencies:" + executionID
}

// ============================================
// 8. 定时任务 Keys
// ============================================

// Cron触发记录
func KeyCronTriggered(jobID uint, timestamp int64) string {
	return PrefixCron + fmt.Sprintf("triggered:%d:%d", jobID, timestamp)
}

// 延迟任务
func KeyDelayedTask(executionID string) string {
	return PrefixDelayed + "task:" + executionID
}

// ============================================
// TTL 常量（秒）
// ============================================

const (
	// Leader选举
	TTLLeaderLock = 15 // 15秒
	TTLHeartbeat  = 30 // 30秒

	// 任务执行
	TTLTaskLock      = 3600  // 1小时
	TTLTaskLockRetry = 600   // 10分钟
	TTLTaskSchedule  = 86400 // 24小时

	// 限流
	TTLRateLimit      = 10  // 10秒
	TTLAlertRateLimit = 300 // 5分钟

	// 缓存
	TTLJobConfig   = 3600  // 1小时
	TTLExecution   = 86400 // 24小时
	TTLStatsHourly = 90000 // 25小时
	TTLConfig      = 600   // 10分钟

	// 会话
	TTLSession      = 86400 // 24小时
	TTLJWTBlacklist = 0     // 永不过期，直到token过期
	TTLLoginLimit   = 3600  // 1小时
)

// ============================================
// 优先级常量
// ============================================

const (
	PriorityHigh   = "high"
	PriorityNormal = "normal"
	PriorityLow    = "low"
)

// PriorityScore 优先级分数
// 数值越小，优先级越高
const (
	ScoreHigh   = 1
	ScoreNormal = 10
	ScoreLow    = 20
)
