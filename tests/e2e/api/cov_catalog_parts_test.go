//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file closes gaps identified in the catalog_parts e2e coverage review:
// duplicate-SKU conflicts (create + update), nonexistent category_id/unit_id
// references, nonexistent attribute_ids, DELETE-already-deleted semantics
// (410, not 200/404), PATCH/DELETE on a never-existed id, the create/update
// include asymmetry (item.category.properties valid on GET but not on
// create/update), the SKU max-length boundary, blanking SKU via PATCH, and a
// concrete-value round trip for item.unit_value (previously only unit_cost's
// value was checked).

// ──────────────────────────────────────────────
// Part — Duplicate SKU conflicts (409)
// ──────────────────────────────────────────────

func TestCovCatalogParts_CreateDuplicateSKUConflict(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-cov-part-dupsku")
	original := createAndCleanup(t, partsPath, validPartBody(sku))
	require.NotEmpty(t, jsonField(original, "id"))

	status, body, err := apiClient.Post(partsPath, validPartBody(sku), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, body)

	errObj := requireErrorResponse(t, body, "resource_conflict", "invalid_request_error")
	assertErrorParam(t, errObj, "sku")
}

func TestCovCatalogParts_UpdateDuplicateSKUConflict(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-cov-part-dupsku-upd")
	holder := createAndCleanup(t, partsPath, validPartBody(sku))
	require.NotEmpty(t, jsonField(holder, "id"))

	other := createAndCleanup(t, partsPath, validPartBody(uniqueName("e2e-cov-part-dupsku-other")))
	otherID := jsonField(other, "id")
	require.NotEmpty(t, otherID)

	status, body, err := apiClient.Patch(partsPath+"/"+otherID, map[string]any{"sku": sku}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, body)

	errObj := requireErrorResponse(t, body, "resource_conflict", "invalid_request_error")
	assertErrorParam(t, errObj, "sku")
}

// ──────────────────────────────────────────────
// Part — SKU max-length boundary and blank-via-PATCH
// ──────────────────────────────────────────────

