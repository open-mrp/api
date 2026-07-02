//go:build e2e

package api_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Gap-closing coverage for POST /v1/auth/actions/login (see
// TASK-auth_login.md). crud_auth_login_test.go already covers happy paths
// (email/username identifier), response shape, cookie-setting, missing/
// malformed-field validation (400), wrong-credential/nonexistent-identifier
// auth failures (401), wrong HTTP method (405), nil body, and same-key/
// same-body idempotency replay. This file closes: value-level assertions
// for the two previously unasserted User fields (image_url,
// email_verified_at), the 429 identifier-based login-throttle path against
// the live gateway, same-key/different-body idempotency conflict, and two
// low-priority identifier format edge cases (length, invalid characters).

// ──────────────────────────────────────────────
// allFields: image_url + email_verified_at asserted by value
// ──────────────────────────────────────────────

// TestCovAuthLogin_ImageURLAndEmailVerifiedAt closes the two unasserted-field
// gaps flagged in the task doc. image_url is a raw relative path passed
// through from auth-service (not a presigned URL) - this is pre-existing,
// intentional-ish behavior outside this task's scope (see project memory
// "Dropdown avatar broken vs chat" for the analogous /me pattern), so we
// assert the exact seeded relative-path string rather than an https:// URL.
func TestCovAuthLogin_ImageURLAndEmailVerifiedAt(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(loginPath, map[string]any{
		"identifier": seedUserEmail,
		"password":   seedUserPassword,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	m := parseJSON(body)
	wantImageURL := fmt.Sprintf("/v1/core/users/%s/photo", SeedUserID)
	assert.Equal(t, wantImageURL, jsonField(m, "image_url"), "image_url should be the raw relative photo path seeded for this user")
	assertValidTimestamp(t, jsonField(m, "email_verified_at"), "email_verified_at")
}

// TestCovAuthLogin_ImageURLAndEmailVerifiedAt_SecondUser repeats the same
// value-level assertions against the second seeded user so the coverage
// isn't tied to a single row's data.
func TestCovAuthLogin_ImageURLAndEmailVerifiedAt_SecondUser(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(loginPath, map[string]any{
		"identifier": seedUser2Email,
		"password":   seedUserPassword,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	m := parseJSON(body)
	wantImageURL := fmt.Sprintf("/v1/core/users/%s/photo", SeedUser2ID)
	assert.Equal(t, wantImageURL, jsonField(m, "image_url"), "image_url should be the raw relative photo path seeded for this user")
	assertValidTimestamp(t, jsonField(m, "email_verified_at"), "email_verified_at")
}

// ──────────────────────────────────────────────
// validation: 429 identifier-based login throttle
// ──────────────────────────────────────────────

// TestCovAuthLogin_ThrottleExceeded exercises the loginFailureLimiter
// (10 failures / 5-minute window, keyed by lower-cased/trimmed identifier)
// through the live HTTP gateway end-to-end - previously this path was only
// unit-tested against a stubbed gRPC client (login_throttle_test.go). Uses a
// synthetic, per-run-unique identifier (uniqueName) so it cannot interfere
// with any other test's failure bucket (e.g. TestLogin_WrongPassword against
// seedUserEmail) and is safe to rerun within the same 5-minute window. Note:
// apiClient retries 429 responses with backoff (up to 5 retries), so the
// final request in this test is expected to take several seconds - the
// throttle window is 5 minutes, so every retry within that window still
// observes 429.
func TestCovAuthLogin_ThrottleExceeded(t *testing.T) {
	t.Parallel()
	identifier := uniqueName("covauthlogin-throttle")

	for i := 0; i < 10; i++ {
		status, body, err := apiClient.Post(loginPath, map[string]any{
			"identifier": identifier,
			"password":   "WrongPass123!",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 401, status, body)
	}

	status, body, err := apiClient.Post(loginPath, map[string]any{
		"identifier": identifier,
		"password":   "WrongPass123!",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 429, status, body)

	requireErrorResponse(t, body, "rate_limit_exceeded", "invalid_request_error")
}

// ──────────────────────────────────────────────
// Additional identifier validation edge cases (low priority, per task doc)
// ──────────────────────────────────────────────

func TestCovAuthLogin_InvalidIdentifier_TooLong(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(loginPath, map[string]any{
		"identifier": strings.Repeat("a", 51),
		"password":   seedUserPassword,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "identifier")
}

func TestCovAuthLogin_InvalidIdentifier_InvalidCharacters(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(loginPath, map[string]any{
		"identifier": seedUserUsername + "!",
		"password":   seedUserPassword,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "identifier")
}

// ──────────────────────────────────────────────
// Idempotency: same key, different body
// ──────────────────────────────────────────────

// TestCovAuthLogin_IdempotencyConflictDifferentBody asserts reusing the same
// Idempotency-Key with a different request body (different identifier) is
// rejected with 400 idempotency_error, matching the pattern already used for
// other create-style endpoints (see idempotency_test.go,
// cov_core_sandboxes_test.go, cov_messaging_blocks_test.go). Verified against
// the live stack that the conflict is detected synchronously (no sleep
// needed between the two requests, unlike TestLogin_Idempotent's same-body
// replay which waits for async cache write).
func TestCovAuthLogin_IdempotencyConflictDifferentBody(t *testing.T) {
	t.Parallel()
	key := newIdempotencyKey()

	status1, body1, err := apiClient.Post(loginPath, map[string]any{
		"identifier": seedUserEmail,
		"password":   seedUserPassword,
	}, key)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)

	status2, body2, err := apiClient.Post(loginPath, map[string]any{
		"identifier": seedUser2Email,
		"password":   seedUserPassword,
	}, key)
	require.NoError(t, err)
	requireStatus(t, 400, status2, body2)
	requireErrorResponse(t, body2, "validation_failed", "idempotency_error")
}
