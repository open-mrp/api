//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const setOrderUnitsPath = "/v1/catalog/items/actions/set-order-units"

func changeStockingUnitPath(itemID string) string {
	return itemsPath + "/" + itemID + "/actions/change-stocking-unit"
}

func dozenOrderUnit(set map[string]any) map[string]any {
	ou := map[string]any{"unit_id": seedDozenUnitID}
	for k, v := range set {
		ou[k] = v
	}
	return ou
}

// orderUnitsOf reads an item's included order_units list.
func orderUnitsOf(t *testing.T, item map[string]any) []map[string]any {
	t.Helper()
	list := jsonObject(item, "order_units")
	require.NotNil(t, list, "order_units is included")
	assert.Equal(t, "list", jsonField(list, "object"))
	var out []map[string]any
	for _, raw := range jsonListData(item, "order_units") {
		if m, ok := raw.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

var itemUnitIncludes = []string{"stocking_unit", "default_order_unit", "order_units"}

// runItemUnitsJob posts an item units action, requires the 202 job and follows it to the end.
func runItemUnitsJob(t *testing.T, path string, body map[string]any) map[string]any {
	t.Helper()
	resp, err := apiClient.PostFull(path, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, resp.StatusCode, resp.Body)
	ack := parseJSON(resp.Body)
	assert.Equal(t, "async_job", jsonField(ack, "object"))
	jobID := jsonField(ack, "id")
	require.NotEmpty(t, jobID)
	assert.True(t, strings.HasSuffix(resp.Header.Get("Location"), "/v1/core/async-jobs/"+jobID))
	return pollJobUntilTerminal(t, jobID)
}

func requireUnits422(t *testing.T, status int, body []byte, param string) {
	t.Helper()
	requireStatus(t, 422, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "", "invalid_request_error"), param)
}

// --- Create / Retrieve ---

func TestItemUnits_CreateAllFields(t *testing.T) {
	t.Parallel()
	body := forgeItemBody("product", uniqueName("e2e-iu-all"))
	body["default_order_unit_id"] = seedDozenUnitID
	body["order_units"] = []any{dozenOrderUnit(map[string]any{
		"portal_visibility": "hidden",
		"discount":          map[string]any{"type": "amount_off", "amount_off": map[string]any{"amount": "1.25", "currency": "usd"}},
		"quantity":          "5",
	})}
	created := createForgeItem(t, itemsPath+"?include="+strings.Join(itemUnitIncludes, ","), body)

	for label, got := range map[string]map[string]any{"create": created, "retrieve": getItem(t, jsonField(created, "id"), itemUnitIncludes...)} {
		assertUnitShape(t, jsonObject(got, "stocking_unit"), SeedUnitID, label+".stocking_unit")
		assertUnitShape(t, jsonObject(got, "default_order_unit"), seedDozenUnitID, label+".default_order_unit")
		units := orderUnitsOf(t, got)
		require.Len(t, units, 1, label)
		ou := units[0]
		assertUnitShape(t, jsonObject(ou, "unit"), seedDozenUnitID, label+".order_units[0].unit")
		assert.Equal(t, "hidden", jsonField(ou, "portal_visibility"), label)
		assertDecimalEqual(t, "5", jsonField(ou, "quantity"), label)
		discount := jsonObject(ou, "discount")
		require.NotNil(t, discount, label)
		assert.Equal(t, "amount_off", jsonField(discount, "type"))
		assertNilField(t, discount, "percent_off")
		amountOff := jsonObject(discount, "amount_off")
		assertDecimalEqual(t, "1.25", jsonField(amountOff, "amount"), label)
		assert.Equal(t, "usd", jsonField(amountOff, "currency"), label)
		assert.NotContains(t, ou, "id", "an order unit is a value, not a resource")
		assert.NotContains(t, ou, "object")
	}
}

func TestItemUnits_OmittedDefaults(t *testing.T) {
	t.Parallel()

	t.Run("physical item defaults to its stocking unit and no order units", func(t *testing.T) {
		got := createForgeItem(t, itemsPath+"?include="+strings.Join(itemUnitIncludes, ","), forgeItemBody("part", uniqueName("e2e-iu-omit")))
		assertUnitShape(t, jsonObject(got, "stocking_unit"), SeedUnitID, "stocking_unit")
		assertUnitShape(t, jsonObject(got, "default_order_unit"), SeedUnitID, "default_order_unit")
		assert.Empty(t, orderUnitsOf(t, got))
	})

	t.Run("an order unit defaults to visible with no discount or quantity", func(t *testing.T) {
		body := forgeItemBody("product", uniqueName("e2e-iu-omit-ou"))
		body["order_units"] = []any{dozenOrderUnit(nil)}
		got := createForgeItem(t, itemsPath+"?include=order_units,default_order_unit", body)
		ou := orderUnitsOf(t, got)[0]
		assert.Equal(t, "visible", jsonField(ou, "portal_visibility"))
		assertNilField(t, ou, "discount")
		assertNilField(t, ou, "quantity")
		assertUnitShape(t, jsonObject(got, "default_order_unit"), SeedUnitID, "default_order_unit")
	})

	t.Run("a service has no stocking unit and defaults to its first order unit", func(t *testing.T) {
		body := forgeItemBody("service", uniqueName("e2e-iu-svc"))
		body["order_units"] = []any{map[string]any{"unit_id": dayUnitID}, map[string]any{"unit_id": "hour"}}
		got := createForgeItem(t, itemsPath+"?include="+strings.Join(itemUnitIncludes, ","), body)
		assertNilField(t, got, "stocking_unit")
		assertUnitShape(t, jsonObject(got, "default_order_unit"), dayUnitID, "default_order_unit")
		assert.Len(t, orderUnitsOf(t, got), 2)
	})
}

func TestItemUnits_NullWithoutInclude(t *testing.T) {
	t.Parallel()
	body := forgeItemBody("product", uniqueName("e2e-iu-noinc"))
	body["order_units"] = []any{dozenOrderUnit(nil)}
	created := createForgeItem(t, itemsPath, body)
	for label, got := range map[string]map[string]any{"create": created, "retrieve": getItem(t, jsonField(created, "id"))} {
		for _, k := range itemUnitIncludes {
			assert.Contains(t, got, k, label)
			assertNilField(t, got, k)
		}
	}
	list, _, err := apiClient.GetList(itemsPath, url.Values{"skus": {jsonField(created, "sku")}, "include": {"order_units"}})
	require.NoError(t, err)
	require.Len(t, list.Data, 1)
	listed := parseJSON(list.Data[0])
	assertNilField(t, listed, "stocking_unit")
	assert.Len(t, orderUnitsOf(t, listed), 1, "a list includes order units too")
}

func TestItemUnits_CreateValidation(t *testing.T) {
	t.Parallel()
	with := func(itemType string, set map[string]any) map[string]any {
		body := forgeItemBody(itemType, uniqueName("e2e-iu-val"))
		for k, v := range set {
			if v == nil {
				delete(body, k)
				continue
			}
			body[k] = v
		}
		return body
	}
	cases := []struct {
		name  string
		body  map[string]any
		param string
	}{
		{"physical item without a stocking unit", with("part", map[string]any{"stocking_unit_id": nil}), "stocking_unit_id"},
		{"service with a stocking unit", with("service", map[string]any{"stocking_unit_id": SeedUnitID}), "stocking_unit_id"},
		{"unknown stocking unit", with("part", map[string]any{"stocking_unit_id": "un_00000000000000000000000000"}), "stocking_unit_id"},
		{"stocked in a currency", with("part", map[string]any{"stocking_unit_id": "dollar"}), "stocking_unit_id"},
		{"the stocking unit as an order unit", with("part", map[string]any{"order_units": []any{map[string]any{"unit_id": SeedUnitID}}}), "order_units[0].unit_id"},
		{"an order unit of another dimension", with("part", map[string]any{"order_units": []any{map[string]any{"unit_id": SeedMaterialUnitID}}}), "order_units[0].unit_id"},
		{"an order unit twice", with("part", map[string]any{"order_units": []any{dozenOrderUnit(nil), dozenOrderUnit(nil)}}), "order_units[1].unit_id"},
		{"a default that is not allowed", with("part", map[string]any{"default_order_unit_id": seedDozenUnitID}), "default_order_unit_id"},
		{"a zero quantity", with("part", map[string]any{"order_units": []any{dozenOrderUnit(map[string]any{"quantity": "0"})}}), "order_units[0].quantity"},
		{"a percent over 100", with("part", map[string]any{"order_units": []any{dozenOrderUnit(map[string]any{
			"discount": map[string]any{"type": "percent_off", "percent_off": "120"},
		})}}), "order_units[0].discount.percent_off"},
		{"a service needs an order unit", with("service", map[string]any{"order_units": []any{}, "default_order_unit_id": nil}), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body, err := apiClient.Post(itemsPath, tc.body, newIdempotencyKey())
			require.NoError(t, err)
			if tc.param == "" {
				if status == 201 {
					apiClient.Delete(itemsPath + "/" + jsonField(parseJSON(body), "id"))
				}
				return
			}
			requireUnits422(t, status, body, tc.param)
		})
	}

	t.Run("an unknown discount type is refused", func(t *testing.T) {
		body := with("part", map[string]any{"order_units": []any{dozenOrderUnit(map[string]any{"discount": map[string]any{"type": "bogo"}})}})
		status, resp, err := apiClient.Post(itemsPath, body, newIdempotencyKey())
		require.NoError(t, err)
		requireClientRejection(t, status, resp)
	})
}

