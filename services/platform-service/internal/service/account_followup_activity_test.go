package service

import (
	"testing"
	"time"

	"github.com/open-mrp/api/services/platform-service/internal/domain"

	"github.com/stretchr/testify/require"
)

func TestSummarizeAccountActivity(t *testing.T) {
	t.Parallel()

	sandbox := "ac_sandbox"
	start := time.Date(2026, 10, 3, 8, 34, 0, 0, time.UTC)
	at := func(d time.Duration) time.Time { return start.Add(d) }
	req := func(account, method, route string, status int, d time.Duration) domain.AccountRequest {
		return domain.AccountRequest{TargetAccountID: account, Method: method, NormalizedRoute: route, StatusCode: status, OccurredAt: at(d)}
	}

	requests := []domain.AccountRequest{
		req("ac_live", "GET", "/v1/identity/me", 200, 0),
		req("ac_live", "GET", "/v1/messaging/notifications", 200, time.Minute),
		req("ac_live", "GET", "/v1/catalog/parts", 200, 2*time.Minute),
		req("ac_live", "POST", "/v1/operations/locations", 201, 3*time.Minute),
		req("ac_live", "GET", "/v1/operations/locations/{id}", 200, 3*time.Minute),
		req("ac_live", "POST", "/v1/operations/locations", 201, 4*time.Minute),
		req("ac_live", "POST", "/v1/catalog/units", 409, 5*time.Minute),
		req("ac_live", "PATCH", "/v1/catalog/units/{id}", 200, 6*time.Minute),
		req("ac_live", "POST", "/v1/auth/api-keys", 201, 7*time.Minute),
		req("ac_live", "GET", "/v1/billing/plans", 200, 8*time.Minute),
		// A new session, entirely in the sandbox, after the gap.
		req(sandbox, "POST", "/v1/auth/api-keys/actions/fetch-doc-api-key", 200, 50*time.Minute),
		req(sandbox, "GET", "/v1/catalog/products", 200, 55*time.Minute),
		// The next day, back in the live account.
		req("ac_live", "DELETE", "/v1/operations/locations/{id}", 204, 26*time.Hour),
	}

	a := summarizeAccountActivity(requests, &sandbox, false)

	require.Equal(t, 13, a.TotalRequests)
	require.Len(t, a.Sessions, 3)
	require.Equal(t, domain.AccountActivitySession{Start: start, Minutes: 8, Requests: 10, Sandbox: false}, a.Sessions[0])
	require.Equal(t, domain.AccountActivitySession{Start: at(50 * time.Minute), Minutes: 5, Requests: 2, Sandbox: true}, a.Sessions[1])
	require.False(t, a.Sessions[2].Sandbox)
	require.Equal(t, 2, a.DaysActive)
	require.True(t, a.ReturnedAfterFirstDay)

	require.Equal(t, []domain.AccountActivityCount{{Resource: "operations/locations", Count: 2}, {Resource: "auth/api-keys", Count: 1}}, a.Created, "most-created first")
	require.Equal(t, []domain.AccountActivityCount{{Resource: "catalog/units", Count: 1}}, a.Updated)
	require.Equal(t, []domain.AccountActivityCount{{Resource: "operations/locations", Count: 1}}, a.Deleted)
	require.Equal(t, []string{"billing/plans", "catalog/parts", "catalog/products", "operations/locations"}, a.Viewed, "dashboard polling is not something the registrant chose to look at")
	require.Equal(t, []domain.AccountActivityError{{Method: "POST", Resource: "catalog/units", StatusCode: 409, Count: 1}}, a.Errors)
	require.True(t, a.CreatedAPIKey)
	require.True(t, a.ReadAPIDocs)
	require.True(t, a.ViewedPlans)
}

func TestSummarizeAccountActivityEmpty(t *testing.T) {
	t.Parallel()

	a := summarizeAccountActivity(nil, nil, false)
	require.Zero(t, a.TotalRequests)
	require.Empty(t, a.Sessions)
	require.Equal(t, []string{"No activity after registering."}, activityTimeline(a))
}

func TestActivityResource(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"/v1/operations/locations":                    "operations/locations",
		"/v1/operations/locations/{id}":               "operations/locations",
		"/v1/catalog/unit-groups/{id}":                "catalog/unit-groups",
		"/v1/auth/api-keys/actions/fetch-doc-api-key": "auth/api-keys",
		"/v1/identity/accounts/{id}/favicon":          "identity/accounts/favicon",
		"/v1/production-runs":                         "production-runs",
	}
	for route, want := range cases {
		require.Equal(t, want, activityResource(route), route)
	}
}

func TestActivityTimeline(t *testing.T) {
	t.Parallel()

	a := domain.AccountActivity{
		TotalRequests: 4,
		Sessions:      []domain.AccountActivitySession{{Start: time.Date(2026, 10, 3, 8, 34, 0, 0, time.UTC), Minutes: 5, Requests: 4}},
		Created:       []domain.AccountActivityCount{{Resource: "operations/locations", Count: 6}},
		Viewed:        []string{"catalog/parts"},
		Errors:        []domain.AccountActivityError{{Method: "POST", Resource: "catalog/units", StatusCode: 409, Count: 1}},
		ReadAPIDocs:   true,
	}
	require.Equal(t, []string{
		"Session 1: Oct 3 08:34 UTC, 5 min, 4 requests (live account)",
		"Created: operations/locations (6)",
		"Looked at: catalog/parts",
		"Opened the API docs",
		"Error: POST catalog/units returned 409 (1×)",
	}, activityTimeline(a))
}
