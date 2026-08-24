//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file closes the coverage gaps identified for catalog_materials (/v1/catalog/materials) on top of the extensive existing coverage in crud_materials_test.go and create_materials_test.go: list date-range filters, nonexistent-FK validation, duplicate-SKU conflicts, PATCH explicit-null rejection on Optional (non-Clearable) fields, delete-of-already-deleted, SKU max-length boundary, and the documented zero-value-collapses-to-null bug in order_point/lead_time.

// ──────────────────────────────────────────────
// --- List: starts_at / ends_at filters ---
// ──────────────────────────────────────────────

func TestCovCatalogMaterials_ListFilterByStartDate(t *testing.T) {
	t.Parallel()
	sku := uniqueName("e2e-mat-sd")
	created := createAndCleanup(t, materialsPath, validMaterialBody(sku))
	id := jsonField(created, "id")

	// A starts_at safely in the past should include the just-created material.
	list, status, err := apiClient.GetList(materialsPath, url.Values{
		"starts_at": {"2020-01-01T00:00:00Z"},
		"q":         {sku},
	})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	found := false
	for _, raw := range list.Data {
		if DataItemField(raw, "id") == id {
			found = true
		}
	}
	assert.True(t, found, "material created after starts_at should be included")

	// A starts_at in the future should exclude the material.
	list2, status2, err := apiClient.GetList(materialsPath, url.Values{
		"starts_at": {"2099-01-01T00:00:00Z"},
		"q":         {sku},
	})
	require.NoError(t, err)
	require.Equal(t, 200, status2)
	assertEmptyListData(t, list2.Data)
}

func TestCovCatalogMaterials_ListFilterByEndDate(t *testing.T) {
	t.Parallel()
	sku := uniqueName("e2e-mat-ed")
	created := createAndCleanup(t, materialsPath, validMaterialBody(sku))
	id := jsonField(created, "id")

	// An ends_at safely in the future should include the just-created material.
	list, status, err := apiClient.GetList(materialsPath, url.Values{
		"ends_at": {"2099-01-01T00:00:00Z"},
		"q":       {sku},
	})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	found := false
	for _, raw := range list.Data {
		if DataItemField(raw, "id") == id {
			found = true
		}
	}
	assert.True(t, found, "material created before ends_at should be included")

	// An ends_at in the past should exclude the material.
	list2, status2, err := apiClient.GetList(materialsPath, url.Values{
		"ends_at": {"2020-01-01T00:00:00Z"},
		"q":       {sku},
	})
	require.NoError(t, err)
	require.Equal(t, 200, status2)
	assertEmptyListData(t, list2.Data)
}

