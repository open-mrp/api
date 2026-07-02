//go:build e2e

package api_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	loginPath = "/v1/auth/actions/login"

	// Seeded user credentials from shared/db/seed/0004_auth.sql.
	seedUserEmail    = "dane@augno.com"
	seedUserUsername = "jdoe"
	seedUserName     = "John Doe"
	seedUserPassword = "Testing123!" // #nosec G101 - Test constant, not a production credential

	seedUser2Email    = "user2@augno.com"
	seedUser2Username = "user2"
	seedUser2Name     = "Sarah Martinez"
)

// ──────────────────────────────────────────────
// Happy Paths
// ──────────────────────────────────────────────

func TestLogin_WithEmail(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(loginPath, map[string]any{
		"identifier": seedUserEmail,
		"password":   seedUserPassword,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	m := parseJSON(body)
	assert.Equal(t, "user", jsonField(m, "object"))
	assert.Equal(t, SeedUserID, jsonField(m, "id"))
	assert.Equal(t, seedUserEmail, jsonField(m, "email"))
	assert.Equal(t, seedUserUsername, jsonField(m, "username"))
	assert.Equal(t, seedUserName, jsonField(m, "name"))
}

func TestLogin_WithUsername(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(loginPath, map[string]any{
		"identifier": seedUserUsername,
		"password":   seedUserPassword,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	m := parseJSON(body)
	assert.Equal(t, "user", jsonField(m, "object"))
	assert.Equal(t, SeedUserID, jsonField(m, "id"))
	assert.Equal(t, seedUserEmail, jsonField(m, "email"))
}

func TestLogin_ResponseFields(t *testing.T) {
	t.Parallel()
	resp, err := apiClient.PostFull(loginPath, map[string]any{
		"identifier": seedUserEmail,
		"password":   seedUserPassword,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)

	m := parseJSON(resp.Body)
	assertObjectField(t, m, "user")
	assertIDFormat(t, jsonField(m, "id"), "us")
	assert.NotEmpty(t, jsonField(m, "email"))
	assert.NotEmpty(t, jsonField(m, "name"))
	assert.NotEmpty(t, jsonField(m, "username"))
	assertValidTimestamp(t, jsonField(m, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(m, "updated_at"), "updated_at")

	// email_verified_at should be present as a key (may be null or a timestamp)
	_, hasEmailVerified := m["email_verified_at"]
	assert.True(t, hasEmailVerified, "email_verified_at field should be present in response")
}

func TestLogin_SetsCookies(t *testing.T) {
	t.Parallel()
	resp, err := apiClient.PostFull(loginPath, map[string]any{
		"identifier": seedUserEmail,
		"password":   seedUserPassword,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)

	cookies := resp.Header["Set-Cookie"]
	require.GreaterOrEqual(t, len(cookies), 2, "should set at least 2 cookies (access + refresh)")

	var hasAccessToken, hasRefreshToken bool
	for _, c := range cookies {
		if strings.Contains(c, "__Secure-augno.access-token") {
			hasAccessToken = true
			assert.Contains(t, c, "HttpOnly", "access token cookie should be HttpOnly")
			assert.Contains(t, c, "Secure", "access token cookie should be Secure")
		}
		if strings.Contains(c, "__Secure-augno.refresh-token") {
			hasRefreshToken = true
			assert.Contains(t, c, "HttpOnly", "refresh token cookie should be HttpOnly")
			assert.Contains(t, c, "Secure", "refresh token cookie should be Secure")
		}
	}
	assert.True(t, hasAccessToken, "response should set access token cookie")
	assert.True(t, hasRefreshToken, "response should set refresh token cookie")
}

func TestLogin_SecondUser(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(loginPath, map[string]any{
		"identifier": seedUser2Email,
		"password":   seedUserPassword,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	m := parseJSON(body)
	assert.Equal(t, "user", jsonField(m, "object"))
	assert.Equal(t, SeedUser2ID, jsonField(m, "id"))
	assert.Equal(t, seedUser2Username, jsonField(m, "username"))
	assert.Equal(t, seedUser2Name, jsonField(m, "name"))
}

// ──────────────────────────────────────────────
// Validation Failures (400)
// ──────────────────────────────────────────────

func TestLogin_MissingIdentifier(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(loginPath, map[string]any{
		"password": seedUserPassword,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
	assertErrorParam(t, errObj, "identifier")
}

func TestLogin_MissingPassword(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(loginPath, map[string]any{
		"identifier": seedUserEmail,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
	assertErrorParam(t, errObj, "password")
}

func TestLogin_EmptyBody(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(loginPath, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	// Multiple missing fields may produce validation_failed or missing_field
	requireErrorResponse(t, body, "", "invalid_request_error")
}

func TestLogin_InvalidIdentifier_TooShort(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(loginPath, map[string]any{
		"identifier": "ab",
		"password":   seedUserPassword,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "identifier")
}

func TestLogin_InvalidIdentifier_BadEmail(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(loginPath, map[string]any{
		"identifier": "not-an-email@",
		"password":   seedUserPassword,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "identifier")
}

func TestLogin_PasswordTooLong(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(loginPath, map[string]any{
		"identifier": seedUserEmail,
		"password":   strings.Repeat("a", 73),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	requireErrorResponse(t, body, "", "invalid_request_error")
}

// ──────────────────────────────────────────────
// Authentication Failures (401)
// ──────────────────────────────────────────────

func TestLogin_WrongPassword(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(loginPath, map[string]any{
		"identifier": seedUserEmail,
		"password":   "WrongPass123!",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 401, status, body)

	requireErrorResponse(t, body, "invalid_credentials", "invalid_request_error")
}

func TestLogin_NonExistentEmail(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(loginPath, map[string]any{
		"identifier": "nonexistent@example.com",
		"password":   seedUserPassword,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 401, status, body)

	requireErrorResponse(t, body, "invalid_credentials", "invalid_request_error")
}

func TestLogin_NonExistentUsername(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(loginPath, map[string]any{
		"identifier": "zzz_no_such_user",
		"password":   seedUserPassword,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 401, status, body)

	requireErrorResponse(t, body, "invalid_credentials", "invalid_request_error")
}

// ──────────────────────────────────────────────
// Other Failures
// ──────────────────────────────────────────────

func TestLogin_WrongHTTPMethod(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Do("GET", loginPath, nil, "")
	require.NoError(t, err)
	requireStatus(t, 405, status, body)

	requireErrorResponse(t, body, "method_not_allowed", "invalid_request_error")
}

func TestLogin_NilBody(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(loginPath, nil, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"nil body should return 400 or 422, got %d: %s", status, string(body))
}

// ──────────────────────────────────────────────
// Idempotency
// ──────────────────────────────────────────────

func TestLogin_Idempotent(t *testing.T) {
	t.Parallel()
	idemKey := newIdempotencyKey()
	reqBody := map[string]any{
		"identifier": seedUserEmail,
		"password":   seedUserPassword,
	}

	status1, body1, err := apiClient.Post(loginPath, reqBody, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")

	// The idempotency response is cached asynchronously after the first
	// request returns. Give the goroutine time to store the response and
	// release the lock before sending the replay request.
	time.Sleep(500 * time.Millisecond)

	status2, body2, err := apiClient.Post(loginPath, reqBody, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)
	id2 := jsonField(parseJSON(body2), "id")

	assert.Equal(t, id1, id2, "idempotent requests should return the same user")
}
