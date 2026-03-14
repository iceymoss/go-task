package engine

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrCircularDependency = errors.New("circular dependency detected")
	ErrDependencyNotFound = errors.New("dependency not found")
)

// DependencyType 依赖类型
type DependencyType int

const (
	DependencyTypeAllSuccess  DependencyType = iota // 所有依赖都成功后执行
	DependencyTypeAnySuccess                        // 任一依赖成功后执行
	DependencyTypeAllComplete                       // 所有依赖完成后执行（无论成功失败）
)

// DependencyRule 任务依赖规则
type DependencyRule struct {
	TaskName       string         // 当前任务名称
	DependsOn      []string       // 依赖的任务名称列表
	DependencyType DependencyType // 依赖类型
	Timeout        time.Duration  // 等待依赖完成的超时时间
	CheckInterval  time.Duration  // 检查依赖状态的间隔
}

// DependencyManager 依赖管理器
type DependencyManager struct {
	dependencies map[string]*DependencyRule // 任务名 -> 依赖规则
	taskStatus   map[string]TaskStatus      // 任务名 -> 任务状态
	graph        map[string][]string        // 任务依赖图（用于检测循环依赖）
	logger       Logger
	mu           sync.RWMutex
}

// TaskStatus 任务依赖状态
type TaskStatus struct {
	Completed  bool      //  是否完成
	Success    bool      //  是否成功
	FinishedAt time.Time //  完成时间
	Error      error     //  错误信息
}

// NewDependencyManager 创建依赖管理器
func NewDependencyManager(logger Logger) *DependencyManager {
	return &DependencyManager{
		dependencies: make(map[string]*DependencyRule),
		taskStatus:   make(map[string]TaskStatus),
		graph:        make(map[string][]string),
		logger:       logger,
	}
}

// AddDependency 添加依赖关系
func (dm *DependencyManager) AddDependency(rule *DependencyRule) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// 检查循环依赖
	if err := dm.checkCircularDependency(rule.TaskName, rule.DependsOn); err != nil {
		return err
	}

	// 添加依赖规则
	dm.dependencies[rule.TaskName] = rule

	// 更新依赖图
	dm.graph[rule.TaskName] = rule.DependsOn

	// 初始化任务状态
	dm.taskStatus[rule.TaskName] = TaskStatus{
		Completed: false,
		Success:   false,
	}

	dm.logger.Info("✅ [Dependency] Added dependency",
		"task", rule.TaskName,
		"depends_on", rule.DependsOn,
		"type", dm.dependencyTypeToString(rule.DependencyType),
	)

	return nil
}

// checkCircularDependency 检查循环依赖
func (dm *DependencyManager) checkCircularDependency(task string, dependencies []string) error {
	visited := make(map[string]bool)
	recursionStack := make(map[string]bool)

	// 构建临时图用于检测
	tempGraph := make(map[string][]string)
	for k, v := range dm.graph {
		tempGraph[k] = make([]string, len(v))
		copy(tempGraph[k], v)
	}
	tempGraph[task] = make([]string, len(dependencies))
	copy(tempGraph[task], dependencies)

	// 对每个依赖任务进行DFS检测
	for _, dep := range dependencies {
		if dm.hasCycle(tempGraph, dep, visited, recursionStack) {
			return ErrCircularDependency
		}
	}

	return nil
}

// hasCycle 检测图中是否存在环
func (dm *DependencyManager) hasCycle(graph map[string][]string, node string, visited, recursionStack map[string]bool) bool {
	visited[node] = true
	recursionStack[node] = true

	for _, neighbor := range graph[node] {
		if !visited[neighbor] {
			if dm.hasCycle(graph, neighbor, visited, recursionStack) {
				return true
			}
		} else if recursionStack[neighbor] {
			return true
		}
	}

	recursionStack[node] = false
	return false
}

