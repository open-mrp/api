//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const rolesPath = "/v1/identity/roles"

// --- List ---

func TestRoles_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(rolesPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Should have at least 1 seeded role")

	found := false
	for _, item := range list.Data {
		if DataItemField(item, "id") == SeedAdminRoleID {
			found = true
			break
		}
	}
	assert.True(t, found, "Seeded admin role should appear in list")
}

func TestRoles_ListResponseShape(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(rolesPath, nil)
	require.NoError(t, err)

	for _, item := range list.Data {
		m := parseJSON(item)
		require.NotNil(t, m)
		assert.Equal(t, "role", jsonField(m, "object"))
		assert.NotEmpty(t, jsonField(m, "id"))
		assert.NotEmpty(t, jsonField(m, "name"))
		assert.NotEmpty(t, jsonField(m, "type"))
		assert.NotEmpty(t, jsonField(m, "created_at"))
		assert.NotEmpty(t, jsonField(m, "updated_at"))
	}
}

func TestRoles_ListFilterByRoleType(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(rolesPath, url.Values{"types": {"admin"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Should have at least 1 admin role")

	for _, item := range list.Data {
		assert.Equal(t, "admin", DataItemField(item, "type"))
	}
}

func TestRoles_ListPagination(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(rolesPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	assert.Len(t, list.Data, 1)
}

func TestRoles_ListSearchByName(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(rolesPath, url.Values{"q": {"Admin"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Search for 'Admin' should return at least 1 result")

	for _, item := range list.Data {
		name := DataItemField(item, "name")
		assert.True(t,
			strings.Contains(strings.ToLower(name), "admin"),
			"Search result %q should contain 'admin'", name,
		)
	}
}

func TestRoles_ListSearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(rolesPath, url.Values{"q": {"zzzznotarole99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

// --- Get ---

func TestRoles_GetByID(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(rolesPath+"/"+SeedAdminRoleID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, SeedAdminRoleID, jsonField(got, "id"))
	assert.Equal(t, "role", jsonField(got, "object"))
	assert.NotEmpty(t, jsonField(got, "name"))
	assert.NotEmpty(t, jsonField(got, "type"))
	assert.NotEmpty(t, jsonField(got, "created_at"))
	assert.NotEmpty(t, jsonField(got, "updated_at"))
}

func TestRoles_GetNotFound(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.GetListRaw(rolesPath+"/rl_000000000000", nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status)
}

// --- Includes ---

func TestRoles_IncludePermissions(t *testing.T) {
	t.Parallel()

	// Create a role with known permissions so we can verify the include value
	name := uniqueName("e2e-role-incl-get")
	createStatus, createBody, err := apiClient.Post(rolesPath, map[string]any{
		"name":        name,
		"permissions": []string{"customers:create", "customers:read"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	roleID := jsonField(parseJSON(createBody), "id")

	// Get with ?include=permissions
	status, body, err := apiClient.GetListRaw(rolesPath+"/"+roleID, url.Values{"include": {"permissions"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	perms, ok := got["permissions"]
	assert.True(t, ok, "permissions field should be present with ?include=permissions")
	permsSlice, ok := perms.([]any)
	require.True(t, ok, "permissions should be a slice")
	assert.GreaterOrEqual(t, len(permsSlice), 1, "permissions should contain at least one entry")

	// Verify permission format is "<domain>:<action>"
	firstPerm, ok := permsSlice[0].(string)
	require.True(t, ok, "each permission should be a string")
	assert.Contains(t, firstPerm, ":", "permission should be in '<domain>:<action>' format")

	// Cleanup
	apiClient.Delete(rolesPath + "/" + roleID)
}

func TestRoles_IncludeOwnerAndPermissions(t *testing.T) {
	t.Parallel()

	// Create a role with permissions
	name := uniqueName("e2e-role-incl-both")
	createStatus, createBody, err := apiClient.Post(rolesPath, map[string]any{
		"name":        name,
		"permissions": []string{"customers:create", "customers:read"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	roleID := jsonField(parseJSON(createBody), "id")

	// Request both includes at once
	status, body, err := apiClient.GetListRaw(rolesPath+"/"+roleID, url.Values{"include": {"owner,permissions"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)

	// Verify owner is populated
	owner := jsonObject(got, "owner")
	require.NotNil(t, owner, "owner should be present with ?include=owner,permissions")
	assert.Equal(t, "owner", jsonField(owner, "object"))

	// Verify permissions is populated
	perms, ok := got["permissions"]
	assert.True(t, ok, "permissions should be present")
	permsSlice, ok := perms.([]any)
	require.True(t, ok, "permissions should be a slice")
	assert.GreaterOrEqual(t, len(permsSlice), 1, "permissions should contain at least one entry")

	// Cleanup
	apiClient.Delete(rolesPath + "/" + roleID)
}

// --- CRUD ---

func TestRoles_CRUD(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-role")

	// Create
	createResp, err := apiClient.PostFull(rolesPath, map[string]any{
		"name":        name,
		"permissions": []string{"customers:read"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	assert.Equal(t, "role", jsonField(created, "object"))
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	assert.Equal(t, name, jsonField(created, "name"))

	// Get
	getStatus, getBody, err := apiClient.GetListRaw(rolesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, name, jsonField(parseJSON(getBody), "name"))

	// Update name
	newName := uniqueName("e2e-role-upd")
	patchStatus, patchBody, err := apiClient.Patch(rolesPath+"/"+id, map[string]any{
		"name": newName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	assert.Equal(t, newName, jsonField(parseJSON(patchBody), "name"))

	// Verify update
	getStatus2, getBody2, err := apiClient.GetListRaw(rolesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus2, getBody2)
	assert.Equal(t, newName, jsonField(parseJSON(getBody2), "name"))

	// Delete
	delStatus, delBody, err := apiClient.Delete(rolesPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// Verify deletion
	getStatus3, _, err := apiClient.GetListRaw(rolesPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus3)
}

func TestRoles_UpdatePermissions(t *testing.T) {
	name := uniqueName("e2e-role-perms")

	// Create with one permission
	createStatus, createBody, err := apiClient.Post(rolesPath+"?include[]=permissions", map[string]any{
		"name":        name,
		"permissions": []string{"customers:read"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	created := parseJSON(createBody)
	permissions, ok := created["permissions"]
	require.True(t, ok, "permissions field should be present")
	permsSlice, ok := permissions.([]any)
	require.True(t, ok && len(permsSlice) == 1, "Should have 1 permission after create")

	// Update with different permissions
	patchStatus, patchBody, err := apiClient.Patch(rolesPath+"/"+id+"?include[]=permissions", map[string]any{
		"permissions": []string{
			"customers:create", "customers:read", "customers:update", "customers:delete",
			"invoices:read",
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	updatedPerms, ok := updated["permissions"]
	require.True(t, ok, "permissions field should be present after update")
	updatedPermsSlice, ok := updatedPerms.([]any)
	// 2 permission codes with multiple CRUD flags: customers (4 actions) + invoices (1 action) = 5 entries
	require.True(t, ok && len(updatedPermsSlice) >= 2, "Should have at least 2 permissions after update, got %d", len(updatedPermsSlice))

	// Cleanup
	apiClient.Delete(rolesPath + "/" + id)
}

func TestRoles_Idempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-idem-role")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Post(rolesPath, map[string]any{"name": name}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")

	status2, body2, err := apiClient.Post(rolesPath, map[string]any{"name": name}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id1, jsonField(parseJSON(body2), "id"))

	apiClient.Delete(rolesPath + "/" + id1)
}

func TestRoles_CreateValidation_MissingName(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(rolesPath, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422, "Missing name should return 400 or 422, got %d: %s", status, string(body))
}

func TestRoles_DeleteGlobalRoleFails(t *testing.T) {
	t.Parallel()
	// The seeded admin role is a global/internal role and should not be deletable
	status, _, err := apiClient.Delete(rolesPath + "/" + SeedAdminRoleID)
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 403 || status == 409 || status == 422,
		"Deleting a global role should fail, got %d", status)
}

// --- Expandable fields ---

func TestRoles_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	// Test on Get
	status, body, err := apiClient.GetListRaw(rolesPath+"/"+SeedAdminRoleID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["owner"], "owner should be null without ?include=owner")
	assert.Nil(t, got["permissions"], "permissions should be null without ?include=permissions")

	// Test on List
	list, _, err := apiClient.GetList(rolesPath, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	for _, item := range list.Data {
		m := parseJSON(item)
		assert.Nil(t, m["owner"], "owner should be null on list items without ?include=owner")
		assert.Nil(t, m["permissions"], "permissions should be null on list items without ?include=permissions")
	}
}

func TestRoles_IncludeOwner(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(rolesPath+"/"+SeedAdminRoleID, url.Values{"include": {"owner"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	owner := jsonObject(got, "owner")
	require.NotNil(t, owner, "owner should be present with ?include=owner")
	assert.Equal(t, "owner", jsonField(owner, "object"))
	ownerType := jsonField(owner, "type")
	assert.Contains(t, []string{"system", "account"}, ownerType)
	if ownerType == "account" {
		acct := jsonObject(owner, "account")
		require.NotNil(t, acct)
		assert.NotEmpty(t, jsonField(acct, "id"))
		assert.Equal(t, "account", jsonField(acct, "object"))
	}
}

func TestRoles_ListWithIncludePermissions(t *testing.T) {
	t.Parallel()

	// Create a role with known permissions so at least one item has them
	name := uniqueName("e2e-role-incl-list")
	createStatus, createBody, err := apiClient.Post(rolesPath, map[string]any{
		"name":        name,
		"permissions": []string{"customers:create", "customers:read"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	roleID := jsonField(parseJSON(createBody), "id")

	list, _, err := apiClient.GetList(rolesPath, url.Values{"include": {"permissions"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1)

	foundWithPerms := false
	for _, item := range list.Data {
		m := parseJSON(item)
		perms, ok := m["permissions"]
		assert.True(t, ok, "permissions field should be present on each item with ?include=permissions")
		if permsSlice, isSlice := perms.([]any); isSlice && len(permsSlice) > 0 {
			foundWithPerms = true
			// Verify format
			firstPerm, isStr := permsSlice[0].(string)
			if isStr {
				assert.Contains(t, firstPerm, ":", "permission should be in '<domain>:<action>' format")
			}
		}
	}
	assert.True(t, foundWithPerms, "at least one role in the list should have non-empty permissions")

	// Cleanup
	apiClient.Delete(rolesPath + "/" + roleID)
}

func TestRoles_ListWithIncludeOwner(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(rolesPath, url.Values{"include": {"owner"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1)

	for _, item := range list.Data {
		m := parseJSON(item)
		owner := jsonObject(m, "owner")
		require.NotNil(t, owner, "owner should be present on each item with ?include=owner")
		assert.Equal(t, "owner", jsonField(owner, "object"))
	}
}

// --- Update ---

func TestRoles_UpdateGlobalRoleFails(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.Patch(rolesPath+"/"+SeedAdminRoleID, map[string]any{
		"name": uniqueName("e2e-global-upd"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 403 || status == 409 || status == 422,
		"Updating a global role should fail, got %d", status)
}

func TestRoles_UpdateIdempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-role-idem-upd")
	createStatus, createBody, err := apiClient.Post(rolesPath, map[string]any{"name": name}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	newName := uniqueName("e2e-role-idem-upd2")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Patch(rolesPath+"/"+id, map[string]any{"name": newName}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)

	status2, body2, err := apiClient.Patch(rolesPath+"/"+id, map[string]any{"name": newName}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)

	assert.Equal(t, jsonField(parseJSON(body1), "name"), jsonField(parseJSON(body2), "name"))
	assert.Equal(t, jsonField(parseJSON(body1), "id"), jsonField(parseJSON(body2), "id"))

	apiClient.Delete(rolesPath + "/" + id)
}

// --- Conflict / Validation ---

func TestRoles_CreateDuplicateNameFails(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-role-dup")

	createStatus, createBody, err := apiClient.Post(rolesPath, map[string]any{"name": name}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	// Attempt to create another role with the same name (different idempotency key)
	status2, body2, err := apiClient.Post(rolesPath, map[string]any{"name": name}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 409, status2, "Duplicate name should return 409, got %d: %s", status2, string(body2))

	apiClient.Delete(rolesPath + "/" + id)
}

func TestRoles_DeleteAlreadyDeletedFails(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-role-deldel")
	createStatus, createBody, err := apiClient.Post(rolesPath, map[string]any{"name": name}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	delStatus, delBody, err := apiClient.Delete(rolesPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// Second delete should fail
	status2, _, err := apiClient.Delete(rolesPath + "/" + id)
	require.NoError(t, err)
	assert.True(t, status2 == 404 || status2 == 410,
		"Deleting an already-deleted role should return 404 or 410, got %d", status2)
}

func TestRoles_DeleteBlockedWhenUsersAssigned(t *testing.T) {
	t.Parallel()

	roleName := uniqueName("e2e-role-assigned")
	createRoleStatus, createRoleBody, err := apiClient.Post(rolesPath, map[string]any{"name": roleName}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createRoleStatus, createRoleBody)
	roleID := jsonField(parseJSON(createRoleBody), "id")

	userName := uniqueName("e2e-role-assigned-user")
	userEmail := userName + "@e2e-test.augno.com"
	createUserStatus, createUserBody, err := apiClient.Post(accountUsersPath, map[string]any{
		"name":    userName,
		"email":   userEmail,
		"role_id": roleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createUserStatus, createUserBody)
	accountUserID := jsonField(parseJSON(createUserBody), "id")
	defer removeAccountUser(accountUserID)

	deleteStatus, deleteBody, err := apiClient.Delete(rolesPath + "/" + roleID)
	require.NoError(t, err)
	requireStatus(t, 409, deleteStatus, deleteBody)
	requireErrorResponse(t, deleteBody, "resource_conflict", "invalid_request_error")

	removeAccountUser(accountUserID)
	delStatus, delBody, err := apiClient.Delete(rolesPath + "/" + roleID)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)
}

func TestRoles_DeleteSucceedsAfterUsersUnassigned(t *testing.T) {
	t.Parallel()

	roleName := uniqueName("e2e-role-unassign")
	createRoleStatus, createRoleBody, err := apiClient.Post(rolesPath, map[string]any{"name": roleName}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createRoleStatus, createRoleBody)
	roleID := jsonField(parseJSON(createRoleBody), "id")

	userName := uniqueName("e2e-role-unassign-user")
	userEmail := userName + "@e2e-test.augno.com"
	createUserStatus, createUserBody, err := apiClient.Post(accountUsersPath, map[string]any{
		"name":    userName,
		"email":   userEmail,
		"role_id": roleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createUserStatus, createUserBody)
	accountUserID := jsonField(parseJSON(createUserBody), "id")
	defer removeAccountUser(accountUserID)

	patchStatus, patchBody, err := apiClient.Patch(accountUsersPath+"/"+accountUserID, map[string]any{
		"role_id": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	deleteStatus, deleteBody, err := apiClient.Delete(rolesPath + "/" + roleID)
	require.NoError(t, err)
	requireStatus(t, 200, deleteStatus, deleteBody)
}

func TestRoles_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()

	// ── CREATE with all fields ──
	name := uniqueName("e2e-role-allf")
	createResp, err := apiClient.PostFull(rolesPath+"?include=permissions", map[string]any{
		"name":        name,
		"permissions": []string{"customers:create", "customers:read"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	got := parseJSON(createResp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	defer apiClient.Delete(rolesPath + "/" + id)

	assert.Equal(t, "role", jsonField(got, "object"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, "user", jsonField(got, "type"))
	assert.NotEmpty(t, jsonField(got, "created_at"))
	assert.NotEmpty(t, jsonField(got, "updated_at"))

	perms, ok := got["permissions"]
	require.True(t, ok, "permissions should be expanded")
	permsSlice, ok := perms.([]any)
	require.True(t, ok, "permissions should be a slice")
	assert.GreaterOrEqual(t, len(permsSlice), 1, "should have at least 1 permission after create")

	// ── UPDATE with different values ──
	updatedName := uniqueName("e2e-role-allf-u")
	patchStatus, patchBody, err := apiClient.Patch(rolesPath+"/"+id+"?include=permissions", map[string]any{
		"name":        updatedName,
		"permissions": []string{"customers:create", "customers:read", "customers:update", "invoices:read"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	assert.Equal(t, id, jsonField(updated, "id"), "ID must not change")
	assert.Equal(t, updatedName, jsonField(updated, "name"))
	assert.Equal(t, "user", jsonField(updated, "type"), "type should be preserved")

	updPerms, ok := updated["permissions"]
	require.True(t, ok, "permissions should be expanded after update")
	updPermsSlice, ok := updPerms.([]any)
	require.True(t, ok, "permissions should be a slice")
	assert.GreaterOrEqual(t, len(updPermsSlice), 2, "should have more permissions after update")
}

func TestRoles_CreateResponseShape(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-role-shape")
	createResp, err := apiClient.PostFull(rolesPath, map[string]any{
		"name":        name,
		"permissions": []string{"customers:create", "customers:read"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	assert.Equal(t, "role", jsonField(created, "object"))
	assert.Equal(t, name, jsonField(created, "name"))
	assert.Equal(t, "user", jsonField(created, "type"), "Custom roles should have type 'user'")
	assert.NotEmpty(t, jsonField(created, "created_at"))
	assert.NotEmpty(t, jsonField(created, "updated_at"))

	apiClient.Delete(rolesPath + "/" + id)
}

func TestRoles_ListCursorPagination(t *testing.T) {
	t.Parallel()
	page1, _, err := apiClient.GetList(rolesPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	requirePageLen(t, page1.Data, 1)

	if !page1.PageInfo.HasNextPage {
		t.Skip("Not enough roles for pagination test")
		return
	}
	require.NotNil(t, page1.PageInfo.NextPageURL, "next_page_url should be set when has_next_page is true")

	page1ID := DataItemField(page1.Data[0], "id")
	assert.NotEmpty(t, page1ID)

	page2, _, err := apiClient.GetListFromPageURL(page1.PageInfo.NextPageURL)
	require.NoError(t, err)
	requirePageLen(t, page2.Data, 1)

	page2ID := DataItemField(page2.Data[0], "id")
	assert.NotEqual(t, page1ID, page2ID, "Page 2 should return a different item than page 1")
}

func TestRoles_CreateValidation_EmptyName(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(rolesPath, map[string]any{"name": ""}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Empty name should return 400 or 422, got %d: %s", status, string(body))
}

func TestRoles_UpdateValidation_EmptyName(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-role-blank")
	createStatus, createBody, err := apiClient.Post(rolesPath, map[string]any{"name": name}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	defer apiClient.Delete(rolesPath + "/" + id)

	status, body, err := apiClient.Patch(rolesPath+"/"+id, map[string]any{"name": ""}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
}

func TestRoles_UpdateValidation_WhitespaceOnlyName(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-role-ws")
	createStatus, createBody, err := apiClient.Post(rolesPath, map[string]any{"name": name}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	defer apiClient.Delete(rolesPath + "/" + id)

	status, body, err := apiClient.Patch(rolesPath+"/"+id, map[string]any{"name": "   "}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
}

func TestRoles_GetNotFound_ErrorShape(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(rolesPath+"/rl_000000000000", nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)

	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

func TestRoles_CreateDuplicate_ErrorShape(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-role-dup")

	// Create first role.
	status1, body1, err := apiClient.Post(rolesPath, map[string]any{"name": name}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")

	// Create second role with same name — expect conflict.
	status2, body2, err := apiClient.Post(rolesPath, map[string]any{"name": name}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status2, body2)

	requireErrorResponse(t, body2, "resource_conflict", "invalid_request_error")

	apiClient.Delete(rolesPath + "/" + id1)
}
