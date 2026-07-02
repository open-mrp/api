//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file extends the existing catalog/product-lines coverage in
// crud_product_lines_test.go with the remaining failure-mode gaps identified
// during the e2e coverage audit: invalid enum values on create/update, a
// name that exceeds the 255-char max, duplicate-name conflicts (409) on
// create+update, nonexistent unit_group_id FK references on create+update,
// and customer-portal write-denial for update/delete (only create-denial was
// previously covered).
//
// It reuses productLinesPath (declared in crud_product_lines_test.go),
// SeedProductLineID / SeedProductLineName / SeedUnitGroupID / SeedAccountID
// (declared in seed_test.go), and getCustomerPortalClient() (declared in
// customer_portal_access_test.go). No new seed rows or package-level
// constants are needed.

// --- Enum validation ---

func TestCovCatalogProductLines_CreateInvalidCommissionPolicy(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(productLinesPath, map[string]any{
		"name":              uniqueName("e2e-cov-pdln-badcp"),
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "bogus_policy",
		"freight_policy":    "billed_freight",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "invalid commission_policy should return 400: %s", string(body))
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "commission_policy")
}

func TestCovCatalogProductLines_CreateInvalidFreightPolicy(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(productLinesPath, map[string]any{
		"name":              uniqueName("e2e-cov-pdln-badfp"),
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "commission_applied",
		"freight_policy":    "bogus_policy",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "invalid freight_policy should return 400: %s", string(body))
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "freight_policy")
}

func TestCovCatalogProductLines_UpdateInvalidCommissionPolicy(t *testing.T) {
	t.Parallel()
	id := covCatalogProductLinesCreate(t, uniqueName("e2e-cov-pdln-upd-badcp"))
	defer apiClient.Delete(productLinesPath + "/" + id)

	status, body, err := apiClient.Patch(productLinesPath+"/"+id, map[string]any{
		"commission_policy": "bogus_policy",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "invalid commission_policy on update should return 400: %s", string(body))
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "commission_policy")
}

func TestCovCatalogProductLines_UpdateInvalidFreightPolicy(t *testing.T) {
	t.Parallel()
	id := covCatalogProductLinesCreate(t, uniqueName("e2e-cov-pdln-upd-badfp"))
	defer apiClient.Delete(productLinesPath + "/" + id)

	status, body, err := apiClient.Patch(productLinesPath+"/"+id, map[string]any{
		"freight_policy": "bogus_policy",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "invalid freight_policy on update should return 400: %s", string(body))
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "freight_policy")
}

// --- Name length validation ---

func TestCovCatalogProductLines_CreateNameTooLong(t *testing.T) {
	t.Parallel()
	longName := make([]byte, 256)
	for i := range longName {
		longName[i] = 'a'
	}
	status, body, err := apiClient.Post(productLinesPath, map[string]any{
		"name":              string(longName),
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "commission_applied",
		"freight_policy":    "billed_freight",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "name > 255 chars should return 400: %s", string(body))
	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "name")
}

// --- Duplicate-name conflict (409) ---

