package engine

import (
	"log"
	"sync"
)

// TaskItem 表示队列中的一个任务
type TaskItem struct {
	Name     string
	Priority int
}

// TaskQueue 简单优先级任务队列，使用内存优先队列 + 固定工作协程
type TaskQueue struct {
	scheduler *Scheduler

	mu        sync.Mutex
	cond      *sync.Cond
	items     []TaskItem
	workerNum int
	wg        sync.WaitGroup
	closed    bool
}

// NewTaskQueue 创建任务队列
func NewTaskQueue(s *Scheduler, workerNum int) *TaskQueue {
	if workerNum <= 0 {
		workerNum = 4
	}
	q := &TaskQueue{
		scheduler: s,
		workerNum: workerNum,
		items:     make([]TaskItem, 0),
	}
	q.cond = sync.NewCond(&q.mu)
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

		log.Printf("🧵 [TaskQueue] Worker-%d handling job: %s (priority=%d)", id, item.Name, item.Priority)
		q.scheduler.runTaskWithStats(item.Name)
	}
}

// pop 从队列中取出一个最高优先级任务
func (q *TaskQueue) pop() (TaskItem, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for len(q.items) == 0 && !q.closed {
		q.cond.Wait()
	}

	if q.closed && len(q.items) == 0 {
		return TaskItem{}, false
	}

	// 找到优先级最高的任务（值越大优先级越高）
	maxIdx := 0
	for i := 1; i < len(q.items); i++ {
		if q.items[i].Priority > q.items[maxIdx].Priority {
			maxIdx = i
		}
	}

	item := q.items[maxIdx]
	// 删除该元素
	q.items = append(q.items[:maxIdx], q.items[maxIdx+1:]...)

	return item, true
}

// Enqueue 入队一个任务
func (q *TaskQueue) Enqueue(name string, priority int) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return nil
	}

	q.items = append(q.items, TaskItem{
		Name:     name,
		Priority: priority,
	})
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
