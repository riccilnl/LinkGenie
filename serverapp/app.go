package serverapp

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/riccilnl/LinkGenie/config"
	"github.com/riccilnl/LinkGenie/db"
	"github.com/riccilnl/LinkGenie/services"
	"github.com/riccilnl/LinkGenie/utils"
)

// App 承载运行期依赖和热重载逻辑。
type App struct {
	mu sync.RWMutex

	cfg *config.Config

	bookmarkRepo   *db.BookmarkRepository
	tagRepo        *db.TagRepository
	folderRepo     *db.FolderRepository
	workflowEngine *services.WorkflowEngine
	tagOptimizer   *services.TagOptimizer
	scraperService *services.ScraperService
	aiService      *services.AIService
	aiRuntime      *services.AIRuntimeManager
	bookmarkSvc    *services.BookmarkService
	bookmarkExchg  *services.BookmarkExchangeService
	rateLimiter    *RateLimiter

	staticRoot string
}

// New 创建应用依赖容器。
func New(cfg *config.Config, staticRoot string) *App {
	if cfg == nil {
		cfg = &config.Config{
			RateLimitPerIP: 60,
			RateLimitBurst: 10,
			AIWorkerCount:  1,
		}
	}
	if staticRoot == "" {
		staticRoot = "."
	}

	app := &App{
		cfg:          cfg.Clone(),
		bookmarkRepo: db.NewBookmarkRepository(),
		tagRepo:      db.NewTagRepository(),
		staticRoot:   staticRoot,
	}
	app.folderRepo = db.NewFolderRepository(app.bookmarkRepo)
	app.scraperService = services.NewScraperService()
	app.aiService = services.NewAIService(app.cfg, app.scraperService)
	app.workflowEngine = services.NewWorkflowEngine(app.bookmarkRepo, app.folderRepo)
	app.tagOptimizer = services.NewTagOptimizer(app.tagRepo, app.bookmarkRepo)
	app.aiRuntime = services.NewAIRuntimeManager(app.EnhanceBookmarkAsync)
	app.refreshRateLimiterLocked()
	app.refreshAIRuntimeLocked()
	app.refreshBookmarkServiceLocked()
	app.refreshBookmarkExchangeLocked()

	return app
}

// Close 关闭运行期资源。
func (a *App) Close() {
	a.mu.Lock()
	runtime := a.aiRuntime
	a.mu.Unlock()

	if runtime != nil {
		runtime.Close()
	}
}

// ConfigSnapshot 返回当前配置快照。
func (a *App) ConfigSnapshot() *config.Config {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.cfg.Clone()
}

// BookmarkRepository 返回书签仓库。
func (a *App) BookmarkRepository() *db.BookmarkRepository {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.bookmarkRepo
}

// TagRepository 返回标签仓库。
func (a *App) TagRepository() *db.TagRepository {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.tagRepo
}

// FolderRepository 返回文件夹仓库。
func (a *App) FolderRepository() *db.FolderRepository {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.folderRepo
}

// WorkflowEngine 返回工作流引擎。
func (a *App) WorkflowEngine() *services.WorkflowEngine {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.workflowEngine
}

// TagOptimizer 返回标签优化服务。
func (a *App) TagOptimizer() *services.TagOptimizer {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.tagOptimizer
}

// ScraperService 返回网页抓取服务。
func (a *App) ScraperService() *services.ScraperService {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.scraperService
}

// AIService 返回 AI 服务。
func (a *App) AIService() *services.AIService {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.aiService
}

// AIRuntime 返回 AI 异步运行时。
func (a *App) AIRuntime() *services.AIRuntimeManager {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.aiRuntime
}

// BookmarkService 返回书签服务。
func (a *App) BookmarkService() *services.BookmarkService {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.bookmarkSvc
}

// BookmarkExchangeService 返回书签导入导出服务。
func (a *App) BookmarkExchangeService() *services.BookmarkExchangeService {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.bookmarkExchg
}