// CheckDependencies 检查任务的所有依赖是否满足
func (dm *DependencyManager) CheckDependencies(taskName string) (bool, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	rule, exists := dm.dependencies[taskName]
	if !exists || len(rule.DependsOn) == 0 {
		// 没有依赖，可以执行
		return true, nil
	}

	switch rule.DependencyType {
	case DependencyTypeAllSuccess:
		return dm.checkAllSuccess(rule.DependsOn), nil
	case DependencyTypeAnySuccess:
		return dm.checkAnySuccess(rule.DependsOn), nil
	case DependencyTypeAllComplete:
		return dm.checkAllComplete(rule.DependsOn), nil
	default:
		return false, fmt.Errorf("unknown dependency type: %v", rule.DependencyType)
	}
}

// checkAllSuccess 检查所有依赖是否都成功
func (dm *DependencyManager) checkAllSuccess(dependencies []string) bool {
	for _, dep := range dependencies {
		status, exists := dm.taskStatus[dep]
		if !exists || !status.Completed || !status.Success {
			return false
		}
	}
	return true
}

// checkAnySuccess 检查是否有任一依赖成功
func (dm *DependencyManager) checkAnySuccess(dependencies []string) bool {
	for _, dep := range dependencies {
		status, exists := dm.taskStatus[dep]
		if exists && status.Completed && status.Success {
			return true
		}
	}
	return false
}

// checkAllComplete 检查所有依赖是否都完成
func (dm *DependencyManager) checkAllComplete(dependencies []string) bool {
	for _, dep := range dependencies {
		status, exists := dm.taskStatus[dep]
		if !exists || !status.Completed {
			return false
		}
	}
	return true
}

// UpdateTaskStatus 更新任务状态
func (dm *DependencyManager) UpdateTaskStatus(taskName string, success bool, err error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.taskStatus[taskName] = TaskStatus{
		Completed:  true,
		Success:    success,
		FinishedAt: time.Now(),
		Error:      err,
	}

	dm.logger.Info("📊 [Dependency] Updated task status",
		"task", taskName,
		"success", success,
		"finished_at", time.Now(),
	)
}

// GetDependencyRule 获取任务的依赖规则
func (dm *DependencyManager) GetDependencyRule(taskName string) (*DependencyRule, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	rule, exists := dm.dependencies[taskName]
	return rule, exists
}

// GetDependencyChain 获取任务的依赖链
func (dm *DependencyManager) GetDependencyChain(taskName string) ([]string, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	var chain []string
	visited := make(map[string]bool)

	dm.buildDependencyChain(taskName, visited, &chain)

	return chain, nil
}

// buildDependencyChain 递归构建依赖链
func (dm *DependencyManager) buildDependencyChain(taskName string, visited map[string]bool, chain *[]string) {
	if visited[taskName] {
		return
	}
	visited[taskName] = true

	if rule, exists := dm.dependencies[taskName]; exists {
		for _, dep := range rule.DependsOn {
			dm.buildDependencyChain(dep, visited, chain)
		}
	}

	*chain = append(*chain, taskName)
}

// GetDependentTasks 获取依赖于指定任务的所有任务
func (dm *DependencyManager) GetDependentTasks(taskName string) []string {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	var dependents []string
	for t, rule := range dm.dependencies {
		for _, dep := range rule.DependsOn {
			if dep == taskName {
				dependents = append(dependents, t)
				break
			}
		}
	}

	return dependents
}

// ClearTaskStatus 清除任务状态（用于重试场景）
func (dm *DependencyManager) ClearTaskStatus(taskName string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.taskStatus[taskName] = TaskStatus{
		Completed: false,
		Success:   false,
	}
}

// dependencyTypeToString 将依赖类型转换为字符串
func (dm *DependencyManager) dependencyTypeToString(dt DependencyType) string {
	switch dt {
	case DependencyTypeAllSuccess:
		return "all_success"
	case DependencyTypeAnySuccess:
		return "any_success"
	case DependencyTypeAllComplete:
		return "all_complete"
	default:
		return "unknown"
	}
}