// --- Update ---

func TestItemUnits_UpdateReplacesOrderUnits(t *testing.T) {
	t.Parallel()
	body := forgeItemBody("product", uniqueName("e2e-iu-upd"))
	body["order_units"] = []any{dozenOrderUnit(nil)}
	body["default_order_unit_id"] = seedDozenUnitID
	itemID := jsonField(createForgeItem(t, itemsPath, body), "id")

	t.Run("replacing the list and default", func(t *testing.T) {
		patchItem(t, itemID, map[string]any{
			"order_units":           []any{map[string]any{"unit_id": seedEachUnitID, "discount": map[string]any{"type": "percent_off", "percent_off": "10"}}},
			"default_order_unit_id": seedEachUnitID,
		})
		got := getItem(t, itemID, itemUnitIncludes...)
		assertUnitShape(t, jsonObject(got, "default_order_unit"), seedEachUnitID, "default_order_unit")
		units := orderUnitsOf(t, got)
		require.Len(t, units, 1)
		assertUnitShape(t, jsonObject(units[0], "unit"), seedEachUnitID, "order_units[0].unit")
		discount := jsonObject(units[0], "discount")
		assert.Equal(t, "percent_off", jsonField(discount, "type"))
		assertDecimalEqual(t, "10", jsonField(discount, "percent_off"))
		assertNilField(t, discount, "amount_off")
	})

	t.Run("a patch of another field keeps the units", func(t *testing.T) {
		patchItem(t, itemID, map[string]any{"description": "units stay"})
		got := getItem(t, itemID, itemUnitIncludes...)
		assertUnitShape(t, jsonObject(got, "stocking_unit"), SeedUnitID, "stocking_unit")
		assertUnitShape(t, jsonObject(got, "default_order_unit"), seedEachUnitID, "default_order_unit")
		assert.Len(t, orderUnitsOf(t, got), 1)
	})

	t.Run("emptying the list drops the default back to the stocking unit", func(t *testing.T) {
		patchItem(t, itemID, map[string]any{"order_units": []any{}})
		got := getItem(t, itemID, itemUnitIncludes...)
		assert.Empty(t, orderUnitsOf(t, got))
		assertUnitShape(t, jsonObject(got, "default_order_unit"), SeedUnitID, "default_order_unit")
	})

	t.Run("a default outside the list is refused", func(t *testing.T) {
		status, resp, err := apiClient.Patch(itemsPath+"/"+itemID, map[string]any{"default_order_unit_id": seedDozenUnitID}, newIdempotencyKey())
		require.NoError(t, err)
		requireUnits422(t, status, resp, "default_order_unit_id")
	})

	t.Run("the stocking unit can't be patched", func(t *testing.T) {
		status, resp, err := apiClient.Patch(itemsPath+"/"+itemID, map[string]any{"stocking_unit_id": seedEachUnitID}, newIdempotencyKey())
		require.NoError(t, err)
		requireClientRejection(t, status, resp)
	})
}

