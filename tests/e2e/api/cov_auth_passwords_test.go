//go:build e2e

package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Coverage for /v1/auth/passwords (UpdatePassword, RequestPasswordReset, ResetPassword),
// see TASK-auth_passwords.md. These are stateless action-style POST endpoints (kind
// "auth"), not a CRUD resource: the response body for all three is EmptyResource ({}),
// so most of the value here is in status codes, side effects (cookies, refresh-token
// revocation, actual password changes verified via re-login), validation, and
// idempotency rather than response-field assertions.
//
// login, register (/v1/auth/users), and add-account-user (/v1/identity/account-users)
// are used only as setup helpers to mint disposable, isolated users for the destructive
// UpdatePassword/ResetPassword scenarios - this avoids ever mutating a shared seed
// user's password, which would race against other e2e test files running in parallel
// against the same seeded credentials.

const (
	covAuthPasswordsUpdatePath       = "/v1/auth/passwords"
	covAuthPasswordsRequestResetPath = "/v1/auth/passwords/actions/request-reset"
	covAuthPasswordsResetPath        = "/v1/auth/passwords/actions/reset"

	// covAuthPasswordsJWTSecretDefault matches JWT_SECRET in docker-compose.e2e.yml.
	// Password reset tokens are stateless JWTs that are never persisted anywhere the
	// e2e harness can read back (unlike registration_session's verification_token), so
	// this is the only way to mint a real, valid reset token for the happy path.
	covAuthPasswordsJWTSecretDefault = "test-token-secret" // #nosec G101 - Test-environment JWT signing secret, not a production credential

	covAuthPasswordsJWTIssuer = "https://augno.com"
)

// covAuthPasswordsJWTSecret resolves the signing secret, allowing CI to override via
// E2E_JWT_SECRET the same way harness_test.go overrides E2E_DB_URL/E2E_API_KEY.
func covAuthPasswordsJWTSecret() string {
	return envOr("E2E_JWT_SECRET", covAuthPasswordsJWTSecretDefault)
}

// covAuthPasswordsClaims mirrors auth-service's token.JWTClaims.
type covAuthPasswordsClaims struct {
	jwt.RegisteredClaims
	TokenType string `json:"token_type"`
}

// covAuthPasswordsMintToken mints an HS512 JWT mirroring auth-service's
// token.EncodeJWT, for subject userID, of the given token_type, expiring in ttl
// (a negative ttl mints an already-expired token), signed with secret.
func covAuthPasswordsMintToken(t *testing.T, userID, tokenType string, ttl time.Duration, secret string) string {
	t.Helper()
	now := time.Now().UTC()
	claims := covAuthPasswordsClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    covAuthPasswordsJWTIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		TokenType: tokenType,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	signed, err := token.SignedString([]byte(secret))
	require.NoError(t, err, "minting local test JWT")
	return signed
}

// covAuthPasswordsResetToken mints a valid 15-minute password_reset JWT for userID,
// signed with the real e2e JWT secret.
func covAuthPasswordsResetToken(t *testing.T, userID string) string {
	t.Helper()
	return covAuthPasswordsMintToken(t, userID, "password_reset", 15*time.Minute, covAuthPasswordsJWTSecret())
}

