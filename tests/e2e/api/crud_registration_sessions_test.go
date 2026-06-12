//go:build e2e

package api_test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	registrationSessionsPath = "/v1/auth/registration-sessions"
	accessTokenCookieName    = "__Secure-augno.access-token"
)

// accessTokenFromSetCookie pulls the access token out of a response's Set-Cookie
// headers. Registration's create-user endpoint logs the user in via cookies
// rather than returning the token in the body.
func accessTokenFromSetCookie(t *testing.T, header http.Header) string {
	t.Helper()
	for _, raw := range header["Set-Cookie"] {
		c, err := http.ParseSetCookie(raw)
		if err == nil && c.Name == accessTokenCookieName && c.Value != "" {
			return c.Value
		}
	}
	t.Fatalf("no %s cookie in response", accessTokenCookieName)
	return ""
}

// uniqueRegistrationEmail returns an address unlikely to collide across runs so
// each test creates a fresh registration session.
func uniqueRegistrationEmail() string {
	return fmt.Sprintf("e2e-reg-%s@example.com", uuid.New().String()[:12])
}

// verifyTokenPath builds the verify-token action path, escaping the opaque token
// since it travels as a path parameter.
func verifyTokenPath(token string) string {
	return fmt.Sprintf("%s/%s/actions/verify-token", registrationSessionsPath, url.PathEscape(token))
}