// --- Set Order Units ---

func TestItemUnits_SetOrderUnits(t *testing.T) {
	t.Parallel()
	first := jsonField(createForgeItem(t, itemsPath, forgeItemBody("product", uniqueName("e2e-iu-set1"))), "id")
	second := jsonField(createForgeItem(t, itemsPath, forgeItemBody("part", uniqueName("e2e-iu-set2"))), "id")
	material := jsonField(createForgeItem(t, itemsPath, forgeItemBody("material", uniqueName("e2e-iu-set3"))), "id")

	job := runItemUnitsJob(t, setOrderUnitsPath, map[string]any{
		"filter":                map[string]any{"item_ids": []string{first, second, material}},
		"order_units":           []any{dozenOrderUnit(map[string]any{"portal_visibility": "hidden"})},
		"default_order_unit_id": seedDozenUnitID,
	})
	require.Equal(t, "completed", jsonField(job, "status"), "%v", job)

	statuses := map[string]string{}
	for _, r := range jobResults(job) {
		statuses[jobResultResourceID(r)] = jsonField(r, "status")
	}
	assert.Equal(t, "updated", statuses[first])
	assert.Equal(t, "updated", statuses[second])
	failed := 0
	for _, r := range jobResults(job) {
		if jsonField(r, "status") == "failed" {
			failed++
		}
	}
	assert.Equal(t, 1, failed, "the material stocked in pounds can't be ordered in dozens: %v", job)

	for _, id := range []string{first, second} {
		got := getItem(t, id, itemUnitIncludes...)
		assertUnitShape(t, jsonObject(got, "default_order_unit"), seedDozenUnitID, id+" default_order_unit")
		units := orderUnitsOf(t, got)
		require.Len(t, units, 1)
		assert.Equal(t, "hidden", jsonField(units[0], "portal_visibility"))
	}
	assert.Empty(t, orderUnitsOf(t, getItem(t, material, "order_units")), "the failed item is unchanged")
}

