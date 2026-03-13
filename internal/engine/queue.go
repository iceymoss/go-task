package engine

import (
	"container/heap"
	"fmt"
	"sync"

	"github.com/iceymoss/go-task/pkg/logger"
)

// DefaultWorkerNum 默认工作协程数量
const defaultWorkerNum = 10

// TaskQueue 简单优先级任务队列，使用内存优先队列 + 固定工作协程
type TaskQueue struct {
	scheduler *Scheduler     // 引用调度器以执行任务
	mu        sync.Mutex     // 保护 items 和 closed 的并发访问
	cond      *sync.Cond     // 条件变量，用于通知 worker 有新任务
	items     *priorityQueue // 任务列表
	workerNum int            // worker 数量
	wg        sync.WaitGroup // 等待 worker 退出
	closed    bool           // 是否已关闭队列
}

// NewTaskQueue 创建任务队列, 会启动固定数量的 worker
func NewTaskQueue(s *Scheduler, workerNum int) *TaskQueue {
	if workerNum <= 0 {
		workerNum = defaultWorkerNum
	}

	// 初始化堆
	pq := make(priorityQueue, 0)
	heap.Init(&pq)

	// 初始化任务队列
	q := &TaskQueue{
		scheduler: s,
		workerNum: workerNum,
		items:     &pq, // 指向堆
	}

	// 初始化条件变量
	q.cond = sync.NewCond(&q.mu)

	// 启动 worker
	q.startWorkers()

	return q
}

// startWorkers 启动固定数量的 worker
func (q *TaskQueue) startWorkers() {
	for i := 0; i < q.workerNum; i++ {
		q.wg.Add(1)
		go q.workerLoop(i)
	}
}

// workerLoop 工作协程循环，从队列中取出最高优先级任务执行
func (q *TaskQueue) workerLoop(id int) {
	defer q.wg.Done()

	for {
		item, ok := q.pop()
		if !ok {
			return
		}
		logger.Info(fmt.Sprintf("🧵 [TaskQueue] Worker-%d handling job: %s (priority=%d)", id, item.Name, item.Priority))
		q.scheduler.runTaskWithStats(item.Name)
	}
}

// pop 从队列中取出一个最高优先级任务
func (q *TaskQueue) pop() (TaskItem, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for len(*q.items) == 0 && !q.closed {
		q.cond.Wait()
	}

	if q.closed && len(*q.items) == 0 {
		return TaskItem{}, false
	}

	// 使用 heap.Pop 直接获取最高优先级的元素，内部会自动平衡树结构
	// 无需 for 循环遍历整个切片寻找最大值
	item := heap.Pop(q.items).(TaskItem)

	return item, true
}

// Enqueue 入队一个任务
func (q *TaskQueue) Enqueue(name string, priority int) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return nil
	}

	// 使用 heap.Push 插入元素，内部会自动调整树结构保持最大堆形态
	heap.Push(q.items, TaskItem{
		Name:     name,
		Priority: priority,
	})

	// 唤醒一个等待的 worker
	q.cond.Signal()

	return nil
}

// Stop 停止队列，等待所有 worker 退出
func (q *TaskQueue) Stop() {
	q.mu.Lock()
	q.closed = true
	q.cond.Broadcast()
	q.mu.Unlock()

	q.wg.Wait()
}