func TestCovCatalogProductLines_CreateDuplicateNameConflict(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(productLinesPath, map[string]any{
		"name":              SeedProductLineName,
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "commission_applied",
		"freight_policy":    "billed_freight",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 409, status, "creating a product line with a duplicate name should return 409: %s", string(body))
	errObj := requireErrorResponse(t, body, "resource_conflict", "invalid_request_error")
	assertErrorParam(t, errObj, "name")
}

func TestCovCatalogProductLines_UpdateDuplicateNameConflict(t *testing.T) {
	t.Parallel()
	id := covCatalogProductLinesCreate(t, uniqueName("e2e-cov-pdln-updconflict"))
	defer apiClient.Delete(productLinesPath + "/" + id)

	status, body, err := apiClient.Patch(productLinesPath+"/"+id, map[string]any{
		"name": SeedProductLineName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 409, status, "renaming a product line to an existing name should return 409: %s", string(body))
	errObj := requireErrorResponse(t, body, "resource_conflict", "invalid_request_error")
	assertErrorParam(t, errObj, "name")

	// Verify the name was NOT actually changed by the rejected update.
	getStatus, getBody, err := apiClient.GetListRaw(productLinesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.NotEqual(t, SeedProductLineName, jsonField(parseJSON(getBody), "name"),
		"name must not change when the rename is rejected as a conflict")
}

// --- Nonexistent unit_group_id FK (create + update) ---
//
// Per the e2e-coverage audit this was flagged as a "likely 500" bug
// (db.MapSQLError not mapping MySQL 1452 FK-violation errors). Verified live
// against the running stack: both create and update already return 404
// (resource_not_found) for a well-formed-but-nonexistent unit_group_id, not
// 500. These assertions lock in the correct behavior; if this regresses to
// a 500, MapSQLError needs to map MySQL error 1452.

func TestCovCatalogProductLines_CreateNonexistentUnitGroupID(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(productLinesPath, map[string]any{
		"name":              uniqueName("e2e-cov-pdln-badug"),
		"unit_group_id":     "ungp_000000000000",
		"commission_policy": "commission_applied",
		"freight_policy":    "billed_freight",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status,
		"creating a product line with a nonexistent unit_group_id should return 404, not 500: %s", string(body))
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

func TestCovCatalogProductLines_UpdateNonexistentUnitGroupID(t *testing.T) {
	t.Parallel()
	id := covCatalogProductLinesCreate(t, uniqueName("e2e-cov-pdln-updbadug"))
	defer apiClient.Delete(productLinesPath + "/" + id)

	status, body, err := apiClient.Patch(productLinesPath+"/"+id, map[string]any{
		"unit_group_id": "ungp_000000000000",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status,
		"updating a product line with a nonexistent unit_group_id should return 404, not 500: %s", string(body))
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// --- Customer-portal write denial (update + delete) ---
//
// TestCustomerPortalAccess_CannotCreateProductLine already covers create
// denial; these cover the remaining update/delete write paths.

func TestCovCatalogProductLines_CustomerPortalCannotUpdate(t *testing.T) {
	t.Parallel()

	// Use a dedicated admin-owned product line rather than the shared seed row
	// for the post-denial integrity check: the customer-portal write denial is
	// a blanket "must be internal user" 403 in the auth middleware (independent
	// of which resource is targeted), so a fresh fixture exercises the same
	// contract while keeping this test self-contained and robust against the
	// seed row's liveness.
	name := uniqueName("e2e-cov-cust-pdln-upd-target")
	id := covCatalogProductLinesCreate(t, name)
	defer apiClient.Delete(productLinesPath + "/" + id)

	client := getCustomerPortalClient()

	status, body, err := client.Patch(productLinesPath+"/"+id, map[string]any{
		"name": uniqueName("e2e-cov-cust-pdln-upd"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 403, status, "customer portal should not be able to update product lines: %s", string(body))

	// Verify the product line's name was not changed by the denied update.
	getStatus, getBody, err := apiClient.GetListRaw(productLinesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, name, jsonField(parseJSON(getBody), "name"),
		"product line name must be unchanged after a denied customer-portal update")
}

func TestCovCatalogProductLines_CustomerPortalCannotDelete(t *testing.T) {
	t.Parallel()

	// Dedicated admin-owned fixture; see CannotUpdate for why the shared seed
	// row is not used for the post-denial existence check.
	id := covCatalogProductLinesCreate(t, uniqueName("e2e-cov-cust-pdln-del-target"))
	defer apiClient.Delete(productLinesPath + "/" + id)

	client := getCustomerPortalClient()

	status, body, err := client.Delete(productLinesPath + "/" + id)
	require.NoError(t, err)
	assert.Equal(t, 403, status, "customer portal should not be able to delete product lines: %s", string(body))

	// Verify the product line still exists after the denied delete.
	getStatus, getBody, err := apiClient.GetListRaw(productLinesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, id, jsonField(parseJSON(getBody), "id"),
		"product line must still exist after a denied customer-portal delete")
}

// --- Response-shape nicety flagged by the audit ---

func TestCovCatalogProductLines_CreateResponseIDFormat(t *testing.T) {
	t.Parallel()
	id := covCatalogProductLinesCreate(t, uniqueName("e2e-cov-pdln-idfmt"))
	defer apiClient.Delete(productLinesPath + "/" + id)

	assertIDFormat(t, id, "pdln")
}

// covCatalogProductLinesCreate creates a minimal valid product line with the
// given name and returns its id. It does not register cleanup; callers must
// defer apiClient.Delete(productLinesPath + "/" + id) themselves.
func covCatalogProductLinesCreate(t *testing.T, name string) string {
	t.Helper()
	status, body, err := apiClient.Post(productLinesPath, map[string]any{
		"name":              name,
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "commission_applied",
		"freight_policy":    "billed_freight",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, id)
	return id
}
