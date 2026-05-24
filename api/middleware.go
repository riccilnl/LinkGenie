package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/riccilnl/LinkGenie/serverapp"
)

// RateLimitMiddleware 限流中间件。
func RateLimitMiddleware(getLimiter func() *serverapp.RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			limiter := getLimiter()
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

// AuthMiddleware 认证中间件。
func AuthMiddleware(getToken func() string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			skipAuthPaths := []string{
				"/health",
				"/api/system/status",
				"/",
				"/index.html",
				"/sw.js",
				"/manifest.json",
				"/icon.svg",
				"/apple-touch-icon.png",
				"/css/",
				"/js/",
			}

			path := r.URL.Path
			for _, skipPath := range skipAuthPaths {
				if path == skipPath {
					next.ServeHTTP(w, r)
					return
				}
				if skipPath != "/" && strings.HasSuffix(skipPath, "/") && strings.HasPrefix(path, skipPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			apiToken := getToken()
			authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
			if authHeader == "" {
				http.Error(w, "Unauthorized: Missing token", http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 {
				http.Error(w, "Unauthorized: Invalid header format", http.StatusUnauthorized)
				return
			}

			prefix := strings.ToLower(parts[0])
			providedToken := strings.TrimSpace(parts[1])
			if (prefix != "bearer" && prefix != "token") || providedToken != apiToken {
				log.Printf("🚫 认证失败: IP=%s, Prefix=%s", r.RemoteAddr, prefix)
				http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RecoveryMiddleware 恢复中间件。
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("🔥 CRITICAL PANIC 捕获: %v", err)
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

// responseWriter 是一个包装器，用于捕获状态码。
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// LoggingMiddleware 日志中间件。
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		referer := r.Header.Get("Referer")
		userAgent := r.Header.Get("User-Agent")
		if referer == "" {
			referer = "(无)"
		}
		if len(userAgent) > 50 {
			userAgent = userAgent[:50] + "..."
		}
		log.Printf("📥 请求: %s %s | IP: %s | Referer: %s | UA: %s",
			r.Method, r.URL.Path, r.RemoteAddr, referer, userAgent)

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		log.Printf("✅ 完成: %s %s | 状态: %d | 耗时: %v", r.Method, r.URL.Path, rw.status, duration)
	})
}

// CORSMiddleware CORS 中间件。
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
