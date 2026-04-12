//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const apiKeysPath = "/v1/auth/api-keys"

func TestAPIKeys_CreateAndGet(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-key")
	createResp, err := apiClient.PostFull(apiKeysPath, map[string]any{
		"name":    name,
		"role_id": SeedAdminRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	m := parseJSON(createResp.Body)
	assert.NotEmpty(t, jsonField(m, "api_key_secret"))
	info := jsonObject(m, "api_key_info")
	require.NotNil(t, info)
	assert.Equal(t, "api_key", jsonField(info, "object"))
	id := jsonField(info, "id")
	assert.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	assert.Equal(t, name, jsonField(info, "name"))
	assert.NotEmpty(t, jsonField(info, "redacted_value"))
	assertValidTimestamp(t, jsonField(info, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(info, "updated_at"), "updated_at")
	assertNilField(t, info, "expires_at")
	assertNilField(t, info, "last_used_at")
	assertNilField(t, info, "revoked_at")
	assertNilField(t, info, "role")

	getStatus, getBody, err := apiClient.GetListRaw(apiKeysPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assert.Equal(t, "api_key", jsonField(got, "object"))
	assert.Equal(t, id, jsonField(got, "id"))

	apiClient.Delete(apiKeysPath + "/" + id)
}

func TestAPIKeys_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(apiKeysPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 1)

	found := false
	for _, item := range list.Data {
		if DataItemField(item, "id") == SeedAPIKeyID {
			found = true
			break
		}
	}
	assert.True(t, found, "Seeded API key should appear in list")
}

func TestAPIKeys_Revoke(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(apiKeysPath, map[string]any{
		"name":    uniqueName("e2e-revoke"),
		"role_id": SeedAdminRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(jsonObject(parseJSON(body), "api_key_info"), "id")

	delStatus, delBody, err := apiClient.Delete(apiKeysPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	getStatus, getBody, err := apiClient.GetListRaw(apiKeysPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.NotNil(t, parseJSON(getBody)["revoked_at"])
}

func TestAPIKeys_Rotate(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(apiKeysPath, map[string]any{
		"name":    uniqueName("e2e-rotate"),
		"role_id": SeedAdminRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	origID := jsonField(jsonObject(parseJSON(body), "api_key_info"), "id")

	rotStatus, rotBody, err := apiClient.Post(apiKeysPath+"/"+origID+"/actions/rotate", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, rotStatus, rotBody)

	rotated := parseJSON(rotBody)
	assert.NotEmpty(t, jsonField(rotated, "api_key_secret"))
	newID := jsonField(jsonObject(rotated, "api_key_info"), "id")
	assert.NotEqual(t, origID, newID)

	getStatus, getBody, err := apiClient.GetListRaw(apiKeysPath+"/"+origID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.NotNil(t, parseJSON(getBody)["revoked_at"])

	apiClient.Delete(apiKeysPath + "/" + newID)
}

func TestAPIKeys_IncludeRole(t *testing.T) {
	t.Parallel()
	getStatus, getBody, err := apiClient.GetListRaw(apiKeysPath+"/"+SeedAPIKeyID, url.Values{"include": {"role"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	role := jsonObject(parseJSON(getBody), "role")
	require.NotNil(t, role)
	assert.NotEmpty(t, jsonField(role, "id"))
	assert.Equal(t, "role", jsonField(role, "object"))
}

func TestAPIKeys_IncludeRolePermissions(t *testing.T) {
	t.Parallel()
	getStatus, getBody, err := apiClient.GetListRaw(apiKeysPath+"/"+SeedAPIKeyID, url.Values{"include": {"role,role.permissions"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	role := jsonObject(got, "role")
	require.NotNil(t, role, "role should be present with ?include=role,role.permissions")
	_, ok := role["permissions"]
	assert.True(t, ok, "role.permissions should be present")
}

func TestAPIKeys_ListSearchByName(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(apiKeysPath, url.Values{"q": {"Admin API Key"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Search for 'Admin API Key' should return at least 1 result")
}

func TestAPIKeys_ListSearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(apiKeysPath, url.Values{"q": {"zzzznotakey99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

func TestAPIKeys_ListResponseShape(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(apiKeysPath, nil)
	require.NoError(t, err)

	for _, item := range list.Data {
		m := parseJSON(item)
		require.NotNil(t, m)
		assert.Equal(t, "api_key", jsonField(m, "object"))
		assert.NotEmpty(t, jsonField(m, "id"))
		assert.NotEmpty(t, jsonField(m, "name"))
		assert.NotEmpty(t, jsonField(m, "redacted_value"))
		assert.NotEmpty(t, jsonField(m, "created_at"))
		assert.NotEmpty(t, jsonField(m, "updated_at"))
	}
}

func TestAPIKeys_ListFilterByStatusActive(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(apiKeysPath, url.Values{"status": {"active"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Should have at least 1 active API key")

	for _, item := range list.Data {
		m := parseJSON(item)
		assert.Nil(t, m["revoked_at"], "Active keys should not have revoked_at set")
	}
}

func TestAPIKeys_ListFilterByStatusRevoked(t *testing.T) {
	t.Parallel()

	// Create and revoke a key so we know one exists
	createStatus, createBody, err := apiClient.Post(apiKeysPath, map[string]any{
		"name":    uniqueName("e2e-status-rev"),
		"role_id": SeedAdminRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(jsonObject(parseJSON(createBody), "api_key_info"), "id")
	apiClient.Delete(apiKeysPath + "/" + id)

	list, _, err := apiClient.GetList(apiKeysPath, url.Values{"status": {"revoked"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Should have at least 1 revoked API key")

	for _, item := range list.Data {
		m := parseJSON(item)
		assert.NotNil(t, m["revoked_at"], "Revoked keys should have revoked_at set")
	}
}

func TestAPIKeys_ListPagination(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(apiKeysPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	assert.Len(t, list.Data, 1)
}

func TestAPIKeys_GetNotFound(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.GetListRaw(apiKeysPath+"/apky_000000000000", nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status)
}

func TestAPIKeys_GetAllFields(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(apiKeysPath+"/"+SeedAPIKeyID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, SeedAPIKeyID, jsonField(got, "id"))
	assert.Equal(t, "api_key", jsonField(got, "object"))
	assert.NotEmpty(t, jsonField(got, "name"))
	assert.NotEmpty(t, jsonField(got, "redacted_value"))
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	assertNilField(t, got, "role")
	// expires_at, last_used_at, revoked_at fields should be present in the response
	_, hasExpiresAt := got["expires_at"]
	assert.True(t, hasExpiresAt, "expires_at field should be present in response")
	_, hasLastUsedAt := got["last_used_at"]
	assert.True(t, hasLastUsedAt, "last_used_at field should be present in response")
	_, hasRevokedAt := got["revoked_at"]
	assert.True(t, hasRevokedAt, "revoked_at field should be present in response")
}

func TestAPIKeys_Idempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-idem-key")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Post(apiKeysPath, map[string]any{
		"name":    name,
		"role_id": SeedAdminRoleID,
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(jsonObject(parseJSON(body1), "api_key_info"), "id")

	status2, body2, err := apiClient.Post(apiKeysPath, map[string]any{
		"name":    name,
		"role_id": SeedAdminRoleID,
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	id2 := jsonField(jsonObject(parseJSON(body2), "api_key_info"), "id")
	assert.Equal(t, id1, id2)

	apiClient.Delete(apiKeysPath + "/" + id1)
}

func TestAPIKeys_CreateValidation_MissingName(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(apiKeysPath, map[string]any{
		"role_id": SeedAdminRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Missing name should return 400 or 422, got %d: %s", status, string(body))
}

func TestAPIKeys_CreateValidation_MissingRoleID(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(apiKeysPath, map[string]any{
		"name": uniqueName("e2e-norole"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Missing role_id should return 400 or 422, got %d: %s", status, string(body))
}

func TestAPIKeys_CreateValidation_EmptyName(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(apiKeysPath, map[string]any{
		"name":    "",
		"role_id": SeedAdminRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Empty name should return 400 or 422, got %d: %s", status, string(body))
}

func TestAPIKeys_CreateValidation_ErrorShape(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(apiKeysPath, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
}

// ──────────────────────────────────────────────
// Omitted Fields
// ──────────────────────────────────────────────

func TestAPIKeys_OmittedFields(t *testing.T) {
	t.Parallel()

	t.Run("CreateWithOnlyRequiredFields", func(t *testing.T) {
		name := uniqueName("e2e-key-omit")
		status, body, err := apiClient.Post(apiKeysPath, map[string]any{
			"name":    name,
			"role_id": SeedAdminRoleID,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)

		m := parseJSON(body)
		assert.NotEmpty(t, jsonField(m, "api_key_secret"))
		info := jsonObject(m, "api_key_info")
		require.NotNil(t, info)
		id := jsonField(info, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(apiKeysPath + "/" + id)

		assertObjectField(t, info, "api_key")
		assert.Equal(t, name, jsonField(info, "name"))
		assert.NotEmpty(t, jsonField(info, "redacted_value"))
		assertValidTimestamp(t, jsonField(info, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(info, "updated_at"), "updated_at")
		assertNilField(t, info, "expires_at")
		assertNilField(t, info, "last_used_at")
		assertNilField(t, info, "revoked_at")
		assertNilField(t, info, "role")
	})

	t.Run("CreateMissingRequiredFields", func(t *testing.T) {
		// Missing name
		status, body, err := apiClient.Post(apiKeysPath, map[string]any{
			"role_id": SeedAdminRoleID,
		}, newIdempotencyKey())
		require.NoError(t, err)
		assert.True(t, status == 400 || status == 422,
			"Missing name should return 400 or 422, got %d: %s", status, string(body))

		// Missing role_id
		status2, body2, err := apiClient.Post(apiKeysPath, map[string]any{
			"name": uniqueName("e2e-key-norole"),
		}, newIdempotencyKey())
		require.NoError(t, err)
		assert.True(t, status2 == 400 || status2 == 422,
			"Missing role_id should return 400 or 422, got %d: %s", status2, string(body2))
	})
}

func TestAPIKeys_RoleNullWithoutInclude(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(apiKeysPath+"/"+SeedAPIKeyID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	// Without ?include=role, the role field should be null (expandable sub-resource)
	assert.Nil(t, got["role"], "role should be null without ?include=role")
}
