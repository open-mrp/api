//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file closes e2e coverage gaps for the auth_api-keys group
// (/v1/auth/api-keys) identified during the e2e coverage audit, layered on
// top of the primary suite in crud_api_keys_test.go: expires_at round-trip
// (create + rotate override), the `expired` status filter, cursor pagination
// for this resource, rotate idempotency, remaining validation cells (name
// too long, malformed timestamps, invalid statuses enum, nonexistent
// role_id), 404s for revoke/rotate on a nonexistent id, authorization (401/
// 403) checks, and proof that revoked/rotated-out secrets stop authenticating.

// ──────────────────────────────────────────────
// allFields: expires_at round-trip (create + rotate override)
// ──────────────────────────────────────────────

// TestCovAuthApiKeys_CreateExpiresAtRoundTrips verifies a user-supplied
// expires_at on create is returned verbatim, not just present/null.
func TestCovAuthApiKeys_CreateExpiresAtRoundTrips(t *testing.T) {
	t.Parallel()
	expiresAt := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Second)

	status, body, err := apiClient.Post(apiKeysPath, map[string]any{
		"name":       uniqueName("cov-ak-expires"),
		"role_id":    SeedAdminRoleID,
		"expires_at": expiresAt.Format(time.RFC3339),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	m := parseJSON(body)
	info := jsonObject(m, "api_key_info")
	require.NotNil(t, info)
	id := jsonField(info, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(apiKeysPath + "/" + id)

	assertIDFormat(t, id, "apke")
	got := jsonField(info, "expires_at")
	gotTime, err := time.Parse(time.RFC3339, got)
	require.NoError(t, err, "expires_at %q should be a valid RFC3339 timestamp", got)
	assert.True(t, gotTime.Equal(expiresAt), "expires_at should round-trip the requested value, got %s want %s", gotTime, expiresAt)

	// Confirm on a fresh GET too, not just the create response.
	getStatus, getBody, err := apiClient.GetListRaw(apiKeysPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	gotAgain := jsonField(parseJSON(getBody), "expires_at")
	assert.Equal(t, got, gotAgain)
}

// TestCovAuthApiKeys_RotateExpiresAtOverrideRoundTrips verifies that rotating
// with an explicit expires_at override produces a rotated key reflecting the
// override, not the old key's expiry.
func TestCovAuthApiKeys_RotateExpiresAtOverrideRoundTrips(t *testing.T) {
	t.Parallel()
	oldExpiresAt := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	newExpiresAt := time.Now().UTC().Add(24 * 30 * time.Hour).Truncate(time.Second)
	require.NotEqual(t, oldExpiresAt, newExpiresAt)

	status, body, err := apiClient.Post(apiKeysPath, map[string]any{
		"name":       uniqueName("cov-ak-rotate-expires"),
		"role_id":    SeedAdminRoleID,
		"expires_at": oldExpiresAt.Format(time.RFC3339),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	origID := jsonField(jsonObject(parseJSON(body), "api_key_info"), "id")

	rotStatus, rotBody, err := apiClient.Post(apiKeysPath+"/"+origID+"/actions/rotate", map[string]any{
		"expires_at": newExpiresAt.Format(time.RFC3339),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, rotStatus, rotBody)

	rotatedInfo := jsonObject(parseJSON(rotBody), "api_key_info")
	require.NotNil(t, rotatedInfo)
	newID := jsonField(rotatedInfo, "id")
	require.NotEmpty(t, newID)
	defer apiClient.Delete(apiKeysPath + "/" + newID)
	defer apiClient.Delete(apiKeysPath + "/" + origID)

	gotStr := jsonField(rotatedInfo, "expires_at")
	got, err := time.Parse(time.RFC3339, gotStr)
	require.NoError(t, err)
	assert.True(t, got.Equal(newExpiresAt),
		"rotated key's expires_at should reflect the override, got %s want %s", got, newExpiresAt)
	assert.False(t, got.Equal(oldExpiresAt),
		"rotated key's expires_at should NOT be the old key's expiry")
}

// ──────────────────────────────────────────────
// responseShape
// ──────────────────────────────────────────────

// TestCovAuthApiKeys_IDFormat asserts newly created API keys use the apke_ id
// prefix (the seeded row SeedAPIKeyID predates a prefix change and uses the
// legacy apky_ prefix; new ids are apke_).
func TestCovAuthApiKeys_IDFormat(t *testing.T) {
	t.Parallel()
	m := createAPIKeyAndCleanup(t, uniqueName("cov-ak-idformat"))
	info := jsonObject(m, "api_key_info")
	require.NotNil(t, info)
	assertIDFormat(t, jsonField(info, "id"), "apke")
}

// ──────────────────────────────────────────────
// list: expired status filter + cursor pagination
// ──────────────────────────────────────────────

// TestCovAuthApiKeys_ListFilterByStatusExpired verifies a key created with a
// past expires_at is surfaced by statuses=expired.
func TestCovAuthApiKeys_ListFilterByStatusExpired(t *testing.T) {
	t.Parallel()
	pastExpiry := time.Now().UTC().Add(-48 * time.Hour).Truncate(time.Second)
	name := uniqueName("cov-ak-expired")

	status, body, err := apiClient.Post(apiKeysPath, map[string]any{
		"name":       name,
		"role_id":    SeedAdminRoleID,
		"expires_at": pastExpiry.Format(time.RFC3339),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(jsonObject(parseJSON(body), "api_key_info"), "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(apiKeysPath + "/" + id)

	assertListContainsID(t, apiKeysPath, url.Values{"statuses": {"expired"}, "q": {name}}, id)

	// It should NOT show under the active filter (still unrevoked, but past
	// its expiry so should not be considered active for auth purposes).
	list, _, err := apiClient.GetList(apiKeysPath, url.Values{"statuses": {"active"}, "q": {name}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "an expired key should not appear under statuses=active")
}

// A key that expired more than 30 days ago still lists under statuses=expired: `expired` means the key passed its expiration time, with no listing window attached.
func TestCovAuthApiKeys_ListExpiredBeyond30DaysStillVisible(t *testing.T) {
	t.Parallel()
	staleExpiry := time.Now().UTC().Add(-40 * 24 * time.Hour).Truncate(time.Second)
	name := uniqueName("cov-ak-staleexpired")

	status, body, err := apiClient.Post(apiKeysPath, map[string]any{
		"name":       name,
		"role_id":    SeedAdminRoleID,
		"expires_at": staleExpiry.Format(time.RFC3339),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(jsonObject(parseJSON(body), "api_key_info"), "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(apiKeysPath + "/" + id)

	// Sanity: the key is still directly retrievable.
	getStatus, getBody, err := apiClient.GetListRaw(apiKeysPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	list, _, err := apiClient.GetList(apiKeysPath, url.Values{"statuses": {"expired"}, "q": {name}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1,
		"a key expired more than 30 days ago should still be listed under statuses=expired")
}

// TestCovAuthApiKeys_ListCursorPaginationAdvances verifies GET
// /v1/auth/api-keys?limit=1 advances the cursor across pages for this
// resource specifically (the shared pagination_errors_test.go suite is
// capped to the first 3 eligible list endpoints and is not guaranteed to
// exercise api-keys).
func TestCovAuthApiKeys_ListCursorPaginationAdvances(t *testing.T) {
	t.Parallel()
	assertCursorPaginationAdvances(t, apiKeysPath, nil)
}

// ──────────────────────────────────────────────
// idempotency: rotate
// ──────────────────────────────────────────────

// TestCovAuthApiKeys_RotateIdempotent verifies replaying the same
// Idempotency-Key on /actions/rotate returns the same rotated key rather
// than minting a second one, and that an independent rotate call (new
// idempotency key) against the resulting key does mint a distinct key.
func TestCovAuthApiKeys_RotateIdempotent(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(apiKeysPath, map[string]any{
		"name":    uniqueName("cov-ak-rotate-idem"),
		"role_id": SeedAdminRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	origID := jsonField(jsonObject(parseJSON(body), "api_key_info"), "id")

	idemKey := newIdempotencyKey()
	rot1Status, rot1Body, err := apiClient.Post(apiKeysPath+"/"+origID+"/actions/rotate", nil, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, rot1Status, rot1Body)
	rot1 := parseJSON(rot1Body)
	id1 := jsonField(jsonObject(rot1, "api_key_info"), "id")
	secret1 := jsonField(rot1, "api_key_secret")
	require.NotEmpty(t, id1)

	rot2Status, rot2Body, err := apiClient.Post(apiKeysPath+"/"+origID+"/actions/rotate", nil, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, rot2Status, rot2Body)
	rot2 := parseJSON(rot2Body)
	id2 := jsonField(jsonObject(rot2, "api_key_info"), "id")
	secret2 := jsonField(rot2, "api_key_secret")

	assert.Equal(t, id1, id2, "replaying the same idempotency key on rotate should return the same key id")
	assert.Equal(t, secret1, secret2, "replaying the same idempotency key on rotate should return the same secret")

	// A third, independent rotate call (fresh idempotency key) should mint a
	// genuinely new key.
	rot3Status, rot3Body, err := apiClient.Post(apiKeysPath+"/"+id1+"/actions/rotate", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, rot3Status, rot3Body)
	id3 := jsonField(jsonObject(parseJSON(rot3Body), "api_key_info"), "id")
	assert.NotEqual(t, id1, id3, "an independent rotate call should mint a distinct key")

	defer apiClient.Delete(apiKeysPath + "/" + origID)
	defer apiClient.Delete(apiKeysPath + "/" + id1)
	defer apiClient.Delete(apiKeysPath + "/" + id3)
}

// ──────────────────────────────────────────────
// validation
// ──────────────────────────────────────────────

// TestCovAuthApiKeys_CreateValidation_NameTooLong verifies a 256-character
// name is rejected (limit is 255).
func TestCovAuthApiKeys_CreateValidation_NameTooLong(t *testing.T) {
	t.Parallel()
	longName := strings.Repeat("a", 256)
	status, body, err := apiClient.Post(apiKeysPath, map[string]any{
		"name":    longName,
		"role_id": SeedAdminRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
}

// TestCovAuthApiKeys_CreateValidation_NameAtMaxLengthAccepted verifies a
// name at the boundary (exactly 255 chars) is accepted.
func TestCovAuthApiKeys_CreateValidation_NameAtMaxLengthAccepted(t *testing.T) {
	t.Parallel()
	maxName := strings.Repeat("b", 255)
	status, body, err := apiClient.Post(apiKeysPath, map[string]any{
		"name":    maxName,
		"role_id": SeedAdminRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(jsonObject(parseJSON(body), "api_key_info"), "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(apiKeysPath + "/" + id)
}

// TestCovAuthApiKeys_CreateValidation_EmptyRoleID verifies an empty-string
// role_id is rejected the same way a missing role_id is.
func TestCovAuthApiKeys_CreateValidation_EmptyRoleID(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(apiKeysPath, map[string]any{
		"name":    uniqueName("cov-ak-emptyrole"),
		"role_id": "",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"empty role_id should return 400 or 422, got %d: %s", status, string(body))
}

// TestCovAuthApiKeys_CreateValidation_MalformedExpiresAt verifies a
// non-RFC3339 expires_at string is rejected with 400/422, not accepted or a
// 5xx.
func TestCovAuthApiKeys_CreateValidation_MalformedExpiresAt(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(apiKeysPath, map[string]any{
		"name":       uniqueName("cov-ak-malexp"),
		"role_id":    SeedAdminRoleID,
		"expires_at": "not-a-date",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"malformed expires_at should return 400 or 422, got %d: %s", status, string(body))
}

// TestCovAuthApiKeys_RotateValidation_MalformedRevokeAt verifies a
// non-RFC3339 revoke_at string on rotate is rejected with 400/422.
func TestCovAuthApiKeys_RotateValidation_MalformedRevokeAt(t *testing.T) {
	t.Parallel()
	m := createAPIKeyAndCleanup(t, uniqueName("cov-ak-malrevat"))
	id := jsonField(jsonObject(m, "api_key_info"), "id")
	require.NotEmpty(t, id)

	status, body, err := apiClient.Post(apiKeysPath+"/"+id+"/actions/rotate", map[string]any{
		"revoke_at": "not-a-date",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"malformed revoke_at should return 400 or 422, got %d: %s", status, string(body))
}

// TestCovAuthApiKeys_RotateValidation_MalformedExpiresAt verifies a
// non-RFC3339 expires_at override string on rotate is rejected with
// 400/422.
func TestCovAuthApiKeys_RotateValidation_MalformedExpiresAt(t *testing.T) {
	t.Parallel()
	m := createAPIKeyAndCleanup(t, uniqueName("cov-ak-malexprot"))
	id := jsonField(jsonObject(m, "api_key_info"), "id")
	require.NotEmpty(t, id)

	status, body, err := apiClient.Post(apiKeysPath+"/"+id+"/actions/rotate", map[string]any{
		"expires_at": "not-a-date",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"malformed expires_at should return 400 or 422, got %d: %s", status, string(body))
}

// TestCovAuthApiKeys_RotatePastRevokeAtCollapsesToImmediate verifies a
// past/now revoke_at on rotate collapses to an immediate revoke of the old
// key, per the mediator's documented behavior, rather than being rejected.
func TestCovAuthApiKeys_RotatePastRevokeAtCollapsesToImmediate(t *testing.T) {
	t.Parallel()
	m := createAPIKeyAndCleanup(t, uniqueName("cov-ak-pastrevat"))
	origID := jsonField(jsonObject(m, "api_key_info"), "id")
	require.NotEmpty(t, origID)

	past := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	status, body, err := apiClient.Post(apiKeysPath+"/"+origID+"/actions/rotate", map[string]any{
		"revoke_at": past,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	newID := jsonField(jsonObject(parseJSON(body), "api_key_info"), "id")
	require.NotEmpty(t, newID)
	defer apiClient.Delete(apiKeysPath + "/" + newID)

	getStatus, getBody, err := apiClient.GetListRaw(apiKeysPath+"/"+origID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.NotNil(t, parseJSON(getBody)["revoked_at"], "old key should be revoked immediately when revoke_at is in the past")
}

// An unrecognized `statuses` value is rejected rather than silently returning an empty list.
func TestCovAuthApiKeys_ListInvalidStatusValueRejected(t *testing.T) {
	t.Parallel()
	// Unrecognized enum values in a list filter are rejected with 400 (platform convention).
	status, body, err := apiClient.GetListRaw(apiKeysPath, url.Values{"statuses": {"bogus"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

// A syntactically valid but nonexistent role_id is rejected on create, rather than producing an API key with no resolvable role.
func TestCovAuthApiKeys_CreateNonexistentRoleIDRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(apiKeysPath, map[string]any{
		"name":    uniqueName("cov-ak-badrole"),
		"role_id": "rl_doesnotexist000000",
	}, newIdempotencyKey())
	require.NoError(t, err)
	if status == 201 {
		id := jsonField(jsonObject(parseJSON(body), "api_key_info"), "id")
		if id != "" {
			defer apiClient.Delete(apiKeysPath + "/" + id)
		}
	}
	assert.True(t, status == 400 || status == 404,
		"nonexistent role_id should be rejected with 400/404, got %d: %s", status, string(body))
}

// ──────────────────────────────────────────────
// 404s: revoke / rotate on nonexistent id
// ──────────────────────────────────────────────

func TestCovAuthApiKeys_RevokeNotFound(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Delete(apiKeysPath + "/apke_doesnotexist0000")
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

func TestCovAuthApiKeys_RotateNotFound(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(apiKeysPath+"/apke_doesnotexist0000/actions/rotate", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

// TestCovAuthApiKeys_RotateAlreadyRevokedKey characterizes the behavior of
// rotating a key that is already revoked (e.g. because a prior rotate call
// already revoked it): the mediator explicitly checks IsRevoked and returns
// a 401 api_key_revoked error (services/auth-service/internal/mediator/
// api_key_mediator.go Rotate, line ~214) rather than 404/409. This is a
// deliberate, documented check (not a code path shared accidentally with
// auth middleware), so this test asserts the actual/current behavior rather
// than flagging it as a bug.
func TestCovAuthApiKeys_RotateAlreadyRevokedKey(t *testing.T) {
	t.Parallel()
	m := createAPIKeyAndCleanup(t, uniqueName("cov-ak-rotaterevoked"))
	id := jsonField(jsonObject(m, "api_key_info"), "id")
	require.NotEmpty(t, id)

	delStatus, delBody, err := apiClient.Delete(apiKeysPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	status, body, err := apiClient.Post(apiKeysPath+"/"+id+"/actions/rotate", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 401, status, body)
	requireErrorResponse(t, body, "api_key_revoked", "invalid_request_error")
}

// ──────────────────────────────────────────────
// crudLifecycle + security: revoked/rotated-out secrets stop authenticating
// ──────────────────────────────────────────────

// TestCovAuthApiKeys_RevokedSecretStopsAuthenticating is the most
// product-relevant gap identified in the audit: a full create -> get ->
// revoke -> (verify old secret 401s) chain, proving the credential is
// actually invalidated and not just marked revoked_at in the read model.
func TestCovAuthApiKeys_RevokedSecretStopsAuthenticating(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(apiKeysPath, map[string]any{
		"name":    uniqueName("cov-ak-revokeauth"),
		"role_id": SeedAdminRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	created := parseJSON(body)
	secret := jsonField(created, "api_key_secret")
	info := jsonObject(created, "api_key_info")
	require.NotNil(t, info)
	id := jsonField(info, "id")
	require.NotEmpty(t, id)
	require.NotEmpty(t, secret)

	getStatus, getBody, err := apiClient.GetListRaw(apiKeysPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	delStatus, delBody, err := apiClient.Delete(apiKeysPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// The revoked secret must no longer authenticate any request.
	revokedClient := apiClient.WithBearerToken(secret, SeedAccountID)
	authStatus, authBody, err := revokedClient.GetListRaw(apiKeysPath, nil)
	require.NoError(t, err)
	requireStatus(t, 401, authStatus, authBody)
	requireErrorResponse(t, authBody, "api_key_revoked", "invalid_request_error")
}

// TestCovAuthApiKeys_RotatedOutSecretStopsAuthenticating proves the old
// key's secret stops authenticating once it has been rotated out.
func TestCovAuthApiKeys_RotatedOutSecretStopsAuthenticating(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(apiKeysPath, map[string]any{
		"name":    uniqueName("cov-ak-rotateauth"),
		"role_id": SeedAdminRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	created := parseJSON(body)
	oldSecret := jsonField(created, "api_key_secret")
	origID := jsonField(jsonObject(created, "api_key_info"), "id")
	require.NotEmpty(t, origID)
	require.NotEmpty(t, oldSecret)

	rotStatus, rotBody, err := apiClient.Post(apiKeysPath+"/"+origID+"/actions/rotate", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, rotStatus, rotBody)
	newID := jsonField(jsonObject(parseJSON(rotBody), "api_key_info"), "id")
	require.NotEmpty(t, newID)
	defer apiClient.Delete(apiKeysPath + "/" + newID)

	rotatedClient := apiClient.WithBearerToken(oldSecret, SeedAccountID)
	authStatus, authBody, err := rotatedClient.GetListRaw(apiKeysPath, nil)
	require.NoError(t, err)
	requireStatus(t, 401, authStatus, authBody)
	requireErrorResponse(t, authBody, "api_key_revoked", "invalid_request_error")
}

// ──────────────────────────────────────────────
// authorization (401/403)
// ──────────────────────────────────────────────

// TestCovAuthApiKeys_NonAdminForbiddenOnList verifies a non-admin-rooted API
// key is rejected on GET (all 5 routes in this group require RoleTypeAdmin).
func TestCovAuthApiKeys_NonAdminForbiddenOnList(t *testing.T) {
	t.Parallel()
	m := createNonAdminAPIKeyAndCleanup(t, uniqueName("cov-ak-nonadmin-list"))
	secret := jsonField(m, "api_key_secret")
	require.NotEmpty(t, secret)

	nonAdminClient := apiClient.WithBearerToken(secret, SeedAccountID)
	status, body, err := nonAdminClient.GetListRaw(apiKeysPath, nil)
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
}

// TestCovAuthApiKeys_NonAdminForbiddenOnCreate verifies a non-admin-rooted
// API key is rejected on POST create.
func TestCovAuthApiKeys_NonAdminForbiddenOnCreate(t *testing.T) {
	t.Parallel()
	m := createNonAdminAPIKeyAndCleanup(t, uniqueName("cov-ak-nonadmin-create"))
	secret := jsonField(m, "api_key_secret")
	require.NotEmpty(t, secret)

	nonAdminClient := apiClient.WithBearerToken(secret, SeedAccountID)
	status, body, err := nonAdminClient.Post(apiKeysPath, map[string]any{
		"name":    uniqueName("cov-ak-should-not-create"),
		"role_id": SeedAdminRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
}

// TestCovAuthApiKeys_EmptyBearerTokenUnauthorized verifies a request with an
// empty bearer token ("Authorization: Bearer ") is rejected with 401
// invalid_credentials.
func TestCovAuthApiKeys_EmptyBearerTokenUnauthorized(t *testing.T) {
	t.Parallel()
	anonClient := apiClient.WithBearerToken("", SeedAccountID)
	status, body, err := anonClient.GetListRaw(apiKeysPath, nil)
	require.NoError(t, err)
	requireStatus(t, 401, status, body)
	requireErrorResponse(t, body, "invalid_credentials", "invalid_request_error")
}

// createNonAdminAPIKeyAndCleanup creates an API key rooted to a non-admin
// role (scanner) and registers cleanup with the (admin) apiClient. Returns
// the parsed create response (api_key_secret + api_key_info at top level).
func createNonAdminAPIKeyAndCleanup(t *testing.T, name string) map[string]any {
	t.Helper()
	status, body, err := apiClient.Post(apiKeysPath, map[string]any{
		"name":    name,
		"role_id": SeedScannerRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	parsed := parseJSON(body)
	info := jsonObject(parsed, "api_key_info")
	require.NotNil(t, info)
	id := jsonField(info, "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { apiClient.Delete(apiKeysPath + "/" + id) })
	return parsed
}
