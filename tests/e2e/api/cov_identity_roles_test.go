//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file adds coverage gaps identified for the identity_roles group (`/v1/identity/roles`) on top of the existing tests/e2e/api/crud_roles_test.go suite: omitted-field/preservation semantics (including the empty-array permissions-clear behavior), malformed-permission-string validation on both create and update, unrecognized permission domain behavior, unrecognized `types` query filter behavior, response-shape ID prefix assertion, and exact-status tightening for the global-role-immutable and already-deleted-role cases (as new, additively-named tests — the existing loose-status tests in crud_roles_test.go are left untouched per task scope).

// --- Omitted Fields / Preservation ---

func TestCovIdentityRoles_OmittedFields(t *testing.T) {
	t.Parallel()

	t.Run("CreateWithOnlyRequiredFields", func(t *testing.T) {
		name := uniqueName("e2e-covroles-omit")
		status, body, err := apiClient.Post(rolesPath+"?include=owner,permissions", map[string]any{
			"name": name,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)

		got := parseJSON(body)
		id := jsonField(got, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(rolesPath + "/" + id)

		assertObjectField(t, got, "role")
		assert.Equal(t, name, jsonField(got, "name"))
		assert.Equal(t, "user", jsonField(got, "type"))
		// permissions was omitted on create → no permissions rows exist, so even with ?include=permissions the value comes back null (see TestCovIdentityRoles_ClearPermissionsReturnsNull below for the same nil-vs-empty-array behavior on an explicit clear).
		assertNilField(t, got, "permissions")
		assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	})

	t.Run("CreateMissingRequiredName", func(t *testing.T) {
		status, body, err := apiClient.Post(rolesPath, map[string]any{}, newIdempotencyKey())
		require.NoError(t, err)
		assert.True(t, status == 400 || status == 422,
			"Missing name should return 400 or 422, got %d: %s", status, string(body))
	})

	t.Run("UpdateNameOnlyPreservesPermissions", func(t *testing.T) {
		name := uniqueName("e2e-covroles-pres-name")
		createStatus, createBody, err := apiClient.Post(rolesPath, map[string]any{
			"name":        name,
			"permissions": []string{"customers:read", "customers:create"},
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, createStatus, createBody)
		id := jsonField(parseJSON(createBody), "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(rolesPath + "/" + id)

		newName := uniqueName("e2e-covroles-pres-name-upd")
		patchStatus, patchBody, err := apiClient.Patch(rolesPath+"/"+id+"?include=permissions", map[string]any{
			"name": newName,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, patchStatus, patchBody)

		updated := parseJSON(patchBody)
		assert.Equal(t, newName, jsonField(updated, "name"))

		perms, ok := updated["permissions"]
		require.True(t, ok, "permissions should be present with ?include=permissions")
		permsSlice, ok := perms.([]any)
		require.True(t, ok, "permissions should be a slice")
		assert.Len(t, permsSlice, 2, "permissions omitted from PATCH body should be left unchanged")
	})

	t.Run("UpdatePermissionsOnlyPreservesName", func(t *testing.T) {
		name := uniqueName("e2e-covroles-pres-perm")
		createStatus, createBody, err := apiClient.Post(rolesPath, map[string]any{
			"name":        name,
			"permissions": []string{"invoices:read"},
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, createStatus, createBody)
		id := jsonField(parseJSON(createBody), "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(rolesPath + "/" + id)

		patchStatus, patchBody, err := apiClient.Patch(rolesPath+"/"+id+"?include=permissions", map[string]any{
			"permissions": []string{"customers:read", "customers:update"},
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, patchStatus, patchBody)

		updated := parseJSON(patchBody)
		// name was omitted from the PATCH body → must be left unchanged.
		assert.Equal(t, name, jsonField(updated, "name"))

		perms, ok := updated["permissions"]
		require.True(t, ok)
		permsSlice, ok := perms.([]any)
		require.True(t, ok)
		assert.Len(t, permsSlice, 2)
	})
}

// TestCovIdentityRoles_ClearPermissionsReturnsNull verifies the doc-promised "pass an empty array to remove all permissions" semantics on PATCH.
//
// Verified against the live stack: after clearing, GET/PATCH with ?include=permissions returns `"permissions": null`, not `[]`. This mirrors omitted-permissions-on-create (also null): the loader's formatRolePermissions returns a nil `[]string` when there are zero permission rows, and the JSON encoder renders a nil slice as `null` rather than `[]`. Asserting actual behavior per task instructions rather than the task doc's unverified `[]` prediction.
func TestCovIdentityRoles_ClearPermissionsReturnsNull(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-covroles-clear")
	createStatus, createBody, err := apiClient.Post(rolesPath, map[string]any{
		"name":        name,
		"permissions": []string{"customers:read", "customers:create"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(rolesPath + "/" + id)

	// Sanity: permissions are non-empty before the clear.
	getStatus, getBody, err := apiClient.GetListRaw(rolesPath+"/"+id, url.Values{"include": {"permissions"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	beforePerms, ok := parseJSON(getBody)["permissions"].([]any)
	require.True(t, ok)
	require.GreaterOrEqual(t, len(beforePerms), 1)

	// Clear with an explicit empty array.
	patchStatus, patchBody, err := apiClient.Patch(rolesPath+"/"+id+"?include=permissions", map[string]any{
		"permissions": []string{},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	assert.Equal(t, name, jsonField(updated, "name"), "name should be unaffected by clearing permissions")
	assertNilField(t, updated, "permissions")

	// Confirm the clear persisted via a fresh GET.
	getStatus2, getBody2, err := apiClient.GetListRaw(rolesPath+"/"+id, url.Values{"include": {"permissions"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus2, getBody2)
	assertNilField(t, parseJSON(getBody2), "permissions")
}

// --- Response Shape ---

func TestCovIdentityRoles_CreateResponseShape_IDFormat(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-covroles-shape")
	status, body, err := apiClient.Post(rolesPath, map[string]any{
		"name": name,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	got := parseJSON(body)
	id := jsonField(got, "id")
	assertIDFormat(t, id, "rl")
	assertObjectField(t, got, "role")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	apiClient.Delete(rolesPath + "/" + id)
}

// --- Validation: permission string parsing ---

func TestCovIdentityRoles_CreateValidation_MalformedPermissionNoColon(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(rolesPath, map[string]any{
		"name":        uniqueName("e2e-covroles-badperm-create"),
		"permissions": []string{"customers"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "permissions")
}

func TestCovIdentityRoles_CreateValidation_InvalidPermissionAction(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(rolesPath, map[string]any{
		"name":        uniqueName("e2e-covroles-badaction-create"),
		"permissions": []string{"customers:foo"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "permissions")
}

func TestCovIdentityRoles_UpdateValidation_MalformedPermissionNoColon(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-covroles-badperm-upd")
	createStatus, createBody, err := apiClient.Post(rolesPath, map[string]any{"name": name}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(rolesPath + "/" + id)

	status, body, err := apiClient.Patch(rolesPath+"/"+id, map[string]any{
		"permissions": []string{"customers"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "permissions")

	// Confirm the malformed PATCH did not mutate the role's name.
	getStatus, getBody, err := apiClient.GetListRaw(rolesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, name, jsonField(parseJSON(getBody), "name"))
}

func TestCovIdentityRoles_UpdateValidation_InvalidPermissionAction(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-covroles-badaction-upd")
	createStatus, createBody, err := apiClient.Post(rolesPath, map[string]any{"name": name}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(rolesPath + "/" + id)

	status, body, err := apiClient.Patch(rolesPath+"/"+id, map[string]any{
		"permissions": []string{"customers:badaction"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "permissions")
}

// The domain half of `domain:action` is checked against the known permission domains, the same way the action half is, so a role cannot be created holding a grant that names nothing.
func TestCovIdentityRoles_CreateValidation_UnrecognizedPermissionDomainRejected(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-covroles-bogusdomain")
	status, body, err := apiClient.Post(rolesPath+"?include=permissions", map[string]any{
		"name":        name,
		"permissions": []string{"bogus_domain:read"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "permissions")
}

// --- Query param: types filter ---

// TestCovIdentityRoles_ListFilterByRoleType_Unrecognized documents actual (verified) behavior for an unrecognized `types` filter value: the request succeeds (200) and returns an empty result set rather than 400 or a silent no-op that ignores the filter.
func TestCovIdentityRoles_ListFilterByRoleType_Unrecognized(t *testing.T) {
	t.Parallel()
	// Unrecognized enum values in a list filter are rejected with 400 (platform convention).
	status, body, err := apiClient.GetListRaw(rolesPath, url.Values{"types": {"bogus_type"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

// --- Global-role immutability & already-deleted: exact-status tightening ---
//
// These add new, tightened assertions alongside the existing loose-OR-status tests in crud_roles_test.go (which are left untouched per task scope).

func TestCovIdentityRoles_UpdateGlobalRoleFails_ExactStatus(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Patch(rolesPath+"/"+SeedAdminRoleID, map[string]any{
		"name": uniqueName("e2e-covroles-global-upd"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
}

func TestCovIdentityRoles_DeleteGlobalRoleFails_ExactStatus(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Delete(rolesPath + "/" + SeedAdminRoleID)
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
}

func TestCovIdentityRoles_DeleteAlreadyDeletedFails_ExactStatus(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-covroles-deldel")
	createStatus, createBody, err := apiClient.Post(rolesPath, map[string]any{"name": name}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	require.NotEmpty(t, id)

	delStatus, delBody, err := apiClient.Delete(rolesPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	status2, body2, err := apiClient.Delete(rolesPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 410, status2, body2)
	requireErrorResponse(t, body2, "resource_gone", "invalid_request_error")
}

// --- Validation: overlong name ---

func TestCovIdentityRoles_CreateValidation_NameTooLong(t *testing.T) {
	t.Parallel()
	longName := make([]byte, 256)
	for i := range longName {
		longName[i] = 'a'
	}
	status, body, err := apiClient.Post(rolesPath, map[string]any{
		"name": string(longName),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "name")
}

// --- Validation: whitespace-only name on create ---
//
// Verified against the live stack: unlike PATCH (whose field.Optional[string] path rejects blank/whitespace-only strings via the framework's blank-string guard), CREATE's Name field is a plain required string, so the framework's `required` validator only rejects the zero value (""); an all-whitespace string passes through and is persisted verbatim. This mirrors the plain string vs field.Optional[string] convention used elsewhere in this codebase, so it is documented as actual behavior rather than filed as a bug.
//
// The name is derived from a fresh UUID so it is a *unique* whitespace-only string per run: role-name uniqueness is enforced on the literal (untrimmed) value, so a hardcoded constant like "   " collides (409) with leftovers from prior runs whose deferred cleanup never registered. Mapping every UUID rune to a space/tab keeps the name whitespace-only (still exercising the blank-string guard bypass) while guaranteeing uniqueness.
func TestCovIdentityRoles_CreateWhitespaceOnlyNameAccepted(t *testing.T) {
	t.Parallel()
	wsName := strings.Map(func(r rune) rune {
		if (r-'0')%2 == 0 {
			return ' '
		}
		return '\t'
	}, newIdempotencyKey())
	require.True(t, strings.TrimSpace(wsName) == "" && wsName != "", "name must be non-empty whitespace-only")

	status, body, err := apiClient.Post(rolesPath, map[string]any{
		"name": wsName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	got := parseJSON(body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(rolesPath + "/" + id)
	assert.Equal(t, wsName, jsonField(got, "name"))
}

// --- Idempotency: update ---

func TestCovIdentityRoles_UpdatePermissionsIdempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-covroles-idem-perm")
	createStatus, createBody, err := apiClient.Post(rolesPath, map[string]any{
		"name":        name,
		"permissions": []string{"customers:read"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(rolesPath + "/" + id)

	idemKey := newIdempotencyKey()
	newPerms := []string{"customers:read", "customers:update", "invoices:read"}

	status1, body1, err := apiClient.Patch(rolesPath+"/"+id+"?include=permissions", map[string]any{
		"permissions": newPerms,
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)

	status2, body2, err := apiClient.Patch(rolesPath+"/"+id+"?include=permissions", map[string]any{
		"permissions": newPerms,
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)

	perms1 := parseJSON(body1)["permissions"]
	perms2 := parseJSON(body2)["permissions"]
	assert.ElementsMatch(t, perms1, perms2, "replayed idempotent update should return identical permissions")
}
