package api

import (
	"net/http"
	"strings"

	"github.com/riccilnl/LinkGenie/serverapp"
)

// NewRouter 统一注册 HTTP 路由和中间件。
func NewRouter(app *serverapp.App, mcpHandler http.Handler) http.Handler {
	systemHandlers := newSystemHandlers(app)
	bookmarkHandlers := newBookmarkHandlers(app)
	tagHandlers := newTagHandlers(app)
	folderHandlers := newFolderHandlers(app.FolderRepository())
	workflowHandlers := newWorkflowHandlers(app.WorkflowEngine())
	tagOptimizerHandlers := newTagOptimizerHandlers(app.TagOptimizer())

	mux := http.NewServeMux()

	mux.HandleFunc("/", app.ServeStatic)
	mux.HandleFunc("/index.html", app.ServeStatic)
	mux.HandleFunc("/sw.js", app.ServeStatic)
	mux.HandleFunc("/manifest.json", app.ServeStatic)
	mux.HandleFunc("/icon.svg", app.ServeStatic)
	mux.HandleFunc("/apple-touch-icon.png", app.ServeStatic)
	mux.HandleFunc("/css/", app.ServeStatic)
	mux.HandleFunc("/js/", app.ServeStatic)

	if mcpHandler != nil {
		mux.Handle("/mcp/", http.StripPrefix("/mcp", mcpHandler))
	}

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		serverapp.EncodeJSON(w, map[string]string{"status": "healthy"})
	})

	mux.HandleFunc("/api/system/status", systemHandlers.handleStatus)
	mux.HandleFunc("/api/system/config", systemHandlers.handleConfig)

	mux.HandleFunc("/api/bookmarks", bookmarkHandlers.handleBookmarks)
	mux.HandleFunc("/api/bookmarks/", func(w http.ResponseWriter, r *http.Request) {
		trimmedPath := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/bookmarks/"), "/")
		if trimmedPath == "" {
			bookmarkHandlers.handleBookmarks(w, r)
			return
		}
		if trimmedPath == "import" {
			bookmarkHandlers.handleImportBookmarks(w, r)
			return
		}
		if trimmedPath == "export" {
			bookmarkHandlers.handleExportBookmarks(w, r)
			return
		}
		if trimmedPath == "check" {
			bookmarkHandlers.handleCheckBookmark(w, r)
			return
		}
		if strings.Contains(trimmedPath, "/folders") {
			switch {
			case r.Method == http.MethodPost && strings.HasSuffix(trimmedPath, "/folders"):
				folderHandlers.handleAddBookmarkToFolders(w, r)
			case r.Method == http.MethodDelete && strings.Contains(trimmedPath, "/folders/"):
				folderHandlers.handleRemoveBookmarkFromFolder(w, r)
			default:
				http.Error(w, "Not found", http.StatusNotFound)
			}
			return
		}
		if strings.HasSuffix(trimmedPath, "/enhance") {
			bookmarkHandlers.handleEnhanceBookmark(w, r)
			return
		}
		bookmarkHandlers.handleBookmarkByID(w, r)
	})

	mux.HandleFunc("/api/tags", tagHandlers.handleTags)
	mux.HandleFunc("/api/tags/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags/" {
			tagHandlers.handleTags(w, r)
			return
		}
		http.Error(w, "Not found", http.StatusNotFound)
	})
	mux.HandleFunc("/api/tags/stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			tagOptimizerHandlers.handleGetTagStats(w, r)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})
	mux.HandleFunc("/api/tags/optimize", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			tagOptimizerHandlers.handleOptimizeTags(w, r)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/api/folders", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			folderHandlers.handleGetFolders(w, r)
		case http.MethodPost:
			folderHandlers.handleCreateFolder(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/folders/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/folders/" {
			switch r.Method {
			case http.MethodGet:
				folderHandlers.handleGetFolders(w, r)
			case http.MethodPost:
				folderHandlers.handleCreateFolder(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}

		switch r.Method {
		case http.MethodPut, http.MethodPatch:
			folderHandlers.handleUpdateFolder(w, r)
		case http.MethodDelete:
			folderHandlers.handleDeleteFolder(w, r)
		case http.MethodGet:
			if strings.HasSuffix(r.URL.Path, "/bookmarks") || strings.HasSuffix(r.URL.Path, "/bookmarks/") {
				folderHandlers.handleGetFolderBookmarks(w, r)
				return
			}
			http.Error(w, "Not found", http.StatusNotFound)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/workflows", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			workflowHandlers.handleGetWorkflows(w, r)
		case http.MethodPost:
			workflowHandlers.handleCreateWorkflow(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/workflows/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/workflows/" {
			switch r.Method {
			case http.MethodGet:
				workflowHandlers.handleGetWorkflows(w, r)
			case http.MethodPost:
				workflowHandlers.handleCreateWorkflow(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}

		switch r.Method {
		case http.MethodPut, http.MethodPatch:
			workflowHandlers.handleUpdateWorkflow(w, r)
		case http.MethodDelete:
			workflowHandlers.handleDeleteWorkflow(w, r)
		case http.MethodPost:
			if strings.HasSuffix(r.URL.Path, "/toggle") {
				workflowHandlers.handleToggleWorkflow(w, r)
				return
			}
			http.Error(w, "Not found", http.StatusNotFound)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/workflows/apply", workflowHandlers.handleApplyWorkflows)

	handler := LoggingMiddleware(mux)
	handler = AuthMiddleware(func() string { return app.ConfigSnapshot().APIToken })(handler)
	handler = RateLimitMiddleware(func() *serverapp.RateLimiter { return app.RateLimiter() })(handler)
	handler = CORSMiddleware(handler)
	handler = RecoveryMiddleware(handler)
	return handler
}
