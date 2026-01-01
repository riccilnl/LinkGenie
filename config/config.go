package config

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config 应用配置
type Config struct {
	AIEnabled        bool
	EnableAsyncAI    bool
	AIAPIKey         string
	AIEndpoint       string
	AIModel          string
	APIToken         string
	DBPath           string
	RateLimitEnabled bool
	RateLimitPerIP   int
	RateLimitBurst   int
	AIWorkerCount    int
}

// Load 加载配置（从 .env 文件和环境变量）
func Load() (*Config, error) {
	// 尝试加载 .env 文件（如果不存在也不报错）
	_ = godotenv.Load()

	cfg := &Config{
		AIEnabled:        getEnvBool("AI_ENABLED", false),
		EnableAsyncAI:    getEnvBool("ENABLE_ASYNC_AI", true),
		AIAPIKey:         getEnv("AI_API_KEY", ""),
		AIEndpoint:       getEnv("AI_ENDPOINT", "https://api.openai.com/v1/chat/completions"),
		AIModel:          getEnv("AI_MODEL", "gpt-3.5-turbo"),
		APIToken:         getEnv("API_TOKEN", "your-secret-token-here"),
		DBPath:           parseDBPath(getEnv("DATABASE_URL", "bookmarks.db")),
		RateLimitEnabled: getEnvBool("RATE_LIMIT_ENABLED", true),
		RateLimitPerIP:   getEnvInt("RATE_LIMIT_PER_IP", 60),
		RateLimitBurst:   getEnvInt("RATE_LIMIT_BURST", 10),
		AIWorkerCount:    getEnvInt("AI_WORKER_COUNT", 5),
	}

	return cfg, nil
}

// LoadFromDB 从数据库加载配置并覆盖当前配置
func (c *Config) LoadFromDB(dbAPI interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
}) error {
	// 这里使用 interface 是为了避免循环引用, 因为 db 包引用了 config
	// 实际上我们会传入 *sql.DB
	rows, err := dbAPI.Query("SELECT key, value FROM system_configs")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}

		switch key {
		case "API_TOKEN":
			if value != "" {
				c.APIToken = value
			}
		case "AI_API_KEY":
			if value != "" {
				c.AIAPIKey = value
				c.AIEnabled = true // 如果设置了 Key，默认开启 AI
			}
		case "AI_ENDPOINT":
			if value != "" {
				c.AIEndpoint = value
			}
		case "AI_MODEL":
			if value != "" {
				c.AIModel = value
			}
		case "AI_ENABLED":
			c.AIEnabled = value == "true" || value == "1"
		}
	}
	return nil
}

// parseDBPath 解析数据库路径（兼容 sqlite:/// 前缀）
func parseDBPath(dbURL string) string {
	return strings.TrimPrefix(dbURL, "sqlite:///")
}

// getEnv 获取环境变量（带默认值）
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool 获取布尔型环境变量
func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	value = strings.ToLower(strings.TrimSpace(value))
	return value == "true" || value == "1" || value == "yes"
}

// getEnvInt 获取整型环境变量
func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	intVal, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intVal
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.APIToken == "" || c.APIToken == "your-secret-token-here" {
		return fmt.Errorf("请设置 API_TOKEN 环境变量")
	}

	if c.AIEnabled && c.AIAPIKey == "" {
		return fmt.Errorf("AI 已启用但未设置 AI_API_KEY")
	}

	// 警告: AI endpoint指向localhost
	if c.AIEnabled && (strings.Contains(c.AIEndpoint, "localhost") ||
		strings.Contains(c.AIEndpoint, "127.0.0.1") ||
		strings.Contains(c.AIEndpoint, "[::1]")) {
		fmt.Println("⚠️  警告: AI_ENDPOINT 指向本地地址,这可能导致请求循环")
		fmt.Printf("   当前配置: %s\n", c.AIEndpoint)
	}

	if c.RateLimitPerIP <= 0 {
		return fmt.Errorf("RATE_LIMIT_PER_IP 必须大于 0")
	}

	return nil
}
