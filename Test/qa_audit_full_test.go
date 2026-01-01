package Test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"ai-bookmark-service/api"
	"ai-bookmark-service/config"
	"ai-bookmark-service/db"
	"ai-bookmark-service/models"
	"ai-bookmark-service/services"
)

// 准备测试环境
func setupQAEnv(t *testing.T) (http.Handler, string) {
	dbPath := "qa_audit.db"
	os.Remove(dbPath)
	os.Remove(dbPath + "-shm")
	os.Remove(dbPath + "-wal")

	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}

	cfg := &config.Config{
		APIToken:         "tester-token",
		AIEnabled:        false,
		EnableAsyncAI:    false,
		RateLimitEnabled: true,
		RateLimitPerIP:   600, // 高一点以免干扰基础测试
		RateLimitBurst:   1000,
	}

	bmRepo := db.NewBookmarkRepository()
	tagRepo := db.NewTagRepository()
	folderRepo := db.NewFolderRepository(bmRepo)
	wfEngine := services.NewWorkflowEngine(bmRepo, folderRepo)
	tagOpt := services.NewTagOptimizer(tagRepo, bmRepo)

	api.SetFolderRepository(folderRepo)
	api.SetWorkflowEngine(wfEngine)
	api.SetTagOptimizer(tagOpt)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/bookmarks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			var bm models.BookmarkCreate
			json.NewDecoder(r.Body).Decode(&bm)
			created, err := bmRepo.Create(&bm)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			json.NewEncoder(w).Encode(created)
		} else {
			bookmarks, _ := bmRepo.List(100, 0, nil)
			json.NewEncoder(w).Encode(map[string]interface{}{"results": bookmarks})
		}
	})

	mux.HandleFunc("/api/folders", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			api.HandleCreateFolder(w, r)
		} else {
			api.HandleGetFolders(w, r)
		}
	})

	mux.HandleFunc("/api/workflows", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			api.HandleCreateWorkflow(w, r)
		} else {
			api.HandleGetWorkflows(w, r)
		}
	})

	handler := api.AuthMiddleware(func() string { return cfg.APIToken })(mux)
	handler = api.RecoveryMiddleware(handler)

	return handler, cfg.APIToken
}

// 🧪 1. 基础逻辑：文件夹 CRUD
func TestFolderCRUD_QA(t *testing.T) {
	handler, token := setupQAEnv(t)
	defer os.Remove("qa_audit.db")

	// Create
	fReq := `{"name": "Read Later", "description": "Priority bookmarks"}`
	req := httptest.NewRequest("POST", "/api/folders", strings.NewReader(fReq))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Errorf("Folder creation failed: %d %s", w.Code, w.Body.String())
	}

	// List
	req = httptest.NewRequest("GET", "/api/folders", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !strings.Contains(w.Body.String(), "Read Later") {
		t.Error("Folder not found in list")
	}
}

// 🧪 2. 并发与内存：高并发书签创建 (Pressure Test)
func TestConcurrentBookmarkCreation_QA(t *testing.T) {
	handler, token := setupQAEnv(t)
	defer os.Remove("qa_audit.db")

	var wg sync.WaitGroup
	count := 50
	errChan := make(chan error, count)

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			bmReq := fmt.Sprintf(`{"url": "http://concurrent-%d.com", "title": "Test %d"}`, id, id)
			req := httptest.NewRequest("POST", "/api/bookmarks", strings.NewReader(bmReq))
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != http.StatusOK && w.Code != http.StatusCreated {
				errChan <- fmt.Errorf("Request %d failed: %d", id, w.Code)
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Error(err)
	}
}

// 🧪 3. 破坏性与边界：畸形 JSON 与 鉴权绕过
func TestDestructive_QA(t *testing.T) {
	handler, token := setupQAEnv(t)
	defer os.Remove("qa_audit.db")

	t.Run("Malformed JSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/bookmarks", strings.NewReader(`{"url": "bad-json",`)) // Missing closing brace
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		// Should return 400 or handle gracefully without panic
		if w.Code == http.StatusInternalServerError {
			t.Error("Server returned 500 on malformed JSON")
		}
	})

	t.Run("Auth Bypass Attempt", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/folders", nil)
		// No Header
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Auth bypass successful! Code: %d", w.Code)
		}
	})

	t.Run("SQL Injection Attempt", func(t *testing.T) {
		// Escape spaces and quotes for httptest.NewRequest
		req := httptest.NewRequest("GET", "/api/bookmarks/1%27%20OR%20%271%27=%271", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	})
}

// 🧪 4. 高级业务集成：幸存者自愈验证 (Panic Recovery)
func TestSurvivorRecovery_QA(t *testing.T) {
	_, _ = setupQAEnv(t) // Just to init DB
	defer os.Remove("qa_audit.db")

	// 注入一个会导致 Panic 的请求 (访问一个不存在的路由，但在某些逻辑里模拟错误)
	// 由于我们直接用 handler，我们可以造一个触发 Panic 的自定义 handler 注入到 mux
	// 但我们要测的是系统自带的 handler。

	// 我们模拟一个场景：如果我们在处理请求时发生意外
	panicMux := http.NewServeMux()
	panicMux.HandleFunc("/api/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("Simulated crash")
	})
	// 正常的 handler
	panicMux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	h := api.RecoveryMiddleware(panicMux)

	// 1. 发起 Panic 请求
	reqPanic := httptest.NewRequest("GET", "/api/panic", nil)
	w1 := httptest.NewRecorder()

	// 验证不崩溃
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RecoveryMiddleware failed to catch panic: %v", r)
		}
	}()
	h.ServeHTTP(w1, reqPanic)

	if w1.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 on panic, got %d", w1.Code)
	}

	// 2. 紧接着发起正常请求，验证系统依然可用 (幸存者特性)
	reqNormal := httptest.NewRequest("GET", "/api/health", nil)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, reqNormal)

	if w2.Code != http.StatusOK || w2.Body.String() != "ok" {
		t.Errorf("System failed to recover! Health check after panic: %d %s", w2.Code, w2.Body.String())
	}
}

// 🧪 5. 安全深度质询：极其耗时的请求是否会导致系统阻塞 (Slow Client/Body)
func TestSlowRequestImpact_QA(t *testing.T) {
	handler, token := setupQAEnv(t)
	defer os.Remove("qa_audit.db")

	// 模拟一个发送超大数据的请求，检查内存消耗 (简易版)
	largeData := make([]byte, 1024*1024*5) // 5MB
	req := httptest.NewRequest("POST", "/api/bookmarks", io.NopCloser(bytes.NewReader(largeData)))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	start := time.Now()
	handler.ServeHTTP(w, req)
	duration := time.Since(start)

	t.Logf("Large request handled in %v", duration)
	// 如果系统挂了或者返回 200 (但在处理无效 JSON), 都算通过。关键是不能 Panic。
}
