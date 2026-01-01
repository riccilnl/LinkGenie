package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"ai-bookmark-service/api"
	"ai-bookmark-service/config"
	"ai-bookmark-service/db"
	"ai-bookmark-service/mcp"
	"ai-bookmark-service/models"
	"ai-bookmark-service/services"
	"ai-bookmark-service/utils"

	"github.com/mark3labs/mcp-go/server"
)

var (
	cfg            *config.Config
	bookmarkRepo   *db.BookmarkRepository
	tagRepo        *db.TagRepository
	folderRepo     *db.FolderRepository
	aiService      *services.AIService
	scraperService *services.ScraperService
	workflowEngine *services.WorkflowEngine
	tagOptimizer   *services.TagOptimizer
	rateLimiter    *api.RateLimiter
	aiWorkerPool   *services.AIWorkerPool
)

func main() {
	// 1. 加载配置
	var err error
	cfg, err = config.Load()
	if err != nil {
		log.Fatalf("❌ 加载配置失败: %v", err)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		log.Printf("⚠️ 配置验证警告: %v", err)
	}

	log.Printf("✅ 配置加载成功")
	log.Printf("📊 AI启用: %v", cfg.AIEnabled)
	log.Printf("📊 异步AI: %v", cfg.EnableAsyncAI)
	log.Printf("📊 限流启用: %v", cfg.RateLimitEnabled)

	// 2. 初始化数据库
	if err := db.Init(cfg.DBPath); err != nil {
		log.Fatalf("❌ 数据库初始化失败: %v", err)
	}
	defer db.Close()

	// 加载动态配置
	if err := cfg.LoadFromDB(db.DB); err != nil {
		log.Printf("⚠️ 从数据库加载动态配置失败: %v", err)
	}

	// 3. 初始化仓库
	bookmarkRepo = db.NewBookmarkRepository()
	tagRepo = db.NewTagRepository()
	folderRepo = db.NewFolderRepository(bookmarkRepo)

	// 4. 初始化服务
	scraperService = services.NewScraperService()
	aiService = services.NewAIService(cfg, scraperService)
	workflowEngine = services.NewWorkflowEngine(bookmarkRepo, folderRepo)
	tagOptimizer = services.NewTagOptimizer(tagRepo, bookmarkRepo)

	// 5. 设置 API 处理器依赖
	api.SetFolderRepository(folderRepo)
	api.SetWorkflowEngine(workflowEngine)
	api.SetTagOptimizer(tagOptimizer)

	// 6. 初始化限流器
	if cfg.RateLimitEnabled {
		rateLimiter = api.NewRateLimiter(cfg.RateLimitPerIP, cfg.RateLimitBurst)
	}

	// 7. 初始化 AI Worker Pool
	aiWorkerPool = services.NewAIWorkerPool(cfg.AIWorkerCount, enhanceBookmarkAsync)
	if cfg.AIEnabled && cfg.EnableAsyncAI {
		aiWorkerPool.Start()
		defer aiWorkerPool.Stop()
	}

	// 8. 初始化 MCP 服务器
	mcpSrv := mcp.NewMCPServer(bookmarkRepo, tagRepo, folderRepo, scraperService)
	httpServer := server.NewStreamableHTTPServer(mcpSrv.Server())
	log.Printf("✅ MCP 服务器初始化成功")

	// 8. 设置路由
	mux := http.NewServeMux()

	// 静态文件
	mux.HandleFunc("/", serveStatic)
	mux.HandleFunc("/index.html", serveStatic)
	mux.HandleFunc("/sw.js", serveStatic)
	mux.HandleFunc("/manifest.json", serveStatic)
	mux.HandleFunc("/icon.svg", serveStatic)

	// CSS 和 JS 模块 (重构后新增)
	mux.HandleFunc("/css/", serveStatic)
	mux.HandleFunc("/js/", serveStatic)

	// MCP HTTP 端点 - 使用 StreamableHTTPServer
	mux.Handle("/mcp/", http.StripPrefix("/mcp", httpServer))

	// 系统状态端点 (用于引导页)
	mux.HandleFunc("/api/system/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		dbStatus := "connected"
		bookmarkCount, err := bookmarkRepo.Count(nil)
		if err != nil {
			dbStatus = "error"
		}

		status := map[string]interface{}{
			"status":          "ok",
			"database":        dbStatus,
			"bookmarks_count": bookmarkCount,
			"ai_enabled":      cfg.AIEnabled,
			"initialized":     bookmarkCount > 0,
		}

		json.NewEncoder(w).Encode(status)
	})

	// 系统配置端点 (支持热重载)
	mux.HandleFunc("/api/system/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			aiKeySet := cfg.AIAPIKey != ""
			json.NewEncoder(w).Encode(map[string]interface{}{
				"ai_enabled":     cfg.AIEnabled,
				"ai_endpoint":    cfg.AIEndpoint,
				"ai_model":       cfg.AIModel,
				"ai_api_key_set": aiKeySet,
			})
			return
		}

		if r.Method == http.MethodPost {
			var newConfig map[string]string
			if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// 持久化到数据库
			for k, v := range newConfig {
				_, err := db.DB.Exec("INSERT OR REPLACE INTO system_configs (key, value) VALUES (?, ?)", k, v)
				if err != nil {
					log.Printf("❌ 无法保存配置 %s: %v", k, err)
				}
			}

			// 内存重载
			if err := cfg.LoadFromDB(db.DB); err != nil {
				log.Printf("⚠️ 内存重载失败: %v", err)
			}

			// 刷新 AI 服务
			aiService = services.NewAIService(cfg, scraperService)
			log.Printf("✅ 系统配置已更新并热重载")

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	// 健康检查端点(不需要认证)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	// API 路由
	mux.HandleFunc("/api/bookmarks", handleBookmarks)
	mux.HandleFunc("/api/bookmarks/", func(w http.ResponseWriter, r *http.Request) {
		// /api/bookmarks/ without ID should list all bookmarks
		if r.URL.Path == "/api/bookmarks/" {
			handleBookmarks(w, r)
			return
		}

		// Check if it's /api/bookmarks/check/ (Linkding validation)
		if r.URL.Path == "/api/bookmarks/check/" || r.URL.Path == "/api/bookmarks/check" {
			handleCheckBookmark(w, r)
			return
		}

		// Check if it's /api/bookmarks/{id}/enhance/
		if len(r.URL.Path) > 9 && r.URL.Path[len(r.URL.Path)-9:] == "/enhance/" {
			handleEnhanceBookmark(w, r)
			return
		}

		// /api/bookmarks/{id}
		handleBookmarkByID(w, r)
	})
	mux.HandleFunc("/api/tags", handleTags)
	mux.HandleFunc("/api/tags/stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			api.HandleGetTagStats(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/tags/optimize", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			api.HandleOptimizeTags(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Folders API (from folder_api.go)
	mux.HandleFunc("/api/folders", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			api.HandleGetFolders(w, r)
		case "POST":
			api.HandleCreateFolder(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/folders/", func(w http.ResponseWriter, r *http.Request) {
		// /api/folders/ without ID should list all folders
		if r.URL.Path == "/api/folders/" {
			switch r.Method {
			case "GET":
				api.HandleGetFolders(w, r)
			case "POST":
				api.HandleCreateFolder(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}

		// /api/folders/{id} or /api/folders/{id}/...
		switch r.Method {
		case "PUT", "PATCH":
			api.HandleUpdateFolder(w, r)
		case "DELETE":
			api.HandleDeleteFolder(w, r)
		case "GET":
			// Check if it's /api/folders/{id}/bookmarks
			if len(r.URL.Path) > 13 && r.URL.Path[len(r.URL.Path)-10:] == "/bookmarks" {
				api.HandleGetFolderBookmarks(w, r)
			} else {
				http.Error(w, "Not found", http.StatusNotFound)
			}
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Workflows API (from workflow_api.go)
	mux.HandleFunc("/api/workflows", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			api.HandleGetWorkflows(w, r)
		case "POST":
			api.HandleCreateWorkflow(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/workflows/", func(w http.ResponseWriter, r *http.Request) {
		// /api/workflows/ without ID should list all workflows or create new workflow
		if r.URL.Path == "/api/workflows/" {
			switch r.Method {
			case "GET":
				api.HandleGetWorkflows(w, r)
			case "POST":
				api.HandleCreateWorkflow(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}

		// /api/workflows/{id} or /api/workflows/{id}/...
		switch r.Method {
		case "PUT", "PATCH":
			api.HandleUpdateWorkflow(w, r)
		case "DELETE":
			api.HandleDeleteWorkflow(w, r)
		case "POST":
			// Check if it's /api/workflows/{id}/toggle
			if len(r.URL.Path) > 7 && r.URL.Path[len(r.URL.Path)-7:] == "/toggle" {
				api.HandleToggleWorkflow(w, r)
			} else {
				http.Error(w, "Not found", http.StatusNotFound)
			}
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/workflows/apply", api.HandleApplyWorkflows)

	// 9. 应用中间件
	handler := api.LoggingMiddleware(mux)
	handler = api.AuthMiddleware(func() string { return cfg.APIToken })(handler)
	handler = api.RateLimitMiddleware(rateLimiter)(handler)
	handler = api.CORSMiddleware(handler) // CORS 必须在最外层
	handler = api.RecoveryMiddleware(handler)

	// 10. 启动服务器
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 服务器启动: http://localhost:%s", port)
	log.Printf("📚 REST API: http://localhost:%s/api/bookmarks", port)
	log.Printf("🔗 MCP 端点: http://localhost:%s/mcp", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("❌ 服务器启动失败: %v", err)
	}
}

// serveStatic 提供静态文件
func serveStatic(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}

	http.ServeFile(w, r, "."+path)
}

// handleBookmarks 处理书签列表和创建
func handleBookmarks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		listBookmarks(w, r)
	case "POST":
		createBookmark(w, r)
	default:
		http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
	}
}

// listBookmarks 获取书签列表
func listBookmarks(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	query := r.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 100
	}

	offset, _ := strconv.Atoi(query.Get("offset"))
	if offset < 0 {
		offset = 0
	}

	// 构建过滤器
	filters := make(map[string]interface{})
	if q := query.Get("q"); q != "" {
		filters["q"] = q
	}
	if query.Get("unread") == "true" {
		filters["unread"] = true
	}
	if query.Get("shared") == "true" {
		filters["shared"] = true
	}

	// 查询书签
	bookmarks, err := bookmarkRepo.List(limit, offset, filters)
	if err != nil {
		log.Printf("❌ 查询书签失败: %v", err)
		http.Error(w, "查询失败", http.StatusInternalServerError)
		return
	}

	// 确保 bookmarks 不是 nil
	if bookmarks == nil {
		bookmarks = []*models.Bookmark{}
	}

	// 统计总数
	count, _ := bookmarkRepo.Count(filters)

	// 返回结果
	response := map[string]interface{}{
		"count":    count,
		"next":     nil,
		"previous": nil,
		"results":  bookmarks,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// createBookmark 创建书签
func createBookmark(w http.ResponseWriter, r *http.Request) {
	var bm models.BookmarkCreate
	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "multipart/form-data") || strings.Contains(contentType, "application/x-www-form-urlencoded") {
		// 1. 处理表单提交 (Linkdy 模式)
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			log.Printf("❌ 解析表单失败: %v", err)
			http.Error(w, "无效的表单数据", http.StatusBadRequest)
			return
		}

		bm.URL = r.FormValue("url")
		bm.Title = r.FormValue("title")
		bm.Description = r.FormValue("description")
		bm.Notes = r.FormValue("notes")
		bm.Unread = r.FormValue("unread") == "true" || r.FormValue("unread") == "1"
		bm.Shared = r.FormValue("shared") == "true" || r.FormValue("shared") == "1"
		bm.IsArchived = r.FormValue("is_archived") == "true" || r.FormValue("is_archived") == "1"
		bm.IsFavorite = r.FormValue("is_favorite") == "true" || r.FormValue("is_favorite") == "1"

		// 处理标签 (Linkdy 发送逗号分隔字符串)
		tagNames := r.FormValue("tag_names")
		if tagNames == "" {
			tagNames = r.FormValue("tags")
		}
		if tagNames != "" {
			parts := strings.Split(tagNames, ",")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					bm.TagNames = append(bm.TagNames, p)
				}
			}
		}
	} else {
		// 2. 处理 JSON 提交 (标准模式)
		var raw map[string]interface{}
		bodyBytes, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(bodyBytes, &raw); err != nil {
			log.Printf("❌ JSON解析失败: %v, Body: %s", err, string(bodyBytes))
			http.Error(w, "无效的JSON数据", http.StatusBadRequest)
			return
		}

		if v, ok := raw["url"].(string); ok {
			bm.URL = v
		}
		if v, ok := raw["title"].(string); ok {
			bm.Title = v
		}
		if v, ok := raw["description"].(string); ok {
			bm.Description = v
		}
		if v, ok := raw["notes"].(string); ok {
			bm.Notes = v
		}
		if v, ok := raw["is_favorite"].(bool); ok {
			bm.IsFavorite = v
		}
		if v, ok := raw["unread"].(bool); ok {
			bm.Unread = v
		}
		if v, ok := raw["shared"].(bool); ok {
			bm.Shared = v
		}
		if v, ok := raw["is_archived"].(bool); ok {
			bm.IsArchived = v
		}

		if tags, ok := raw["tag_names"].([]interface{}); ok {
			for _, t := range tags {
				if ts, ok := t.(string); ok {
					bm.TagNames = append(bm.TagNames, ts)
				}
			}
		} else if tagStr, ok := raw["tag_names"].(string); ok {
			parts := strings.Split(tagStr, ",")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					bm.TagNames = append(bm.TagNames, p)
				}
			}
		}
		if tags, ok := raw["tags"].([]interface{}); ok {
			for _, t := range tags {
				if ts, ok := t.(string); ok {
					bm.TagNames = append(bm.TagNames, ts)
				}
			}
		}
	}

	// 映射归档到收藏
	if bm.IsArchived {
		bm.IsFavorite = true
	}

	// 自动截断
	if len(bm.Title) > 200 {
		bm.Title = bm.Title[:197] + "..."
	}
	if len(bm.Description) > 1000 {
		bm.Description = bm.Description[:997] + "..."
	}

	// 验证并创建
	if err := utils.ValidateBookmarkCreate(&bm); err != nil {
		log.Printf("⚠️ 验证失败: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	created, err := bookmarkRepo.Create(&bm)
	if err != nil {
		log.Printf("❌ 创建书签失败: %v", err)
		http.Error(w, "创建失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if cfg.EnableAsyncAI {
		aiWorkerPool.Submit(created.ID)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

// handleCheckBookmark 检查URL是否已保存并返回元数据 (Linkding 兼容)
func handleCheckBookmark(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	urlStr := r.URL.Query().Get("url")
	if urlStr == "" {
		http.Error(w, "缺少url参数", http.StatusBadRequest)
		return
	}

	// 规范化URL
	normalizedURL, err := utils.NormalizeURL(urlStr)
	if err != nil {
		// 如果无法规范化，返回未保存即可，不要报错，避免拦截客户端
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"already_bookmarked": false,
			"bookmark_id":        nil,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")

	bm, err := bookmarkRepo.GetByURL(normalizedURL)
	if err == nil {
		// 已存在
		json.NewEncoder(w).Encode(map[string]interface{}{
			"already_bookmarked": true,
			"bookmark_id":        bm.ID,
			"metadata": map[string]string{
				"url":         bm.URL,
				"title":       bm.Title,
				"description": bm.Description,
			},
		})
		return
	}

	// 不存在，尝试快速抓取元数据以供客户端预填充
	// 这里使用 ScraperService, 但增加较短的超时，避免阻塞客户端过久
	metadata, err := scraperService.ScrapeWebPage(normalizedURL)
	if err != nil {
		// 抓取失败也返回 200，只是 metadata 里的内容不全
		json.NewEncoder(w).Encode(map[string]interface{}{
			"already_bookmarked": false,
			"bookmark_id":        nil,
			"metadata": map[string]string{
				"url": normalizedURL,
			},
		})
		return
	}

	// 返回抓取到的元数据
	json.NewEncoder(w).Encode(map[string]interface{}{
		"already_bookmarked": false,
		"bookmark_id":        nil,
		"metadata": map[string]string{
			"url":         normalizedURL,
			"title":       metadata.Title,
			"description": metadata.Description,
		},
	})
}

// handleBookmarkByID 处理单个书签操作
func handleBookmarkByID(w http.ResponseWriter, r *http.Request) {
	// 提取ID
	idStr := r.URL.Path[len("/api/bookmarks/"):]
	// 去除末尾的斜杠(如果有)
	idStr = strings.TrimSuffix(idStr, "/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "无效的ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		getBookmark(w, r, id)
	case "PATCH", "PUT":
		updateBookmark(w, r, id)
	case "DELETE":
		deleteBookmark(w, r, id)
	default:
		http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
	}
}

// getBookmark 获取单个书签
func getBookmark(w http.ResponseWriter, r *http.Request, id int) {
	bookmark, err := bookmarkRepo.GetByID(id)
	if err != nil {
		http.Error(w, "书签不存在", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bookmark)
}

// updateBookmark 更新书签
func updateBookmark(w http.ResponseWriter, r *http.Request, id int) {
	var bm models.BookmarkCreate
	if err := json.NewDecoder(r.Body).Decode(&bm); err != nil {
		http.Error(w, "无效的请求数据", http.StatusBadRequest)
		return
	}

	// 验证输入
	if err := utils.ValidateBookmarkCreate(&bm); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 更新书签
	updated, err := bookmarkRepo.Update(id, &bm)
	if err != nil {
		log.Printf("❌ 更新书签失败: %v", err)
		http.Error(w, "更新失败", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

// deleteBookmark 删除书签
func deleteBookmark(w http.ResponseWriter, r *http.Request, id int) {
	if err := bookmarkRepo.Delete(id); err != nil {
		log.Printf("❌ 删除书签失败: %v", err)
		http.Error(w, "删除失败", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleTags 处理标签列表
func handleTags(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	tags, err := tagRepo.List()
	if err != nil {
		log.Printf("❌ 查询标签失败: %v", err)
		http.Error(w, "查询失败", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tags)
}

// handleEnhanceBookmark 手动触发AI增强
func handleEnhanceBookmark(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	// 提取ID from /api/bookmarks/{id}/enhance/
	path := r.URL.Path
	// Remove /api/bookmarks/ prefix and /enhance/ suffix
	idStr := path[len("/api/bookmarks/") : len(path)-len("/enhance/")]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "无效的ID", http.StatusBadRequest)
		return
	}

	// 检查AI是否启用
	if !cfg.AIEnabled {
		http.Error(w, "AI功能未启用", http.StatusServiceUnavailable)
		return
	}

	// 异步触发AI增强
	aiWorkerPool.Submit(id)

	// 立即返回成功响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "AI增强已开始处理",
		"id":      id,
	})
}

// enhanceBookmarkAsync 异步增强书签
func enhanceBookmarkAsync(bookmarkID int) {
	log.Printf("🔄 后台任务开始: 增强书签 ID=%d", bookmarkID)

	// 获取书签
	bm, err := bookmarkRepo.GetByID(bookmarkID)
	if err != nil {
		log.Printf("❌ 后台任务: 书签不存在 ID=%d, 错误: %v", bookmarkID, err)
		return
	}

	// AI增强
	log.Printf("🤖 触发AI增强: Title='%s' Desc='%s'", bm.Title, bm.Description)
	aiResp, err := aiService.Enhance(bm.URL)
	if err != nil {
		log.Printf("⚠️ 后台AI增强失败: %v", err)
		return
	}

	// 更新书签
	needsUpdate := false
	updateReq := &models.BookmarkCreate{
		URL:         bm.URL,
		Title:       bm.Title,
		Description: bm.Description,
		Notes:       bm.Notes,
		IsFavorite:  bm.IsFavorite,
		Unread:      bm.Unread,
		Shared:      bm.Shared,
		TagNames:    bm.TagNames,
	}

	if aiResp.Title != "" {
		updateReq.Title = aiResp.Title
		needsUpdate = true
		log.Printf("✨ AI优化标题: %s", aiResp.Title)
	}

	if aiResp.Description != "" {
		updateReq.Description = aiResp.Description
		needsUpdate = true
		log.Printf("✨ AI优化描述: %s", aiResp.Description[:utils.Min(150, len(aiResp.Description))])
	}

	if len(aiResp.Tags) > 0 {
		// 合并标签（去重）
		tagMap := make(map[string]bool)
		for _, tag := range updateReq.TagNames {
			tagMap[tag] = true
		}
		for _, tag := range aiResp.Tags {
			if !tagMap[tag] {
				updateReq.TagNames = append(updateReq.TagNames, tag)
				tagMap[tag] = true
			}
		}
		needsUpdate = true
		log.Printf("✨ AI添加标签: %v", aiResp.Tags)
	}

	if needsUpdate {
		_, err := bookmarkRepo.Update(bookmarkID, updateReq)
		if err != nil {
			log.Printf("❌ 后台任务更新失败: %v", err)
		} else {
			log.Printf("✅ 后台任务完成: 书签已更新 ID=%d", bookmarkID)
		}
	} else {
		log.Printf("ℹ️ 后台任务完成: 无需更新 ID=%d", bookmarkID)
	}
}
