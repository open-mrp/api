//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file closes e2e coverage gaps for /v1/operations/shipping-terms
// identified in the gap-analysis task. It extends, but does not duplicate,
// the coverage already present in crud_shipping_terms_test.go and
// crud_partial_includes_test.go — see TASK-operations_shipping-terms.md.

// --- allFields: minimum_order_value, free_shipping_service_levels contents, nested unit/owner.account ---

func TestCovOperationsShippingTerms_CreateAndUpdateAllFieldsPopulated(t *testing.T) {
	t.Parallel()

	includeParams := "?include=flat_rate.unit,minimum_order_value.unit,free_shipping_service_levels,owner,owner.account"

	// ── CREATE with flat_rate, minimum_order_value, and free_shipping_service_level_ids all set ──
	name := covOperationsShippingTermsUniqueName("allf-pop")
	createResp, err := apiClient.PostFull(shippingTermsPath+includeParams, map[string]any{
		"name": name,
		"type": "flat_rate_freight",
		"flat_rate": map[string]any{
			"value":   "9.99",
			"unit_id": SeedUnitID,
		},
		"minimum_order_value": map[string]any{
			"value":   "100.00",
			"unit_id": SeedUnitID,
		},
		"free_shipping_service_level_ids": []string{SeedServiceLevelID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	got := parseJSON(createResp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	assertIDFormat(t, id, "shtm")
	defer apiClient.Delete(shippingTermsPath + "/" + id)

	assertObjectField(t, got, "shipping_term")
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, "flat_rate_freight", jsonField(got, "type"))
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	// owner: top-level type + nested account (both requested via include)
	owner := jsonObject(got, "owner")
	require.NotNil(t, owner, "owner should be present with ?include=owner")
	assertObjectField(t, owner, "owner")
	assert.Equal(t, "account", jsonField(owner, "type"))
	ownerAccount := jsonObject(owner, "account")
	require.NotNil(t, ownerAccount, "owner.account should be present with ?include=owner.account")
	assert.Equal(t, SeedAccountID, jsonField(ownerAccount, "id"))
	assertObjectField(t, ownerAccount, "account")

	// flat_rate + nested unit
	flatRate := jsonObject(got, "flat_rate")
	require.NotNil(t, flatRate, "flat_rate must be set after create")
	assertObjectField(t, flatRate, "quantity")
	assert.Equal(t, "9.99", jsonField(flatRate, "value"))
	assert.NotEmpty(t, jsonField(flatRate, "display_value"))
	flatRateUnit := jsonObject(flatRate, "unit")
	require.NotNil(t, flatRateUnit, "flat_rate.unit should be present with ?include=flat_rate.unit")
	assert.Equal(t, SeedUnitID, jsonField(flatRateUnit, "id"))
	assertObjectField(t, flatRateUnit, "unit")

	// minimum_order_value + nested unit
	minOrderValue := jsonObject(got, "minimum_order_value")
	require.NotNil(t, minOrderValue, "minimum_order_value must be set after create")
	assertObjectField(t, minOrderValue, "quantity")
	assert.Equal(t, "100.00", jsonField(minOrderValue, "value"))
	assert.NotEmpty(t, jsonField(minOrderValue, "display_value"))
	minOrderValueUnit := jsonObject(minOrderValue, "unit")
	require.NotNil(t, minOrderValueUnit, "minimum_order_value.unit should be present with ?include=minimum_order_value.unit")
	assert.Equal(t, SeedUnitID, jsonField(minOrderValueUnit, "id"))
	assertObjectField(t, minOrderValueUnit, "unit")

	// free_shipping_service_levels contents
	freeLevels := jsonListData(got, "free_shipping_service_levels")
	require.Len(t, freeLevels, 1, "free_shipping_service_levels should contain exactly the assigned service level")
	firstLevel, ok := freeLevels[0].(map[string]any)
	require.True(t, ok, "free_shipping_service_levels item should decode as an object")
	assert.Equal(t, SeedServiceLevelID, jsonField(firstLevel, "id"))
	assertObjectField(t, firstLevel, "service_level")

	// ── UPDATE: change flat_rate/minimum_order_value values, keep same service level ──
	updatedName := covOperationsShippingTermsUniqueName("allf-pop-u")
	patchResp, err := apiClient.PatchFull(shippingTermsPath+"/"+id+includeParams, map[string]any{
		"name": updatedName,
		"flat_rate": map[string]any{
			"value":   "19.99",
			"unit_id": SeedUnitID,
		},
		"minimum_order_value": map[string]any{
			"value":   "200.00",
			"unit_id": SeedUnitID,
		},
		"free_shipping_service_level_ids": []string{SeedServiceLevelID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchResp.StatusCode, patchResp.Body)

	updated := parseJSON(patchResp.Body)
	assert.Equal(t, id, jsonField(updated, "id"), "ID must not change")
	assert.Equal(t, updatedName, jsonField(updated, "name"))
	assert.Equal(t, "flat_rate_freight", jsonField(updated, "type"), "type should be preserved")

	updFlatRate := jsonObject(updated, "flat_rate")
	require.NotNil(t, updFlatRate, "flat_rate should be present after update")
	assert.Equal(t, "19.99", jsonField(updFlatRate, "value"))
	require.NotNil(t, jsonObject(updFlatRate, "unit"), "flat_rate.unit should be present after update")

	updMinOrderValue := jsonObject(updated, "minimum_order_value")
	require.NotNil(t, updMinOrderValue, "minimum_order_value should be present after update")
	assert.Equal(t, "200.00", jsonField(updMinOrderValue, "value"))
	require.NotNil(t, jsonObject(updMinOrderValue, "unit"), "minimum_order_value.unit should be present after update")

	updFreeLevels := jsonListData(updated, "free_shipping_service_levels")
	require.Len(t, updFreeLevels, 1, "free_shipping_service_levels should still contain the assigned service level after update")
	updFirstLevel, ok := updFreeLevels[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, SeedServiceLevelID, jsonField(updFirstLevel, "id"))
}

// --- expandable: owner.account nested include against a real account-owned seed row ---

func TestCovOperationsShippingTerms_IncludeOwnerAccount(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(shippingTermsPath+"/"+SeedCustomShippingTermID, url.Values{"include": {"owner", "owner.account"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	owner := jsonObject(got, "owner")
	require.NotNil(t, owner, "owner should be present with ?include=owner")
	assert.Equal(t, "account", jsonField(owner, "type"))

	account := jsonObject(owner, "account")
	require.NotNil(t, account, "owner.account should be present with ?include=owner.account")
	assert.NotEmpty(t, jsonField(account, "id"))
	assertObjectField(t, account, "account")
}

// --- expandable: nested flat_rate.unit / minimum_order_value.unit against seeded populated quantities ---

func TestCovOperationsShippingTerms_IncludeFlatRateAndMinimumOrderValueUnit(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(shippingTermsPath+"/"+SeedCustomShippingTermID, url.Values{"include": {"flat_rate.unit", "minimum_order_value.unit"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)

	flatRate := jsonObject(got, "flat_rate")
	require.NotNil(t, flatRate, "seeded custom shipping term should have flat_rate set")
	flatRateUnit := jsonObject(flatRate, "unit")
	require.NotNil(t, flatRateUnit, "flat_rate.unit should be present with ?include=flat_rate.unit")
	assert.NotEmpty(t, jsonField(flatRateUnit, "id"))
	assertObjectField(t, flatRateUnit, "unit")

	minOrderValue := jsonObject(got, "minimum_order_value")
	require.NotNil(t, minOrderValue, "seeded custom shipping term should have minimum_order_value set")
	minOrderValueUnit := jsonObject(minOrderValue, "unit")
	require.NotNil(t, minOrderValueUnit, "minimum_order_value.unit should be present with ?include=minimum_order_value.unit")
	assert.NotEmpty(t, jsonField(minOrderValueUnit, "id"))
	assertObjectField(t, minOrderValueUnit, "unit")
}

// --- expandable: free_shipping_service_levels contents against seeded populated list ---

func TestCovOperationsShippingTerms_IncludeFreeShippingServiceLevelsContents(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(shippingTermsPath+"/"+SeedCustomShippingTermID, url.Values{"include": {"free_shipping_service_levels"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	levels := jsonListData(got, "free_shipping_service_levels")
	require.GreaterOrEqual(t, len(levels), 1, "seeded custom shipping term should have at least one free shipping service level")

	found := false
	for _, item := range levels {
		m, ok := item.(map[string]any)
		require.True(t, ok, "free_shipping_service_levels item should decode as an object")
		assertObjectField(t, m, "service_level")
		assert.NotEmpty(t, jsonField(m, "id"))
		if jsonField(m, "id") == SeedServiceLevelID {
			found = true
		}
	}
	assert.True(t, found, "expected free_shipping_service_levels to contain SeedServiceLevelID=%s", SeedServiceLevelID)
}

// --- validation: invalid type enum, oversized name, malformed QuantityInput ---

func TestCovOperationsShippingTerms_CreateValidation_InvalidTypeEnum(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(shippingTermsPath, map[string]any{
		"name": covOperationsShippingTermsUniqueName("badtype"),
		"type": "bogus_type",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Invalid type enum should return 400 or 422, got %d: %s", status, string(body))
}

func TestCovOperationsShippingTerms_CreateValidation_NameTooLong(t *testing.T) {
	t.Parallel()
	longName := covOperationsShippingTermsRepeat("a", 300)
	status, body, err := apiClient.Post(shippingTermsPath, map[string]any{
		"name": longName,
		"type": "free_freight",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"name > 255 chars should return 400 or 422, got %d: %s", status, string(body))
}

// QuantityInput.Value is validate:"required", so a flat_rate carrying only a unit_id is a validation error rather than a persisted half-quantity.
func TestCovOperationsShippingTerms_CreateValidation_FlatRateMissingValue(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(shippingTermsPath, map[string]any{
		"name": covOperationsShippingTermsUniqueName("flatrate-noval"),
		"type": "flat_rate_freight",
		"flat_rate": map[string]any{
			"unit_id": SeedUnitID,
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	if status >= 200 && status < 300 {
		id := jsonField(parseJSON(body), "id")
		if id != "" {
			defer apiClient.Delete(shippingTermsPath + "/" + id)
		}
	}
	assert.True(t, status == 400 || status == 422,
		"flat_rate missing value should return 400 or 422, got %d: %s", status, string(body))
}

// QuantityInput.UnitID is validate:"required", so a flat_rate carrying only a value is a validation error rather than a quantity with a null unit.
func TestCovOperationsShippingTerms_CreateValidation_FlatRateMissingUnitID(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(shippingTermsPath, map[string]any{
		"name": covOperationsShippingTermsUniqueName("flatrate-nounit"),
		"type": "flat_rate_freight",
		"flat_rate": map[string]any{
			"value": "9.99",
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	if status >= 200 && status < 300 {
		id := jsonField(parseJSON(body), "id")
		if id != "" {
			defer apiClient.Delete(shippingTermsPath + "/" + id)
		}
	}
	assert.True(t, status == 400 || status == 422,
		"flat_rate missing unit_id should return 400 or 422, got %d: %s", status, string(body))
}

// A flat_rate.unit_id naming a unit that does not exist is rejected, rather than persisting a quantity with no unit association.
func TestCovOperationsShippingTerms_CreateValidation_FlatRateUnknownUnitID(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(shippingTermsPath, map[string]any{
		"name": covOperationsShippingTermsUniqueName("flatrate-fkunit"),
		"type": "flat_rate_freight",
		"flat_rate": map[string]any{
			"value":   "9.99",
			"unit_id": "un_000000000000000000",
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	if status >= 200 && status < 300 {
		id := jsonField(parseJSON(body), "id")
		if id != "" {
			defer apiClient.Delete(shippingTermsPath + "/" + id)
		}
	}
	assert.True(t, status == 400 || status == 404 || status == 422,
		"flat_rate.unit_id referencing a non-existent unit should return 400, 404, or 422, got %d: %s", status, string(body))
}

// A free_shipping_service_level_ids entry naming a service level that does not exist is rejected, rather than silently dropped from the created term.
func TestCovOperationsShippingTerms_CreateValidation_FreeShippingServiceLevelUnknownID(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(shippingTermsPath, map[string]any{
		"name":                            covOperationsShippingTermsUniqueName("svclvl-fk"),
		"type":                            "free_freight",
		"free_shipping_service_level_ids": []string{"crop_00000000000000000"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	if status >= 200 && status < 300 {
		id := jsonField(parseJSON(body), "id")
		if id != "" {
			defer apiClient.Delete(shippingTermsPath + "/" + id)
		}
	}
	assert.True(t, status == 400 || status == 404 || status == 422,
		"free_shipping_service_level_ids with a non-existent ID should return 400, 404, or 422, got %d: %s", status, string(body))
}

// --- validation: PATCH explicit null on non-clearable fields ---

func TestCovOperationsShippingTerms_UpdateValidation_NullNameRejected(t *testing.T) {
	t.Parallel()
	name := covOperationsShippingTermsUniqueName("nullname")
	createStatus, createBody, err := apiClient.Post(shippingTermsPath, map[string]any{
		"name": name,
		"type": "free_freight",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	defer apiClient.Delete(shippingTermsPath + "/" + id)

	patchStatus, patchBody, err := apiClient.Patch(shippingTermsPath+"/"+id, map[string]any{
		"name": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, patchStatus, "explicit null on non-clearable 'name' should return 400, got body: %s", string(patchBody))
}

func TestCovOperationsShippingTerms_UpdateValidation_NullTypeRejected(t *testing.T) {
	t.Parallel()
	name := covOperationsShippingTermsUniqueName("nulltype")
	createStatus, createBody, err := apiClient.Post(shippingTermsPath, map[string]any{
		"name": name,
		"type": "free_freight",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	defer apiClient.Delete(shippingTermsPath + "/" + id)

	patchStatus, patchBody, err := apiClient.Patch(shippingTermsPath+"/"+id, map[string]any{
		"type": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, patchStatus, "explicit null on non-clearable 'type' should return 400, got body: %s", string(patchBody))
}

// --- update: clearing minimum_order_value and free_shipping_service_level_ids via explicit null ---

func TestCovOperationsShippingTerms_UpdateClearMinimumOrderValueAndServiceLevels(t *testing.T) {
	t.Parallel()
	name := covOperationsShippingTermsUniqueName("clearboth")
	createStatus, createBody, err := apiClient.Post(shippingTermsPath, map[string]any{
		"name": name,
		"type": "free_freight",
		"minimum_order_value": map[string]any{
			"value":   "50.00",
			"unit_id": SeedUnitID,
		},
		"free_shipping_service_level_ids": []string{SeedServiceLevelID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	created := parseJSON(createBody)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(shippingTermsPath + "/" + id)
	require.NotNil(t, jsonObject(created, "minimum_order_value"), "minimum_order_value should be set after create")

	patchStatus, patchBody, err := apiClient.Patch(shippingTermsPath+"/"+id, map[string]any{
		"minimum_order_value":             nil,
		"free_shipping_service_level_ids": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	patched := parseJSON(patchBody)
	assert.Equal(t, name, jsonField(patched, "name"), "name should be preserved")
	assert.Nil(t, patched["minimum_order_value"], "minimum_order_value should be cleared after sending null")
	assert.Nil(t, patched["free_shipping_service_levels"], "free_shipping_service_levels should be null without ?include, regardless of clear")
}

// --- helpers local to this file ---

func covOperationsShippingTermsUniqueName(prefix string) string {
	return uniqueName("e2e-cov-shipterm-" + prefix)
}

func covOperationsShippingTermsRepeat(s string, n int) string {
	out := make([]byte, 0, n*len(s))
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}
