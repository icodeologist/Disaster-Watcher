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

type HttpRateLimiterMIddleware struct {
	Visitors map[string]*rate.Limiter
	Rate     rate.Limit
	Burst    int
	mu       sync.RWMutex
}

type ipMap struct {
	ratelimiter *rate.Limiter
	LastSeen time.Time
}

func NewHTTPRateLimiterMiddleware(r rate.Limit, burst int) *HttpRateLimiterMIddleware {
	return &HttpRateLimiterMIddleware{
		Visitors: make(map[string]),
		Rate:     r,
		Burst:    burst,
	}
}

func (rl *HttpRateLimiterMIddleware) GetLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.Visitors[ip]
	if !exists {
		limiter = rate.NewLimiter(rl.Rate, rl.Burst)
		rl.Visitors[ip] = limiter
	}

	return limiter
}

func (rl *HttpRateLimiterMIddleware) RateLimitingMiddelware(c *gin.Context) {
	ip := c.ClientIP()
	log.Println("IP : \n", ip)
	limiter := rl.GetLimiter(ip)
	if !limiter.Allow() {
		c.JSON(http.StatusTooManyRequests, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode: "TOO_MANY_REQS",
				Message:   "Too many request at the moment.Rate limit exceeded",
			},
		})
		return
	}
	c.Next()
}
