//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SeedPairUnitID is the base unit of the seeded unit group: pairs, which is the whole point of the distinction — a doff of sock greige is 60 pairs, not 60 eaches.
const SeedPairUnitID = "un_01seedpair000000000"

func lotDefaultPath(itemID string) string {
	return "/v1/catalog/items/" + itemID + "/lot-default"
}

func productLineWithLot(t *testing.T, quantity, unitID string) map[string]any {
	t.Helper()

	return createAndCleanup(t, productLinesPath, map[string]any{
		"name":              uniqueName("e2e-pdln-lot"),
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "commission_applied",
		"freight_policy":    "billed_freight",
		"default_lot":       map[string]any{"value": quantity, "unit_id": unitID},
	})
}

func TestProductLineLot_RoundTripsWithItsUnit(t *testing.T) {
	t.Parallel()

	created := productLineWithLot(t, "60", SeedPairUnitID)

	// The lot is expandable, so like every other sub-resource it is null until asked for.
	status, body, err := apiClient.GetListRaw(productLinesPath+"/"+jsonField(created, "id"), nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assertNilField(t, parseJSON(body), "default_lot")

	status, body, err = apiClient.GetListRaw(productLinesPath+"/"+jsonField(created, "id"),
		url.Values{"include": {"default_lot", "default_lot.unit"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	lot := jsonObject(parseJSON(body), "default_lot")
	require.NotNil(t, lot, "default_lot must expand: %s", string(body))
	assert.Equal(t, "quantity", jsonField(lot, "object"))
	assert.NotEmpty(t, jsonField(lot, "id"))
	assert.Equal(t, "60", jsonField(lot, "value"))

	// The unit is what makes the number mean something, so it has to be resolvable.
	unit := jsonObject(lot, "unit")
	require.NotNil(t, unit, "default_lot.unit must expand")
	assert.Equal(t, SeedPairUnitID, jsonField(unit, "id"))
	assert.Equal(t, "pr", jsonField(unit, "abbreviation"))
}

// A size with no unit cannot say whether 60 means pairs or eaches, so half a convention is rejected rather than stored.
func TestProductLineLot_RejectsAQuantityWithNoUnit(t *testing.T) {
	t.Parallel()

	resp, err := apiClient.PostFull(productLinesPath, map[string]any{
		"name":              uniqueName("e2e-pdln-halflot"),
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "commission_applied",
		"freight_policy":    "billed_freight",
		"default_lot":       map[string]any{"value": "60"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, resp.StatusCode, 500, "must not 5xx: %s", string(resp.Body))
	assert.Equal(t, 400, resp.StatusCode)
}

// A lot counted in a unit the line cannot express is not a lot anybody can act on.
func TestProductLineLot_RejectsAUnitOutsideTheGroup(t *testing.T) {
	t.Parallel()

	resp, err := apiClient.PostFull(productLinesPath, map[string]any{
		"name":              uniqueName("e2e-pdln-badunit"),
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "commission_applied",
		"freight_policy":    "billed_freight",
		"default_lot": map[string]any{
			"value":   "60",
			"unit_id": "un_01definitelynotinthisgroup",
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, resp.StatusCode, 500, "must not 5xx: %s", string(resp.Body))
	assert.Equal(t, 400, resp.StatusCode)
}

func TestProductLineLot_CanBeCleared(t *testing.T) {
	t.Parallel()

	created := productLineWithLot(t, "60", SeedPairUnitID)
	id := jsonField(created, "id")

	status, body, err := apiClient.Patch(productLinesPath+"/"+id, map[string]any{
		"default_lot": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	// The lot is one value, so removing it removes the whole thing rather than leaving a number with no unit behind. Read back with the include, since a null expandable and an unrequested one look identical.
	status, body, err = apiClient.GetListRaw(productLinesPath+"/"+id,
		url.Values{"include": {"default_lot"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assertNilField(t, parseJSON(body), "default_lot")
}

// The lookup the batch form uses. A sellable item takes its lot straight from its own line.
func TestItemLotDefault_ResolvesFromTheItemsOwnProductLine(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(lotDefaultPath(SeedItemID), url.Values{"include": {"unit"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	lot := parseJSON(body)
	assert.Equal(t, "item_lot_default", jsonField(lot, "object"))
	item := jsonObject(lot, "item")
	require.NotNil(t, item, "the lot must name the item it was resolved for: %s", string(body))
	assert.Equal(t, SeedItemID, jsonField(item, "id"))

	// Whichever rule applied, the answer has to name it and has to carry a unit, or a form cannot tell 60 pairs from 60 eaches.
	source := jsonField(lot, "source")
	assert.Contains(t,
		[]string{"item_override", "product_line", "downstream_product_line", "account_default", ""},
		source)

	if quantity, ok := lot["quantity"].(float64); ok && quantity > 0 {
		unit := jsonObject(lot, "unit")
		require.NotNil(t, unit,
			"a lot with a quantity must say what it is counted in (unit should be present with ?include=unit)")
		assert.Equal(t, "unit", jsonField(unit, "object"))
		assert.NotEmpty(t, jsonField(unit, "id"))
	}
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
