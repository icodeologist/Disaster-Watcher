package auth

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/icodeologist/disasterwatch/internal/models"
	"golang.org/x/time/rate"
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
}

func NewRateLimiterMiddleware(noOfevents rate.Limit, tCapacity int) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		Visitors:         make(map[string]*LimiterInfo),
		NoOfEventsPerSec: noOfevents,
		TokenCap:         tCapacity,
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
	for _, limitinfo := range rl.Visitors {
		t := time.Now()
		lastSeemTime := t.Sub(limitinfo.LastSeen)
		log.Println("TIME ELASPED : ", lastSeemTime)
	}
}

// Go func sleep until 7 day is up and then it runs CleanUpNotActiveIps
func (rl *RateLimitMiddleware) RUNCleanUPEvery7Days() {
	ticker := time.NewTicker(24 * 7 * time.Hour)
	go func() {
		for {
			select {
			case <-ticker.C:
				rl.mu.Lock()
				defer rl.mu.Unlock()
				rl.CleanUpNotActiveIps()
			}
		}
	}()
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
