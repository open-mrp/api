package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/augno/api/services/api-gateway/internal/header"
	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/retry"
)

// A rate limiter with exponential backoff and jitter
type RateLimiter struct {
	requests map[string][]time.Time
	mutex    sync.RWMutex
	// Number of violations per client
	violations map[string]int
	// Last violation time per client
	lastViolation map[string]time.Time
	// Limit of requests per window
	limit int
	// Window of time to track requests
	window time.Duration
	// Backoff configuration for retry-after calculation
	backoffCfg retry.Config
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests:      make(map[string][]time.Time),
		violations:    make(map[string]int),
		lastViolation: make(map[string]time.Time),
		limit:         limit,
		window:        window,
		backoffCfg: retry.Config{
			InitialWait:    1 * time.Second,
			MaxWait:        15 * time.Minute,
			Multiplier:     1.5,
			JitterFraction: 0.1,
		},
	}
}

func (rl *RateLimiter) IsAllowed(key string) (bool, int, int) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now().UTC()
	cutoff := now.Add(-rl.window)

	if requests, exists := rl.requests[key]; exists {
		var validRequests []time.Time
		for _, reqTime := range requests {
			if reqTime.After(cutoff) {
				validRequests = append(validRequests, reqTime)
			}
		}
		rl.requests[key] = validRequests
	}

	if len(rl.requests[key]) < rl.limit {
		rl.requests[key] = append(rl.requests[key], now)
		rl.violations[key] = 0
		remaining := rl.limit - len(rl.requests[key])
		return true, 0, remaining
	}

	rl.violations[key]++
	rl.lastViolation[key] = now

	delay := rl.calculateBackoffDelay(rl.violations[key])
	retryAfterSeconds := int(delay.Seconds())

	return false, retryAfterSeconds, 0
}

func (rl *RateLimiter) GetResetAfterSeconds(key string) int {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	now := time.Now().UTC()
	cutoff := now.Add(-rl.window)

	requests, exists := rl.requests[key]
	if !exists || len(requests) == 0 {
		return 0
	}

	var earliest time.Time
	for _, reqTime := range requests {
		if reqTime.After(cutoff) {
			earliest = reqTime
			break
		}
	}

	if earliest.IsZero() {
		return 0
	}

	resetAt := earliest.Add(rl.window)
	if resetAt.Before(now) {
		return 0
	}

	return int(resetAt.Sub(now).Seconds())
}

func (rl *RateLimiter) calculateBackoffDelay(violationCount int) time.Duration {
	return retry.CalculateDelay(&rl.backoffCfg, violationCount)
}

var globalRateLimiter *RateLimiter

func getGlobalRateLimiter() *RateLimiter {
	if globalRateLimiter == nil {
		globalRateLimiter = NewRateLimiter(20, time.Second)
	}
	return globalRateLimiter
}

func RateLimitMiddleware() func(http.HandlerFunc) http.HandlerFunc {
	rateLimiter := getGlobalRateLimiter()

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Skip rate limiting for health check endpoint.
			if r.URL.Path == "/healthz" {
				next.ServeHTTP(w, r)
				return
			}

			clientIP := header.GetClientIP(r)

			allowed, retryAfterSeconds, remaining := rateLimiter.IsAllowed(clientIP.String())

			w.Header().Set(header.RateLimitLimitHeader, fmt.Sprintf("%d", rateLimiter.limit))
			w.Header().Set(header.RateLimitRemainingHeader, fmt.Sprintf("%d", remaining))
			w.Header().Set(header.RateLimitResetHeader, fmt.Sprintf("%d", rateLimiter.GetResetAfterSeconds(clientIP.String())))

			if !allowed {
				w.Header().Set(header.ContentTypeHeader, "application/json")
				w.Header().Set(header.RetryAfterHeader, fmt.Sprintf("%d", retryAfterSeconds))
				w.WriteHeader(http.StatusTooManyRequests)
				httptransport.RespondWithAPIError(r.Context(), w, apierror.NewRateLimitExceededError("Too many requests. Please try again later."))
				return
			}

			next.ServeHTTP(w, r)
		}
	}
}
