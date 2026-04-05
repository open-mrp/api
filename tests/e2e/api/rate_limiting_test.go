//go:build e2e

package api_test

import (
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Rate limiting tests verify the API handles burst traffic gracefully.
// The e2e environment may not have a rate limiter configured, so these tests
// focus on verifying no 500s under load rather than asserting 429 responses.

// TestRateLimiting_BurstDoesNotCause500 fires a burst of concurrent requests
// and verifies none return 500.
func TestRateLimiting_BurstDoesNotCause500(t *testing.T) {
	t.Parallel()

	const burst = 50
	var wg sync.WaitGroup
	statuses := make([]int, burst)
	errs := make([]error, burst)

	for i := 0; i < burst; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			resp, err := apiClient.Get(customersPath+"/"+SeedCustomerAccountID, nil)
			if err != nil {
				errs[idx] = err
				return
			}
			resp.Body.Close()
			statuses[idx] = resp.StatusCode
		}(i)
	}
	wg.Wait()

	for i := 0; i < burst; i++ {
		require.NoError(t, errs[i], "Request %d failed with error", i)
		assert.NotEqual(t, 500, statuses[i],
			"Request %d returned 500 — burst should not cause server errors", i)
	}

	// Count responses by type.
	var ok, rateLimited, other int
	for _, s := range statuses {
		switch {
		case s == http.StatusOK:
			ok++
		case s == http.StatusTooManyRequests:
			rateLimited++
		case s > 0:
			other++
		}
	}

	assert.GreaterOrEqual(t, ok, 1, "At least 1 request in the burst should succeed")

	if rateLimited > 0 {
		t.Logf("Rate limiter triggered: %d/%d requests returned 429", rateLimited, burst)
	}
}

// TestRateLimiting_429ErrorShapeIfTriggered validates error shape when rate limited.
func TestRateLimiting_429ErrorShapeIfTriggered(t *testing.T) {
	t.Parallel()

	// Use a no-retry client so we actually see 429s.
	baseURL := envOr("E2E_BASE_URL", defaultBaseURL)
	apiKey := envOr("E2E_API_KEY", SeedAPIKey)
	accountID := envOr("E2E_ACCOUNT_ID", SeedAccountID)
	noRetryClient := NewClient(baseURL, apiKey, accountID)
	noRetryClient.retries = 0

	const burst = 100
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		errBody []byte
	)

	for i := 0; i < burst; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := noRetryClient.Get(customersPath+"/"+SeedCustomerAccountID, nil)
			if err != nil {
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusTooManyRequests {
				mu.Lock()
				if errBody == nil {
					body := make([]byte, 4096)
					n, _ := resp.Body.Read(body)
					errBody = body[:n]
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if errBody == nil {
		// Rate limiter not configured in this environment — just verify the burst completed.
		return
	}

	// If we got a 429, validate its error shape.
	errObj := requireErrorResponse(t, errBody, "", "")
	_, hasTransient := errObj["is_transient"]
	assert.True(t, hasTransient, "429 error should have is_transient field")
}
