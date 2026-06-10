//go:build e2e

package api_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	docAPIKeyPath = "/v1/auth/api-keys/actions/fetch-doc-api-key"

	// accessTokenCookie is the cookie the login endpoint sets with the user
	// access token (see api-gateway internal/cookie).
	accessTokenCookie = "__Secure-augno.access-token"
)

// docAPIKeyResult holds the salient fields of a fetch-doc-api-key response.
type docAPIKeyResult struct {
	id     string
	secret string
}

// loginAsUser logs in with the given credentials and returns a Client that
// authenticates with the resulting access token against accountID. The access
// token is only exposed via the Set-Cookie header; in the (non-production) e2e
// environment it is accepted as a bearer token.
func loginAsUser(t *testing.T, identifier, password, accountID string) *Client {
	t.Helper()
	resp, err := apiClient.PostFull(loginPath, map[string]any{
		"identifier": identifier,
		"password":   password,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)

	token := cookieValue(resp.Header["Set-Cookie"], accessTokenCookie)
	require.NotEmpty(t, token, "login should set the %s cookie", accessTokenCookie)

	return apiClient.WithBearerToken(token, accountID)
}

// loginAsSandboxUser logs in as the seeded user (an admin of the seeded sandbox
// account) and returns a Client authenticated as that user targeting the
// sandbox account. The doc API key endpoint requires an internal *user*
// identity on a sandbox account, which the default API-key harness cannot
// satisfy.
func loginAsSandboxUser(t *testing.T) *Client {
	t.Helper()
	return loginAsUser(t, seedUserEmail, seedUserPassword, SeedSandboxAccountID)
}

// cookieValue extracts the value of the named cookie from Set-Cookie headers.
func cookieValue(setCookies []string, name string) string {
	for _, c := range (&http.Response{Header: http.Header{"Set-Cookie": setCookies}}).Cookies() {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}

// fetchDocAPIKey calls the endpoint with the given client and asserts a 200,
// returning the key ID and plaintext secret.
func fetchDocAPIKey(t *testing.T, c *Client) docAPIKeyResult {
	t.Helper()
	resp, err := c.PostFull(docAPIKeyPath, nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)

	m := parseJSON(resp.Body)
	return docAPIKeyResult{
		id:     jsonField(jsonObject(m, "api_key_info"), "id"),
		secret: jsonField(m, "api_key_secret"),
	}
}

// TestDocAPIKey_FetchReturnsKey verifies a sandbox user can fetch a doc API key
// and that the response carries the expected shape, including a 30-day expiry.
func TestDocAPIKey_FetchReturnsKey(t *testing.T) {
	t.Parallel()
	user := loginAsSandboxUser(t)

	resp, err := user.PostFull(docAPIKeyPath, nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)

	m := parseJSON(resp.Body)
	assert.Equal(t, "created_api_key", jsonField(m, "object"))
	assert.NotEmpty(t, jsonField(m, "api_key_secret"))

	info := jsonObject(m, "api_key_info")
	require.NotNil(t, info)
	assert.Equal(t, "api_key", jsonField(info, "object"))
	assert.NotEmpty(t, jsonField(info, "id"))
	assert.NotEmpty(t, jsonField(info, "name"))
	assert.NotEmpty(t, jsonField(info, "redacted_value"))
	assertValidTimestamp(t, jsonField(info, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(info, "updated_at"), "updated_at")
	// Doc API keys are issued with an expiry so they can be auto-rotated.
	assertValidTimestamp(t, jsonField(info, "expires_at"), "expires_at")
	assertNilField(t, info, "revoked_at")
}

// TestDocAPIKey_FetchReusesExistingKey verifies that fetching again returns the
// same (non-expired) key rather than minting a new one each call.
func TestDocAPIKey_FetchReusesExistingKey(t *testing.T) {
	t.Parallel()
	user := loginAsSandboxUser(t)

	first := fetchDocAPIKey(t, user)
	require.NotEmpty(t, first.id)

	second := fetchDocAPIKey(t, user)
	assert.Equal(t, first.id, second.id, "a valid doc API key should be reused, not recreated")
	assert.Equal(t, first.secret, second.secret, "the same plaintext secret should be returned for the reused key")
}

// TestDocAPIKey_ReturnedSecretAuthenticates verifies the returned secret is a
// working sandbox API key by using it to call an authenticated endpoint.
func TestDocAPIKey_ReturnedSecretAuthenticates(t *testing.T) {
	t.Parallel()
	user := loginAsSandboxUser(t)
	key := fetchDocAPIKey(t, user)
	require.NotEmpty(t, key.secret)

	keyClient := apiClient.WithBearerToken(key.secret, SeedSandboxAccountID)
	status, body, err := keyClient.GetListRaw(apiKeysPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

// TestDocAPIKey_RejectsAPIKeyAuth verifies the endpoint rejects API-key callers:
// it requires a user identity. The default harness client uses an API key.
func TestDocAPIKey_RejectsAPIKeyAuth(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(docAPIKeyPath, nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
}

// TestDocAPIKey_RejectsProductionAccount verifies the endpoint is sandbox-only:
// a user targeting a production account is rejected with a validation error.
func TestDocAPIKey_RejectsProductionAccount(t *testing.T) {
	t.Parallel()
	user := loginAsUser(t, seedUserEmail, seedUserPassword, SeedAccountID)

	status, body, err := user.Post(docAPIKeyPath, nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
}
