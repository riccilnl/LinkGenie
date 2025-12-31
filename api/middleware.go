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
func AuthMiddleware(apiToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 跳过认证的路径
			skipAuthPaths := []string{
				"/health",
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
			for _, path := range skipAuthPaths {
				// 特殊处理根路径：必须精确匹配
				if path == "/" {
					if r.URL.Path == "/" {
						next.ServeHTTP(w, r)
						return
					}
					continue
				}

				// 后缀为 / 的通常是文件夹或 API 前缀
				if strings.HasSuffix(path, "/") {
					if strings.HasPrefix(r.URL.Path, path) {
						next.ServeHTTP(w, r)
						return
					}
				} else {
					// 否则要求完全匹配 (针对文件或特定路径)
					if r.URL.Path == path {
						next.ServeHTTP(w, r)
						return
					}
				}
			}

			// 对于需要认证的路径,检查 token
			token := r.Header.Get("Authorization")
			validToken := "Bearer " + apiToken
			validTokenAlt := "Token " + apiToken
			if token != validToken && token != validTokenAlt {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
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

// LoggingMiddleware 日志中间件
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

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

		next.ServeHTTP(w, r)

		// 记录响应时间
		duration := time.Since(start)
		log.Printf("✅ 完成: %s %s 耗时: %v", r.Method, r.URL.Path, duration)
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
