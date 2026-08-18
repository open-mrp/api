//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/augno/api/shared/id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const volumeDiscountsPath = "/v1/sales/volume-discounts"

// ──────────────────────────────────────────────
// VolumeDiscount — Include Tests
// ──────────────────────────────────────────────

func TestVolumeDiscounts_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(volumeDiscountsPath+"/"+SeedVolumeDiscountID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["customer_groups"], "customer_groups should be null without include")
	assert.Nil(t, got["product_lines"], "product_lines should be null without include")
	assert.Nil(t, got["categories"], "categories should be null without include")
	assert.Nil(t, got["attributes"], "attributes should be null without include")
	assert.Nil(t, got["acceptable_units"], "acceptable_units should be null without include")
}

func TestVolumeDiscounts_IncludeCustomerGroups(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(volumeDiscountsPath+"/"+SeedVolumeDiscountID, url.Values{"include": {"customer_groups"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	cg := jsonObject(got, "customer_groups")
	require.NotNil(t, cg, "customer_groups should be present with ?include=customer_groups")
	assert.Equal(t, "list", jsonField(cg, "object"))
}

func TestVolumeDiscounts_IncludeProductLines(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(volumeDiscountsPath+"/"+SeedVolumeDiscountID, url.Values{"include": {"product_lines"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	pl := jsonObject(got, "product_lines")
	require.NotNil(t, pl, "product_lines should be present with ?include=product_lines")
	assert.Equal(t, "list", jsonField(pl, "object"))
}

func TestVolumeDiscounts_IncludeCategories(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(volumeDiscountsPath+"/"+SeedVolumeDiscountID, url.Values{"include": {"categories"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	cats := jsonObject(got, "categories")
	require.NotNil(t, cats, "categories should be present with ?include=categories")
	assert.Equal(t, "list", jsonField(cats, "object"))
}

func TestVolumeDiscounts_IncludeAttributes(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(volumeDiscountsPath+"/"+SeedVolumeDiscountID, url.Values{"include": {"attributes"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	attrs := jsonObject(got, "attributes")
	require.NotNil(t, attrs, "attributes should be present with ?include=attributes")
	assert.Equal(t, "list", jsonField(attrs, "object"))
}

func TestVolumeDiscounts_IncludeAcceptableUnits(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(volumeDiscountsPath+"/"+SeedVolumeDiscountID, url.Values{"include": {"acceptable_units"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	units := jsonObject(got, "acceptable_units")
	require.NotNil(t, units, "acceptable_units should be present with ?include=acceptable_units")
	assert.Equal(t, "list", jsonField(units, "object"))
}

// ──────────────────────────────────────────────
// VolumeDiscount — List
// ──────────────────────────────────────────────

func TestVolumeDiscounts_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(volumeDiscountsPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assertListContainsID(t, volumeDiscountsPath, nil, SeedVolumeDiscountID)
}

func TestVolumeDiscounts_ListResponseShape(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(volumeDiscountsPath, nil)
	require.NoError(t, err)

	for _, item := range list.Data {
		m := parseJSON(item)
		assert.Equal(t, "volume_discount", jsonField(m, "object"))
		assertIDFormat(t, jsonField(m, "id"), id.QuantityDiscountIDPrefix)
		assert.NotEmpty(t, jsonField(m, "name"))
		assertValidTimestamp(t, jsonField(m, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(m, "updated_at"), "updated_at")
		// Tiers are required (not expandable), so they are always populated.
		assert.NotNil(t, jsonObject(m, "tiers"), "tiers should always be present")
	}
}

func TestVolumeDiscounts_ListCursorPagination(t *testing.T) {
	t.Parallel()
	assertCursorPaginationAdvances(t, volumeDiscountsPath, nil)
}

func TestVolumeDiscounts_ListSearchByName(t *testing.T) {
	t.Parallel()
	created := createVolumeDiscount(t, map[string]any{})
	name := jsonField(created, "name")

	list, _, err := apiClient.GetList(volumeDiscountsPath, url.Values{"q": {name}})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1, "searching for an exact name should match it")
	assert.Equal(t, jsonField(created, "id"), DataItemField(list.Data[0], "id"))
}

func TestVolumeDiscounts_ListWithIncludesHydratesEveryRow(t *testing.T) {
	t.Parallel()
	// Guarantees at least one row carries every scope, so the assertions below are not
	// vacuously true against discounts that happen to have empty scopes.
	createVolumeDiscount(t, map[string]any{
		"customer_group_ids": []string{SeedCustomerGroupID},
		"product_line_ids":   []string{SeedProductLineID},
		"category_ids":       []string{SeedItemCategoryID},
		"attribute_ids":      []string{SeedAttributeID},
		"unit_ids":           []string{SeedUnitID},
	})

	params := url.Values{"include": {"customer_groups", "product_lines", "categories", "attributes", "acceptable_units"}}
	list, _, err := apiClient.GetList(volumeDiscountsPath, params)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	for _, item := range list.Data {
		m := parseJSON(item)
		assert.NotNil(t, jsonObject(m, "customer_groups"), "customer_groups should hydrate on list")
		assert.NotNil(t, jsonObject(m, "product_lines"), "product_lines should hydrate on list")
		assert.NotNil(t, jsonObject(m, "categories"), "categories should hydrate on list")
		assert.NotNil(t, jsonObject(m, "attributes"), "attributes should hydrate on list")
		assert.NotNil(t, jsonObject(m, "acceptable_units"), "acceptable_units should hydrate on list")
	}
}

// ──────────────────────────────────────────────
// VolumeDiscount — Create
// ──────────────────────────────────────────────

// createVolumeDiscount creates a discount with one tier, merging any overrides over the
// defaults, and registers cleanup.
func createVolumeDiscount(t *testing.T, overrides map[string]any) map[string]any {
	t.Helper()
	body := map[string]any{
		"name": uniqueName("e2e-quds"),
		"tiers": []map[string]any{
			{"name": "Tier 1", "discount_percentage": "0.05", "threshold": "100"},
		},
	}
	for k, v := range overrides {
		body[k] = v
	}
	return createAndCleanup(t, volumeDiscountsPath, body)
}

func TestVolumeDiscounts_CreateAllFieldsAndResponseShape(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-quds-allf")
	resp, err := apiClient.PostFull(volumeDiscountsPath+"?include=customer_groups&include=product_lines&include=categories&include=attributes&include=acceptable_units", map[string]any{
		"name": name,
		"tiers": []map[string]any{
			{"name": "Tier 1", "discount_percentage": "0.05", "threshold": "100"},
			{"name": "Tier 2", "discount_percentage": "0.10", "threshold": "500"},
		},
		"customer_group_ids": []string{SeedCustomerGroupID},
		"product_line_ids":   []string{SeedProductLineID},
		"category_ids":       []string{SeedItemCategoryID},
		"attribute_ids":      []string{SeedAttributeID},
		"unit_ids":           []string{SeedUnitID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	got := parseJSON(resp.Body)
	discountID := jsonField(got, "id")
	require.NotEmpty(t, discountID)
	t.Cleanup(func() { apiClient.Delete(volumeDiscountsPath + "/" + discountID) })

	assertIDFormat(t, discountID, id.QuantityDiscountIDPrefix)
	assertCreatedLocation(t, resp.Header, discountID)
	assertObjectField(t, got, "volume_discount")
	assert.Equal(t, name, jsonField(got, "name"))

	assert.Len(t, jsonListData(got, "tiers"), 2, "both tiers should be created")
	assert.Len(t, jsonListData(got, "product_lines"), 1)
	assert.Len(t, jsonListData(got, "categories"), 1)
	assert.Len(t, jsonListData(got, "attributes"), 1)
	assert.Len(t, jsonListData(got, "acceptable_units"), 1)
}

// The legacy Express implementation had its customer-group write commented out, so groups
// selected in the UI were silently dropped and every discount stayed account-wide. This
// pins that the Go endpoint actually persists them.
func TestVolumeDiscounts_CreatePersistsCustomerGroups(t *testing.T) {
	t.Parallel()
	created := createVolumeDiscount(t, map[string]any{
		"customer_group_ids": []string{SeedCustomerGroupID},
	})
	discountID := jsonField(created, "id")

	status, body, err := apiClient.GetListRaw(volumeDiscountsPath+"/"+discountID, url.Values{"include": {"customer_groups"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	groups := jsonListData(parseJSON(body), "customer_groups")
	require.Len(t, groups, 1, "the customer group sent on create must be persisted, not dropped")
	assert.Equal(t, SeedCustomerGroupID, jsonField(groups[0].(map[string]any), "id"))
}

func TestVolumeDiscounts_CreateDuplicateNameConflict(t *testing.T) {
	t.Parallel()
	created := createVolumeDiscount(t, map[string]any{})

	status, body, err := apiClient.Post(volumeDiscountsPath, map[string]any{
		"name":  jsonField(created, "name"),
		"tiers": []map[string]any{{"name": "Tier 1", "discount_percentage": "0.05", "threshold": "100"}},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assertErrorParam(t, errObj, "name")
}

func TestVolumeDiscounts_CreateValidation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		body map[string]any
	}{
		{"missing name", map[string]any{
			"tiers": []map[string]any{{"name": "Tier 1", "discount_percentage": "0.05", "threshold": "100"}},
		}},
		{"missing tiers", map[string]any{"name": uniqueName("e2e-quds-notiers")}},
		{"tier missing name", map[string]any{
			"name":  uniqueName("e2e-quds-badtier"),
			"tiers": []map[string]any{{"discount_percentage": "0.05", "threshold": "100"}},
		}},
		{"tier missing threshold", map[string]any{
			"name":  uniqueName("e2e-quds-nothresh"),
			"tiers": []map[string]any{{"name": "Tier 1", "discount_percentage": "0.05"}},
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			status, body, err := apiClient.Post(volumeDiscountsPath, tc.body, newIdempotencyKey())
			require.NoError(t, err)
			assert.True(t, status == 400 || status == 422,
				"%s should be rejected, got %d: %s", tc.name, status, string(body))
		})
	}
}

func TestVolumeDiscounts_CreateIdempotent(t *testing.T) {
	t.Parallel()
	idemKey := newIdempotencyKey()
	body := map[string]any{
		"name":  uniqueName("e2e-quds-idem"),
		"tiers": []map[string]any{{"name": "Tier 1", "discount_percentage": "0.05", "threshold": "100"}},
	}

	status1, body1, err := apiClient.Post(volumeDiscountsPath, body, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	firstID := jsonField(parseJSON(body1), "id")
	t.Cleanup(func() { apiClient.Delete(volumeDiscountsPath + "/" + firstID) })

	status2, body2, err := apiClient.Post(volumeDiscountsPath, body, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, firstID, jsonField(parseJSON(body2), "id"),
		"replaying the key must not create a second discount, which would also trip the name conflict")
}

// ──────────────────────────────────────────────
// VolumeDiscount — Update
// ──────────────────────────────────────────────

// Every scope list is gated behind its own has_* flag. With the flag false the list is
// ignored entirely, which is what lets a caller PATCH the name without wiping its scopes.
func TestVolumeDiscounts_UpdateWithoutFlagsPreservesScopes(t *testing.T) {
	t.Parallel()
	created := createVolumeDiscount(t, map[string]any{
		"customer_group_ids": []string{SeedCustomerGroupID},
		"product_line_ids":   []string{SeedProductLineID},
		"unit_ids":           []string{SeedUnitID},
	})
	discountID := jsonField(created, "id")

	newName := uniqueName("e2e-quds-renamed")
	status, body, err := apiClient.Patch(volumeDiscountsPath+"/"+discountID+"?include=customer_groups&include=product_lines&include=acceptable_units", map[string]any{
		"name": newName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	updated := parseJSON(body)
	assert.Equal(t, newName, jsonField(updated, "name"))
	assert.Len(t, jsonListData(updated, "customer_groups"), 1, "customer_groups survive a flagless PATCH")
	assert.Len(t, jsonListData(updated, "product_lines"), 1, "product_lines survive a flagless PATCH")
	assert.Len(t, jsonListData(updated, "acceptable_units"), 1, "acceptable_units survive a flagless PATCH")
	assert.Len(t, jsonListData(updated, "tiers"), 1, "tiers survive a flagless PATCH")
}

func TestVolumeDiscounts_UpdateWithFlagReplacesScope(t *testing.T) {
	t.Parallel()
	created := createVolumeDiscount(t, map[string]any{
		"product_line_ids": []string{SeedProductLineID},
	})
	discountID := jsonField(created, "id")

	status, body, err := apiClient.Patch(volumeDiscountsPath+"/"+discountID+"?include=product_lines", map[string]any{
		"has_product_lines": true,
		"product_line_ids":  []string{},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	assert.Empty(t, jsonListData(parseJSON(body), "product_lines"),
		"has_product_lines with an empty list clears the scope")
}

// Tiers upsert: entries carrying an id are updated, new ones created, and any existing tier
// absent from the list is deleted. The legacy implementation only upserted, so removing a
// tier in the UI silently kept it.
func TestVolumeDiscounts_UpdateTiersReplacesSet(t *testing.T) {
	t.Parallel()
	created := createVolumeDiscount(t, map[string]any{
		"tiers": []map[string]any{
			{"name": "Tier 1", "discount_percentage": "0.05", "threshold": "100"},
			{"name": "Tier 2", "discount_percentage": "0.10", "threshold": "500"},
		},
	})
	discountID := jsonField(created, "id")

	tiers := jsonListData(created, "tiers")
	require.Len(t, tiers, 2)
	keptTierID := jsonField(tiers[0].(map[string]any), "id")

	// Send only the first tier back, renamed: the second must be deleted.
	status, body, err := apiClient.Patch(volumeDiscountsPath+"/"+discountID, map[string]any{
		"has_tiers": true,
		"tiers": []map[string]any{
			{"id": keptTierID, "name": "Tier 1 renamed", "discount_percentage": "0.07", "threshold": "150"},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	remaining := jsonListData(parseJSON(body), "tiers")
	require.Len(t, remaining, 1, "a tier omitted from the list is deleted")
	kept := remaining[0].(map[string]any)
	assert.Equal(t, keptTierID, jsonField(kept, "id"), "the tier that was sent keeps its id")
	assert.Equal(t, "Tier 1 renamed", jsonField(kept, "name"))
}

func TestVolumeDiscounts_UpdateDuplicateNameConflict(t *testing.T) {
	t.Parallel()
	first := createVolumeDiscount(t, map[string]any{})
	second := createVolumeDiscount(t, map[string]any{})

	status, body, err := apiClient.Patch(volumeDiscountsPath+"/"+jsonField(second, "id"), map[string]any{
		"name": jsonField(first, "name"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "", "invalid_request_error"), "name")
}

func TestVolumeDiscounts_UpdateNonexistentReturns404(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Patch(volumeDiscountsPath+"/"+mustGenID(t, id.QuantityDiscountIDPrefix),
		map[string]any{"name": uniqueName("e2e-quds-missing")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

// ──────────────────────────────────────────────
// VolumeDiscount — Delete
// ──────────────────────────────────────────────

func TestVolumeDiscounts_DeleteRemovesDiscount(t *testing.T) {
	t.Parallel()
	created := createVolumeDiscount(t, map[string]any{
		"customer_group_ids": []string{SeedCustomerGroupID},
		"product_line_ids":   []string{SeedProductLineID},
	})
	discountID := jsonField(created, "id")

	status, body, err := apiClient.Delete(volumeDiscountsPath + "/" + discountID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	getStatus, getBody, err := apiClient.GetListRaw(volumeDiscountsPath+"/"+discountID, nil)
	require.NoError(t, err)
	requireStatus(t, 404, getStatus, getBody)
}

func TestVolumeDiscounts_DeleteNonexistentReturns404(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Delete(volumeDiscountsPath + "/" + mustGenID(t, id.QuantityDiscountIDPrefix))
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

func TestVolumeDiscounts_RetrieveNonexistentReturns404(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(volumeDiscountsPath+"/"+mustGenID(t, id.QuantityDiscountIDPrefix), nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}