func TestItemUnits_SetOrderUnitsRejections(t *testing.T) {
	t.Parallel()
	itemID := jsonField(createForgeItem(t, itemsPath, forgeItemBody("part", uniqueName("e2e-iu-setval"))), "id")
	cases := []struct {
		name  string
		body  map[string]any
		param string
	}{
		{"no filter", map[string]any{"filter": map[string]any{}, "order_units": []any{}}, "filter"},
		{"an unknown item", map[string]any{"filter": map[string]any{"item_ids": []string{itemID, "it_00000000000000000000000000"}}, "order_units": []any{}}, "filter.item_ids[1]"},
		{"an unknown unit", map[string]any{"filter": map[string]any{"item_ids": []string{itemID}}, "order_units": []any{map[string]any{"unit_id": "un_00000000000000000000000000"}}}, "order_units[0].unit_id"},
		{"a unit twice", map[string]any{"filter": map[string]any{"item_ids": []string{itemID}}, "order_units": []any{dozenOrderUnit(nil), dozenOrderUnit(nil)}}, "order_units[1].unit_id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body, err := apiClient.Post(setOrderUnitsPath, tc.body, newIdempotencyKey())
			require.NoError(t, err)
			requireUnits422(t, status, body, tc.param)
		})
	}
}

// --- Change Stocking Unit ---

