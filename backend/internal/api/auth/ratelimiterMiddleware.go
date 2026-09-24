package auth

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/icodeologist/disasterwatch/internal/models"
	"golang.org/x/time/rate"
)

const (
	defaultVisitorInactiveAfter = 7 * 24 * time.Hour
	defaultCleanupInterval      = 7 * 24 * time.Hour
)

// Each ip will have LimiterInfo
// This contains like the ratelimiter that ip has and the last time its been active
// Last seen is just to delete ips that are just sitting in map without doing anything
type LimiterInfo struct {
	rateLimiter *rate.Limiter
	LastSeen    time.Time
}

type RateLimitMiddleware struct {
	Visitors         map[string]*LimiterInfo
	NoOfEventsPerSec rate.Limit
	TokenCap         int
	mu               sync.Mutex
	inactiveAfter    time.Duration
	cleanupInterval  time.Duration
}

func NewRateLimiterMiddleware(noOfevents rate.Limit, tCapacity int) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		Visitors:         make(map[string]*LimiterInfo),
		NoOfEventsPerSec: noOfevents,
		TokenCap:         tCapacity,
		inactiveAfter:    defaultVisitorInactiveAfter,
		cleanupInterval:  defaultCleanupInterval,
	}
}

func (rl *RateLimitMiddleware) GetLimiter(ip string) *LimiterInfo {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	limiterInfo, exists := rl.Visitors[ip]
	if !exists {
		limiterInfo = &LimiterInfo{
			rateLimiter: rate.NewLimiter(rl.NoOfEventsPerSec, rl.TokenCap),
		}
		rl.Visitors[ip] = limiterInfo
	}
	limiterInfo.LastSeen = time.Now()
	return limiterInfo
}

func (rl *RateLimitMiddleware) CleanUpNotActiveIps() {
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()

	for ip, limiterInfo := range rl.Visitors {
		if limiterInfo == nil || now.Sub(limiterInfo.LastSeen) >= rl.inactiveAfter {
			delete(rl.Visitors, ip)
		}
	}
}

// StartCleanup owns the cleanup ticker and stops it when ctx is canceled.
// The returned channel closes after the ticker and goroutine have stopped.
func (rl *RateLimitMiddleware) StartCleanup(ctx context.Context) <-chan struct{} {
	if ctx == nil {
		ctx = context.Background()
	}
	interval := rl.cleanupInterval
	if interval <= 0 {
		interval = defaultCleanupInterval
	}
	ticker := time.NewTicker(interval)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				rl.CleanUpNotActiveIps()
			case <-ctx.Done():
				return
			}
		}
	}()
	return done
}

func (rl *RateLimitMiddleware) RateLimitingMiddelware(c *gin.Context) {
	ip := c.ClientIP()
	log.Println("IP : \n", ip)
	limiter := rl.GetLimiter(ip)
	if !limiter.rateLimiter.Allow() {
		c.JSON(http.StatusTooManyRequests, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode: "TOO_MANY_REQS",
				Message:   "Too many request at the moment.Rate limit exceeded",
			},
		})
		return
	}
	log.Println("calling next handler for Ip :", ip)
	c.Next()
	log.Println("next handler returned ip : ", ip)
}
