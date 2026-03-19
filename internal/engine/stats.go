package engine

import (
	"sort"
	"sync"
	"time"
)

// status 状态常量
type jobStatus string

const (
	Idle    jobStatus = "Idle"
	Waiting jobStatus = "Waiting"
	Queued  jobStatus = "Queued"
	Running jobStatus = "Running"
	Error   jobStatus = "Error"
	Success jobStatus = "Success"
)

const (
	LastResultSuccess         string = "Success"
	LastResultError           string = "Error: %v"
	LastResultPending         string = "Pending"
	LastResultDependencyCheck string = "Dependency check failed: %v"
)

// JobStats 任务运行时状态
type JobStats struct {
	Name        string    `json:"name"`
	CronExpr    string    `json:"cron_expr"`
	Status      jobStatus `json:"status"`
	LastRunTime string    `json:"last_run"`
	NextRunTime string    `json:"next_run"`
	LastResult  string    `json:"last_result"`
	RunCount    int64     `json:"run_count"`
	Source      string    `json:"source"`
	RawNext     time.Time `json:"raw_next"`
}

// StatManager 任务状态管理器
type StatManager struct {
	stats map[string]*JobStats
	mu    sync.RWMutex
}

func NewStatManager() *StatManager {
	return &StatManager{
		stats: make(map[string]*JobStats),
	}
}

// Set 初始化/重置任务状态
func (m *StatManager) Set(name string, stat *JobStats) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats[name] = stat
}

// Update 处于锁的保护之下,Update 安全地更新任务状态
func (m *StatManager) Update(name string, fn func(stat *JobStats)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if stat, exists := m.stats[name]; exists {
		fn(stat)
	}
}

// Get 修改：Get 获取状态的安全拷贝
func (m *StatManager) Get(name string) (JobStats, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	stat, exists := m.stats[name]
	if !exists {
		return JobStats{}, false
	}
	return *stat, true // 返回解引用的值（深拷贝）
}

// GetAll 获取所有状态的安全拷贝列表
func (m *StatManager) GetAll() []JobStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]JobStats, 0, len(m.stats))
	for _, s := range m.stats {
		list = append(list, *s) // 每次循环都拷贝一份值放进 slice
	}

	// 按名称排序
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	return list
}
