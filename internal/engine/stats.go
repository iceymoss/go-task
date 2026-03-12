package engine

import (
	"sort"
	"sync"
	"time"
)

// JobStats 任务运行时状态
type JobStats struct {
	Name        string    `json:"name"`
	CronExpr    string    `json:"cron_expr"`
	Status      string    `json:"status"`      // Idle, Running, Error
	LastRunTime string    `json:"last_run"`    // 格式化后的时间
	NextRunTime string    `json:"next_run"`    // 格式化后的时间
	LastResult  string    `json:"last_result"` // 成功或错误信息
	RunCount    int64     `json:"run_count"`
	rawNext     time.Time // 用于内部计算
	Source      string    `json:"source"` // 任务来源 (例如: "SYSTEM", "YAML", "API")
}

// StatManager 任务状态管理器
type StatManager struct {
	stats map[string]*JobStats // 任务名称 -> 状态
	mu    sync.RWMutex         // 保护 stats 的读写
}

// NewStatManager 创建状态管理器
func NewStatManager() *StatManager {
	return &StatManager{
		stats: make(map[string]*JobStats),
	}
}

// Set 设置任务状态
func (m *StatManager) Set(name string, stat *JobStats) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats[name] = stat
}

// Get 获取任务状态
func (m *StatManager) Get(name string) *JobStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stats[name]
}

// GetAll 获取所有任务状态
func (m *StatManager) GetAll() []*JobStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]*JobStats, 0, len(m.stats))
	for _, s := range m.stats {
		list = append(list, s)
	}
	// 按名称排序
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	return list
}
