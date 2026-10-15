//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SeedPairUnitID is the base unit of the seeded socks unit group: pairs, which is the whole point of the distinction — a doff of sock greige is 60 pairs, not 60 eaches.
const SeedPairUnitID = "un_01seedpair000000000"

func lotDefaultPath(itemID string) string {
	return "/v1/catalog/items/" + itemID + "/lot-default"
}

// categoryWithLot creates a product category counted in pairs, with the given lot when lot is non-nil.
func categoryWithLot(t *testing.T, lot map[string]any) map[string]any {
	t.Helper()
	body := map[string]any{
		"name":          uniqueName("e2e-ic-lot"),
		"type":          "product",
		"unit_group_id": SeedUnitGroupID,
	}
	if lot != nil {
		body["default_lot_size"] = lot
	}
	return createAndCleanup(t, itemCategoriesPath, body)
}

func TestItemCategoryLot_RoundTripsWithItsUnit(t *testing.T) {
	t.Parallel()

	created := categoryWithLot(t, map[string]any{"value": "60", "unit_id": SeedPairUnitID})
	lot := jsonObject(created, "default_lot_size")
	require.NotNil(t, lot, "default_lot_size is a value, present without an include: %v", created)
	assertValueShapeBare(t, lot, "default_lot_size")
	assert.Equal(t, "60", jsonField(lot, "value"))

	// The unit is what makes the number mean something, so it always comes with it.
	unit := jsonObject(lot, "unit")
	require.NotNil(t, unit, "default_lot_size carries its unit")
	assert.Equal(t, SeedPairUnitID, jsonField(unit, "id"))

	status, body, err := apiClient.GetListRaw(itemCategoriesPath+"/"+jsonField(created, "id"), nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	read := jsonObject(parseJSON(body), "default_lot_size")
	require.NotNil(t, read)
	assert.Equal(t, "60", jsonField(read, "value"))
}

func TestItemCategoryLot_OmittedIsNull(t *testing.T) {
	t.Parallel()

	created := categoryWithLot(t, nil)
	assertNilField(t, created, "default_lot_size")
}

// A lot must be positive and counted in the same kind of unit as the category's items.
func TestItemCategoryLot_Validation(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		lot   map[string]any
		param string
	}{
		"zero":           {map[string]any{"value": "0", "unit_id": SeedPairUnitID}, "default_lot_size.value"},
		"other kind":     {map[string]any{"value": "60", "unit_id": SeedMaterialUnitID}, "default_lot_size.unit_id"},
		"unknown unit":   {map[string]any{"value": "60", "unit_id": "un_01definitelynotreal00"}, "default_lot_size.unit_id"},
		"missing unit":   {map[string]any{"value": "60"}, ""},
		"missing amount": {map[string]any{"unit_id": SeedPairUnitID}, ""},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			resp, err := apiClient.PostFull(itemCategoriesPath, map[string]any{
				"name":             uniqueName("e2e-ic-badlot"),
				"type":             "product",
				"unit_group_id":    SeedUnitGroupID,
				"default_lot_size": tc.lot,
			}, newIdempotencyKey())
			require.NoError(t, err)
			require.Less(t, resp.StatusCode, 500, "must not 5xx: %s", string(resp.Body))
			requireStatus(t, 422, resp.StatusCode, resp.Body)
			if tc.param != "" {
				assertErrorParam(t, requireErrorResponse(t, resp.Body, "validation_failed", "invalid_request_error"), tc.param)
			}
		})
	}
}

func TestItemCategoryLot_UpdateAndClear(t *testing.T) {
	t.Parallel()

	created := categoryWithLot(t, nil)
	id := jsonField(created, "id")

	status, body, err := apiClient.Patch(itemCategoriesPath+"/"+id, map[string]any{
		"default_lot_size": map[string]any{"value": "120", "unit_id": SeedPairUnitID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	lot := jsonObject(parseJSON(body), "default_lot_size")
	require.NotNil(t, lot)
	assert.Equal(t, "120", jsonField(lot, "value"))

	// An unrelated edit leaves it alone.
	status, body, err = apiClient.Patch(itemCategoriesPath+"/"+id, map[string]any{"name": uniqueName("e2e-ic-lot-rn")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, "120", jsonField(jsonObject(parseJSON(body), "default_lot_size"), "value"))

	status, body, err = apiClient.Patch(itemCategoriesPath+"/"+id, map[string]any{"default_lot_size": nil}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assertNilField(t, parseJSON(body), "default_lot_size")
}

// An item's lot comes from its category when it sets none of its own.
func TestItemLotDefault_ResolvesFromTheItemsCategory(t *testing.T) {
	t.Parallel()

	category := categoryWithLot(t, map[string]any{"value": "48", "unit_id": SeedPairUnitID})
	body := validProductBody(uniqueName("e2e-ic-lot-sku"))
	body["category_id"] = jsonField(category, "id")
	status, respBody, err := apiClient.Post(productsPath+"?include=item", body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)
	product := parseJSON(respBody)
	t.Cleanup(func() { _, _, _ = apiClient.Delete(productsPath + "/" + jsonField(product, "id")) })
	itemID := jsonField(jsonObject(product, "item"), "id")
	require.NotEmpty(t, itemID)

	status, lotBody, err := apiClient.GetListRaw(lotDefaultPath(itemID), url.Values{"include": {"unit"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, lotBody)

	lot := parseJSON(lotBody)
	assert.Equal(t, "item_lot_default", jsonField(lot, "object"))
	assert.Equal(t, itemID, jsonField(jsonObject(lot, "item"), "id"))
	assert.Equal(t, "item_category", jsonField(lot, "source"))
	assert.Equal(t, float64(48), lot["quantity"])
	assert.Equal(t, jsonField(category, "id"), jsonField(jsonObject(lot, "item_category"), "id"))
	unit := jsonObject(lot, "unit")
	require.NotNil(t, unit)
	assert.Equal(t, SeedPairUnitID, jsonField(unit, "id"))
}

// The unit is expandable, so like every other sub-resource it is null until asked for.
func TestItemLotDefault_UnitNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(lotDefaultPath(SeedItemID), nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assertNilField(t, parseJSON(body), "unit")
}

func TestItemLotDefault_RejectsAnUnknownItem(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(lotDefaultPath("it_01definitelynotreal"), nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 404, status)
}
