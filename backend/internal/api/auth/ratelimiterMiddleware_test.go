package auth

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestCleanUpNotActiveIpsRemovesExpiredVisitorsAndKeepsActiveVisitors(t *testing.T) {
	ratelimiter := NewRateLimiterMiddleware(10, 5)
	ratelimiter.inactiveAfter = time.Hour
	ratelimiter.Visitors["active"] = &LimiterInfo{LastSeen: time.Now().Add(-30 * time.Minute)}
	ratelimiter.Visitors["expired"] = &LimiterInfo{LastSeen: time.Now().Add(-2 * time.Hour)}

	ratelimiter.CleanUpNotActiveIps()

	if _, exists := ratelimiter.Visitors["active"]; !exists {
		t.Fatal("cleanup removed an active visitor")
	}
	if _, exists := ratelimiter.Visitors["expired"]; exists {
		t.Fatal("cleanup kept an expired visitor")
	}
}

func TestCleanUpNotActiveIpsCanRunRepeatedly(t *testing.T) {
	ratelimiter := NewRateLimiterMiddleware(10, 5)
	ratelimiter.inactiveAfter = time.Hour
	ratelimiter.Visitors["expired"] = &LimiterInfo{LastSeen: time.Now().Add(-2 * time.Hour)}

	for range 3 {
		ratelimiter.CleanUpNotActiveIps()
	}

	if len(ratelimiter.Visitors) != 0 {
		t.Fatalf("visitor count after repeated cleanup = %d, want 0", len(ratelimiter.Visitors))
	}
}

func TestStartCleanupStopsWhenContextIsCanceled(t *testing.T) {
	ratelimiter := NewRateLimiterMiddleware(10, 5)
	ratelimiter.inactiveAfter = time.Hour
	ratelimiter.cleanupInterval = time.Millisecond
	ratelimiter.Visitors["expired"] = &LimiterInfo{LastSeen: time.Now().Add(-2 * time.Hour)}
	ctx, cancel := context.WithCancel(context.Background())
	done := ratelimiter.StartCleanup(ctx)

	deadline := time.After(time.Second)
	for {
		ratelimiter.mu.Lock()
		_, exists := ratelimiter.Visitors["expired"]
		ratelimiter.mu.Unlock()
		if !exists {
			break
		}
		select {
		case <-deadline:
			t.Fatal("cleanup ticker did not remove the expired visitor")
		case <-time.After(time.Millisecond):
		}
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cleanup goroutine did not stop after context cancellation")
	}
}

func TestRateLimiterSupportsConcurrentRequestsAndCleanup(t *testing.T) {
	ratelimiter := NewRateLimiterMiddleware(10, 5)
	ratelimiter.inactiveAfter = time.Hour

	var workers sync.WaitGroup
	for workerID := 0; workerID < 16; workerID++ {
		workers.Add(1)
		go func(workerID int) {
			defer workers.Done()
			for request := 0; request < 200; request++ {
				ip := fmt.Sprintf("192.0.2.%d", workerID%4)
				limiter := ratelimiter.GetLimiter(ip)
				limiter.rateLimiter.Allow()
			}
		}(workerID)
	}

	for cleanup := 0; cleanup < 100; cleanup++ {
		ratelimiter.CleanUpNotActiveIps()
	}
	workers.Wait()
}
