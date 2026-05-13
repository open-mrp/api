//go:build e2e

package api_test

import (
	"net/url"
	"strings"
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
	adminList, _, err := apiClient.GetList(accountUsersPath, url.Values{"role_type": {"admin"}})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(adminList.Data), 1, "Should have at least 1 admin user")

	adminIDs := make(map[string]bool, len(adminList.Data))
	for _, item := range adminList.Data {
		adminIDs[DataItemField(item, "id")] = true
	}

	// Users returned for a different role_type must not overlap with the admin set.
	scannerList, _, err := apiClient.GetList(accountUsersPath, url.Values{"role_type": {"scanner"}})
	require.NoError(t, err)
	for _, item := range scannerList.Data {
		id := DataItemField(item, "id")
		assert.False(t, adminIDs[id],
			"User %q appears in both admin and scanner role-type filter results", id)
	}
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

	lowerName := strings.ToLower(name)
	for _, item := range list.Data {
		n := DataItemField(item, "name")
		assert.True(t,
			strings.Contains(strings.ToLower(n), lowerName),
			"Search result %q should contain %q", n, name,
		)
	}
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

	createResp, err := apiClient.PostFull(accountUsersPath, map[string]any{
		"name":    name,
		"email":   email,
		"role_id": SeedAdminRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	assert.Equal(t, "account_user", jsonField(created, "object"))
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
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
	removeAccountUser(id)
}

func TestAccountUsers_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()

	// ── CREATE with all fields ──
	name := uniqueName("e2e-au-allf")
	email := name + "@e2e-test.augno.com"
	createResp, err := apiClient.PostFull(accountUsersPath, map[string]any{
		"name":          name,
		"email":         email,
		"role_id":       SeedAdminRoleID,
		"department_id": SeedDepartmentID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	got := parseJSON(createResp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	defer removeAccountUser(id)

	assert.Equal(t, "account_user", jsonField(got, "object"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, email, jsonField(got, "email"))
	assert.NotEmpty(t, jsonField(got, "status"))
	_, hasIsVerified := got["is_verified"]
	assert.False(t, hasIsVerified, "is_verified should not be exposed on the resource")
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
	removeAccountUser(id)
}

func TestAccountUsers_Remove(t *testing.T) {
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

	delStatus, delBody, err := apiClient.Put(accountUsersPath+"/"+id+"/actions/remove", nil)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// Verify shows as removed or 404
	getStatus, _, err := apiClient.GetListRaw(accountUsersPath+"/"+id, nil)
	require.NoError(t, err)
	assert.True(t, getStatus == 200 || getStatus == 404,
		"Removed user should return 200 (with removed status) or 404, got %d", getStatus)

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

	// Lock via disable action.
	lockStatus, lockBody, err := apiClient.Put(accountUsersPath+"/"+id+"/actions/disable", nil)
	require.NoError(t, err)
	requireStatus(t, 200, lockStatus, lockBody)

	// Verify locked status
	getStatus, getBody, err := apiClient.GetListRaw(accountUsersPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, "disabled", jsonField(parseJSON(getBody), "status"))

	// Re-applying disable is an idempotent no-op.
	idemStatus, idemBody, err := apiClient.Put(accountUsersPath+"/"+id+"/actions/disable", nil)
	require.NoError(t, err)
	requireStatus(t, 200, idemStatus, idemBody)

	// Unlock via activate action.
	unlockStatus, unlockBody, err := apiClient.Put(accountUsersPath+"/"+id+"/actions/activate", nil)
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
	removeAccountUser(id)
}

func TestAccountUsers_RemoveAndRestore(t *testing.T) {
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

	// Remove via remove action.
	delStatus, delBody, err := apiClient.Put(accountUsersPath+"/"+id+"/actions/remove", nil)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// Restore via activate action.
	restoreStatus, restoreBody, err := apiClient.Put(accountUsersPath+"/"+id+"/actions/activate", nil)
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
	removeAccountUser(id)
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

	removeAccountUser(id1)
}

func TestAccountUsers_ListIncludeRemoved(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-acuser-rem")
	email := name + "@e2e-test.augno.com"

	// Create and remove a user
	createStatus, createBody, err := apiClient.Post(accountUsersPath, map[string]any{
		"name":    name,
		"email":   email,
		"role_id": SeedAdminRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	removeAccountUser(id)

	// List with removed_scope=included should find the removed user
	list, _, err := apiClient.GetList(accountUsersPath, url.Values{"removed_scope": {"included"}})
	require.NoError(t, err)

	found := false
	for _, item := range list.Data {
		if DataItemField(item, "id") == id {
			found = true
			break
		}
	}
	assert.True(t, found, "Removed user should appear when removed_scope=included")
}

// removeAccountUser is a test helper that soft-deletes a user via the remove action.
func removeAccountUser(id string) {
	_, _, _ = apiClient.Put(accountUsersPath+"/"+id+"/actions/remove", nil)
}

// --- Scanner creation ---

func TestAccountUsers_CreateScannerWithPassword(t *testing.T) {
	t.Parallel()
	username := uniqueName("e2e-scn-user")

	status, body, err := apiClient.Post(accountUsersPath, map[string]any{
		"username": username,
		"password": "ScannerPass123!",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	got := parseJSON(body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	defer removeAccountUser(id)

	assert.Equal(t, username, jsonField(got, "username"))
	assertNilField(t, got, "email")
}

func TestAccountUsers_CreateScannerMissingPasswordFails(t *testing.T) {
	t.Parallel()
	username := uniqueName("e2e-scn-nopw")

	status, _, err := apiClient.Post(accountUsersPath, map[string]any{
		"username": username,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "scanner users (username-only) must provide a password")
}

func TestAccountUsers_CreateNonScannerWithPasswordFails(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-acu-nopw")
	email := name + "@e2e-test.augno.com"

	status, _, err := apiClient.Post(accountUsersPath, map[string]any{
		"name":     name,
		"email":    email,
		"role_id":  SeedAdminRoleID,
		"password": "SomePassword123!",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "non-scanner users cannot set a password directly")
}

// --- Status transitions ---

func TestAccountUsers_StatusNoopIsIdempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-acu-noop")
	email := name + "@e2e-test.augno.com"

	createStatus, createBody, err := apiClient.Post(accountUsersPath, map[string]any{
		"name":    name,
		"email":   email,
		"role_id": SeedSalesRepRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	defer removeAccountUser(id)

	// User is already active; activate must be a no-op.
	status, body, err := apiClient.Put(accountUsersPath+"/"+id+"/actions/activate", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	// Status must still be active.
	_, getBody, err := apiClient.GetListRaw(accountUsersPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, "active", jsonField(parseJSON(getBody), "status"))
}

func TestAccountUsers_StatusLockAdminFails(t *testing.T) {
	t.Parallel()
	// The seeded admin account user cannot be disabled.
	status, _, err := apiClient.Put(accountUsersPath+"/"+SeedAccountUserID+"/actions/disable", nil)
	require.NoError(t, err)
	assert.Equal(t, 400, status, "admin users cannot be locked")
}

func TestAccountUsers_StatusLockAfterRemoveFails(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-acu-lock-rm")
	email := name + "@e2e-test.augno.com"

	createStatus, createBody, err := apiClient.Post(accountUsersPath, map[string]any{
		"name":    name,
		"email":   email,
		"role_id": SeedSalesRepRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	// Remove first.
	rmStatus, rmBody, err := apiClient.Put(accountUsersPath+"/"+id+"/actions/remove", nil)
	require.NoError(t, err)
	requireStatus(t, 200, rmStatus, rmBody)

	// Attempting to lock a removed user must fail.
	lockStatus, _, err := apiClient.Put(accountUsersPath+"/"+id+"/actions/disable", nil)
	require.NoError(t, err)
	assert.Equal(t, 400, lockStatus, "removed users cannot be locked")
}

// --- Scanner password endpoint ---
//
// The scanner-password endpoint (POST /v1/auth/scanner-passwords) lives under
// the auth group with session-based middleware, and requires a requester
// password. The e2e harness authenticates with an API key (no password), so
// the endpoint cannot be fully exercised here. A schema-level happy-path test
// is sufficient at the e2e layer; the behavioral guards (scanner-role only,
// requester-password verification) are covered by unit tests in the core
// service and the auth middleware tests.

func TestAuth_ScannerPasswordsEndpointRegistered(t *testing.T) {
	t.Parallel()
	// Sending the request without a session should be rejected at the auth
	// layer with 401/403, proving the route is wired up without requiring a
	// full session-auth setup in e2e.
	status, _, err := apiClient.Post("/v1/auth/scanner-passwords", map[string]any{
		"account_user_id":    SeedAccountUserID,
		"requester_password": "irrelevant-Password1!",
		"new_password":       "irrelevant-NewPassword1!",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Contains(t, []int{400, 401, 403}, status,
		"scanner-passwords route should be reachable; got %d", status)
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
		defer removeAccountUser(id)

		assert.Equal(t, "account_user", jsonField(got, "object"))
		assert.Equal(t, name, jsonField(got, "name"))
		assert.Equal(t, email, jsonField(got, "email"))
		assert.NotEmpty(t, jsonField(got, "status"))
		_, hasIsVerified := got["is_verified"]
		assert.False(t, hasIsVerified, "is_verified should not be exposed on the resource")
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
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, createStatus, createBody)

		created := parseJSON(createBody)
		id := jsonField(created, "id")
		require.NotEmpty(t, id)
		defer removeAccountUser(id)
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
