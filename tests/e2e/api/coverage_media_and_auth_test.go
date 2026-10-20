//go:build e2e

package api_test

import (
	"encoding/base64"
	"net/http"
	"net/url"
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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

	status, body, err := apiClient.PutBytes("/v1/identity/users/"+SeedUserID+"/photo", "image/png", onePixelPNG(t))
	require.NoError(t, err)
	require.Less(t, status, 500, "upload must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.GetListRaw("/v1/identity/users/"+SeedUserID+"/photo", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "read must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)
	photoURL := jsonField(parseJSON(body), "url")
	assert.NotEmpty(t, photoURL, "an uploaded photo must yield a URL: %s", string(body))

	// /me signs the photo it shows the user. It must be the object the upload wrote: the two used to
	// key it by different accounts, so a user in two accounts uploaded where /me never looked.
	status, body, err = loginAsSeedUser(t).GetListRaw("/v1/identity/me", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	meURL := jsonField(parseJSON(body), "image_url")
	require.NotEmpty(t, meURL, "/me signs the uploaded photo: %s", string(body))
	assert.Equal(t, signedObjectPath(t, photoURL), signedObjectPath(t, meURL), "the photo endpoint and /me sign the same object")
}

// signedObjectPath is a presigned URL without its signature: the object it points at.
func signedObjectPath(t *testing.T, signed string) string {
	t.Helper()
	u, err := url.Parse(signed)
	require.NoError(t, err)
	return u.Host + u.Path
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

// An avatar an identity provider supplied at sign-up is an image hosted elsewhere, and is shown as it
// is. Signing it as though it were an uploaded photo pointed the browser at an object that was never
// stored, and the avatar went blank.
func TestUserPhoto_AnExternalAvatarIsShownAsIs(t *testing.T) {
	t.Parallel()
	email := covAuthUsersUniqueEmail("e2e-external-avatar")
	avatar := "https://lh3.googleusercontent.com/a/e2e-external-avatar=s96-c"

	status, body, err := apiClient.Post(covAuthUsersRegisterPath, map[string]any{
		"email":    email,
		"password": covAuthUsersPassword,
		"name":     "E2E External Avatar",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	userID := jsonField(parseJSON(body), "id")

	status, body, err = apiClient.Post(accountUsersPath, map[string]any{
		"name":    "E2E External Avatar",
		"email":   email,
		"role_id": SeedAdminRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	accountUserID := jsonField(parseJSON(body), "id")
	t.Cleanup(func() { removeAccountUser(accountUserID) })

	_, err = authDB(t).Exec("UPDATE user SET image_url = ? WHERE id = ?", avatar, userID)
	require.NoError(t, err)

	status, body, err = loginAsUser(t, email, covAuthUsersPassword, SeedAccountID).GetListRaw("/v1/identity/me", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, avatar, jsonField(parseJSON(body), "image_url"), "/me")

	status, body, err = apiClient.GetListRaw(accountUsersPath+"/"+accountUserID, url.Values{"include[]": {"user"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	user, ok := parseJSON(body)["user"].(map[string]any)
	require.True(t, ok, "the user is included: %s", body)
	assert.Equal(t, avatar, user["image_url"], "the team member")
}