// RateLimiter 返回当前限流器。
func (a *App) RateLimiter() *RateLimiter {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.rateLimiter
}

// IsAIRuntimeAvailable 返回当前 AI 队列是否可用。
func (a *App) IsAIRuntimeAvailable() bool {
	a.mu.RLock()
	runtime := a.aiRuntime
	a.mu.RUnlock()
	return runtime != nil && runtime.IsAvailable()
}

// SaveSystemConfig 持久化并热重载系统配置。
func (a *App) SaveSystemConfig(overrides map[string]string) error {
	nextCfg := a.ConfigSnapshot()
	if err := nextCfg.ApplyOverrides(overrides); err != nil {
		return err
	}
	if err := nextCfg.Validate(); err != nil {
		return err
	}

	for key, value := range overrides {
		if _, err := db.DB.Exec("INSERT OR REPLACE INTO system_configs (key, value) VALUES (?, ?)", key, value); err != nil {
			return err
		}
	}

	return a.ReloadConfigFromDB()
}

// ReloadConfigFromDB 从数据库重载配置并刷新运行时依赖。
func (a *App) ReloadConfigFromDB() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	nextCfg := a.cfg.Clone()
	if err := nextCfg.LoadFromDB(db.DB); err != nil {
		return err
	}

	a.cfg = nextCfg
	a.aiService = services.NewAIService(a.cfg, a.scraperService)
	a.refreshRateLimiterLocked()
	a.refreshAIRuntimeLocked()
	a.refreshBookmarkServiceLocked()
	a.refreshBookmarkExchangeLocked()
	return nil
}

// ServeStatic 提供静态资源。
func (a *App) ServeStatic(w http.ResponseWriter, r *http.Request) {
	cleanPath := path.Clean("/" + r.URL.Path)
	if cleanPath == "/" {
		cleanPath = "/index.html"
	}

	switch cleanPath {
	case "/index.html", "/sw.js", "/manifest.json":
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
	}

	fullPath := filepath.Join(a.staticRoot, filepath.FromSlash(strings.TrimPrefix(cleanPath, "/")))
	http.ServeFile(w, r, fullPath)
}

// EnhanceBookmarkAsync 执行后台 AI 增强。
func (a *App) EnhanceBookmarkAsync(bookmarkID int) {
	log.Printf("🔄 后台任务开始: 增强书签 ID=%d", bookmarkID)

	bookmarkRepo := a.BookmarkRepository()
	bookmarkSvc := a.BookmarkService()
	aiService := a.AIService()
	if bookmarkRepo == nil || bookmarkSvc == nil || aiService == nil {
		log.Printf("❌ 后台任务依赖未就绪: bookmark_id=%d", bookmarkID)
		return
	}

	bm, err := bookmarkRepo.GetByID(bookmarkID)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Printf("❌ 后台任务: 获取书签失败 ID=%d, 错误: %v", bookmarkID, err)
		} else {
			log.Printf("❌ 后台任务: 书签不存在 ID=%d", bookmarkID)
		}
		return
	}

	log.Printf("🤖 触发AI增强: Title='%s' Desc='%s'", bm.Title, bm.Description)
	aiResp, err := aiService.Enhance(bm.URL, bm.Title, bm.Description)
	if err != nil {
		log.Printf("⚠️ 后台AI增强失败: %v", err)
		return
	}

	patchInput := services.PatchBookmarkInput{}
	needsUpdate := false

	if aiResp.Title != "" {
		title := aiResp.Title
		patchInput.Title = &title
		needsUpdate = true
		log.Printf("✨ AI优化标题: %s", aiResp.Title)
	}
	if aiResp.Description != "" {
		description := aiResp.Description
		patchInput.Description = &description
		needsUpdate = true
		log.Printf("✨ AI优化描述: %s", aiResp.Description[:utils.Min(150, len(aiResp.Description))])
	}
	if len(aiResp.Tags) > 0 {
		tagMap := make(map[string]bool, len(bm.TagNames)+len(aiResp.Tags))
		tagNames := append([]string{}, bm.TagNames...)
		for _, tag := range tagNames {
			tagMap[tag] = true
		}
		for _, tag := range aiResp.Tags {
			if !tagMap[tag] {
				tagNames = append(tagNames, tag)
				tagMap[tag] = true
			}
		}
		patchInput.TagNames = &tagNames
		needsUpdate = true
		log.Printf("✨ AI添加标签: %v", aiResp.Tags)
	}

	if !needsUpdate {
		log.Printf("ℹ️ 后台任务完成: 无需更新 ID=%d", bookmarkID)
		return
	}

	if _, err := bookmarkSvc.PatchBookmark(context.Background(), bookmarkID, patchInput); err != nil {
		log.Printf("❌ 后台任务更新失败: %v", err)
		return
	}

	log.Printf("✅ 后台任务完成: 书签已更新 ID=%d", bookmarkID)
}

