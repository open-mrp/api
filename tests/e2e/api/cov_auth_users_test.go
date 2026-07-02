//go:build e2e

package api_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Coverage for POST /v1/auth/users ("Register User", see TASK-auth_users.md).
// This is a create-only auth action (no GET/PATCH/DELETE/list/actions/*
// under this route, and the User response has no expandable fields), so
// this single file covers the full request-body matrix, response-field
// coverage, omitted-field defaults, validation, the anti-enumeration
// duplicate-email path, idempotency, and cookie-setting for this endpoint.
// Zero prior test file referenced this route or RegisterRequest.

const covAuthUsersRegisterPath = "/v1/auth/users"

// covAuthUsersPassword matches apiresource.SampleNewUserPassword
// ("50iR2X0r@bvIH") - a valid 8-72 char password with upper/lower/digit/
// special char.
const covAuthUsersPassword = "50iR2X0r@bvIH" // #nosec G101 - Test constant, not a production credential

func covAuthUsersUniqueEmail(prefix string) string {
	return strings.ToLower(uniqueName(prefix)) + "@example.com"
}

// ──────────────────────────────────────────────
// Happy path / full field coverage
// ──────────────────────────────────────────────

// TestCovAuthUsers_Register_AllFields asserts every json field of the User
// response struct (id/object/email/name/username/email_verified_at/
// image_url/created_at/updated_at) for the minimal (required-fields-only)
// registration request.
func TestCovAuthUsers_Register_AllFields(t *testing.T) {
	t.Parallel()
	email := covAuthUsersUniqueEmail("e2e-register-allfields")
	name := "E2E Register AllFields"

	status, body, err := apiClient.Post(covAuthUsersRegisterPath, map[string]any{
		"email":    email,
		"password": covAuthUsersPassword,
		"name":     name,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	m := parseJSON(body)
	assertObjectField(t, m, "user")
	assertIDFormat(t, jsonField(m, "id"), "us")
	assert.Equal(t, email, jsonField(m, "email"))
	assert.Equal(t, name, jsonField(m, "name"))
	assertNilField(t, m, "username")
	assertNilField(t, m, "email_verified_at")
	assertNilField(t, m, "image_url")
	assertValidTimestamp(t, jsonField(m, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(m, "updated_at"), "updated_at")
}

// TestCovAuthUsers_Register_WithAccountSlug proves a valid (non-FK-checked)
// account_slug is accepted and does not alter the response shape.
func TestCovAuthUsers_Register_WithAccountSlug(t *testing.T) {
	t.Parallel()
	email := covAuthUsersUniqueEmail("e2e-register-slug")

	status, body, err := apiClient.Post(covAuthUsersRegisterPath, map[string]any{
		"email":        email,
		"password":     covAuthUsersPassword,
		"name":         "E2E Register Slug",
		"account_slug": SeedAccountSlug,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	m := parseJSON(body)
	assert.Equal(t, email, jsonField(m, "email"))
	assertIDFormat(t, jsonField(m, "id"), "us")
}

// TestCovAuthUsers_Register_UnvalidatedAccountSlug proves an account_slug
// with no matching account_portal row still succeeds - the field is
// intentionally not FK-checked (only used to scope the "already
// registered" magic-login email when the duplicate-email path fires).
func TestCovAuthUsers_Register_UnvalidatedAccountSlug(t *testing.T) {
	t.Parallel()
	email := covAuthUsersUniqueEmail("e2e-register-badslug")

	status, body, err := apiClient.Post(covAuthUsersRegisterPath, map[string]any{
		"email":        email,
		"password":     covAuthUsersPassword,
		"name":         "E2E Register BadSlug",
		"account_slug": "this-slug-does-not-exist-zzz",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	m := parseJSON(body)
	assert.Equal(t, email, jsonField(m, "email"))
}

// TestCovAuthUsers_Register_SetsCookies mirrors TestLogin_SetsCookies:
// a successful registration also logs the new user in.
func TestCovAuthUsers_Register_SetsCookies(t *testing.T) {
	t.Parallel()
	email := covAuthUsersUniqueEmail("e2e-register-cookies")

	resp, err := apiClient.PostFull(covAuthUsersRegisterPath, map[string]any{
		"email":    email,
		"password": covAuthUsersPassword,
		"name":     "E2E Register Cookies",
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

// ──────────────────────────────────────────────
// Omitted / optional fields
// ──────────────────────────────────────────────

// TestCovAuthUsers_Register_OmittedAccountSlug proves account_slug is truly
// optional: omitting it entirely still succeeds and leaves the response
// unaffected.
func TestCovAuthUsers_Register_OmittedAccountSlug(t *testing.T) {
	t.Parallel()
	email := covAuthUsersUniqueEmail("e2e-register-noslug")

	status, body, err := apiClient.Post(covAuthUsersRegisterPath, map[string]any{
		"email":    email,
		"password": covAuthUsersPassword,
		"name":     "E2E Register NoSlug",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	m := parseJSON(body)
	assert.Equal(t, email, jsonField(m, "email"))
	assertNilField(t, m, "username")
}

// TestCovAuthUsers_Register_AccountSlugNull proves an explicit JSON null for
// the optional account_slug field is rejected 400 (field.Optional[string]
// null-rejection semantics).
func TestCovAuthUsers_Register_AccountSlugNull(t *testing.T) {
	t.Parallel()
	email := covAuthUsersUniqueEmail("e2e-register-slugnull")

	status, body, err := apiClient.Post(covAuthUsersRegisterPath, map[string]any{
		"email":        email,
		"password":     covAuthUsersPassword,
		"name":         "E2E Register SlugNull",
		"account_slug": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "account_slug")
}

// TestCovAuthUsers_Register_AccountSlugBlank proves an explicit empty string
// for the optional account_slug field is rejected 400.
func TestCovAuthUsers_Register_AccountSlugBlank(t *testing.T) {
	t.Parallel()
	email := covAuthUsersUniqueEmail("e2e-register-slugblank")

	status, body, err := apiClient.Post(covAuthUsersRegisterPath, map[string]any{
		"email":        email,
		"password":     covAuthUsersPassword,
		"name":         "E2E Register SlugBlank",
		"account_slug": "",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "account_slug")
}

// ──────────────────────────────────────────────
// Validation: missing required fields
// ──────────────────────────────────────────────

func TestCovAuthUsers_Register_MissingEmail(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(covAuthUsersRegisterPath, map[string]any{
		"password": covAuthUsersPassword,
		"name":     "E2E Register MissingEmail",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
	assertErrorParam(t, errObj, "email")
}

func TestCovAuthUsers_Register_MissingPassword(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(covAuthUsersRegisterPath, map[string]any{
		"email": covAuthUsersUniqueEmail("e2e-register-missingpw"),
		"name":  "E2E Register MissingPassword",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
	assertErrorParam(t, errObj, "password")
}

func TestCovAuthUsers_Register_MissingName(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(covAuthUsersRegisterPath, map[string]any{
		"email":    covAuthUsersUniqueEmail("e2e-register-missingname"),
		"password": covAuthUsersPassword,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
	assertErrorParam(t, errObj, "name")
}

func TestCovAuthUsers_Register_EmptyName(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(covAuthUsersRegisterPath, map[string]any{
		"email":    covAuthUsersUniqueEmail("e2e-register-emptyname"),
		"password": covAuthUsersPassword,
		"name":     "",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	// required tag rejects blank strings with missing_field, per
	// nullable-field-patterns.md's "T + required" row.
	errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
	assertErrorParam(t, errObj, "name")
}

// ──────────────────────────────────────────────
// Validation: malformed email
// ──────────────────────────────────────────────

func TestCovAuthUsers_Register_InvalidEmail_TrailingAt(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(covAuthUsersRegisterPath, map[string]any{
		"email":    "not-an-email@",
		"password": covAuthUsersPassword,
		"name":     "E2E Register BadEmail",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "email")
}

func TestCovAuthUsers_Register_InvalidEmail_ConsecutiveDots(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(covAuthUsersRegisterPath, map[string]any{
		"email":    "a..b@example.com",
		"password": covAuthUsersPassword,
		"name":     "E2E Register BadEmail",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "email")
}

func TestCovAuthUsers_Register_InvalidEmail_NumericTLD(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(covAuthUsersRegisterPath, map[string]any{
		"email":    "test@example.c0m",
		"password": covAuthUsersPassword,
		"name":     "E2E Register BadEmail",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "email")
}

// ──────────────────────────────────────────────
// Validation: weak/malformed password
// ──────────────────────────────────────────────

func TestCovAuthUsers_Register_PasswordTooShort(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(covAuthUsersRegisterPath, map[string]any{
		"email":    covAuthUsersUniqueEmail("e2e-register-shortpw"),
		"password": "short1!",
		"name":     "E2E Register ShortPW",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "password")
}

func TestCovAuthUsers_Register_PasswordTooLong(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(covAuthUsersRegisterPath, map[string]any{
		"email":    covAuthUsersUniqueEmail("e2e-register-longpw"),
		"password": strings.Repeat("a", 73),
		"name":     "E2E Register LongPW",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "password")
}

func TestCovAuthUsers_Register_PasswordNoDigitOrCase(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(covAuthUsersRegisterPath, map[string]any{
		"email":    covAuthUsersUniqueEmail("e2e-register-weakpw"),
		"password": "onlylowercase",
		"name":     "E2E Register WeakPW",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "password")
}

// ──────────────────────────────────────────────
// Anti-enumeration duplicate-email path
// ──────────────────────────────────────────────

// TestCovAuthUsers_Register_DuplicateEmail proves the anti-enumeration
// design: registering with an already-registered email returns a generic
// 400 validation_failed with NO param (not 409, not 500, not 200) so the
// existence of the account is never revealed.
func TestCovAuthUsers_Register_DuplicateEmail(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(covAuthUsersRegisterPath, map[string]any{
		"email":    seedUserEmail,
		"password": covAuthUsersPassword,
		"name":     "E2E Register Duplicate",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	param, hasParam := errObj["param"]
	assert.True(t, !hasParam || param == nil, "duplicate-email error should not name a param, got %v", param)
}

// TestCovAuthUsers_Register_DuplicateEmail_CaseInsensitive proves the
// duplicate-email check is not bypassable via case variation: registering
// with a differently-cased version of an existing email must still hit the
// generic 400 anti-enumeration path, never a 500 (unique-constraint
// collision) or a 200 (silently creating a second account for the same
// inbox). See TASK-auth_users.md prod-bug-suspect #1.
func TestCovAuthUsers_Register_DuplicateEmail_CaseInsensitive(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(covAuthUsersRegisterPath, map[string]any{
		"email":    strings.ToUpper(seedUserEmail),
		"password": covAuthUsersPassword,
		"name":     "E2E Register DuplicateCased",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	param, hasParam := errObj["param"]
	assert.True(t, !hasParam || param == nil, "duplicate-email error should not name a param, got %v", param)
}

// ──────────────────────────────────────────────
// Unknown field / nil body / wrong method
// ──────────────────────────────────────────────

func TestCovAuthUsers_Register_UnknownField(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(covAuthUsersRegisterPath, map[string]any{
		bogusE2EJSONField: "x",
		"email":           covAuthUsersUniqueEmail("e2e-register-unknown"),
		"password":        covAuthUsersPassword,
		"name":            "E2E Register Unknown",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assertJSONUnknownFieldRejected(t, "POST", covAuthUsersRegisterPath, status, body)
}

func TestCovAuthUsers_Register_NilBody(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(covAuthUsersRegisterPath, nil, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"nil body should return 400 or 422, got %d: %s", status, string(body))
}

func TestCovAuthUsers_Register_WrongHTTPMethod(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Do("GET", covAuthUsersRegisterPath, nil, "")
	require.NoError(t, err)
	requireStatus(t, 405, status, body)

	requireErrorResponse(t, body, "method_not_allowed", "invalid_request_error")
}

// ──────────────────────────────────────────────
// Idempotency
// ──────────────────────────────────────────────

// TestCovAuthUsers_Register_Idempotent mirrors TestLogin_Idempotent: the
// same idempotency key replayed after the caching goroutine has had time
// to store the response must return the identical user id/created_at, not
// create a second user.
func TestCovAuthUsers_Register_Idempotent(t *testing.T) {
	t.Parallel()
	idemKey := newIdempotencyKey()
	reqBody := map[string]any{
		"email":    covAuthUsersUniqueEmail("e2e-register-idem"),
		"password": covAuthUsersPassword,
		"name":     "E2E Register Idem",
	}

	status1, body1, err := apiClient.Post(covAuthUsersRegisterPath, reqBody, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)
	m1 := parseJSON(body1)
	id1 := jsonField(m1, "id")
	createdAt1 := jsonField(m1, "created_at")

	// The idempotency response is cached asynchronously after the first
	// request returns. Give the goroutine time to store the response and
	// release the lock before sending the replay request.
	time.Sleep(500 * time.Millisecond)

	status2, body2, err := apiClient.Post(covAuthUsersRegisterPath, reqBody, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)
	m2 := parseJSON(body2)
	id2 := jsonField(m2, "id")
	createdAt2 := jsonField(m2, "created_at")

	assert.Equal(t, id1, id2, "idempotent replay should return the same user id")
	assert.Equal(t, createdAt1, createdAt2, "idempotent replay should return the same created_at (no second row created)")
}

// TestCovAuthUsers_Register_Idempotent_NoSleep is a variant of the
// idempotency test without an inter-request sleep, specifically to probe
// the RecoveryPointStarted/RecoveryPointFinished state machine for a race
// that would fall through to the default 500 branch (see
// TASK-auth_users.md prod-bug-suspect #3). A correct implementation must
// still return 200 with the identical id both times (or, if genuinely
// concurrent, a 409 idempotency_in_progress - never a 500).
func TestCovAuthUsers_Register_Idempotent_NoSleep(t *testing.T) {
	t.Parallel()
	idemKey := newIdempotencyKey()
	reqBody := map[string]any{
		"email":    covAuthUsersUniqueEmail("e2e-register-idemrace"),
		"password": covAuthUsersPassword,
		"name":     "E2E Register IdemNoSleep",
	}

	status1, body1, err := apiClient.Post(covAuthUsersRegisterPath, reqBody, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")

	status2, body2, err := apiClient.Post(covAuthUsersRegisterPath, reqBody, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)
	id2 := jsonField(parseJSON(body2), "id")

	assert.Equal(t, id1, id2, "back-to-back idempotent replay (no sleep) should return the same user id")
}
