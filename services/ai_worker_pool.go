package services

import (
	"log"
	"sync"
)

// AIWorkerPool AI 任务工作池
type AIWorkerPool struct {
	taskChan    chan int
	workerCount int
	wg          sync.WaitGroup
	handler     func(int) // 实际处理任务的函数
	stopChan    chan struct{}
	enabled     bool
}

// NewAIWorkerPool 创建一个新的工作池
func NewAIWorkerPool(workerCount int, handler func(int)) *AIWorkerPool {
	if workerCount <= 0 {
		workerCount = 1
	}
	return &AIWorkerPool{
		taskChan:    make(chan int, 1000), // 缓冲队列，防止短时间流量高峰
		workerCount: workerCount,
		handler:     handler,
		stopChan:    make(chan struct{}),
	}
}

// Start 启动工作池
func (p *AIWorkerPool) Start() {
	if p.enabled {
		return
	}
	log.Printf("🧵 AI Worker Pool 启动: %d workers", p.workerCount)
	p.enabled = true
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// Submit 提交任务
func (p *AIWorkerPool) Submit(bookmarkID int) {
	if !p.enabled {
		log.Printf("ℹ️ AI Worker Pool 未启动，跳过任务: %d", bookmarkID)
		return
	}
	select {
	case p.taskChan <- bookmarkID:
		// 成功入队
	default:
		log.Printf("⚠️ AI 任务队列已满 (size=1000)，忽略书签 ID: %d", bookmarkID)
	}
}

// Stop 停止工作池
func (p *AIWorkerPool) Stop() {
	close(p.stopChan)
	close(p.taskChan)
	p.wg.Wait()
	log.Printf("🛑 AI Worker Pool 已停止")
}

func (p *AIWorkerPool) worker(id int) {
	defer p.wg.Done()
	log.Printf("👷 Worker %d 准备就绪", id)
	for {
		select {
		case bookmarkID, ok := <-p.taskChan:
			if !ok {
				return
			}
			// 执行繁重的 AI 处理逻辑
			p.handler(bookmarkID)
		case <-p.stopChan:
			return
		}
	}
}
