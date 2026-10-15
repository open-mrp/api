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

// --- allFields: free_shipping contents, owner.account ---

func TestCovOperationsShippingTerms_CreateAndUpdateAllFieldsPopulated(t *testing.T) {
	t.Parallel()

	includeParams := "?include=free_shipping.service_levels,owner,owner.account"

	// ── CREATE with flat_rate, free_shipping and status all set ──
	name := covOperationsShippingTermsUniqueName("allf-pop")
	createResp, err := apiClient.PostFull(shippingTermsPath+includeParams, map[string]any{
		"name": name,
		"type": "flat_rate",
		"flat_rate": map[string]any{
			"amount":   "9.99",
			"currency": "usd",
		},
		"free_shipping": map[string]any{
			"minimum_order_value": map[string]any{"amount": "100.00", "currency": "usd"},
			"service_level_ids":   []string{SeedServiceLevelID},
		},
		"status": "archived",
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
	assert.Equal(t, "flat_rate", jsonField(got, "type"))
	assert.Equal(t, "archived", jsonField(got, "status"))
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

	assertShippingMoney(t, jsonObject(got, "flat_rate"), "9.99", "flat_rate after create")
	freeShipping := jsonObject(got, "free_shipping")
	require.NotNil(t, freeShipping)
	assertShippingMoney(t, jsonObject(freeShipping, "minimum_order_value"), "100.00", "free_shipping.minimum_order_value after create")
	assert.Equal(t, "selected", jsonField(freeShipping, "service_level_scope"))

	freeLevels := jsonListData(freeShipping, "service_levels")
	require.Len(t, freeLevels, 1, "free_shipping_service_levels should contain exactly the assigned service level")
	firstLevel, ok := freeLevels[0].(map[string]any)
	require.True(t, ok, "free_shipping_service_levels item should decode as an object")
	assert.Equal(t, SeedServiceLevelID, jsonField(firstLevel, "id"))
	assertObjectField(t, firstLevel, "service_level")

	// ── UPDATE: change flat_rate and the threshold, drop the service levels, reactivate ──
	updatedName := covOperationsShippingTermsUniqueName("allf-pop-u")
	patchResp, err := apiClient.PatchFull(shippingTermsPath+"/"+id+includeParams, map[string]any{
		"name": updatedName,
		"flat_rate": map[string]any{
			"amount":   "19.99",
			"currency": "usd",
		},
		"free_shipping": map[string]any{
			"minimum_order_value": map[string]any{"amount": "200.00", "currency": "usd"},
		},
		"status": "active",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchResp.StatusCode, patchResp.Body)

	updated := parseJSON(patchResp.Body)
	assert.Equal(t, id, jsonField(updated, "id"), "ID must not change")
	assert.Equal(t, updatedName, jsonField(updated, "name"))
	assert.Equal(t, "flat_rate", jsonField(updated, "type"), "type should be preserved")
	assert.Equal(t, "active", jsonField(updated, "status"))

	assertShippingMoney(t, jsonObject(updated, "flat_rate"), "19.99", "flat_rate after update")
	updFreeShipping := jsonObject(updated, "free_shipping")
	require.NotNil(t, updFreeShipping)
	assertShippingMoney(t, jsonObject(updFreeShipping, "minimum_order_value"), "200.00", "free_shipping.minimum_order_value after update")
	assert.Equal(t, "all", jsonField(updFreeShipping, "service_level_scope"), "free_shipping is replaced as a whole")
	assert.Empty(t, jsonListData(updFreeShipping, "service_levels"))
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

// --- money: the seeded term's flat_rate and free-shipping threshold are always present as Money ---

func TestCovOperationsShippingTerms_FlatRateAndMinimumOrderValueAreMoney(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(shippingTermsPath+"/"+SeedCustomShippingTermID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	for field, money := range map[string]map[string]any{
		"flat_rate":                         jsonObject(got, "flat_rate"),
		"free_shipping.minimum_order_value": jsonObject(jsonObject(got, "free_shipping"), "minimum_order_value"),
	} {
		require.NotNil(t, money, "seeded custom shipping term should have %s set", field)
		assertValueShapeBare(t, money, field)
		assert.NotEmpty(t, jsonField(money, "amount"), "%s.amount", field)
		assert.Equal(t, "usd", jsonField(money, "currency"), "%s.currency", field)
	}

	// Money has no unit to include.
	status, body, err = apiClient.GetListRaw(shippingTermsPath+"/"+SeedCustomShippingTermID, url.Values{"include": {"flat_rate.unit"}})
	require.NoError(t, err)
	require.Less(t, status, 500, string(body))
	assert.Equal(t, 400, status, string(body))
}

// --- expandable: free_shipping.service_levels contents against seeded populated list ---

func TestCovOperationsShippingTerms_IncludeFreeShippingServiceLevelsContents(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(shippingTermsPath+"/"+SeedCustomShippingTermID, url.Values{"include": {"free_shipping.service_levels"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	freeShipping := jsonObject(got, "free_shipping")
	require.NotNil(t, freeShipping)
	assert.Equal(t, "selected", jsonField(freeShipping, "service_level_scope"))
	levels := jsonListData(freeShipping, "service_levels")
	require.GreaterOrEqual(t, len(levels), 1, "seeded custom shipping term should have at least one free shipping service level")

	found := false
	for _, item := range levels {
		m, ok := item.(map[string]any)
		require.True(t, ok, "service_levels item should decode as an object")
		assertObjectField(t, m, "service_level")
		assert.NotEmpty(t, jsonField(m, "id"))
		if jsonField(m, "id") == SeedServiceLevelID {
			found = true
		}
	}
	assert.True(t, found, "expected free_shipping.service_levels to contain SeedServiceLevelID=%s", SeedServiceLevelID)
}

// --- validation: invalid type enum, oversized name, malformed MoneyInput ---

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
		"type": "free",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"name > 255 chars should return 400 or 422, got %d: %s", status, string(body))
}

// A flat_rate carrying only a currency is a validation error rather than a persisted half-amount.
func TestCovOperationsShippingTerms_CreateValidation_FlatRateMissingAmount(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(shippingTermsPath, map[string]any{
		"name": covOperationsShippingTermsUniqueName("flatrate-noval"),
		"type": "flat_rate",
		"flat_rate": map[string]any{
			"currency": "usd",
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	if status >= 200 && status < 300 {
		id := jsonField(parseJSON(body), "id")
		if id != "" {
			defer apiClient.Delete(shippingTermsPath + "/" + id)
		}
	}
	requireStatus(t, 422, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "validation_failed", "invalid_request_error"), "flat_rate.amount")
}

// A flat_rate carrying only an amount is a validation error rather than money in no currency.
func TestCovOperationsShippingTerms_CreateValidation_FlatRateMissingCurrency(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(shippingTermsPath, map[string]any{
		"name": covOperationsShippingTermsUniqueName("flatrate-nounit"),
		"type": "flat_rate",
		"flat_rate": map[string]any{
			"amount": "9.99",
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	if status >= 200 && status < 300 {
		id := jsonField(parseJSON(body), "id")
		if id != "" {
			defer apiClient.Delete(shippingTermsPath + "/" + id)
		}
	}
	requireStatus(t, 422, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "validation_failed", "invalid_request_error"), "flat_rate.currency")
}

// A flat_rate in a currency other than dollars is rejected rather than stored as dollars.
func TestCovOperationsShippingTerms_CreateValidation_FlatRateUnsupportedCurrency(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(shippingTermsPath, map[string]any{
		"name": covOperationsShippingTermsUniqueName("flatrate-fkunit"),
		"type": "flat_rate",
		"flat_rate": map[string]any{
			"amount":   "9.99",
			"currency": "eur",
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	if status >= 200 && status < 300 {
		id := jsonField(parseJSON(body), "id")
		if id != "" {
			defer apiClient.Delete(shippingTermsPath + "/" + id)
		}
	}
	requireStatus(t, 422, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "currency_not_supported", "invalid_request_error"), "flat_rate.currency")
}

// A free_shipping.service_level_ids entry naming a service level that does not exist is rejected, rather than silently dropped from the created term.
func TestCovOperationsShippingTerms_CreateValidation_FreeShippingServiceLevelUnknownID(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(shippingTermsPath, map[string]any{
		"name": covOperationsShippingTermsUniqueName("svclvl-fk"),
		"type": "carrier_rate",
		"free_shipping": map[string]any{
			"minimum_order_value": map[string]any{"amount": "1.00", "currency": "usd"},
			"service_level_ids":   []string{"crop_00000000000000000"},
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
		"free_shipping.service_level_ids with a non-existent ID should return 400, 404, or 422, got %d: %s", status, string(body))
}

// --- validation: PATCH explicit null on non-clearable fields ---

func TestCovOperationsShippingTerms_UpdateValidation_NullNameRejected(t *testing.T) {
	t.Parallel()
	name := covOperationsShippingTermsUniqueName("nullname")
	createStatus, createBody, err := apiClient.Post(shippingTermsPath, map[string]any{
		"name": name,
		"type": "free",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	defer apiClient.Delete(shippingTermsPath + "/" + id)

	patchStatus, patchBody, err := apiClient.Patch(shippingTermsPath+"/"+id, map[string]any{
		"name": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 422, patchStatus, "explicit null on non-clearable 'name' should return 422, got body: %s", string(patchBody))
}

func TestCovOperationsShippingTerms_UpdateValidation_NullTypeRejected(t *testing.T) {
	t.Parallel()
	name := covOperationsShippingTermsUniqueName("nulltype")
	createStatus, createBody, err := apiClient.Post(shippingTermsPath, map[string]any{
		"name": name,
		"type": "free",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	defer apiClient.Delete(shippingTermsPath + "/" + id)

	patchStatus, patchBody, err := apiClient.Patch(shippingTermsPath+"/"+id, map[string]any{
		"type": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 422, patchStatus, "explicit null on non-clearable 'type' should return 422, got body: %s", string(patchBody))
}

// --- update: clearing free_shipping via explicit null ---

func TestCovOperationsShippingTerms_UpdateClearFreeShipping(t *testing.T) {
	t.Parallel()
	name := covOperationsShippingTermsUniqueName("clearboth")
	createStatus, createBody, err := apiClient.Post(shippingTermsPath, map[string]any{
		"name": name,
		"type": "carrier_rate",
		"free_shipping": map[string]any{
			"minimum_order_value": map[string]any{"amount": "50.00", "currency": "usd"},
			"service_level_ids":   []string{SeedServiceLevelID},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	created := parseJSON(createBody)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(shippingTermsPath + "/" + id)
	require.NotNil(t, jsonObject(created, "free_shipping"), "free_shipping should be set after create")

	patchStatus, patchBody, err := apiClient.Patch(shippingTermsPath+"/"+id, map[string]any{
		"free_shipping": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	patched := parseJSON(patchBody)
	assert.Equal(t, name, jsonField(patched, "name"), "name should be preserved")
	assertNilField(t, patched, "free_shipping")
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