func TestCovCatalogParts_CreateSKUTooLong(t *testing.T) {
	t.Parallel()

	longSKU := make([]byte, 256)
	for i := range longSKU {
		longSKU[i] = 'a'
	}

	status, body, err := apiClient.Post(partsPath, map[string]any{
		"sku":         string(longSKU),
		"category_id": SeedItemCategoryID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "sku")
}

func TestCovCatalogParts_UpdateSKUEmptyStringRejected(t *testing.T) {
	t.Parallel()

	created := createAndCleanup(t, partsPath, validPartBody(uniqueName("e2e-cov-part-blanksku")))
	id := jsonField(created, "id")
	require.NotEmpty(t, id)

	// UpdatePartRequest.SKU is field.Optional[string] with validate:"omitempty,max=255",
	// which could in principle exempt an explicit "" from length validation. The live
	// stack in fact rejects it (400 invalid_format, "must not be blank"), confirming the
	// SKU-blanking concern flagged in the coverage review is not a real gap. Assert the
	// documented/correct behavior so a future regression to a silent 200 is caught.
	status, body, err := apiClient.Patch(partsPath+"/"+id, map[string]any{"sku": ""}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "sku")
}

// ──────────────────────────────────────────────
// Part — Nonexistent FK references
// ──────────────────────────────────────────────

func TestCovCatalogParts_CreateNonexistentCategoryID(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(partsPath, map[string]any{
		"sku":         uniqueName("e2e-cov-part-badcat"),
		"category_id": "itcg_nonexistent0000000",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)

	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

func TestCovCatalogParts_CreateNonexistentUnitPriceUnit(t *testing.T) {
	t.Parallel()

	body := validPartBody(uniqueName("e2e-cov-part-badunit-price"))
	body["unit_price"] = map[string]any{
		"value":               "5.00",
		"numerator_unit_id":   currencyUnitID,
		"denominator_unit_id": "nonexistent_unit_id",
	}

	status, respBody, err := apiClient.Post(partsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)

	errObj := requireErrorResponse(t, respBody, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "unit_price.denominator_unit_id")
}

func TestCovCatalogParts_CreateNonexistentUnitCostUnit(t *testing.T) {
	t.Parallel()

	body := validPartBody(uniqueName("e2e-cov-part-badunit-cost"))
	body["unit_cost"] = map[string]any{
		"value":               "2.50",
		"numerator_unit_id":   "nonexistent_unit_id",
		"denominator_unit_id": nonCurrencyUnitID,
	}

	status, respBody, err := apiClient.Post(partsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)

	errObj := requireErrorResponse(t, respBody, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "unit_cost.numerator_unit_id")
}

// ──────────────────────────────────────────────
// Part — Nonexistent attribute_ids
// ──────────────────────────────────────────────

// TestCovCatalogParts_CreateWithNonexistentAttributeIDRejected covers the create-side attribute check. The _item_attributes join table has no FK constraint on either column, so an unchecked id would be accepted as a silent orphan join row instead of a clean client error.
func TestCovCatalogParts_CreateWithNonexistentAttributeIDRejected(t *testing.T) {
	t.Parallel()

	body := validPartBody(uniqueName("e2e-cov-part-badattr"))
	body["attribute_ids"] = []string{"at_nonexistent00000000"}

	status, respBody, err := apiClient.Post(partsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, respBody)
	requireErrorResponse(t, respBody, "resource_not_found", "invalid_request_error")
}

// ──────────────────────────────────────────────
// Part — DELETE-already-deleted (410 Gone, not 200/404/500)
// ──────────────────────────────────────────────

// TestCovCatalogParts_DeleteAlreadyDeletedReturns410 asserts the documented,
// parts-specific contract: unlike most resources in this codebase (which
// tolerate a repeat DELETE as an idempotent 200/404), DeletePartEndpoint's doc
// comment states re-deleting an already-deleted part is an error — the service
// returns apierror.NewAlreadyDeletedError, which maps to 410 Gone with code
// "resource_gone". Do NOT loosen this to also accept 200/404 like the generic
// double-delete pattern used elsewhere; that would mask a real regression.
func TestCovCatalogParts_DeleteAlreadyDeletedReturns410(t *testing.T) {
	t.Parallel()

	created := createAndCleanup(t, partsPath, validPartBody(uniqueName("e2e-cov-part-deldel")))
	id := jsonField(created, "id")
	require.NotEmpty(t, id)

	delStatus, delBody, err := apiClient.Delete(partsPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	status2, body2, err := apiClient.Delete(partsPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 410, status2, body2)

	errObj := requireErrorResponse(t, body2, "resource_gone", "invalid_request_error")
	assert.Nil(t, errObj["param"])
}

// ──────────────────────────────────────────────
// Part — PATCH/DELETE on a fabricated, never-existed id (404, not the
// delete-then-verify-404 shortcut already covered elsewhere)
// ──────────────────────────────────────────────

func TestCovCatalogParts_PatchNeverExistedIDReturns404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Patch(
		partsPath+"/pt_covneverexisted0001",
		map[string]any{"sku": uniqueName("e2e-cov-part-ghost")},
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

func TestCovCatalogParts_DeleteNeverExistedIDReturns404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Delete(partsPath + "/pt_covneverexisted0002")
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// ──────────────────────────────────────────────
// Part — Create/update include asymmetry: item.category.properties is valid
// on GET/LIST but not on create/update.
// ──────────────────────────────────────────────

func TestCovCatalogParts_CreateRejectsGetOnlyIncludeKey(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(
		partsPath+"?include=item.category.properties",
		validPartBody(uniqueName("e2e-cov-part-badinc-create")),
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "include[]")
}

func TestCovCatalogParts_UpdateRejectsGetOnlyIncludeKey(t *testing.T) {
	t.Parallel()

	created := createAndCleanup(t, partsPath, validPartBody(uniqueName("e2e-cov-part-badinc-update")))
	id := jsonField(created, "id")
	require.NotEmpty(t, id)

	status, body, err := apiClient.Patch(
		partsPath+"/"+id+"?include=item.category.properties",
		map[string]any{"sku": uniqueName("e2e-cov-part-badinc-update-sku")},
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "include[]")
}

// ──────────────────────────────────────────────
// Part — item.unit_value.value concrete round trip (previously only
// item.unit_cost.value was asserted to a specific number).
// ──────────────────────────────────────────────

func TestCovCatalogParts_CreateWithValidRates_UnitValueRoundTrip(t *testing.T) {
	t.Parallel()

	body := validPartBody(uniqueName("e2e-cov-part-unitvalue"))
	body["unit_price"] = map[string]any{
		"value":               "5.00",
		"numerator_unit_id":   currencyUnitID,
		"denominator_unit_id": nonCurrencyUnitID,
	}

	resp, err := apiClient.PostFull(partsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	got := parseJSON(resp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { apiClient.Delete(partsPath + "/" + id) })

	status, respBody, err := apiClient.GetListRaw(partsPath+"/"+id, url.Values{"include": {"item.unit_value"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, respBody)

	uv := jsonObject(jsonObject(parseJSON(respBody), "item"), "unit_value")
	require.NotNil(t, uv, "item.unit_value should be populated with ?include=item.unit_value")
	assert.Equal(t, "rate", jsonField(uv, "object"))
	assert.Equal(t, "5.00", jsonField(uv, "value"))
	assert.NotEmpty(t, jsonField(uv, "id"))
	assertValidTimestamp(t, jsonField(uv, "created_at"), "item.unit_value.created_at")
	assertValidTimestamp(t, jsonField(uv, "updated_at"), "item.unit_value.updated_at")
}

// ──────────────────────────────────────────────
// Part — allFields: every Part + Item field with concrete expected values for
// both item.unit_value.value and item.unit_cost.value in a single response.
// ──────────────────────────────────────────────

func TestCovCatalogParts_CreateAndUpdateAllFields_ConcreteRateValues(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-cov-part-allf2")
	desc := "All-fields concrete-rates description"
	notes := "All-fields concrete-rates notes"
	body := validPartBody(sku)
	body["description"] = desc
	body["notes"] = notes
	body["unit_price"] = map[string]any{
		"value":               "5.00",
		"numerator_unit_id":   currencyUnitID,
		"denominator_unit_id": nonCurrencyUnitID,
	}
	body["unit_cost"] = map[string]any{
		"value":               "2.50",
		"numerator_unit_id":   currencyUnitID,
		"denominator_unit_id": nonCurrencyUnitID,
	}
	body["attribute_ids"] = []string{SeedAttributeID}

	resp, err := apiClient.PostFull(
		partsPath+"?include=item,item.category,item.unit_value,item.unit_cost,item.burn_rate,item.attributes",
		body, newIdempotencyKey(),
	)
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	got := parseJSON(resp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { apiClient.Delete(partsPath + "/" + id) })

	// Part-level fields.
	assertIDFormat(t, id, "pt")
	assertObjectField(t, got, "part")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	item := jsonObject(got, "item")
	require.NotNil(t, item)
	assert.Equal(t, "item", jsonField(item, "object"))
	assert.NotEmpty(t, jsonField(item, "id"))
	assert.Equal(t, sku, jsonField(item, "sku"))
	assert.Equal(t, desc, jsonField(item, "description"))
	assert.Equal(t, notes, jsonField(item, "notes"))
	assert.Equal(t, "part", jsonField(item, "type"))
	assertValidTimestamp(t, jsonField(item, "created_at"), "item.created_at")
	assertValidTimestamp(t, jsonField(item, "updated_at"), "item.updated_at")

	cat := jsonObject(item, "category")
	require.NotNil(t, cat)
	assert.Equal(t, SeedItemCategoryID, jsonField(cat, "id"))

	uv := jsonObject(item, "unit_value")
	require.NotNil(t, uv)
	assert.Equal(t, "rate", jsonField(uv, "object"))
	assert.Equal(t, "5.00", jsonField(uv, "value"))

	uc := jsonObject(item, "unit_cost")
	require.NotNil(t, uc)
	assert.Equal(t, "rate", jsonField(uc, "object"))
	assert.Equal(t, "2.50", jsonField(uc, "value"))

	br := jsonObject(item, "burn_rate")
	require.NotNil(t, br)
	assert.Equal(t, "rate", jsonField(br, "object"))
	assert.Equal(t, "0.00", jsonField(br, "value"), "burn_rate is always initialized to 0 per day at part creation")

	attrs := jsonObject(item, "attributes")
	require.NotNil(t, attrs)
	assert.Equal(t, "list", jsonField(attrs, "object"))
	attrData := jsonArray(attrs, "data")
	require.NotEmpty(t, attrData)
	found := false
	for _, raw := range attrData {
		obj, ok := raw.(map[string]any)
		require.True(t, ok)
		if jsonField(obj, "id") == SeedAttributeID {
			found = true
		}
	}
	assert.True(t, found, "expected linked attribute %s in item.attributes", SeedAttributeID)

	// Update the sku and description, re-fetch with the same includes, and
	// confirm every field (including the two rate values) is preserved or
	// updated as expected — closing the gap where only unit_cost.value was
	// checked against a concrete number.
	newSKU := uniqueName("e2e-cov-part-allf2-upd")
	patchStatus, patchBody, err := apiClient.Patch(
		partsPath+"/"+id+"?include=item,item.category,item.unit_value,item.unit_cost,item.burn_rate,item.attributes",
		map[string]any{"sku": newSKU},
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	assert.Equal(t, id, jsonField(updated, "id"))
	assertObjectField(t, updated, "part")
	assertValidTimestamp(t, jsonField(updated, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(updated, "updated_at"), "updated_at")

	itemU := jsonObject(updated, "item")
	require.NotNil(t, itemU)
	assert.Equal(t, newSKU, jsonField(itemU, "sku"))
	assert.Equal(t, desc, jsonField(itemU, "description"), "description should be preserved across the sku-only patch")
	assert.Equal(t, notes, jsonField(itemU, "notes"), "notes should be preserved across the sku-only patch")

	uvU := jsonObject(itemU, "unit_value")
	require.NotNil(t, uvU)
	assert.Equal(t, "5.00", jsonField(uvU, "value"), "unit_value should be unaffected by an sku-only patch")

	ucU := jsonObject(itemU, "unit_cost")
	require.NotNil(t, ucU)
	assert.Equal(t, "2.50", jsonField(ucU, "value"), "unit_cost should be unaffected by an sku-only patch")
}
