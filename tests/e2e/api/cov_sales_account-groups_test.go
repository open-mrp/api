//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// accountGroupsPath is declared in crud_account_groups_test.go (same package)
// and reused here.

// --- Response Shape ---

// TestCovSalesAccountGroups_CreateResponseShape asserts the acgp_ ID prefix,
// object field, and valid timestamps on a minimal create.
func TestCovSalesAccountGroups_CreateResponseShape(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-covacgp-shape")
	got := createAndCleanup(t, accountGroupsPath, map[string]any{
		"name": name,
		"type": "pricing_group",
	})

	id := jsonField(got, "id")
	assert.NotEmpty(t, id)
	assertIDFormat(t, id, "acgp")
	assertObjectField(t, got, "account_group")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
}

// --- List Pagination ---

// TestCovSalesAccountGroups_ListPagination creates two account groups sharing
// a unique name prefix and walks the list one row per page via the "q"
// search param, asserting the cursor advances and visits each row exactly
// once.
func TestCovSalesAccountGroups_ListPagination(t *testing.T) {
	t.Parallel()

	prefix := uniqueName("e2e-covacgp-pg")
	var ids []string
	for i := 0; i < 2; i++ {
		status, body, err := apiClient.Post(accountGroupsPath, map[string]any{
			"name": fmt.Sprintf("%s-%d", prefix, i),
			"type": "type_group",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)
		id := jsonField(parseJSON(body), "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(accountGroupsPath + "/" + id)
		ids = append(ids, id)
	}

	assertScopedCursorPagination(t, accountGroupsPath, url.Values{"q": {prefix}}, ids)
}

// --- Idempotency ---

// TestCovSalesAccountGroups_UpdateIdempotent PATCHes twice with the same
// idempotency key and body, asserting the second call is a no-op that
// returns the identical result rather than erroring or double-applying.
func TestCovSalesAccountGroups_UpdateIdempotent(t *testing.T) {
	t.Parallel()

	created := createAndCleanup(t, accountGroupsPath, map[string]any{
		"name": uniqueName("e2e-covacgp-idemupd"),
		"type": "type_group",
	})
	id := jsonField(created, "id")

	idemKey := newIdempotencyKey()
	newName := uniqueName("e2e-covacgp-idemupd-new")
	body := map[string]any{"name": newName}

	status1, body1, err := apiClient.Patch(accountGroupsPath+"/"+id, body, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)
	got1 := parseJSON(body1)
	assert.Equal(t, newName, jsonField(got1, "name"))

	status2, body2, err := apiClient.Patch(accountGroupsPath+"/"+id, body, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)
	got2 := parseJSON(body2)

	assert.Equal(t, jsonField(got1, "id"), jsonField(got2, "id"))
	assert.Equal(t, jsonField(got1, "name"), jsonField(got2, "name"))
	assert.Equal(t, jsonField(got1, "updated_at"), jsonField(got2, "updated_at"),
		"replayed PATCH with the same idempotency key should not re-apply the update")
}

// --- Create Validation: enums, length, duplicate name ---

// TestCovSalesAccountGroups_CreateValidation_InvalidTypeEnum asserts an
// invalid `type` enum value on create is rejected with 400, not silently
// accepted or a 5xx.
func TestCovSalesAccountGroups_CreateValidation_InvalidTypeEnum(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(accountGroupsPath, map[string]any{
		"name": uniqueName("e2e-covacgp-badtype"),
		"type": "bogus_type",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "type")
}

// TestCovSalesAccountGroups_CreateValidation_InvalidCommissionPolicyEnum
// asserts an invalid `commission_policy` enum value on create is rejected
// with 400.
func TestCovSalesAccountGroups_CreateValidation_InvalidCommissionPolicyEnum(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(accountGroupsPath, map[string]any{
		"name":              uniqueName("e2e-covacgp-badcp"),
		"type":              "type_group",
		"commission_policy": "not_a_policy",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "commission_policy")
}

// TestCovSalesAccountGroups_CreateValidation_InvalidFreightPolicyEnum
// asserts an invalid `freight_policy` enum value on create is rejected with
// 400.
func TestCovSalesAccountGroups_CreateValidation_InvalidFreightPolicyEnum(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(accountGroupsPath, map[string]any{
		"name":           uniqueName("e2e-covacgp-badfp"),
		"type":           "type_group",
		"freight_policy": "not_a_policy",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "freight_policy")
}

// TestCovSalesAccountGroups_CreateValidation_NameTooLong asserts a 256-char
// name is rejected with 400 on create.
func TestCovSalesAccountGroups_CreateValidation_NameTooLong(t *testing.T) {
	t.Parallel()
	longName := strings.Repeat("a", 256)
	status, body, err := apiClient.Post(accountGroupsPath, map[string]any{
		"name": longName,
		"type": "type_group",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "name")
}

// TestCovSalesAccountGroups_CreateValidation_DuplicateName asserts creating
// a second account group with the same name as an existing one returns 409
// resource_conflict with param=name, per the documented behavior on
// CreateAccountGroupRequest.
func TestCovSalesAccountGroups_CreateValidation_DuplicateName(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-covacgp-dupe")
	createAndCleanup(t, accountGroupsPath, map[string]any{
		"name": name,
		"type": "type_group",
	})

	status, body, err := apiClient.Post(accountGroupsPath, map[string]any{
		"name": name,
		"type": "pricing_group",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
	errObj := requireErrorResponse(t, body, "resource_conflict", "invalid_request_error")
	assertErrorParam(t, errObj, "name")
}

// --- Update Validation: enums, length, description clear, type no-op ---

// TestCovSalesAccountGroups_UpdateValidation_InvalidCommissionPolicyEnum
// asserts an invalid `commission_policy` enum value on update is rejected
// with 400.
func TestCovSalesAccountGroups_UpdateValidation_InvalidCommissionPolicyEnum(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, accountGroupsPath, map[string]any{
		"name": uniqueName("e2e-covacgp-updbadcp"),
		"type": "type_group",
	})
	id := jsonField(created, "id")

	status, body, err := apiClient.Patch(accountGroupsPath+"/"+id, map[string]any{
		"commission_policy": "not_a_policy",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "commission_policy")
}

// TestCovSalesAccountGroups_UpdateValidation_InvalidFreightPolicyEnum
// asserts an invalid `freight_policy` enum value on update is rejected with
// 400.
func TestCovSalesAccountGroups_UpdateValidation_InvalidFreightPolicyEnum(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, accountGroupsPath, map[string]any{
		"name": uniqueName("e2e-covacgp-updbadfp"),
		"type": "type_group",
	})
	id := jsonField(created, "id")

	status, body, err := apiClient.Patch(accountGroupsPath+"/"+id, map[string]any{
		"freight_policy": "not_a_policy",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "freight_policy")
}

// TestCovSalesAccountGroups_UpdateValidation_NameTooLong asserts a 256-char
// name is rejected with 400 on update.
func TestCovSalesAccountGroups_UpdateValidation_NameTooLong(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, accountGroupsPath, map[string]any{
		"name": uniqueName("e2e-covacgp-updlong"),
		"type": "type_group",
	})
	id := jsonField(created, "id")

	longName := strings.Repeat("b", 256)
	status, body, err := apiClient.Patch(accountGroupsPath+"/"+id, map[string]any{
		"name": longName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "name")
}

// TestCovSalesAccountGroups_UpdateDescriptionNullClears asserts PATCHing
// description: null clears a previously-set description on a live server
// round trip (the endpoint-package unit test only covers JSON marshaling,
// not the actual persisted/returned value).
func TestCovSalesAccountGroups_UpdateDescriptionNullClears(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, accountGroupsPath, map[string]any{
		"name":        uniqueName("e2e-covacgp-clear"),
		"type":        "type_group",
		"description": "will be cleared",
	})
	id := jsonField(created, "id")
	require.Equal(t, "will be cleared", jsonField(created, "description"))

	status, body, err := apiClient.Patch(accountGroupsPath+"/"+id, map[string]any{
		"description": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assertNilField(t, got, "description")

	// Confirm the clear persisted via a follow-up GET, not just the PATCH response.
	getStatus, getBody, err := apiClient.GetListRaw(accountGroupsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assertNilField(t, parseJSON(getBody), "description")
}

// TestCovSalesAccountGroups_UpdateTypeFieldRejected asserts that sending a
// `type` field in a PATCH body is rejected with 400 parameter_unknown
// (UpdateAccountGroupRequest has no Type field, and the JSON decoder used by
// this endpoint rejects unknown fields rather than silently dropping them),
// and that the account group's type is left unchanged.
func TestCovSalesAccountGroups_UpdateTypeFieldRejected(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, accountGroupsPath, map[string]any{
		"name": uniqueName("e2e-covacgp-typenoop"),
		"type": "pricing_group",
	})
	id := jsonField(created, "id")

	status, body, err := apiClient.Patch(accountGroupsPath+"/"+id, map[string]any{
		"type": "type_group",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_unknown", "invalid_request_error")
	assertErrorParam(t, errObj, "type")

	getStatus, getBody, err := apiClient.GetListRaw(accountGroupsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, "pricing_group", jsonField(parseJSON(getBody), "type"),
		"type should remain unchanged after a rejected PATCH")
}

// --- Not Found (404) ---

const covSalesAccountGroupsFakeID = "acgp_00000000000000000000"

func TestCovSalesAccountGroups_GetNotFound(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountGroupsPath+"/"+covSalesAccountGroupsFakeID, nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

func TestCovSalesAccountGroups_UpdateNotFound(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Patch(accountGroupsPath+"/"+covSalesAccountGroupsFakeID, map[string]any{
		"name": uniqueName("e2e-covacgp-notfound"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

func TestCovSalesAccountGroups_DeleteNotFound(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Delete(accountGroupsPath + "/" + covSalesAccountGroupsFakeID)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// --- Delete Blocked While In Use ---

// TestCovSalesAccountGroups_DeleteConflictWhenCustomersExist asserts that
// deleting an account group still referenced by customer records fails with
// 400 validation_failed (not a 5xx), and that the group is not deleted.
// Uses the stable seeded SeedCustomerGroupID, which multiple seeded customer
// rows reference as their customer_type_group_id FK — it must never actually
// be deleted by this or any other test.
func TestCovSalesAccountGroups_DeleteConflictWhenCustomersExist(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Delete(accountGroupsPath + "/" + SeedCustomerGroupID)
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "validation_failed", "invalid_request_error")

	// Confirm the group still exists (was not deleted despite the error).
	getStatus, getBody, err := apiClient.GetListRaw(accountGroupsPath+"/"+SeedCustomerGroupID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, SeedCustomerGroupName, jsonField(parseJSON(getBody), "name"))
}

// --- Tenant Isolation ---

// TestCovSalesAccountGroups_TenantIsolation_Update verifies tenant B cannot
// PATCH an account group belonging to tenant A.
func TestCovSalesAccountGroups_TenantIsolation_Update(t *testing.T) {
	t.Parallel()
	clientB := getTenantBClient()

	created := createAndCleanup(t, accountGroupsPath, map[string]any{
		"name": uniqueName("e2e-covacgp-iso-patch"),
		"type": "type_group",
	})
	id := jsonField(created, "id")

	status, body, err := clientB.Patch(accountGroupsPath+"/"+id, map[string]any{
		"name": "cross-tenant update attempt",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status,
		"Tenant B should get 404 for PATCH on tenant A's account group: %s", string(body))
}

// TestCovSalesAccountGroups_TenantIsolation_Delete verifies tenant B cannot
// DELETE an account group belonging to tenant A, and that the group still
// exists in tenant A afterward.
func TestCovSalesAccountGroups_TenantIsolation_Delete(t *testing.T) {
	t.Parallel()
	clientB := getTenantBClient()

	created := createAndCleanup(t, accountGroupsPath, map[string]any{
		"name": uniqueName("e2e-covacgp-iso-delete"),
		"type": "type_group",
	})
	id := jsonField(created, "id")

	status, body, err := clientB.Delete(accountGroupsPath + "/" + id)
	require.NoError(t, err)
	assert.Equal(t, 404, status,
		"Tenant B should get 404 for DELETE on tenant A's account group: %s", string(body))

	getStatus, _, err := apiClient.GetListRaw(accountGroupsPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 200, getStatus, "account group should still exist in tenant A after tenant B's delete attempt")
}

// TestCovSalesAccountGroups_TenantIsolation_ListDoesNotLeak verifies tenant
// B's list results do not contain tenant A's account groups.
func TestCovSalesAccountGroups_TenantIsolation_ListDoesNotLeak(t *testing.T) {
	t.Parallel()
	clientB := getTenantBClient()

	distinctName := uniqueName("e2e-covacgp-iso-leak")
	createAndCleanup(t, accountGroupsPath, map[string]any{
		"name": distinctName,
		"type": "type_group",
	})

	list, _, err := clientB.GetList(accountGroupsPath, url.Values{"q": {distinctName}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data,
		"Tenant B's list should not contain tenant A's account group with name "+distinctName)
}

// --- Query Param Boundaries ---

// TestCovSalesAccountGroups_ListLimitBoundaries asserts out-of-range `limit`
// values are rejected with 400 (PaginationRequest validate:"min=1,max=1000").
func TestCovSalesAccountGroups_ListLimitBoundaries(t *testing.T) {
	t.Parallel()

	t.Run("Zero", func(t *testing.T) {
		t.Parallel()
		status, body, err := apiClient.GetListRaw(accountGroupsPath, url.Values{"limit": {"0"}})
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
	})

	t.Run("Negative", func(t *testing.T) {
		t.Parallel()
		status, body, err := apiClient.GetListRaw(accountGroupsPath, url.Values{"limit": {"-1"}})
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
	})

	t.Run("TooLarge", func(t *testing.T) {
		t.Parallel()
		status, body, err := apiClient.GetListRaw(accountGroupsPath, url.Values{"limit": {"1001"}})
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
	})
}

// TestCovSalesAccountGroups_ListMalformedCursor asserts a malformed cursor
// value returns 400 rather than a 5xx.
func TestCovSalesAccountGroups_ListMalformedCursor(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountGroupsPath, url.Values{"cursor": {"not-a-valid-cursor"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

// TestCovSalesAccountGroups_ListInvalidTypeFilter asserts an invalid `type`
// query filter value is rejected with 400. Note the error.param is the Go
// struct field name ("Type"), not the query tag ("type") — this matches the
// established convention for query-param enum validation elsewhere in the
// codebase (see TestCovCatalogUnits_ListInvalidTypeFilter).
func TestCovSalesAccountGroups_ListInvalidTypeFilter(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountGroupsPath, url.Values{"type": {"bogus"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "type")
}