func TestItemUnits_ChangeStockingUnitConvert(t *testing.T) {
	t.Parallel()
	body := forgeItemBody("part", uniqueName("e2e-iu-conv"))
	body["unit_cost"] = map[string]any{"amount": "10", "currency": "usd", "per_unit_id": SeedUnitID}
	body["order_units"] = []any{dozenOrderUnit(nil)}
	itemID := jsonField(createForgeItem(t, itemsPath, body), "id")
	status, raw, err := apiClient.Patch(itemsPath+"/"+itemID+"/inventory", map[string]any{
		"quantity": map[string]any{"value": "3", "unit_id": SeedUnitID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)

	job := runItemUnitsJob(t, changeStockingUnitPath(itemID), map[string]any{"stocking_unit_id": seedEachUnitID, "mode": "convert"})
	require.Equal(t, "completed", jsonField(job, "status"), "%v", job)
	results := jobResults(job)
	require.Len(t, results, 1)
	assert.Equal(t, itemID, jobResultResourceID(results[0]))

	got := getItem(t, itemID, itemUnitIncludes...)
	assertUnitShape(t, jsonObject(got, "stocking_unit"), seedEachUnitID, "stocking_unit")
	assertUnitShape(t, jsonObject(got, "default_order_unit"), seedEachUnitID, "default_order_unit")
	var orderUnitIDs []string
	for _, ou := range orderUnitsOf(t, got) {
		orderUnitIDs = append(orderUnitIDs, jsonField(jsonObject(ou, "unit"), "id"))
	}
	assert.ElementsMatch(t, []string{seedDozenUnitID, SeedUnitID}, orderUnitIDs, "the old stocking unit stays orderable")
	assertItemPrice(t, jsonObject(got, "unit_cost"), "5", seedEachUnitID, "unit_cost restated per each")

	status, raw, err = apiClient.GetListRaw(itemsPath+"/"+itemID+"/inventory", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	onHand := jsonObject(parseJSON(raw), "on_hand")
	assertDecimalEqual(t, "6", jsonField(onHand, "value"), "3 pairs are 6 each")
	assert.Equal(t, seedEachUnitID, jsonField(jsonObject(onHand, "unit"), "id"))
}

func TestItemUnits_ChangeStockingUnitRelabel(t *testing.T) {
	t.Parallel()
	body := forgeItemBody("part", uniqueName("e2e-iu-relabel"))
	body["order_units"] = []any{dozenOrderUnit(nil)}
	itemID := jsonField(createForgeItem(t, itemsPath, body), "id")
	status, raw, err := apiClient.Patch(itemsPath+"/"+itemID+"/inventory", map[string]any{
		"quantity": map[string]any{"value": "4", "unit_id": SeedUnitID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)

	job := runItemUnitsJob(t, changeStockingUnitPath(itemID), map[string]any{"stocking_unit_id": SeedMaterialUnitID, "mode": "relabel"})
	require.Equal(t, "completed", jsonField(job, "status"), "%v", job)
	results := jobResults(job)
	require.Len(t, results, 1)
	subs := jsonListData(results[0], "sub_resources")
	require.Len(t, subs, 1, "the dozen order unit measured another dimension and was removed: %v", results[0])
	sub, _ := subs[0].(map[string]any)
	assert.Equal(t, seedDozenUnitID, jsonField(sub, "id"))
	assert.Equal(t, "unit", jsonField(sub, "type"))

	got := getItem(t, itemID, itemUnitIncludes...)
	assertUnitShape(t, jsonObject(got, "stocking_unit"), SeedMaterialUnitID, "stocking_unit")
	assertUnitShape(t, jsonObject(got, "default_order_unit"), SeedMaterialUnitID, "default_order_unit")
	assert.Empty(t, orderUnitsOf(t, got))

	status, raw, err = apiClient.GetListRaw(itemsPath+"/"+itemID+"/inventory", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	onHand := jsonObject(parseJSON(raw), "on_hand")
	assertDecimalEqual(t, "4", jsonField(onHand, "value"), "relabel keeps the number")
	assert.Equal(t, SeedMaterialUnitID, jsonField(jsonObject(onHand, "unit"), "id"))
}

func TestItemUnits_ChangeStockingUnitRejections(t *testing.T) {
	t.Parallel()
	partID := jsonField(createForgeItem(t, itemsPath, forgeItemBody("part", uniqueName("e2e-iu-csval"))), "id")
	serviceBody := forgeItemBody("service", uniqueName("e2e-iu-csval-svc"))
	serviceBody["order_units"] = []any{map[string]any{"unit_id": dayUnitID}}
	serviceID := jsonField(createForgeItem(t, itemsPath, serviceBody), "id")

	cases := []struct {
		name   string
		itemID string
		body   map[string]any
		param  string
	}{
		{"convert across dimensions", partID, map[string]any{"stocking_unit_id": SeedMaterialUnitID, "mode": "convert"}, "stocking_unit_id"},
		{"the current unit", partID, map[string]any{"stocking_unit_id": SeedUnitID, "mode": "convert"}, "stocking_unit_id"},
		{"an unknown unit", partID, map[string]any{"stocking_unit_id": "un_00000000000000000000000000", "mode": "relabel"}, "stocking_unit_id"},
		{"a service", serviceID, map[string]any{"stocking_unit_id": SeedUnitID, "mode": "relabel"}, "stocking_unit_id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body, err := apiClient.Post(changeStockingUnitPath(tc.itemID), tc.body, newIdempotencyKey())
			require.NoError(t, err)
			requireUnits422(t, status, body, tc.param)
		})
	}

	t.Run("an unknown mode", func(t *testing.T) {
		status, body, err := apiClient.Post(changeStockingUnitPath(partID), map[string]any{"stocking_unit_id": seedEachUnitID, "mode": "scale"}, newIdempotencyKey())
		require.NoError(t, err)
		requireClientRejection(t, status, body)
	})
	t.Run("an unknown item", func(t *testing.T) {
		status, body, err := apiClient.Post(changeStockingUnitPath("it_00000000000000000000000000"), map[string]any{"stocking_unit_id": seedEachUnitID, "mode": "convert"}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 404, status, body)
	})
}

func TestItemUnits_ChangeStockingUnitIdempotent(t *testing.T) {
	t.Parallel()
	itemID := jsonField(createForgeItem(t, itemsPath, forgeItemBody("part", uniqueName("e2e-iu-csidem"))), "id")
	key := newIdempotencyKey()
	body := map[string]any{"stocking_unit_id": seedEachUnitID, "mode": "convert"}
	s1, b1, err := apiClient.Post(changeStockingUnitPath(itemID), body, key)
	require.NoError(t, err)
	requireStatus(t, 202, s1, b1)
	s2, b2, err := apiClient.Post(changeStockingUnitPath(itemID), body, key)
	require.NoError(t, err)
	requireStatus(t, 202, s2, b2)
	assert.Equal(t, jsonField(parseJSON(b1), "id"), jsonField(parseJSON(b2), "id"), "the same key answers the same job")
	pollJobUntilTerminal(t, jsonField(parseJSON(b1), "id"))
}

// --- Pricing ---

// A line in an order unit is priced at the list price converted to that unit (or to the order unit's own quantity of
// stocking units), less the order unit's discount.
func TestItemUnits_OrderUnitPricing(t *testing.T) {
	t.Parallel()
	lockPricingRead(t)
	productID, itemID := newSellableProductIDs(t, "e2e-iu-price")
	patchItem(t, itemID, map[string]any{
		"sales": map[string]any{"unit_price": map[string]any{"amount": "10", "currency": "usd", "per_unit_id": SeedUnitID}},
		"order_units": []any{
			dozenOrderUnit(map[string]any{"discount": map[string]any{"type": "amount_off", "amount_off": map[string]any{"amount": "5", "currency": "usd"}}}),
			map[string]any{"unit_id": seedEachUnitID, "quantity": "4", "discount": map[string]any{"type": "percent_off", "percent_off": "50"}},
		},
	})

	quote := func(unitID string) float64 {
		t.Helper()
		status, body, err := apiClient.Post(salesOrderPriceQuotePath, map[string]any{
			"buyer_account_id": SeedCustomerAccountID,
			"lines":            []map[string]any{quoteLine(productID, "1", unitID)},
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		return quoteUnitPriceAt(t, body, 0)
	}
	assert.InDelta(t, 10.0, quote(SeedUnitID), 0.001, "the stocking unit pays list")
	assert.InDelta(t, 55.0, quote(seedDozenUnitID), 0.001, "a dozen is 6 pairs: 60 less 5")
	assert.InDelta(t, 20.0, quote(seedEachUnitID), 0.001, "this item's 'each' holds 4 pairs: 40 less 50%")
}

// --- Units in use ---

func TestItemUnits_UnitInUseByAnItemCantBeDeleted(t *testing.T) {
	t.Parallel()
	unitID := jsonField(createAndCleanup(t, unitsPath, map[string]any{
		"name": uniqueName("e2e-iu-unit"), "abbreviation": uniqueName("iu"), "dimension": "quantity",
		"ratio_numerator": "3", "ratio_denominator": "1", "offset_numerator": "0", "offset_denominator": "1",
	}), "id")
	body := forgeItemBody("part", uniqueName("e2e-iu-inuse"))
	body["order_units"] = []any{map[string]any{"unit_id": unitID}}
	createForgeItem(t, itemsPath, body)

	status, resp, err := apiClient.Delete(unitsPath + "/" + unitID)
	require.NoError(t, err)
	requireStatus(t, 409, status, resp)
	requireErrorResponse(t, resp, "resource_in_use", "")
}

// --- Version compatibility ---

func TestItemUnits_Preview9(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)

	t.Run("a preview.9 product takes its units from its category's group", func(t *testing.T) {
		_, itemID := newProductItemIDs(t, "e2e-iu-p9")
		status, resp, err := preview.GetListRaw(itemsPath+"/"+itemID, nil)
		require.NoError(t, err)
		requireStatus(t, 200, status, resp)
		got := parseJSON(resp)
		for _, k := range itemUnitIncludes {
			assert.NotContains(t, got, k, "preview.9 items have no %s", k)
		}
		forge := getItem(t, itemID, itemUnitIncludes...)
		assertUnitShape(t, jsonObject(forge, "stocking_unit"), SeedUnitID, "the Socks group's base unit")
		assertUnitShape(t, jsonObject(forge, "default_order_unit"), SeedUnitID, "default_order_unit")
		var orderUnitIDs []string
		for _, ou := range orderUnitsOf(t, forge) {
			orderUnitIDs = append(orderUnitIDs, jsonField(jsonObject(ou, "unit"), "id"))
		}
		assert.Subset(t, orderUnitIDs, []string{seedEachUnitID, seedDozenUnitID}, "the Socks group's other members")
		assert.NotContains(t, orderUnitIDs, SeedUnitID, "the stocking unit is not an order unit")
	})

	t.Run("unit groups answer preview.9 only", func(t *testing.T) {
		route := itemCategoriesPath + "/" + SeedItemCategoryID + "/unit-groups/" + SeedUnitGroupID
		status, body, err := apiClient.Put(route, nil)
		require.NoError(t, err)
		requireStatus(t, 404, status, body)
		status, body, err = preview.GetListRaw("/v1/catalog/unit-groups", nil)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
	})
}
