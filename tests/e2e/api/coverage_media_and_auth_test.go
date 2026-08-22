//go:build e2e

package api_test

import (
	"encoding/base64"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Image upload and session endpoints the suite had never called.
//
// Both groups are ones a unit test cannot really cover: an upload is only meaningful once
// the bytes reach a store and come back out, and a refresh only means something when the
// cookie a real login issued is the one presented.

// A one-pixel PNG. Small enough to be inline, and a real image rather than arbitrary bytes,
// so a decoder anywhere along the path has something valid to work with.
const onePixelPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="

func onePixelPNG(t *testing.T) []byte {
	t.Helper()
	b, err := base64.StdEncoding.DecodeString(onePixelPNGBase64)
	require.NoError(t, err)
	return b
}

// ──────────────────────────────────────────────
// Account logo and favicon
// ──────────────────────────────────────────────

// The upload route is /photo while the read route is /logo — an asymmetry worth pinning, since
// a caller guessing the pair from either half alone gets a 404.
func TestAccountLogo_UploadThenRead(t *testing.T) {
	status, body, err := apiClient.PutBytes("/v1/identity/accounts/"+SeedAccountID+"/photo", "image/png", onePixelPNG(t))
	require.NoError(t, err)
	require.Less(t, status, 500, "upload must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.GetListRaw("/v1/identity/accounts/"+SeedAccountID+"/logo", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "read must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	// The stored object is private, so what comes back is a temporary link rather than the image.
	assert.NotEmpty(t, jsonField(parseJSON(body), "url"), "an uploaded logo must yield a URL: %s", string(body))
}

func TestAccountFavicon_UploadThenRead(t *testing.T) {
	status, body, err := apiClient.PutBytes("/v1/identity/accounts/"+SeedAccountID+"/favicon", "image/png", onePixelPNG(t))
	require.NoError(t, err)
	require.Less(t, status, 500, "upload must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.GetListRaw("/v1/identity/accounts/"+SeedAccountID+"/favicon", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "read must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)
	assert.NotEmpty(t, jsonField(parseJSON(body), "url"), "an uploaded favicon must yield a URL: %s", string(body))
}

// Branding is per-account, so one tenant must not be able to overwrite another's logo.
func TestAccountLogo_RejectsAnotherTenant(t *testing.T) {
	t.Parallel()

	tenantB := apiClient.WithBearerToken(SeedTenantBAPIKey, SeedTenantBAccountID)

	status, body, err := tenantB.PutBytes("/v1/identity/accounts/"+SeedAccountID+"/photo", "image/png", onePixelPNG(t))
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.NotEqual(t, 200, status, "another tenant must not replace this account's logo: %s", string(body))

}

// A logo is what the customer portal renders, so it is deliberately readable by anyone
// signed in. An account with no logo — or no account at all — reports no URL rather than an
// error, which is what lets the portal fall back to a default without special-casing.
func TestAccountLogo_UnknownAccountReportsNoURL(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw("/v1/identity/accounts/ac_doesnotexist00000/logo", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)
	assert.Nil(t, parseJSON(body)["url"], "an unknown account has no logo: %s", string(body))
}

// ──────────────────────────────────────────────
// User photo
// ──────────────────────────────────────────────

func TestUserPhoto_UploadThenRead(t *testing.T) {
	status, body, err := apiClient.PutBytes("/v1/identity/users/"+SeedUserID+"/photo", "image/png", onePixelPNG(t))
	require.NoError(t, err)
	require.Less(t, status, 500, "upload must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.GetListRaw("/v1/identity/users/"+SeedUserID+"/photo", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "read must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)
	assert.NotEmpty(t, jsonField(parseJSON(body), "url"), "an uploaded photo must yield a URL: %s", string(body))
}

// A user outside the caller's account is not theirs to read or replace, whatever their own
// permissions say about their own team. SeedUser2 belongs to one account only, which is what
// makes this a genuine cross-tenancy attempt rather than an admin reaching their own second account.
func TestUserPhoto_RejectsAnotherTenantsUser(t *testing.T) {
	t.Parallel()

	tenantB := apiClient.WithBearerToken(SeedTenantBAPIKey, SeedTenantBAccountID)

	status, body, err := tenantB.PutBytes("/v1/identity/users/"+SeedUser2ID+"/photo", "image/png", onePixelPNG(t))
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.NotEqual(t, 200, status, "another tenant must not replace this user's photo: %s", string(body))

	status, body, err = tenantB.GetListRaw("/v1/identity/users/"+SeedUser2ID+"/photo", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.NotEqual(t, 200, status, "another tenant must not resolve this user's photo: %s", string(body))
}

func TestUserPhoto_UnknownUserIsRefused(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw("/v1/identity/users/us_doesnotexist00000/photo", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 404, status, "an unknown user is not a member of this account: %s", string(body))
}

// ──────────────────────────────────────────────
// Session lifecycle
// ──────────────────────────────────────────────

// loginCookies signs the seeded user in and returns the cookies the response set.
func loginCookies(t *testing.T) []*http.Cookie {
	t.Helper()

	resp, err := apiClient.PostFull(loginPath, map[string]any{
		"identifier": seedUserEmail,
		"password":   seedUserPassword,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)

	cookies := (&http.Response{Header: resp.Header}).Cookies()
	require.NotEmpty(t, cookies, "login must hand back session cookies")
	return cookies
}

func TestRefreshAccessToken_IssuesAFreshSession(t *testing.T) {
	t.Parallel()

	session := apiClient.WithCookies(loginCookies(t), SeedAccountID)

	resp, err := session.PutFull("/v1/auth/access-tokens", nil)
	require.NoError(t, err)
	require.Less(t, resp.StatusCode, 500, "refresh must not 5xx: %s", string(resp.Body))
	requireStatus(t, 200, resp.StatusCode, resp.Body)

	// The point of a refresh is a new access token, and it arrives the same way it did at login.
	var refreshed bool
	for _, c := range (&http.Response{Header: resp.Header}).Cookies() {
		if c.Name == "__Secure-openmrp.access-token" && c.Value != "" {
			refreshed = true
		}
	}
	assert.True(t, refreshed, "a refresh must set a new access-token cookie: %v", resp.Header["Set-Cookie"])
}

// Without the refresh cookie there is nothing to refresh from, and the request must be
// refused rather than quietly minting a session.
func TestRefreshAccessToken_RejectsAMissingCookie(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.PutRaw("/v1/auth/access-tokens", nil, nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Contains(t, []int{400, 401}, status, "a refresh without a cookie must be refused: %s", string(body))
}

func TestRefreshAccessToken_RejectsAForgedCookie(t *testing.T) {
	t.Parallel()

	forged := apiClient.WithCookies([]*http.Cookie{{
		Name:  "__Secure-openmrp.refresh-token",
		Value: "not-a-real-refresh-token",
	}}, SeedAccountID)

	status, body, err := forged.PutRaw("/v1/auth/access-tokens", nil, nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Contains(t, []int{400, 401}, status, "a forged refresh token must be refused: %s", string(body))
}

// Revoking has to actually end the session: if the refresh token survives a logout, signing
// out on a shared machine leaves the next person able to mint a new session from it.
func TestRevokeRefreshToken_EndsTheSession(t *testing.T) {
	t.Parallel()

	cookies := loginCookies(t)
	session := apiClient.WithCookies(cookies, SeedAccountID)

	status, body, err := session.Delete("/v1/auth/refresh-tokens")
	require.NoError(t, err)
	require.Less(t, status, 500, "revoke must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	status, body, err = session.PutRaw("/v1/auth/access-tokens", nil, nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Contains(t, []int{400, 401}, status, "a revoked refresh token must not mint a session: %s", string(body))
}

func TestRevokeRefreshToken_WithoutACookieIsRefused(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Delete("/v1/auth/refresh-tokens")
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Contains(t, []int{400, 401}, status, "there is no session to revoke: %s", string(body))
}

// ──────────────────────────────────────────────
// Registration sessions
// ──────────────────────────────────────────────

func TestRegistrationSessions_List(t *testing.T) {
	t.Parallel()

	list, status, err := loginAsSeedUser(t).GetList("/v1/auth/registration-sessions", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	assert.NotNil(t, list, "the list must be well-formed even when empty")
}

func TestRegistrationSessions_ResendVerificationForUnknownSessionIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(registrationSessionsPath+"/rgfw_doesnotexist/actions/resend-verification-email", nil, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 404, status, "an unknown registration session must 404: %s", string(body))
}

// ──────────────────────────────────────────────
// Paid-plan registration billing
// ──────────────────────────────────────────────

// paidRegistrationSession runs a registration up to the point a payment method is collected,
// and returns the session path plus a client acting as the newly created user. The billing
// actions authorize against that user, so an API key cannot stand in for them.
func paidRegistrationSession(t *testing.T) (sessionPath string, asUser *Client) {
	t.Helper()

	status, body, err := apiClient.Post(registrationSessionsPath, map[string]any{
		"email":     uniqueRegistrationEmail(),
		"plan_code": "starter",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	sessionID := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, sessionID)
	sessionPath = registrationSessionsPath + "/" + sessionID

	status, body, err = apiClient.Put(verifyTokenPath(registrationVerificationToken(t, sessionID)), nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	resp, err := apiClient.PostFull(sessionPath+"/users", map[string]any{
		"name":     "E2E Billing Registrant",
		"password": "P@ssw0rd123!",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	return sessionPath, apiClient.WithBearerToken(accessTokenFromSetCookie(t, resp.Header), "")
}

// setupIntentIDFromClientSecret reads the intent ID back out of its client secret, which is how
// the browser identifies the intent it just confirmed — the setup-billing response carries the
// secret rather than the bare ID.
func setupIntentIDFromClientSecret(t *testing.T, clientSecret string) string {
	t.Helper()

	id, _, found := strings.Cut(clientSecret, "_secret_")
	require.True(t, found, "a client secret is shaped <intent id>_secret_<key>, got %q", clientSecret)
	return id
}

func TestRegistrationBilling_SetupReturnsWhatStripeNeeds(t *testing.T) {
	t.Parallel()

	sessionPath, asUser := paidRegistrationSession(t)

	status, body, err := asUser.Post(sessionPath+"/actions/setup-billing", nil, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "setup-billing must not 5xx: %s", string(body))
	requireStatus(t, 201, status, body)

	// Both halves are needed to mount Stripe's card form: the secret identifies the intent,
	// the publishable key is what the browser authenticates with.
	setup := parseJSON(body)
	assert.NotEmpty(t, jsonField(setup, "client_secret"), "the setup intent must be identified: %s", string(body))
	assert.NotEmpty(t, jsonField(setup, "publishable_key"), "the browser needs a publishable key: %s", string(body))
}

func TestRegistrationBilling_ConfirmMarksThePaymentComplete(t *testing.T) {
	t.Parallel()

	sessionPath, asUser := paidRegistrationSession(t)

	status, body, err := asUser.Post(sessionPath+"/actions/setup-billing", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	status, body, err = asUser.Post(sessionPath+"/actions/confirm-payment", map[string]any{
		"setup_intent_id": setupIntentIDFromClientSecret(t, jsonField(parseJSON(body), "client_secret")),
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "confirm-payment must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)
}

// A setup intent is what proves a card was actually collected. Confirming against one that is
// not this session's would let a paid registration complete without a payment method.
func TestRegistrationBilling_ConfirmRejectsAnUnknownSetupIntent(t *testing.T) {
	t.Parallel()

	sessionPath, asUser := paidRegistrationSession(t)

	status, body, err := asUser.Post(sessionPath+"/actions/confirm-payment", map[string]any{
		"setup_intent_id": "seti_not_a_real_intent",
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.NotEqual(t, 200, status, "an unrelated setup intent must not confirm payment: %s", string(body))
}

// The billing actions decide whether an account gets provisioned on a paid plan, so they are
// gated on the registering user rather than on whoever holds an API key.
func TestRegistrationBilling_RequiresTheRegisteringUser(t *testing.T) {
	t.Parallel()

	sessionPath, _ := paidRegistrationSession(t)

	for name, tc := range map[string]struct {
		path string
		body map[string]any
	}{
		"setup-billing":   {sessionPath + "/actions/setup-billing", nil},
		"confirm-payment": {sessionPath + "/actions/confirm-payment", map[string]any{"setup_intent_id": "seti_stub_1"}},
	} {
		t.Run(name, func(t *testing.T) {
			status, body, err := apiClient.Post(tc.path, tc.body, newIdempotencyKey())
			require.NoError(t, err)
			require.Less(t, status, 500, "must not 5xx: %s", string(body))
			assert.Contains(t, []int{401, 403}, status, "an API key must not drive %s: %s", name, string(body))
		})
	}
}

func TestRegistrationSessions_ResendVerificationEmail(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(registrationSessionsPath, map[string]any{
		"email":     uniqueRegistrationEmail(),
		"plan_code": "free",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	sessionID := jsonField(parseJSON(body), "id")

	// Accepted rather than OK: the mail is handed to the sending pipeline, not delivered inline.
	status, body, err = apiClient.Post(registrationSessionsPath+"/"+sessionID+"/actions/resend-verification-email", nil, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "resend must not 5xx: %s", string(body))
	assert.Equal(t, 202, status, "the resend must be accepted: %s", string(body))
}
