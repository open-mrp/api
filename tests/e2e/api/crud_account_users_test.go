//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const accountUsersPath = "/v1/identity/account-users"

// --- List ---

func TestAccountUsers_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(accountUsersPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Should have at least 1 seeded account user")

	found := false
	for _, item := range list.Data {
		if DataItemField(item, "id") == SeedAccountUserID {
			found = true
			break
		}
	}
	assert.True(t, found, "Seeded account user should appear in list")
}

func TestAccountUsers_ListResponseShape(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(accountUsersPath, nil)
	require.NoError(t, err)

	for _, item := range list.Data {
		m := parseJSON(item)
		require.NotNil(t, m)
		assert.Equal(t, "account_user", jsonField(m, "object"))
		assert.NotEmpty(t, jsonField(m, "id"))
		assert.NotEmpty(t, jsonField(m, "status"))
		assert.NotEmpty(t, jsonField(m, "created_at"))
		assert.NotEmpty(t, jsonField(m, "updated_at"))
	}
}

func TestAccountUsers_ListPagination(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(accountUsersPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	assert.Len(t, list.Data, 1)
}

func TestAccountUsers_ListFilterByRoleType(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(accountUsersPath, url.Values{"role_type": {"admin"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Should have at least 1 admin user")
}

func TestAccountUsers_ListSearchByName(t *testing.T) {
	t.Parallel()
	// First get the seeded user's name to search for
	getStatus, getBody, err := apiClient.GetListRaw(accountUsersPath+"/"+SeedAccountUserID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	name := jsonField(parseJSON(getBody), "name")
	if name == "" {
		t.Skip("Seeded account user has no name set")
	}

	list, _, err := apiClient.GetList(accountUsersPath, url.Values{"q": {name}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Search by name should return at least 1 result")
}

func TestAccountUsers_ListSearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(accountUsersPath, url.Values{"q": {"zzzznotauser99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

// --- Get ---

func TestAccountUsers_GetByID(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountUsersPath+"/"+SeedAccountUserID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, SeedAccountUserID, jsonField(got, "id"))
	assert.Equal(t, "account_user", jsonField(got, "object"))
	assert.NotEmpty(t, jsonField(got, "status"))
	assert.NotEmpty(t, jsonField(got, "created_at"))
	assert.NotEmpty(t, jsonField(got, "updated_at"))
}

func TestAccountUsers_GetNotFound(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.GetListRaw(accountUsersPath+"/acus_000000000000", nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status)
}

// --- Includes ---

func TestAccountUsers_IncludeRole(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountUsersPath+"/"+SeedAccountUserID, url.Values{"include": {"role"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	role := jsonObject(parseJSON(body), "role")
	require.NotNil(t, role, "role should be present with ?include=role")
	assert.NotEmpty(t, jsonField(role, "id"))
	assert.Equal(t, "role", jsonField(role, "object"))
}

func TestAccountUsers_IncludeDepartment(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountUsersPath+"/"+SeedAccountUserID, url.Values{"include": {"department"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	// department may be null if the user has no department assigned, but the field should be present
	_, ok := got["department"]
	assert.True(t, ok, "department field should be present with ?include=department")
}

// --- CRUD ---

func TestAccountUsers_CreateAndGet(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-acuser")
	email := name + "@e2e-test.augno.com"

	createStatus, createBody, err := apiClient.Post(accountUsersPath, map[string]any{
		"name":    name,
		"email":   email,
		"role_id": SeedAdminRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	created := parseJSON(createBody)
	assert.Equal(t, "account_user", jsonField(created, "object"))
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assert.Equal(t, name, jsonField(created, "name"))

	// Get
	getStatus, getBody, err := apiClient.GetListRaw(accountUsersPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assert.Equal(t, id, jsonField(got, "id"))
	assert.Equal(t, name, jsonField(got, "name"))

	// Audit
	expectAuditEvent(t, id, "account_user", "create")

	// Cleanup
	apiClient.Delete(accountUsersPath + "/" + id)
}

func TestAccountUsers_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()

	// ── CREATE with all fields ──
	name := uniqueName("e2e-au-allf")
	email := name + "@e2e-test.augno.com"
	createStatus, createBody, err := apiClient.Post(accountUsersPath, map[string]any{
		"name":          name,
		"email":         email,
		"role_id":       SeedAdminRoleID,
		"department_id": SeedDepartmentID,
		"is_sales_rep":  true,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	got := parseJSON(createBody)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(accountUsersPath + "/" + id)

	assert.Equal(t, "account_user", jsonField(got, "object"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, email, jsonField(got, "email"))
	assert.NotEmpty(t, jsonField(got, "status"))
	assert.Equal(t, "false", jsonField(got, "is_verified"))
	assertNilField(t, got, "image_url")
	assertNilField(t, got, "last_used_at")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	// Verify via GET with include
	getStatus, getBody, err := apiClient.GetListRaw(accountUsersPath+"/"+id, url.Values{"include": {"role,department"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got = parseJSON(getBody)

	role := jsonObject(got, "role")
	require.NotNil(t, role, "role must be set after create")
	assert.Equal(t, SeedAdminRoleID, jsonField(role, "id"))
	assert.Equal(t, "role", jsonField(role, "object"))

	dept := jsonObject(got, "department")
	require.NotNil(t, dept, "department must be set after create")
	assert.Equal(t, SeedDepartmentID, jsonField(dept, "id"))
	assert.Equal(t, "department", jsonField(dept, "object"))

	// ── UPDATE with different values ──
	updatedName := uniqueName("e2e-au-allf-u")
	updatedEmail := updatedName + "@e2e-test.augno.com"
	patchStatus, patchBody, err := apiClient.Patch(accountUsersPath+"/"+id, map[string]any{
		"name":    updatedName,
		"email":   updatedEmail,
		"role_id": SeedSalesRepRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	assert.Equal(t, id, jsonField(updated, "id"), "ID must not change")
	assert.Equal(t, "account_user", jsonField(updated, "object"))
	assert.Equal(t, updatedName, jsonField(updated, "name"))
	assert.Equal(t, updatedEmail, jsonField(updated, "email"))
	assert.NotEmpty(t, jsonField(updated, "status"), "status should be preserved")
	assertValidTimestamp(t, jsonField(updated, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(updated, "updated_at"), "updated_at")

	// Verify via GET with include
	getStatus2, getBody2, err := apiClient.GetListRaw(accountUsersPath+"/"+id, url.Values{"include": {"role,department"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus2, getBody2)
	updated = parseJSON(getBody2)

	updRole := jsonObject(updated, "role")
	require.NotNil(t, updRole, "role should be updated")
	assert.Equal(t, SeedSalesRepRoleID, jsonField(updRole, "id"))

	// Department should be preserved
	updDept := jsonObject(updated, "department")
	require.NotNil(t, updDept, "department should be preserved")
	assert.Equal(t, SeedDepartmentID, jsonField(updDept, "id"))

	// Audit
	expectAuditEvent(t, id, "account_user", "create")
	expectAuditEvent(t, id, "account_user", "update")
}

func TestAccountUsers_Update(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-acuser-upd")
	email := name + "@e2e-test.augno.com"

	createStatus, createBody, err := apiClient.Post(accountUsersPath, map[string]any{
		"name":    name,
		"email":   email,
		"role_id": SeedAdminRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	newName := uniqueName("e2e-acuser-new")
	patchStatus, patchBody, err := apiClient.Patch(accountUsersPath+"/"+id, map[string]any{
		"name": newName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	assert.Equal(t, newName, jsonField(parseJSON(patchBody), "name"))

	// Audit
	expectAuditEvent(t, id, "account_user", "update")

	// Cleanup
	apiClient.Delete(accountUsersPath + "/" + id)
}

func TestAccountUsers_Delete(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-acuser-del")
	email := name + "@e2e-test.augno.com"

	createStatus, createBody, err := apiClient.Post(accountUsersPath, map[string]any{
		"name":    name,
		"email":   email,
		"role_id": SeedAdminRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	delStatus, delBody, err := apiClient.Delete(accountUsersPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// Verify shows as removed or 404
	getStatus, _, err := apiClient.GetListRaw(accountUsersPath+"/"+id, nil)
	require.NoError(t, err)
	assert.True(t, getStatus == 200 || getStatus == 404,
		"Deleted user should return 200 (with removed status) or 404, got %d", getStatus)

	// Audit
	expectAuditEvent(t, id, "account_user", "create")
	expectAuditEvent(t, id, "account_user", "delete")
}

func TestAccountUsers_LockAndUnlock(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-acuser-lock")
	email := name + "@e2e-test.augno.com"

	// Use a non-admin role since admin users cannot be locked.
	createStatus, createBody, err := apiClient.Post(accountUsersPath, map[string]any{
		"name":    name,
		"email":   email,
		"role_id": SeedSalesRepRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	// Lock
	lockStatus, lockBody, err := apiClient.Post(accountUsersPath+"/"+id+"/lock", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, lockStatus, lockBody)

	// Verify locked status
	getStatus, getBody, err := apiClient.GetListRaw(accountUsersPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, "disabled", jsonField(parseJSON(getBody), "status"))

	// Unlock
	unlockStatus, unlockBody, err := apiClient.Post(accountUsersPath+"/"+id+"/unlock", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, unlockStatus, unlockBody)

	// Verify unlocked
	getStatus2, getBody2, err := apiClient.GetListRaw(accountUsersPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus2, getBody2)
	assert.Equal(t, "active", jsonField(parseJSON(getBody2), "status"))

	// Audit — lock and unlock both emit "update" actions
	expectAuditEvent(t, id, "account_user", "create")
	expectAuditEvent(t, id, "account_user", "update")

	// Cleanup
	apiClient.Delete(accountUsersPath + "/" + id)
}

func TestAccountUsers_DeleteAndRestore(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-acuser-rest")
	email := name + "@e2e-test.augno.com"

	createStatus, createBody, err := apiClient.Post(accountUsersPath, map[string]any{
		"name":    name,
		"email":   email,
		"role_id": SeedAdminRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	// Delete
	delStatus, delBody, err := apiClient.Delete(accountUsersPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// Restore
	restoreStatus, restoreBody, err := apiClient.Post(accountUsersPath+"/"+id+"/restore", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, restoreStatus, restoreBody)

	// Verify restored
	getStatus, getBody, err := apiClient.GetListRaw(accountUsersPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	status := jsonField(parseJSON(getBody), "status")
	assert.True(t, status == "active" || status == "pending",
		"Restored user should have active or pending status, got %q", status)

	// Audit
	expectAuditEvent(t, id, "account_user", "create")
	expectAuditEvent(t, id, "account_user", "delete")
	expectAuditEvent(t, id, "account_user", "update") // restore emits update

	// Cleanup
	apiClient.Delete(accountUsersPath + "/" + id)
}

func TestAccountUsers_Idempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-idem-acuser")
	email := name + "@e2e-test.augno.com"
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Post(accountUsersPath, map[string]any{
		"name":    name,
		"email":   email,
		"role_id": SeedAdminRoleID,
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")

	status2, body2, err := apiClient.Post(accountUsersPath, map[string]any{
		"name":    name,
		"email":   email,
		"role_id": SeedAdminRoleID,
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id1, jsonField(parseJSON(body2), "id"))

	apiClient.Delete(accountUsersPath + "/" + id1)
}

func TestAccountUsers_ListIncludeRemoved(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-acuser-rem")
	email := name + "@e2e-test.augno.com"

	// Create and delete a user
	createStatus, createBody, err := apiClient.Post(accountUsersPath, map[string]any{
		"name":    name,
		"email":   email,
		"role_id": SeedAdminRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	apiClient.Delete(accountUsersPath + "/" + id)

	// List with include_removed should find the deleted user
	list, _, err := apiClient.GetList(accountUsersPath, url.Values{"include_removed": {"true"}})
	require.NoError(t, err)

	found := false
	for _, item := range list.Data {
		if DataItemField(item, "id") == id {
			found = true
			break
		}
	}
	assert.True(t, found, "Removed user should appear when include_removed=true")
}

// ──────────────────────────────────────────────
// Omitted Fields
// ──────────────────────────────────────────────

func TestAccountUsers_OmittedFields(t *testing.T) {
	t.Parallel()

	t.Run("CreateWithOnlyRequiredFields", func(t *testing.T) {
		name := uniqueName("e2e-au-omit")
		email := name + "@e2e-test.augno.com"

		status, body, err := apiClient.Post(accountUsersPath, map[string]any{
			"name":    name,
			"email":   email,
			"role_id": SeedAdminRoleID,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)

		got := parseJSON(body)
		id := jsonField(got, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(accountUsersPath + "/" + id)

		assert.Equal(t, "account_user", jsonField(got, "object"))
		assert.Equal(t, name, jsonField(got, "name"))
		assert.Equal(t, email, jsonField(got, "email"))
		assert.NotEmpty(t, jsonField(got, "status"))
		assert.Equal(t, "false", jsonField(got, "is_verified"))
		assertNilField(t, got, "image_url")
		assertNilField(t, got, "last_used_at")
		assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

		// Expandable fields should be nil without ?include
		assertNilField(t, got, "role")
		assertNilField(t, got, "department")
	})

	t.Run("UpdatePreservesOmittedFields", func(t *testing.T) {
		name := uniqueName("e2e-au-pres")
		email := name + "@e2e-test.augno.com"

		// Create with all fields
		createStatus, createBody, err := apiClient.Post(accountUsersPath, map[string]any{
			"name":          name,
			"email":         email,
			"role_id":       SeedAdminRoleID,
			"department_id": SeedDepartmentID,
			"is_sales_rep":  true,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, createStatus, createBody)

		created := parseJSON(createBody)
		id := jsonField(created, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(accountUsersPath + "/" + id)
		origCreatedAt := jsonField(created, "created_at")

		// Update ONLY name
		newName := uniqueName("e2e-au-pres-u")
		patchStatus, patchBody, err := apiClient.Patch(accountUsersPath+"/"+id, map[string]any{
			"name": newName,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, patchStatus, patchBody)

		// GET with include to verify preservation
		getStatus, getBody, err := apiClient.GetListRaw(accountUsersPath+"/"+id, url.Values{"include": {"role,department"}})
		require.NoError(t, err)
		requireStatus(t, 200, getStatus, getBody)

		got := parseJSON(getBody)

		// Updated field
		assert.Equal(t, newName, jsonField(got, "name"))

		// Preserved fields
		assert.Equal(t, id, jsonField(got, "id"))
		assert.Equal(t, "account_user", jsonField(got, "object"))
		assert.Equal(t, email, jsonField(got, "email"), "email should be preserved")
		assert.NotEmpty(t, jsonField(got, "status"), "status should be preserved")
		assert.Equal(t, origCreatedAt, jsonField(got, "created_at"), "created_at should not change")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

		// Preserved role
		role := jsonObject(got, "role")
		require.NotNil(t, role, "role should be preserved")
		assert.Equal(t, SeedAdminRoleID, jsonField(role, "id"))

		// Preserved department
		dept := jsonObject(got, "department")
		require.NotNil(t, dept, "department should be preserved")
		assert.Equal(t, SeedDepartmentID, jsonField(dept, "id"))
	})
}
