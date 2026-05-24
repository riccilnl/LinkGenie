package services

import (
	"log"
	"sync"
)

// AIRuntimeSettings 描述 AI 异步运行时的目标状态
// 这里刻意与配置对象解耦，避免服务层直接依赖配置来源。
type AIRuntimeSettings struct {
	AIEnabled    bool
	AsyncEnabled bool
	WorkerCount  int
}

// AIRuntimeManager 统一管理 AI worker pool 的启停与重载。
type AIRuntimeManager struct {
	mu       sync.RWMutex
	handler  func(int)
	pool     *AIWorkerPool
	settings AIRuntimeSettings
}

// NewAIRuntimeManager 创建 AI 运行时管理器。
func NewAIRuntimeManager(handler func(int)) *AIRuntimeManager {
	return &AIRuntimeManager{handler: handler}
}

// Reload 按目标配置重建异步运行时。
func (m *AIRuntimeManager) Reload(settings AIRuntimeSettings) {
	if m == nil {
		return
	}

	normalized := normalizeAIRuntimeSettings(settings)
	shouldRun := normalized.AIEnabled && normalized.AsyncEnabled

	m.mu.Lock()
	currentPool := m.pool
	currentSettings := m.settings
	if shouldRun && currentPool != nil && currentPool.IsEnabled() && currentSettings == normalized {
		m.mu.Unlock()
		return
	}

	m.settings = normalized
	if !shouldRun {
		m.pool = nil
		m.mu.Unlock()
		if currentPool != nil {
			currentPool.Stop()
		}
		log.Printf("ℹ️ AI 异步运行时已停用: ai_enabled=%v async_enabled=%v", normalized.AIEnabled, normalized.AsyncEnabled)
		return
	}

	newPool := NewAIWorkerPool(normalized.WorkerCount, m.handler)
	newPool.Start()
	m.pool = newPool
	m.mu.Unlock()

	if currentPool != nil {
		currentPool.Stop()
	}
	log.Printf("✅ AI 异步运行时已刷新: workers=%d", normalized.WorkerCount)
}

// Submit 向当前运行时投递任务。
func (m *AIRuntimeManager) Submit(bookmarkID int) error {
	if m == nil {
		return ErrEnhancementUnavailable
	}

	m.mu.RLock()
	pool := m.pool
	m.mu.RUnlock()

	if pool == nil {
		return ErrEnhancementUnavailable
	}

	return pool.Submit(bookmarkID)
}

// IsAvailable 返回当前是否可接收异步任务。
func (m *AIRuntimeManager) IsAvailable() bool {
	if m == nil {
		return false
	}

	m.mu.RLock()
	pool := m.pool
	m.mu.RUnlock()

	return pool != nil && pool.IsEnabled()
}

// Close 关闭当前运行时。
func (m *AIRuntimeManager) Close() {
	if m == nil {
		return
	}

	m.mu.Lock()
	pool := m.pool
	m.pool = nil
	m.mu.Unlock()

	if pool != nil {
		pool.Stop()
	}
}

func normalizeAIRuntimeSettings(settings AIRuntimeSettings) AIRuntimeSettings {
	if settings.WorkerCount <= 0 {
		settings.WorkerCount = 1
	}
	return settings
}
