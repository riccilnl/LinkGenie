package testsupport

import (
	"net/http"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/riccilnl/LinkGenie/api"
	"github.com/riccilnl/LinkGenie/config"
	"github.com/riccilnl/LinkGenie/db"
	"github.com/riccilnl/LinkGenie/serverapp"
)

// AppOptions 描述测试应用装配参数。
type AppOptions struct {
	AIEnabled        bool
	EnableAsyncAI    bool
	AIAPIKey         string
	AIEndpoint       string
	AIModel          string
	APIToken         string
	RateLimitEnabled bool
	RateLimitPerIP   int
	RateLimitBurst   int
	AIWorkerCount    int
}

// AppFixture 包装测试应用和核心依赖。
type AppFixture struct {
	App     *serverapp.App
	Handler http.Handler
	Config  *config.Config
}

// NewAppFixture 创建集成测试用应用。
func NewAppFixture(t *testing.T, opts AppOptions) *AppFixture {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "integration-test.db")
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("初始化测试数据库失败: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("关闭测试数据库失败: %v", err)
		}
	})

	cfg := &config.Config{
		AIEnabled:        opts.AIEnabled,
		EnableAsyncAI:    opts.EnableAsyncAI,
		AIAPIKey:         firstNonEmpty(opts.AIAPIKey, ""),
		AIEndpoint:       firstNonEmpty(opts.AIEndpoint, "https://api.openai.com/v1/chat/completions"),
		AIModel:          firstNonEmpty(opts.AIModel, "test-model"),
		APIToken:         firstNonEmpty(opts.APIToken, "test-token"),
		DBPath:           dbPath,
		RateLimitEnabled: opts.RateLimitEnabled,
		RateLimitPerIP:   firstPositive(opts.RateLimitPerIP, 60),
		RateLimitBurst:   firstPositive(opts.RateLimitBurst, 10),
		AIWorkerCount:    firstPositive(opts.AIWorkerCount, 1),
	}

	app := serverapp.New(cfg, repoRoot(t))
	t.Cleanup(app.Close)

	return &AppFixture{
		App:     app,
		Handler: api.NewRouter(app, http.NotFoundHandler()),
		Config:  cfg,
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("无法定位测试仓库根目录")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}

func firstPositive(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func firstNonEmpty(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
