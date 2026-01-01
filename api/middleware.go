package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// tokenBucket 令牌桶
type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
	mu         sync.Mutex
}

// RateLimiter 限流器
type RateLimiter struct {
	buckets sync.Map // map[string]*tokenBucket
	rate    float64  // tokens per second
	burst   int      // max tokens
}

// NewRateLimiter 创建限流器
func NewRateLimiter(ratePerMinute, burst int) *RateLimiter {
	rl := &RateLimiter{
		rate:  float64(ratePerMinute) / 60.0, // 转换为每秒
		burst: burst,
	}

	// 定期清理过期的bucket
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()

	log.Printf("🛡️ 限流器已启动: %d请求/分钟, 突发容量: %d", ratePerMinute, burst)
	return rl
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow(ip string) bool {
	if rl == nil {
		return true
	}

	now := time.Now()

	// 获取或创建bucket
	value, _ := rl.buckets.LoadOrStore(ip, &tokenBucket{
		tokens:     float64(rl.burst),
		lastRefill: now,
	})

	bucket := value.(*tokenBucket)
	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	// 补充令牌
	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.tokens += elapsed * rl.rate
	if bucket.tokens > float64(rl.burst) {
		bucket.tokens = float64(rl.burst)
	}
	bucket.lastRefill = now

	// 消耗令牌
	if bucket.tokens >= 1.0 {
		bucket.tokens -= 1.0
		return true
	}

	return false
}

// cleanup 清理过期的bucket
func (rl *RateLimiter) cleanup() {
	if rl == nil {
		return
	}

	now := time.Now()
	rl.buckets.Range(func(key, value interface{}) bool {
		bucket := value.(*tokenBucket)
		bucket.mu.Lock()
		// 删除5分钟未使用的bucket
		if now.Sub(bucket.lastRefill) > 5*time.Minute {
			rl.buckets.Delete(key)
		}
		bucket.mu.Unlock()
		return true
	})
}

// RateLimitMiddleware 限流中间件
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limiter == nil {
				next.ServeHTTP(w, r)
				return
			}

			ip := r.RemoteAddr
			if !limiter.Allow(ip) {
				log.Printf("🚫 限流: IP=%s", ip)
				http.Error(w, "请求过于频繁，请稍后再试", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// AuthMiddleware 认证中间件
func AuthMiddleware(getToken func() string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 跳过认证的路径
			skipAuthPaths := []string{
				"/health",
				"/api/system/status",
				"/api/system/config", // 允许在引导页设置配置
				"/",
				"/index.html",
				"/sw.js",
				"/manifest.json",
				"/icon.svg",
				"/css/",
				"/js/",
				"/mcp/", // MCP 端点不需要认证
			}

			// 检查是否是跳过认证的路径
			path := r.URL.Path

			// 核心：强制放行系统管理接口，确保引导页可用
			if strings.HasPrefix(path, "/api/system/") {
				next.ServeHTTP(w, r)
				return
			}

			for _, skipPath := range skipAuthPaths {
				// 精确匹配
				if path == skipPath {
					next.ServeHTTP(w, r)
					return
				}
				// 前缀匹配 (仅针对非根路径的文件夹型路径，如 /static/)
				if skipPath != "/" && strings.HasSuffix(skipPath, "/") && strings.HasPrefix(path, skipPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// 对于需要认证的路径,检查 token
			apiToken := getToken()

			// 处理 Authorization 头
			authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
			if authHeader == "" {
				http.Error(w, "Unauthorized: Missing token", http.StatusUnauthorized)
				return
			}

			// 支持 "Bearer <token>" 和 "Token <token>" 格式
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 {
				http.Error(w, "Unauthorized: Invalid header format", http.StatusUnauthorized)
				return
			}

			prefix := strings.ToLower(parts[0])
			providedToken := strings.TrimSpace(parts[1])

			if (prefix != "bearer" && prefix != "token") || providedToken != apiToken {
				log.Printf("🚫 认证失败: Prefix=%s, Header=%s", prefix, authHeader)
				http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RecoveryMiddleware 恢复中间件 (防止进程崩溃，实现幸存者自愈)
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("🔥 CRITICAL PANIC 捕获: %v", err)
				// 记录错误详情到响应
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "服务器内部错误 (已自动恢复)",
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// responseWriter 是一个包装器，用于捕获状态码
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// LoggingMiddleware 日志中间件
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 包装 ResponseWriter 以获取状态码
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		// 记录请求(不记录敏感信息)
		referer := r.Header.Get("Referer")
		userAgent := r.Header.Get("User-Agent")
		if referer == "" {
			referer = "(无)"
		}
		// 截取 User-Agent 前 50 个字符以避免日志过长
		if len(userAgent) > 50 {
			userAgent = userAgent[:50] + "..."
		}
		log.Printf("📥 请求: %s %s | IP: %s | Referer: %s | UA: %s",
			r.Method, r.URL.Path, r.RemoteAddr, referer, userAgent)

		next.ServeHTTP(rw, r)

		// 记录响应时间和状态码
		duration := time.Since(start)
		log.Printf("✅ 完成: %s %s | 状态: %d | 耗时: %v", r.Method, r.URL.Path, rw.status, duration)
	})
}

// CORSMiddleware CORS 中间件
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
