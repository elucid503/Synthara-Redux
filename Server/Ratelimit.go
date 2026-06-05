package Server

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateBucket struct {

	Count int
	ResetTime time.Time

}

type RateLimiter struct {

	Limit int

	Window time.Duration
	Buckets map[string]*rateBucket

	Mutex sync.Mutex

}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {

	return &RateLimiter{

		Limit: limit,
		Window: window,
		Buckets: make(map[string]*rateBucket),

	}

}

func (limiter *RateLimiter) Allow(Key string) bool {

	Now := time.Now()

	limiter.Mutex.Lock()
	defer limiter.Mutex.Unlock()

	Bucket, Exists := limiter.Buckets[Key]

	if !Exists || Now.After(Bucket.ResetTime) {

		limiter.Buckets[Key] = &rateBucket{Count: 1, ResetTime: Now.Add(limiter.Window)}
		return true

	}

	if Bucket.Count >= limiter.Limit {

		return false

	}

	Bucket.Count++

	return true

}

func clientKey(Context *gin.Context) string {

	if Context == nil || Context.Request == nil {

		return "unknown"

	}

	IP := Context.ClientIP()

	if IP != "" {

		return IP

	}

	return "unknown"

}

func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {

	return func(Context *gin.Context) {

		if limiter == nil {

			Context.Next()
			return

		}

		if !limiter.Allow(clientKey(Context)) {

			Context.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"Error": "Rate limit exceeded. Please try again shortly."})
			return

		}

		Context.Next()

	}

}
