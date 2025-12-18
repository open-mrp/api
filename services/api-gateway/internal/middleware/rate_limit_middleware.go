package middleware

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/augno/api/services/api-gateway/internal/header"
	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	"github.com/augno/api/shared/contracts"
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
	// Base delay for exponential backoff
	baseDelay time.Duration
	// Maximum delay for exponential backoff
	maxDelay time.Duration
	// Multiplier for exponential backoff
	multiplier float64
	// Percentage of jitter to add (0.0 to 1.0)
	jitterPercent float64
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests:      make(map[string][]time.Time),
		violations:    make(map[string]int),
		lastViolation: make(map[string]time.Time),
		limit:         limit,
		window:        window,
		baseDelay:     1 * time.Second,
		maxDelay:      15 * time.Minute,
		multiplier:    1.5,
		jitterPercent: 0.1,
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
	if violationCount <= 0 {
		return rl.baseDelay
	}

	delay := float64(rl.baseDelay) * rl.multiplier
	for i := 1; i < violationCount; i++ {
		delay *= rl.multiplier
	}

	if delay > float64(rl.maxDelay) {
		delay = float64(rl.maxDelay)
	}

	if rl.jitterPercent > 0 {
		jitterRange := delay * rl.jitterPercent
		var buf [8]byte
		if _, err := rand.Read(buf[:]); err != nil {
			jitter := 0.0
			delay += jitter
		} else {
			randomUint64 := binary.BigEndian.Uint64(buf[:])
			randomFloat := float64(randomUint64) / float64(^uint64(0))
			jitter := (randomFloat - 0.5) * 2 * jitterRange
			delay += jitter
		}

		if delay < float64(rl.baseDelay) {
			delay = float64(rl.baseDelay)
		}
	}

	return time.Duration(delay)
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

			w.Header().Set("RateLimit-Limit", fmt.Sprintf("%d", rateLimiter.limit))
			w.Header().Set("RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			w.Header().Set("RateLimit-Reset", fmt.Sprintf("%d", rateLimiter.GetResetAfterSeconds(clientIP.String())))

			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfterSeconds))
				w.WriteHeader(http.StatusTooManyRequests)
				httptransport.RespondWithAPIError(r.Context(), w, contracts.NewRateLimitExceededError("Too many requests. Please try again later."))
				return
			}

			next.ServeHTTP(w, r)
		}
	}
}
