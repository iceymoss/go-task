package engine

// TaskItem 表示队列中的一个任务
type TaskItem struct {
	Name     string // 任务名称
	Priority int    // 优先级，值越大优先级越高
}

// 定义一个基于 TaskItem 切片的类型，用于实现堆接口
type priorityQueue []TaskItem

// Len 获取长度
func (pq *priorityQueue) Len() int { return len(*pq) }

// Less 决定了堆的排序方式。
// 我们希望优先级高的排在前面（最大堆），所以这里用大于号 (>)
func (pq *priorityQueue) Less(i, j int) bool {
	return (*pq)[i].Priority > (*pq)[j].Priority
}

// Swap 交换两个元素
func (pq *priorityQueue) Swap(i, j int) {
	(*pq)[i], (*pq)[j] = (*pq)[j], (*pq)[i]
}

// Push 往切片末尾追加元素
func (pq *priorityQueue) Push(x any) {
	*pq = append(*pq, x.(TaskItem))
}

// Pop 弹出切片最后一个元素
func (pq *priorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]

	// 防止内存泄漏, 如果 TaskItem 中有指针（包括 string 底层的指针），将其置空可以切断引用，帮助 GC 回收
	old[n-1] = TaskItem{}

	// 缩小切片
	*pq = old[0 : n-1]
	return item
}
