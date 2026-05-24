package main

import (
	"log"
	"net/http"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/riccilnl/LinkGenie/api"
	"github.com/riccilnl/LinkGenie/config"
	"github.com/riccilnl/LinkGenie/db"
	"github.com/riccilnl/LinkGenie/mcp"
	"github.com/riccilnl/LinkGenie/serverapp"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ 加载配置失败: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		log.Printf("⚠️ 配置验证警告: %v", err)
	}

	log.Printf("✅ 配置加载成功")
	log.Printf("📊 AI启用: %v", cfg.AIEnabled)
	log.Printf("📊 异步AI: %v", cfg.EnableAsyncAI)
	log.Printf("📊 限流启用: %v", cfg.RateLimitEnabled)

	if err := db.Init(cfg.DBPath); err != nil {
		log.Fatalf("❌ 数据库初始化失败: %v", err)
	}
	defer db.Close()

	if err := cfg.LoadFromDB(db.DB); err != nil {
		log.Printf("⚠️ 从数据库加载动态配置失败: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		log.Printf("⚠️ 动态配置验证警告: %v", err)
	}

	app := serverapp.New(cfg, ".")
	defer app.Close()

	mcpSrv := mcp.NewMCPServer(
		app.BookmarkRepository(),
		app.TagRepository(),
		app.FolderRepository(),
		app.ScraperService(),
	)
	httpServer := server.NewStreamableHTTPServer(mcpSrv.Server())
	log.Printf("✅ MCP 服务器初始化成功")

	handler := api.NewRouter(app, httpServer)

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
