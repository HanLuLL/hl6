package middleware

import (
	"hash/fnv"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"hl6-server/pkg/response"
)

const shardCount = 64

type visitor struct {
	count    int
	lastSeen time.Time
}

type shard struct {
	mu       sync.Mutex
	visitors map[string]*visitor
}

type RateLimiter struct {
	shards [shardCount]shard
	limit  int
	window time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		limit:  limit,
		window: window,
	}
	for i := range rl.shards {
		rl.shards[i].visitors = make(map[string]*visitor)
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) getShard(key string) *shard {
	h := fnv.New32a()
	h.Write([]byte(key))
	return &rl.shards[h.Sum32()%shardCount]
}

func (rl *RateLimiter) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		s := rl.getShard(ip)
		s.mu.Lock()
		v, exists := s.visitors[ip]
		if !exists || time.Since(v.lastSeen) > rl.window {
			s.visitors[ip] = &visitor{count: 1, lastSeen: time.Now()}
			s.mu.Unlock()
			c.Next()
			return
		}
		v.count++
		v.lastSeen = time.Now()
		if v.count > rl.limit {
			s.mu.Unlock()
			response.ErrorWithKey(c, http.StatusTooManyRequests, "rate limit exceeded", "error.rateLimitExceeded")
			c.Abort()
			return
		}
		s.mu.Unlock()
		c.Next()
	}
}

func (rl *RateLimiter) cleanup() {
	for {
		time.Sleep(rl.window)
		for i := range rl.shards {
			s := &rl.shards[i]
			s.mu.Lock()
			for ip, v := range s.visitors {
				if time.Since(v.lastSeen) > rl.window {
					delete(s.visitors, ip)
				}
			}
			s.mu.Unlock()
		}
	}
}
