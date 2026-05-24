package serverapp

import (
	"log"
	"sync"
	"time"
)

// tokenBucket 令牌桶。
type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
	mu         sync.Mutex
}

// RateLimiter 基于内存的简单令牌桶限流器。
type RateLimiter struct {
	buckets sync.Map
	rate    float64
	burst   int
}

// NewRateLimiter 创建限流器。
func NewRateLimiter(ratePerMinute, burst int) *RateLimiter {
	rl := &RateLimiter{
		rate:  float64(ratePerMinute) / 60.0,
		burst: burst,
	}

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

// Allow 检查是否允许请求。
func (rl *RateLimiter) Allow(ip string) bool {
	if rl == nil {
		return true
	}

	now := time.Now()
	value, _ := rl.buckets.LoadOrStore(ip, &tokenBucket{
		tokens:     float64(rl.burst),
		lastRefill: now,
	})

	bucket := value.(*tokenBucket)
	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.tokens += elapsed * rl.rate
	if bucket.tokens > float64(rl.burst) {
		bucket.tokens = float64(rl.burst)
	}
	bucket.lastRefill = now

	if bucket.tokens >= 1.0 {
		bucket.tokens -= 1.0
		return true
	}

	return false
}

func (rl *RateLimiter) cleanup() {
	if rl == nil {
		return
	}

	now := time.Now()
	rl.buckets.Range(func(key, value interface{}) bool {
		bucket := value.(*tokenBucket)
		bucket.mu.Lock()
		if now.Sub(bucket.lastRefill) > 5*time.Minute {
			rl.buckets.Delete(key)
		}
		bucket.mu.Unlock()
		return true
	})
}