// covAuthPasswordsRegisterUser registers a brand-new disposable user (via the
// out-of-scope but reusable /v1/auth/users endpoint - see cov_auth_users_test.go) with
// covAuthUsersPassword and returns its id and email.
func covAuthPasswordsRegisterUser(t *testing.T, prefix string) (userID, email string) {
	t.Helper()
	email = covAuthUsersUniqueEmail(prefix)
	status, body, err := apiClient.Post(covAuthUsersRegisterPath, map[string]any{
		"email":    email,
		"password": covAuthUsersPassword,
		"name":     "E2E " + prefix,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	userID = jsonField(parseJSON(body), "id")
	require.NotEmpty(t, userID)
	return userID, email
}

// covAuthPasswordsNewAccountMember registers a disposable user, grants it membership on
// SeedAccountID (required for UpdatePassword: the Augno-Account header is resolved
// against real account membership before the endpoint's own account-agnostic
// CheckHasUserActor logic ever runs, so a freshly-registered account-less user 403s),
// logs in, and returns a bearer-authenticated Client plus the user's id/email/password.
func covAuthPasswordsNewAccountMember(t *testing.T, prefix string) (client *Client, userID, email, password string) {
	t.Helper()
	userID, email = covAuthPasswordsRegisterUser(t, prefix)

	status, body, err := apiClient.Post(accountUsersPath, map[string]any{
		"email": email,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	client = loginAsUser(t, email, covAuthUsersPassword, SeedAccountID)
	return client, userID, email, covAuthUsersPassword
}

// covAuthPasswordsRawPost performs a POST with no Authorization and no Augno-Account
// header, bypassing Client (which always injects both), to prove an endpoint is truly
// callable unauthenticated. apiClient.baseURL/apiVersion are unexported fields on
// *Client but accessible here since this file is in the same package.
func covAuthPasswordsRawPost(t *testing.T, path string, body map[string]any) (int, []byte) {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, apiClient.baseURL+path, bytes.NewReader(b))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Augno-Version", apiClient.apiVersion)
	req.Header.Set("Idempotency-Key", newIdempotencyKey())

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, respBody
}

// covAuthPasswordsAssertLogin asserts that logging in with identifier/password
// returns the given status (200 or 401), used to verify the actual password-change
// side effect end-to-end rather than trusting the 200 on the mutating call alone.
func covAuthPasswordsAssertLogin(t *testing.T, identifier, password string, wantStatus int) {
	t.Helper()
	status, body, err := apiClient.Post(loginPath, map[string]any{
		"identifier": identifier,
		"password":   password,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, wantStatus, status, body)
}

// ──────────────────────────────────────────────
// UpdatePassword: happy path + side effects
// ──────────────────────────────────────────────

func TestCovAuthPasswords_UpdatePassword_HappyPath(t *testing.T) {
	t.Parallel()
	client, _, email, oldPassword := covAuthPasswordsNewAccountMember(t, "e2e-updpw-happy")
	const newPassword = "NewTesting123!" // #nosec G101 - Test constant, not a production credential

	status, body, err := client.Post(covAuthPasswordsUpdatePath, map[string]any{
		"old_password": oldPassword,
		"new_password": newPassword,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Empty(t, parseJSON(body), "UpdatePassword response body should be an empty object")

	// Side effect: old password now fails, new password succeeds.
	covAuthPasswordsAssertLogin(t, email, oldPassword, 401)
	covAuthPasswordsAssertLogin(t, email, newPassword, 200)
}

// TestCovAuthPasswords_UpdatePassword_SamePasswordAllowed proves setting new_password
// equal to old_password is not rejected by validation (per TASK-auth_passwords.md, the
// service allows this - there is no "must differ" check).
func TestCovAuthPasswords_UpdatePassword_SamePasswordAllowed(t *testing.T) {
	t.Parallel()
	client, _, email, password := covAuthPasswordsNewAccountMember(t, "e2e-updpw-samepw")

	status, body, err := client.Post(covAuthPasswordsUpdatePath, map[string]any{
		"old_password": password,
		"new_password": password,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	covAuthPasswordsAssertLogin(t, email, password, 200)
}

func TestCovAuthPasswords_UpdatePassword_WrongOldPassword(t *testing.T) {
	t.Parallel()
	client, _, _, _ := covAuthPasswordsNewAccountMember(t, "e2e-updpw-wrongold")

	status, body, err := client.Post(covAuthPasswordsUpdatePath, map[string]any{
		"old_password": "TotallyWrong123!",
		"new_password": "NewTesting123!",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 401, status, body)
	requireErrorResponse(t, body, "invalid_credentials", "invalid_request_error")
}

// TestCovAuthPasswords_UpdatePassword_NonUserCaller proves the default API-key
// identity (not a user actor) is rejected with 403, using the correct error code -
// contrast with RequestPasswordReset, which has no such gate.
func TestCovAuthPasswords_UpdatePassword_NonUserCaller(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(covAuthPasswordsUpdatePath, map[string]any{
		"old_password": "Testing123!",
		"new_password": "NewTesting123!",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
}

func TestCovAuthPasswords_UpdatePassword_Idempotent(t *testing.T) {
	t.Parallel()
	client, _, email, oldPassword := covAuthPasswordsNewAccountMember(t, "e2e-updpw-idem")
	const newPassword = "IdemUpdate123!" // #nosec G101 - Test constant, not a production credential
	idemKey := newIdempotencyKey()
	reqBody := map[string]any{
		"old_password": oldPassword,
		"new_password": newPassword,
	}

	status1, body1, err := client.Post(covAuthPasswordsUpdatePath, reqBody, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)

	// The idempotency response is cached asynchronously after the first request
	// returns. Give the goroutine time to store the response before replaying.
	time.Sleep(500 * time.Millisecond)

	status2, body2, err := client.Post(covAuthPasswordsUpdatePath, reqBody, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)

	// The password was only actually rotated once: the new password still works and
	// the account was not left in a broken (e.g. double-rotated) state.
	covAuthPasswordsAssertLogin(t, email, newPassword, 200)
}

// ──────────────────────────────────────────────
// UpdatePassword: validation
// ──────────────────────────────────────────────

func TestCovAuthPasswords_UpdatePassword_Validation(t *testing.T) {
	t.Parallel()
	client, _, email, password := covAuthPasswordsNewAccountMember(t, "e2e-updpw-validation")

	t.Run("MissingOldPassword", func(t *testing.T) {
		status, body, err := client.Post(covAuthPasswordsUpdatePath, map[string]any{
			"new_password": "NewTesting123!",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
		assertErrorParam(t, errObj, "old_password")
	})

	t.Run("MissingNewPassword", func(t *testing.T) {
		status, body, err := client.Post(covAuthPasswordsUpdatePath, map[string]any{
			"old_password": password,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
		assertErrorParam(t, errObj, "new_password")
	})

	t.Run("EmptyOldPassword", func(t *testing.T) {
		status, body, err := client.Post(covAuthPasswordsUpdatePath, map[string]any{
			"old_password": "",
			"new_password": "NewTesting123!",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
		assertErrorParam(t, errObj, "old_password")
	})

	t.Run("WeakOldPasswordFormat", func(t *testing.T) {
		status, body, err := client.Post(covAuthPasswordsUpdatePath, map[string]any{
			"old_password": "short",
			"new_password": "NewTesting123!",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
		assertErrorParam(t, errObj, "old_password")
	})

	weakNewPasswords := map[string]string{
		"NoUppercase": "alllowercase1!",
		"NoLowercase": "NOUPPERCASE1!",
		"NoDigit":     "NoDigitsHere!",
		"NoSpecial":   "NoSpecial123",
	}
	for name, weak := range weakNewPasswords {
		t.Run("WeakNewPassword_"+name, func(t *testing.T) {
			status, body, err := client.Post(covAuthPasswordsUpdatePath, map[string]any{
				"old_password": password,
				"new_password": weak,
			}, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 400, status, body)
			errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
			assertErrorParam(t, errObj, "new_password")
		})
	}

	// 256+ chars: cleanly 400s via the password validator's own 72-char ceiling
	// (bcrypt's hard limit) rather than 500ing downstream - see prod-bug-suspect #1.
	t.Run("NewPasswordTooLong", func(t *testing.T) {
		status, body, err := client.Post(covAuthPasswordsUpdatePath, map[string]any{
			"old_password": password,
			"new_password": strings.Repeat("A1a!", 64), // 256 chars
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	})

	// None of the above cases should have mutated the account's password.
	covAuthPasswordsAssertLogin(t, email, password, 200)
}

func TestCovAuthPasswords_UpdatePassword_UnknownField(t *testing.T) {
	t.Parallel()
	client, _, _, password := covAuthPasswordsNewAccountMember(t, "e2e-updpw-unknown")
	status, body, err := client.Post(covAuthPasswordsUpdatePath, map[string]any{
		bogusE2EJSONField: "x",
		"old_password":    password,
		"new_password":    "NewTesting123!",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assertJSONUnknownFieldRejected(t, "POST", covAuthPasswordsUpdatePath, status, body)
}

func TestCovAuthPasswords_UpdatePassword_NilBody(t *testing.T) {
	t.Parallel()
	client, _, _, _ := covAuthPasswordsNewAccountMember(t, "e2e-updpw-nilbody")
	status, body, err := client.Post(covAuthPasswordsUpdatePath, nil, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"nil body should return 400 or 422, got %d: %s", status, string(body))
}

func TestCovAuthPasswords_UpdatePassword_WrongHTTPMethod(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Do("GET", covAuthPasswordsUpdatePath, nil, "")
	require.NoError(t, err)
	requireStatus(t, 405, status, body)
	requireErrorResponse(t, body, "method_not_allowed", "invalid_request_error")
}

// ──────────────────────────────────────────────
// RequestPasswordReset: happy path + enumeration safety
// ──────────────────────────────────────────────

// TestCovAuthPasswords_RequestReset_EnumerationSafety proves the endpoint always
// returns 202 for a well-formed identifier, whether it is a known email, a known
// username, or an identifier that matches no user at all - the mediator swallows
// not-found so the response never reveals which identifiers exist. All calls use the
// default API-key apiClient (not a user identity), proving there is no user-actor gate
// here (contrast with UpdatePassword's 403 for the same caller).
func TestCovAuthPasswords_RequestReset_EnumerationSafety(t *testing.T) {
	t.Parallel()

	t.Run("KnownEmail", func(t *testing.T) {
		status, body, err := apiClient.Post(covAuthPasswordsRequestResetPath, map[string]any{
			"identifier": seedUserEmail,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 202, status, body)
		assert.Empty(t, parseJSON(body))
	})

	t.Run("KnownUsername", func(t *testing.T) {
		status, body, err := apiClient.Post(covAuthPasswordsRequestResetPath, map[string]any{
			"identifier": seedUserUsername,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 202, status, body)
	})

	t.Run("UnknownIdentifier", func(t *testing.T) {
		status, body, err := apiClient.Post(covAuthPasswordsRequestResetPath, map[string]any{
			"identifier": "nonexistent-e2e-user@example.com",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 202, status, body)
		assert.Empty(t, parseJSON(body))
	})
}

// TestCovAuthPasswords_RequestReset_CallableUnauthenticated proves the endpoint has no
// identity check at all: a raw request with no Authorization and no Augno-Account
// header still succeeds.
func TestCovAuthPasswords_RequestReset_CallableUnauthenticated(t *testing.T) {
	t.Parallel()
	status, body := covAuthPasswordsRawPost(t, covAuthPasswordsRequestResetPath, map[string]any{
		"identifier": seedUserEmail,
	})
	requireStatus(t, 202, status, body)
}

// ──────────────────────────────────────────────
// RequestPasswordReset: account_slug (optional, create-style, unvalidated)
// ──────────────────────────────────────────────

func TestCovAuthPasswords_RequestReset_AccountSlug(t *testing.T) {
	t.Parallel()

	t.Run("ValidSlug", func(t *testing.T) {
		status, body, err := apiClient.Post(covAuthPasswordsRequestResetPath, map[string]any{
			"identifier":   seedUserEmail,
			"account_slug": SeedAccountSlug,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 202, status, body)
	})

	t.Run("Omitted", func(t *testing.T) {
		status, body, err := apiClient.Post(covAuthPasswordsRequestResetPath, map[string]any{
			"identifier": seedUserEmail,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 202, status, body)
	})

	// account_slug is never FK-checked (it's only interpolated into the reset link
	// URL path, see prod-bug-suspect #3) - a path-traversal-ish or space-containing
	// value must still not 500.
	t.Run("UnvalidatedWeirdChars", func(t *testing.T) {
		for _, slug := range []string{"../../evil", "a b", "weird/slug?x=1"} {
			status, body, err := apiClient.Post(covAuthPasswordsRequestResetPath, map[string]any{
				"identifier":   seedUserEmail,
				"account_slug": slug,
			}, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 202, status, body)
		}
	})

	// Explicit null for a create-style field.Optional field must 400, not be silently
	// treated as absent - see nullable-field-patterns.md and prod-bug-suspect #4.
	t.Run("ExplicitNullRejected", func(t *testing.T) {
		status, body, err := apiClient.Post(covAuthPasswordsRequestResetPath, map[string]any{
			"identifier":   seedUserEmail,
			"account_slug": nil,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
		assertErrorParam(t, errObj, "account_slug")
	})
}

// ──────────────────────────────────────────────
// RequestPasswordReset: validation
// ──────────────────────────────────────────────

func TestCovAuthPasswords_RequestReset_Validation(t *testing.T) {
	t.Parallel()

	t.Run("MissingIdentifier", func(t *testing.T) {
		status, body, err := apiClient.Post(covAuthPasswordsRequestResetPath, map[string]any{}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
		assertErrorParam(t, errObj, "identifier")
	})

	t.Run("EmptyIdentifier", func(t *testing.T) {
		status, body, err := apiClient.Post(covAuthPasswordsRequestResetPath, map[string]any{
			"identifier": "",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
		assertErrorParam(t, errObj, "identifier")
	})

	t.Run("TooShortIdentifier", func(t *testing.T) {
		status, body, err := apiClient.Post(covAuthPasswordsRequestResetPath, map[string]any{
			"identifier": "ab",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
		assertErrorParam(t, errObj, "identifier")
	})

	t.Run("MalformedEmail", func(t *testing.T) {
		status, body, err := apiClient.Post(covAuthPasswordsRequestResetPath, map[string]any{
			"identifier": "not-an-email@",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
		assertErrorParam(t, errObj, "identifier")
	})
}

func TestCovAuthPasswords_RequestReset_UnknownField(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(covAuthPasswordsRequestResetPath, map[string]any{
		bogusE2EJSONField: "x",
		"identifier":      seedUserEmail,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assertJSONUnknownFieldRejected(t, "POST", covAuthPasswordsRequestResetPath, status, body)
}

func TestCovAuthPasswords_RequestReset_NilBody(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(covAuthPasswordsRequestResetPath, nil, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"nil body should return 400 or 422, got %d: %s", status, string(body))
}

func TestCovAuthPasswords_RequestReset_WrongHTTPMethod(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Do("GET", covAuthPasswordsRequestResetPath, nil, "")
	require.NoError(t, err)
	requireStatus(t, 405, status, body)
	requireErrorResponse(t, body, "method_not_allowed", "invalid_request_error")
}

func TestCovAuthPasswords_RequestReset_Idempotent(t *testing.T) {
	t.Parallel()
	idemKey := newIdempotencyKey()
	reqBody := map[string]any{"identifier": seedUserEmail}

	status1, body1, err := apiClient.Post(covAuthPasswordsRequestResetPath, reqBody, idemKey)
	require.NoError(t, err)
	requireStatus(t, 202, status1, body1)

	time.Sleep(500 * time.Millisecond)

	status2, body2, err := apiClient.Post(covAuthPasswordsRequestResetPath, reqBody, idemKey)
	require.NoError(t, err)
	requireStatus(t, 202, status2, body2)
}

// ──────────────────────────────────────────────
// ResetPassword: happy path + side effects
// ──────────────────────────────────────────────

func TestCovAuthPasswords_ResetPassword_HappyPath(t *testing.T) {
	t.Parallel()
	userID, email := covAuthPasswordsRegisterUser(t, "e2e-resetpw-happy")
	token := covAuthPasswordsResetToken(t, userID)
	const newPassword = "ResetPw123!" // #nosec G101 - Test constant, not a production credential

	resp, err := apiClient.PostFull(covAuthPasswordsResetPath, map[string]any{
		"token":    token,
		"password": newPassword,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	assert.Empty(t, parseJSON(resp.Body), "ResetPassword response body should be an empty object")

	// Cookie side effects mirror TestLogin_SetsCookies: ResetPassword is the only one
	// of the three password endpoints that logs the caller in.
	cookies := resp.Header["Set-Cookie"]
	require.GreaterOrEqual(t, len(cookies), 2, "should set at least 2 cookies (access + refresh)")
	var hasAccessToken, hasRefreshToken bool
	for _, c := range cookies {
		if strings.Contains(c, accessTokenCookie) {
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

	// Side effect: original registration password now fails, the reset password
	// succeeds.
	covAuthPasswordsAssertLogin(t, email, covAuthUsersPassword, 401)
	covAuthPasswordsAssertLogin(t, email, newPassword, 200)
}

func TestCovAuthPasswords_ResetPassword_Idempotent(t *testing.T) {
	t.Parallel()
	userID, email := covAuthPasswordsRegisterUser(t, "e2e-resetpw-idem")
	token := covAuthPasswordsResetToken(t, userID)
	idemKey := newIdempotencyKey()
	reqBody := map[string]any{"token": token, "password": "IdemReset123!"}

	resp1, err := apiClient.PostFull(covAuthPasswordsResetPath, reqBody, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, resp1.StatusCode, resp1.Body)
	access1 := cookieValue(resp1.Header["Set-Cookie"], accessTokenCookie)
	require.NotEmpty(t, access1)

	time.Sleep(500 * time.Millisecond)

	resp2, err := apiClient.PostFull(covAuthPasswordsResetPath, reqBody, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, resp2.StatusCode, resp2.Body)
	access2 := cookieValue(resp2.Header["Set-Cookie"], accessTokenCookie)

	assert.Equal(t, access1, access2, "idempotent replay should return the identical cached access token")
	covAuthPasswordsAssertLogin(t, email, "IdemReset123!", 200)
}

// TestCovAuthPasswords_ResetPassword_ReusableUntilExpiry documents the actual (as
// opposed to assumed single-use) behavior: ValidatePasswordResetToken only checks JWT
// signature/type/expiry, never marks the token as consumed, so replaying the same
// still-valid token with a *different* Idempotency-Key succeeds again and rotates the
// password a second time. This is prod-bug-suspect #2 in TASK-auth_passwords.md -
// asserted here as the current, actual behavior (not fabricated as a defect); a
// reviewer should confirm this stateless design is intentional.
func TestCovAuthPasswords_ResetPassword_ReusableUntilExpiry(t *testing.T) {
	t.Parallel()
	userID, email := covAuthPasswordsRegisterUser(t, "e2e-resetpw-reuse")
	token := covAuthPasswordsResetToken(t, userID)

	status1, body1, err := apiClient.Post(covAuthPasswordsResetPath, map[string]any{
		"token":    token,
		"password": "FirstReset123!",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)

	status2, body2, err := apiClient.Post(covAuthPasswordsResetPath, map[string]any{
		"token":    token,
		"password": "SecondReset123!",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)

	covAuthPasswordsAssertLogin(t, email, "FirstReset123!", 401)
	covAuthPasswordsAssertLogin(t, email, "SecondReset123!", 200)
}

// ──────────────────────────────────────────────
// ResetPassword: token validity
// ──────────────────────────────────────────────

func TestCovAuthPasswords_ResetPassword_TokenValidity(t *testing.T) {
	t.Parallel()
	userID, _ := covAuthPasswordsRegisterUser(t, "e2e-resetpw-tokval")

	t.Run("GarbageString", func(t *testing.T) {
		status, body, err := apiClient.Post(covAuthPasswordsResetPath, map[string]any{
			"token":    "not-a-jwt",
			"password": "ValidPass123!",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 401, status, body)
		requireErrorResponse(t, body, "invalid_credentials", "invalid_request_error")
	})

	t.Run("WrongSecret", func(t *testing.T) {
		token := covAuthPasswordsMintToken(t, userID, "password_reset", 15*time.Minute, "wrong-secret-entirely")
		status, body, err := apiClient.Post(covAuthPasswordsResetPath, map[string]any{
			"token":    token,
			"password": "ValidPass123!",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 401, status, body)
		requireErrorResponse(t, body, "invalid_credentials", "invalid_request_error")
	})

	wrongTypes := []string{"access", "magic_login"}
	for _, tokenType := range wrongTypes {
		t.Run("WrongType_"+tokenType, func(t *testing.T) {
			token := covAuthPasswordsMintToken(t, userID, tokenType, 15*time.Minute, covAuthPasswordsJWTSecret())
			status, body, err := apiClient.Post(covAuthPasswordsResetPath, map[string]any{
				"token":    token,
				"password": "ValidPass123!",
			}, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 401, status, body)
			requireErrorResponse(t, body, "invalid_credentials", "invalid_request_error")
		})
	}

	t.Run("Expired", func(t *testing.T) {
		token := covAuthPasswordsMintToken(t, userID, "password_reset", -15*time.Minute, covAuthPasswordsJWTSecret())
		status, body, err := apiClient.Post(covAuthPasswordsResetPath, map[string]any{
			"token":    token,
			"password": "ValidPass123!",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 401, status, body)
		// An expired token now surfaces as the specific expired_token code (previously
		// collapsed into the generic invalid_credentials).
		requireErrorResponse(t, body, "expired_token", "invalid_request_error")
	})

	t.Run("UnknownSubject", func(t *testing.T) {
		token := covAuthPasswordsMintToken(t, "us_nonexistentuserzz", "password_reset", 15*time.Minute, covAuthPasswordsJWTSecret())
		status, body, err := apiClient.Post(covAuthPasswordsResetPath, map[string]any{
			"token":    token,
			"password": "ValidPass123!",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 401, status, body)
		requireErrorResponse(t, body, "invalid_credentials", "invalid_request_error")
	})
}

// ──────────────────────────────────────────────
// ResetPassword: validation
// ──────────────────────────────────────────────

func TestCovAuthPasswords_ResetPassword_Validation(t *testing.T) {
	t.Parallel()
	userID, _ := covAuthPasswordsRegisterUser(t, "e2e-resetpw-validation")

	t.Run("MissingToken", func(t *testing.T) {
		status, body, err := apiClient.Post(covAuthPasswordsResetPath, map[string]any{
			"password": "ValidPass123!",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
		assertErrorParam(t, errObj, "token")
	})

	t.Run("MissingPassword", func(t *testing.T) {
		status, body, err := apiClient.Post(covAuthPasswordsResetPath, map[string]any{
			"token": "sometoken",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
		assertErrorParam(t, errObj, "password")
	})

	t.Run("EmptyToken", func(t *testing.T) {
		status, body, err := apiClient.Post(covAuthPasswordsResetPath, map[string]any{
			"token":    "",
			"password": "ValidPass123!",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
		assertErrorParam(t, errObj, "token")
	})

	t.Run("EmptyPassword", func(t *testing.T) {
		status, body, err := apiClient.Post(covAuthPasswordsResetPath, map[string]any{
			"token":    "sometoken",
			"password": "",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
		assertErrorParam(t, errObj, "password")
	})

	t.Run("WeakPassword", func(t *testing.T) {
		token := covAuthPasswordsResetToken(t, userID)
		status, body, err := apiClient.Post(covAuthPasswordsResetPath, map[string]any{
			"token":    token,
			"password": "weak",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
		assertErrorParam(t, errObj, "password")
	})

	// ResetPasswordRequest.Password has no max= tag (unlike UpdatePasswordRequest's
	// max=255 - see prod-bug-suspect #1), but the shared "password" validator tag
	// already caps length at 72 internally, so a 300+ char password must still cleanly
	// 400 via invalid_format rather than 500ing downstream (e.g. in bcrypt).
	t.Run("PasswordTooLong_NoServerError", func(t *testing.T) {
		token := covAuthPasswordsResetToken(t, userID)
		status, body, err := apiClient.Post(covAuthPasswordsResetPath, map[string]any{
			"token":    token,
			"password": strings.Repeat("A1a!", 100), // 400 chars
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
		assertErrorParam(t, errObj, "password")
	})
}

func TestCovAuthPasswords_ResetPassword_UnknownField(t *testing.T) {
	t.Parallel()
	userID, _ := covAuthPasswordsRegisterUser(t, "e2e-resetpw-unknown")
	token := covAuthPasswordsResetToken(t, userID)
	status, body, err := apiClient.Post(covAuthPasswordsResetPath, map[string]any{
		bogusE2EJSONField: "x",
		"token":           token,
		"password":        "ValidPass123!",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assertJSONUnknownFieldRejected(t, "POST", covAuthPasswordsResetPath, status, body)
}

func TestCovAuthPasswords_ResetPassword_NilBody(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(covAuthPasswordsResetPath, nil, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"nil body should return 400 or 422, got %d: %s", status, string(body))
}

func TestCovAuthPasswords_ResetPassword_WrongHTTPMethod(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Do("GET", covAuthPasswordsResetPath, nil, "")
	require.NoError(t, err)
	requireStatus(t, 405, status, body)
	requireErrorResponse(t, body, "method_not_allowed", "invalid_request_error")
}