func (a *App) refreshBookmarkServiceLocked() {
	var enqueueEnhancement func(int) error
	if a.aiRuntime != nil {
		enqueueEnhancement = func(id int) error {
			return a.aiRuntime.Submit(id)
		}
	}

	a.bookmarkSvc = services.NewBookmarkService(
		a.bookmarkRepo,
		a.tagRepo,
		a.folderRepo,
		a.workflowEngine,
		enqueueEnhancement,
	)
}

func (a *App) refreshBookmarkExchangeLocked() {
	a.bookmarkExchg = services.NewBookmarkExchangeService(
		a.bookmarkSvc,
		a.bookmarkRepo,
		a.folderRepo,
	)
}

func (a *App) refreshAIRuntimeLocked() {
	if a.aiRuntime == nil {
		return
	}

	a.aiRuntime.Reload(services.AIRuntimeSettings{
		AIEnabled:    a.cfg.AIEnabled,
		AsyncEnabled: a.cfg.EnableAsyncAI,
		WorkerCount:  a.cfg.AIWorkerCount,
	})
}

func (a *App) refreshRateLimiterLocked() {
	if !a.cfg.RateLimitEnabled {
		a.rateLimiter = nil
		return
	}

	a.rateLimiter = NewRateLimiter(a.cfg.RateLimitPerIP, a.cfg.RateLimitBurst)
}

// MarshalStatus 生成系统状态响应。
func (a *App) MarshalStatus() map[string]interface{} {
	cfg := a.ConfigSnapshot()
	bookmarkCount := 0
	dbStatus := "connected"
	if repo := a.BookmarkRepository(); repo != nil {
		count, err := repo.Count(nil)
		if err != nil {
			dbStatus = "error"
		} else {
			bookmarkCount = count
		}
	} else {
		dbStatus = "error"
	}

	return map[string]interface{}{
		"status":               "ok",
		"database":             dbStatus,
		"bookmarks_count":      bookmarkCount,
		"ai_enabled":           cfg.AIEnabled,
		"enable_async_ai":      cfg.EnableAsyncAI,
		"ai_worker_count":      cfg.AIWorkerCount,
		"ai_runtime_available": a.IsAIRuntimeAvailable(),
		"initialized":          bookmarkCount > 0,
	}
}

// MarshalConfig 生成系统配置响应。
func (a *App) MarshalConfig() map[string]interface{} {
	cfg := a.ConfigSnapshot()
	return map[string]interface{}{
		"ai_enabled":           cfg.AIEnabled,
		"enable_async_ai":      cfg.EnableAsyncAI,
		"ai_worker_count":      cfg.AIWorkerCount,
		"ai_runtime_available": a.IsAIRuntimeAvailable(),
		"ai_endpoint":          cfg.AIEndpoint,
		"ai_model":             cfg.AIModel,
		"ai_api_key_set":       cfg.AIAPIKey != "",
	}
}

// EncodeJSON 统一 JSON 输出。
func EncodeJSON(w http.ResponseWriter, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
