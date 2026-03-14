package engine

import (
	"fmt"
	"sync"

	"github.com/iceymoss/go-task/internal/core"
)

// TaskRegistry 任务模板注册表 (只管名字和构造函数的映射)
type TaskRegistry struct {
	creators map[string]core.TaskCreator
	mu       sync.RWMutex
}

func NewTaskRegistry() *TaskRegistry {
	return &TaskRegistry{
		creators: make(map[string]core.TaskCreator),
	}
}

// Register 注册一个任务模板
func (r *TaskRegistry) Register(name string, creator core.TaskCreator) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.creators[name] = creator
}

// Get 获取任务模板的构造函数
func (r *TaskRegistry) Get(name string) (core.TaskCreator, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	creator, ok := r.creators[name]
	if !ok {
		return nil, fmt.Errorf("task implementation '%s' not found in registry", name)
	}
	return creator, nil
}