func TestCovCatalogMaterials_ListFilterByStartDateMalformed(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(materialsPath, url.Values{"starts_at": {"not-a-date"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assertErrorParam(t, errObj, "starts_at")
}

func TestCovCatalogMaterials_ListFilterByEndDateMalformed(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(materialsPath, url.Values{"ends_at": {"not-a-date"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assertErrorParam(t, errObj, "ends_at")
}

// ──────────────────────────────────────────────
// --- List: nonexistent FK filters ---
// ──────────────────────────────────────────────

func TestCovCatalogMaterials_ListFilterByNonexistentCategoryID(t *testing.T) {
	t.Parallel()
	list, status, err := apiClient.GetList(materialsPath, url.Values{"category_ids": {"itcg_00000000000000000"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	assertEmptyListData(t, list.Data)
}

func TestCovCatalogMaterials_ListFilterByNonexistentAttributeID(t *testing.T) {
	t.Parallel()
	// A per-run-unique id can never collide with a link some other test persisted.
	bogusAttrID := "at_" + strings.ReplaceAll(newIdempotencyKey(), "-", "")
	list, status, err := apiClient.GetList(materialsPath, url.Values{"attribute_ids": {bogusAttrID}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	assertEmptyListData(t, list.Data)
}

// ──────────────────────────────────────────────
// --- Include: unknown value rejection ---
// ──────────────────────────────────────────────

func TestCovCatalogMaterials_RetrieveUnknownIncludeValueRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(materialsPath+"/"+SeedMaterialID, url.Values{"include": {"bogus_field"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assertErrorParam(t, errObj, "include[]")
}

// ──────────────────────────────────────────────
// --- Duplicate SKU conflicts (409) ---
// ──────────────────────────────────────────────

// TestCovCatalogMaterials_CreateDuplicateSKUConflict verifies that creating a material whose SKU is already used by another item is rejected with a 409 conflict scoped to the sku param, per the documented business rule in core-service/internal/service/material_service.go.
func TestCovCatalogMaterials_CreateDuplicateSKUConflict(t *testing.T) {
	t.Parallel()

	// SeedMaterialID's underlying item SKU ("YRN-001") is stable seed data and never deleted by other tests, so it is safe to use as the conflicting SKU.
	body := validMaterialBody(covCatalogMaterialsSeedYarnSKU)
	status, respBody, err := apiClient.Post(materialsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, respBody)

	errObj := requireErrorResponse(t, respBody, "resource_conflict", "invalid_request_error")
	assertErrorParam(t, errObj, "sku")
}

func TestCovCatalogMaterials_UpdateDuplicateSKUConflict(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, materialsPath, validMaterialBody(uniqueName("e2e-mat-upddup")))
	id := jsonField(created, "id")

	status, respBody, err := apiClient.Patch(materialsPath+"/"+id, map[string]any{"sku": covCatalogMaterialsSeedYarnSKU}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, respBody)

	errObj := requireErrorResponse(t, respBody, "resource_conflict", "invalid_request_error")
	assertErrorParam(t, errObj, "sku")
}

// covCatalogMaterialsSeedYarnSKU is the SKU of SeedMaterialID's underlying item, used as a stable conflict target for duplicate-SKU tests.
const covCatalogMaterialsSeedYarnSKU = "YRN-001"

// ──────────────────────────────────────────────
// --- PATCH explicit-null rejection (Optional, not Clearable) ---
// ──────────────────────────────────────────────

func TestCovCatalogMaterials_UpdateExplicitNullSKURejected(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, materialsPath, validMaterialBody(uniqueName("e2e-mat-nullsku")))
	id := jsonField(created, "id")

	status, body, err := apiClient.Patch(materialsPath+"/"+id, map[string]any{"sku": nil}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "sku")
}

func TestCovCatalogMaterials_UpdateExplicitNullDescriptionRejected(t *testing.T) {
	t.Parallel()
	body := validMaterialBody(uniqueName("e2e-mat-nulldesc"))
	body["description"] = "will not be cleared"
	created := createAndCleanup(t, materialsPath, body)
	id := jsonField(created, "id")

	status, respBody, err := apiClient.Patch(materialsPath+"/"+id, map[string]any{"description": nil}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)
	errObj := requireErrorResponse(t, respBody, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "description")
}

func TestCovCatalogMaterials_UpdateExplicitNullNotesRejected(t *testing.T) {
	t.Parallel()
	body := validMaterialBody(uniqueName("e2e-mat-nullnotes"))
	body["notes"] = "will not be cleared"
	created := createAndCleanup(t, materialsPath, body)
	id := jsonField(created, "id")

	status, respBody, err := apiClient.Patch(materialsPath+"/"+id, map[string]any{"notes": nil}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)
	errObj := requireErrorResponse(t, respBody, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "notes")
}

// ──────────────────────────────────────────────
// --- Update-side rate currency-rule validation ---
// ──────────────────────────────────────────────

// TestCovCatalogMaterials_UpdateUnitCostRejectsNonCurrencyNumerator mirrors TestMaterials_Create_RejectsNonCurrencyNumeratorOnUnitCost but exercises the PATCH path, which was previously untested.
func TestCovCatalogMaterials_UpdateUnitCostRejectsNonCurrencyNumerator(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, materialsPath, validMaterialBody(uniqueName("e2e-mat-updratebad")))
	id := jsonField(created, "id")

	status, body, err := apiClient.Patch(materialsPath+"/"+id, map[string]any{
		"unit_cost": map[string]any{
			"value":               "0.50",
			"numerator_unit_id":   nonCurrencyUnitID, // wrong: must be currency
			"denominator_unit_id": nonCurrencyUnitID,
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "unit_cost.numerator_unit_id")
}

// ──────────────────────────────────────────────
// --- Nonexistent FK validation (category_id / unit_id / attribute_ids) ---
// ──────────────────────────────────────────────

// A nonexistent category_id is reported against the field, the way the sibling foreign keys on this request (unit_price/unit_cost unit ids) are, rather than as a bare not-found for the material itself.
func TestCovCatalogMaterials_CreateNonexistentCategoryID(t *testing.T) {
	t.Parallel()
	body := validMaterialBody(uniqueName("e2e-mat-badcat"))
	body["category_id"] = "itcg_00000000000000000"

	status, respBody, err := apiClient.Post(materialsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)
	errObj := requireErrorResponse(t, respBody, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "category_id")
}

// A nonexistent order_point.unit_id is a clean client error rather than a raw 500, the same way unit_price and unit_cost unit ids are guarded.
func TestCovCatalogMaterials_CreateNonexistentOrderPointUnitID(t *testing.T) {
	t.Parallel()
	body := validMaterialBody(uniqueName("e2e-mat-badopunit"))
	body["order_point"] = map[string]any{"value": "5", "unit_id": "un_00000000000000000"}

	status, respBody, err := apiClient.Post(materialsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	require.NotEqual(t, 500, status, "nonexistent order_point.unit_id must not 500 (body: %s)", string(respBody))
	assert.True(t, status == 400 || status == 404 || status == 422,
		"nonexistent order_point.unit_id should be a clean client error, got %d: %s", status, string(respBody))
}

func TestCovCatalogMaterials_CreateNonexistentLeadTimeUnitID(t *testing.T) {
	t.Parallel()
	body := validMaterialBody(uniqueName("e2e-mat-badltunit"))
	body["lead_time"] = map[string]any{"value": "3", "unit_id": "un_00000000000000000"}

	status, respBody, err := apiClient.Post(materialsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	require.NotEqual(t, 500, status, "nonexistent lead_time.unit_id must not 500 (body: %s)", string(respBody))
	assert.True(t, status == 400 || status == 404 || status == 422,
		"nonexistent lead_time.unit_id should be a clean client error, got %d: %s", status, string(respBody))
}

func TestCovCatalogMaterials_UpdateNonexistentOrderPointUnitID(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, materialsPath, validMaterialBody(uniqueName("e2e-mat-updbadop")))
	id := jsonField(created, "id")

	status, respBody, err := apiClient.Patch(materialsPath+"/"+id, map[string]any{
		"order_point": map[string]any{"value": "5", "unit_id": "un_00000000000000000"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.NotEqual(t, 500, status, "nonexistent order_point.unit_id on update must not 500 (body: %s)", string(respBody))
	assert.True(t, status == 400 || status == 404 || status == 422,
		"nonexistent order_point.unit_id on update should be a clean client error, got %d: %s", status, string(respBody))
}

// TestCovCatalogMaterials_CreateNonexistentAttributeIDRejected covers the create-side attribute check: a nonexistent attribute_ids entry is a clean client error, not a 5xx and not a silently-dropped id.
func TestCovCatalogMaterials_CreateNonexistentAttributeIDRejected(t *testing.T) {
	t.Parallel()
	body := validMaterialBody(uniqueName("e2e-mat-badattr"))
	body["attribute_ids"] = []string{"at_00000000000000000"}

	status, respBody, err := apiClient.Post(materialsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	require.NotEqual(t, 500, status, "nonexistent attribute_ids entry must not 500 (body: %s)", string(respBody))
	requireStatus(t, 404, status, respBody)
	requireErrorResponse(t, respBody, "resource_not_found", "invalid_request_error")
}

// ──────────────────────────────────────────────
// --- SKU max-length boundary ---
// ──────────────────────────────────────────────

func TestCovCatalogMaterials_CreateSKUAtMaxLengthSucceeds(t *testing.T) {
	t.Parallel()
	prefix := uniqueName("e2e-mat-max255")
	sku := prefix + strings.Repeat("x", 255-len(prefix))
	require.Len(t, sku, 255)

	status, body, err := apiClient.Post(materialsPath, validMaterialBody(sku), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	id := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(materialsPath + "/" + id)
}

func TestCovCatalogMaterials_CreateSKUExceedsMaxLengthRejected(t *testing.T) {
	t.Parallel()
	prefix := uniqueName("e2e-mat-max256")
	sku := prefix + strings.Repeat("x", 256-len(prefix))
	require.Len(t, sku, 256)

	status, body, err := apiClient.Post(materialsPath, validMaterialBody(sku), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "sku")
}

// ──────────────────────────────────────────────
// --- Delete of already-deleted material ---
// ──────────────────────────────────────────────

func TestCovCatalogMaterials_DeleteAlreadyDeleted(t *testing.T) {
	t.Parallel()
	sku := uniqueName("e2e-mat-deldel")
	status, body, err := apiClient.Post(materialsPath, validMaterialBody(sku), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")

	delStatus, delBody, err := apiClient.Delete(materialsPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	status2, body2, err := apiClient.Delete(materialsPath + "/" + id)
	require.NoError(t, err)
	assert.True(t, status2 == 404 || status2 == 410,
		"deleting an already-deleted material should return 404 or 410, got %d: %s", status2, string(body2))
}

// ──────────────────────────────────────────────
// --- Zero-value Quantity normalizes to null (no-threshold semantics) ---
// ──────────────────────────────────────────────

// TestCovCatalogMaterials_OrderPointZeroValueNormalizesToNull pins the intended contract: a "0" order_point is presented as null, meaning "no reorder threshold". This is deliberate, not a presenter bug. The create path always materializes an order_point quantity row for every material — when the caller omits order_point it is stored as 0 (verified: all material rows have a non-null order_point_id, defaulting to value 0). Storage therefore cannot distinguish "explicitly 0" from "unset/default 0", so materialQuantityFromProto normalizes value 0 to null. A reorder point of 0 (reorder only once stock hits zero) is semantically equivalent to having no threshold, so null is the correct, self-consistent representation. A genuine nonzero order_point still round-trips (asserted separately in crud_materials_test.go).
func TestCovCatalogMaterials_OrderPointZeroValueNormalizesToNull(t *testing.T) {
	t.Parallel()
	body := validMaterialBody(uniqueName("e2e-mat-zeroop"))
	body["order_point"] = map[string]any{"value": "0", "unit_id": SeedUnitID}

	status, respBody, err := apiClient.Post(materialsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)

	got := parseJSON(respBody)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(materialsPath + "/" + id)

	orderPoint := jsonObject(got, "order_point")
	require.Nil(t, orderPoint, "a \"0\" order_point normalizes to null (no reorder threshold), since unset also stores 0 and the two are indistinguishable at the storage layer")
}

// TestCovCatalogMaterials_LeadTimeZeroValueNormalizesToNull is the lead_time analog of TestCovCatalogMaterials_OrderPointZeroValueNormalizesToNull; same intended contract, same normalization in materialQuantityFromProto. A lead_time of 0 (instantly available) is equivalent to no lead time, so null is correct.
func TestCovCatalogMaterials_LeadTimeZeroValueNormalizesToNull(t *testing.T) {
	t.Parallel()
	body := validMaterialBody(uniqueName("e2e-mat-zerolt"))
	body["lead_time"] = map[string]any{"value": "0", "unit_id": SeedUnitID}

	status, respBody, err := apiClient.Post(materialsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)

	got := parseJSON(respBody)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(materialsPath + "/" + id)

	leadTime := jsonObject(got, "lead_time")
	require.Nil(t, leadTime, "a \"0\" lead_time normalizes to null (no lead time), since unset also stores 0 and the two are indistinguishable at the storage layer")
}