// TestRegistration_FullJourney exercises the self-serve registration flow end to
// end against the gateway: create a session, retrieve it, read the verification
// token from the database (standing in for the email link), verify the token,
// then create the user. This mirrors what the frontend does and guards the
// email-verification path that has no other automated coverage.
func TestRegistration_FullJourney(t *testing.T) {
	t.Parallel()

	email := uniqueRegistrationEmail()

	// 1. Create the registration session.
	status, body, err := apiClient.Post(registrationSessionsPath, map[string]any{
		"email":     email,
		"plan_code": "free",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	created := parseJSON(body)
	assert.Equal(t, "registration_session", jsonField(created, "object"))
	sessionID := jsonField(created, "id")
	require.NotEmpty(t, sessionID)

	// 2. Retrieve the session and assert its pre-verification shape.
	sessionPath := registrationSessionsPath + "/" + sessionID
	status, body, err = apiClient.Do("GET", sessionPath, nil, "")
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	retrieved := parseJSON(body)
	assert.Equal(t, sessionID, jsonField(retrieved, "id"))
	assert.Equal(t, "verification", jsonField(retrieved, "step"))
	assert.Equal(t, "free", jsonField(retrieved, "plan_code"))

	user := jsonObject(retrieved, "user")
	require.NotNil(t, user)
	assert.Equal(t, email, jsonField(user, "email"))
	assert.Empty(t, jsonField(user, "email_verified_at"), "email should not be verified before the token is used")

	// Read the verification token from the database, standing in for the link
	// the user would receive by email (the token is never exposed by the API).
	token := registrationVerificationToken(t, sessionID)

	// 3. Verify the email token. The token is a path parameter on a PUT action.
	status, body, err = apiClient.Put(verifyTokenPath(token), nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	verified := parseJSON(body)
	assert.Equal(t, sessionID, jsonField(verified, "id"))
	assert.Equal(t, "user_details", jsonField(verified, "step"), "step should advance to user_details after verification")
	verifiedUser := jsonObject(verified, "user")
	require.NotNil(t, verifiedUser)
	assert.NotEmpty(t, jsonField(verifiedUser, "email_verified_at"), "email_verified_at should be set after verification")

	// 4. Create the user for the now-verified session. The response logs the new
	//    user in by setting auth cookies; capture the access token so later steps
	//    act as that user (completion is authorized against the session's user).
	resp, err := apiClient.PostFull(sessionPath+"/users", map[string]any{
		"name":     "E2E Registration User",
		"password": "P@ssw0rd123!",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	createdUser := parseJSON(resp.Body)
	assert.Equal(t, "user", jsonField(createdUser, "object"))
	assert.NotEmpty(t, jsonField(createdUser, "id"))

	userToken := accessTokenFromSetCookie(t, resp.Header)
	asUser := apiClient.WithBearerToken(userToken, "")

	// 5. Provide account details (required before completion).
	accountName := "E2E Reg Co " + uuid.New().String()[:8]
	status, body, err = asUser.Patch(sessionPath, map[string]any{
		"session_data": map[string]any{"account_name": accountName},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	// 6. Complete registration. The free plan skips billing, so this provisions
	//    the account directly. Acts as the registered user.
	status, body, err = asUser.Post(sessionPath+"/accounts", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	completed := parseJSON(body)
	assert.Equal(t, "account", jsonField(completed, "object"))
	accountID := jsonField(completed, "id")
	require.NotEmpty(t, accountID)

	// 7. Completing registration provisions a sandbox account alongside the new
	//    production account, named "<account> Sandbox".
	sandboxID, sandboxName := sandboxForOwnerAccount(t, accountID)
	assert.NotEmpty(t, sandboxID)
	assert.Equal(t, accountName+" Sandbox", sandboxName)
}

// TestRegistration_VerifyIsIdempotent confirms a token can be verified twice
// without error — the frontend may replay the verify link.
func TestRegistration_VerifyIsIdempotent(t *testing.T) {
	t.Parallel()

	email := uniqueRegistrationEmail()
	status, body, err := apiClient.Post(registrationSessionsPath, map[string]any{
		"email":     email,
		"plan_code": "starter",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	sessionID := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, sessionID)

	token := registrationVerificationToken(t, sessionID)

	// First verification succeeds.
	status, body, err = apiClient.Put(verifyTokenPath(token), nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	// Re-verifying the same token is idempotent, not an error.
	status, body, err = apiClient.Put(verifyTokenPath(token), nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, sessionID, jsonField(parseJSON(body), "id"))
}

// TestRegistration_DuplicateEmailReturnsSameSession confirms re-registering an
// email with an active session returns that session rather than creating a new
// one.
func TestRegistration_DuplicateEmailReturnsSameSession(t *testing.T) {
	t.Parallel()

	reqBody := map[string]any{"email": uniqueRegistrationEmail(), "plan_code": "free"}

	status, body, err := apiClient.Post(registrationSessionsPath, reqBody, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	firstID := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, firstID)

	// Distinct idempotency key so the gateway runs create logic again; the
	// service returns the existing active session for the same email.
	status, body, err = apiClient.Post(registrationSessionsPath, reqBody, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	secondID := jsonField(parseJSON(body), "id")
	assert.Equal(t, firstID, secondID, "duplicate email should return the existing session id")
}

// TestRegistration_CreateSessionValidation covers the request-validation guards
// on session creation.
func TestRegistration_CreateSessionValidation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		body map[string]any
	}{
		{"invalid email", map[string]any{"email": "not-an-email", "plan_code": "free"}},
		{"missing plan_code", map[string]any{"email": uniqueRegistrationEmail()}},
		{"invalid plan_code", map[string]any{"email": uniqueRegistrationEmail(), "plan_code": "not-a-plan"}},
		{"missing email", map[string]any{"plan_code": "free"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			status, body, err := apiClient.Post(registrationSessionsPath, tc.body, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 400, status, body)
		})
	}
}

// TestRegistration_VerifyUnknownTokenNotFound confirms an unrecognized token is
// rejected as not-found rather than surfacing a 5xx.
func TestRegistration_VerifyUnknownTokenNotFound(t *testing.T) {
	t.Parallel()

	bogus := "rt_" + uuid.New().String()
	status, body, err := apiClient.Put(verifyTokenPath(bogus), nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

// TestRegistration_CreateUserBeforeVerificationRejected confirms a user cannot be
// created for a session whose email has not been verified.
func TestRegistration_CreateUserBeforeVerificationRejected(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(registrationSessionsPath, map[string]any{
		"email":     uniqueRegistrationEmail(),
		"plan_code": "free",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	sessionID := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, sessionID)

	status, body, err = apiClient.Post(registrationSessionsPath+"/"+sessionID+"/users", map[string]any{
		"name":     "Premature User",
		"password": "P@ssw0rd123!",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

// TestRegistration_RetrieveUnknownSessionNotFound confirms retrieving a
// nonexistent session id returns not-found.
func TestRegistration_RetrieveUnknownSessionNotFound(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Do("GET", registrationSessionsPath+"/rf_"+uuid.New().String(), nil, "")
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}
