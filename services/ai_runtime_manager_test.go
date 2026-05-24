package services

import (
	"errors"
	"testing"
	"time"
)

func TestAIRuntimeManagerReloadControlsAvailability(t *testing.T) {
	processed := make(chan int, 2)
	manager := NewAIRuntimeManager(func(id int) {
		processed <- id
	})
	t.Cleanup(manager.Close)

	if manager.IsAvailable() {
		t.Fatalf("初始状态不应可用")
	}
	if err := manager.Submit(1); !errors.Is(err, ErrEnhancementUnavailable) {
		t.Fatalf("未启用时应返回 ErrEnhancementUnavailable，实际为 %v", err)
	}

	manager.Reload(AIRuntimeSettings{AIEnabled: true, AsyncEnabled: true, WorkerCount: 2})
	if !manager.IsAvailable() {
		t.Fatalf("启用后运行时应可用")
	}
	if err := manager.Submit(42); err != nil {
		t.Fatalf("投递任务失败: %v", err)
	}

	select {
	case id := <-processed:
		if id != 42 {
			t.Fatalf("预期处理书签 ID 42，实际为 %d", id)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("等待 worker 处理任务超时")
	}

	manager.Reload(AIRuntimeSettings{AIEnabled: true, AsyncEnabled: false, WorkerCount: 2})
	if manager.IsAvailable() {
		t.Fatalf("关闭异步后运行时不应可用")
	}
	if err := manager.Submit(43); !errors.Is(err, ErrEnhancementUnavailable) {
		t.Fatalf("停用后应返回 ErrEnhancementUnavailable，实际为 %v", err)
	}
}

func TestAIRuntimeManagerNormalizesWorkerCount(t *testing.T) {
	manager := NewAIRuntimeManager(func(int) {})
	t.Cleanup(manager.Close)

	manager.Reload(AIRuntimeSettings{AIEnabled: true, AsyncEnabled: true, WorkerCount: 0})
	if !manager.IsAvailable() {
		t.Fatalf("worker_count=0 时应归一化后启动")
	}
}
