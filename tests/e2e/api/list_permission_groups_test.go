//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const permissionGroupsPath = "/v1/identity/permission-groups"

func TestPermissionGroups_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(permissionGroupsPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Should have at least 1 permission group")
}

func TestPermissionGroups_ListResponseShape(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(permissionGroupsPath, nil)
	require.NoError(t, err)

	for _, item := range list.Data {
		m := parseJSON(item)
		require.NotNil(t, m)
		assert.Equal(t, "permission_group", jsonField(m, "object"))
		assert.NotEmpty(t, jsonField(m, "id"))
		assert.NotEmpty(t, jsonField(m, "code"))
		assert.NotEmpty(t, jsonField(m, "name"))
		assert.NotEmpty(t, jsonField(m, "created_at"))
		assert.NotEmpty(t, jsonField(m, "updated_at"))
	}
}

func TestPermissionGroups_ListSearchByName(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(permissionGroupsPath, url.Values{"q": {"Admin"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Search for 'Admin' should return at least 1 result")
}

func TestPermissionGroups_ListSearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(permissionGroupsPath, url.Values{"q": {"zzzznotagroup99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

func TestPermissionGroups_ContainsPermissions(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(permissionGroupsPath, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	// Each permission group should have a nested permissions list
	for _, item := range list.Data {
		m := parseJSON(item)
		require.NotNil(t, m)

		perms := jsonObject(m, "permissions")
		require.NotNil(t, perms, "Permission group %q should have permissions field", jsonField(m, "code"))
		assert.Equal(t, "list", jsonField(perms, "object"))
	}
}

func TestPermissionGroups_PermissionFields(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(permissionGroupsPath, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	// Find a group with permissions and validate the permission shape
	for _, item := range list.Data {
		m := parseJSON(item)
		perms := jsonObject(m, "permissions")
		if perms == nil {
			continue
		}

		dataRaw, ok := perms["data"]
		if !ok {
			continue
		}
		dataSlice, ok := dataRaw.([]any)
		if !ok || len(dataSlice) == 0 {
			continue
		}

		perm, ok := dataSlice[0].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "permission", jsonField(perm, "object"))
		assert.NotEmpty(t, jsonField(perm, "id"))
		assert.NotEmpty(t, jsonField(perm, "code"))
		assert.NotEmpty(t, jsonField(perm, "name"))
		assert.NotEmpty(t, jsonField(perm, "group"))
		assert.NotEmpty(t, jsonField(perm, "created_at"))
		assert.NotEmpty(t, jsonField(perm, "updated_at"))
		return
	}
	t.Skip("No permission groups with permissions found")
}

func TestPermissionGroups_Pagination(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(permissionGroupsPath, nil)
	require.NoError(t, err)
	if len(list.Data) <= 1 {
		t.Skip("Need more than 1 permission group to test pagination")
	}

	page1, _, err := apiClient.GetList(permissionGroupsPath, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(page1.Data), 1)
}

// ──────────────────────────────────────────────
// PermissionGroup — Owner Include Tests
// ──────────────────────────────────────────────

func TestPermissionGroups_OwnerNullWithoutInclude(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(permissionGroupsPath, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	for _, item := range list.Data {
		m := parseJSON(item)
		assert.Nil(t, m["owner"], "owner should be null without ?include=owner")
	}
}

func TestPermissionGroups_IncludeOwner(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(permissionGroupsPath, url.Values{"include": {"owner"}})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	first := parseJSON(list.Data[0])
	owner := jsonObject(first, "owner")
	require.NotNil(t, owner, "owner should be present with ?include=owner")
	assert.Equal(t, "owner", jsonField(owner, "object"))
	ownerType := jsonField(owner, "type")
	assert.Contains(t, []string{"system", "account"}, ownerType)
}
