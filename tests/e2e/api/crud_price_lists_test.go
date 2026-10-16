//go:build e2e

package api_test

import (
	"net/url"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/shared/id"
)

const priceListsPath = "/v1/sales/price-lists"

func priceListRulesPath(listID string) string { return priceListsPath + "/" + listID + "/rules" }

// mirrorPricingOnce makes sure the seeded account prices from its price lists before a test pins a quote: creating
// a list mirrors the account's legacy prices first, whether or not the startup backfill has reached it yet.
var mirrorPricingOnce sync.Once

func mirrorPricing(t *testing.T) {
	t.Helper()
	mirrorPricingOnce.Do(func() {
		status, body, err := apiClient.Post(priceListsPath, map[string]any{"name": uniqueName("e2e-mirror")}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)
		_, _, _ = apiClient.Delete(priceListsPath + "/" + jsonField(parseJSON(body), "id"))
	})
}

func createPriceList(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	if body == nil {
		body = map[string]any{}
	}
	if _, ok := body["name"]; !ok {
		body["name"] = uniqueName("e2e-prls")
	}
	return createAndCleanup(t, priceListsPath, body)
}

func createPriceListRule(t *testing.T, listID string, body map[string]any) map[string]any {
	t.Helper()
	status, respBody, err := apiClient.Post(priceListRulesPath(listID), body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)
	created := parseJSON(respBody)
	ruleID := jsonField(created, "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(priceListRulesPath(listID) + "/" + ruleID) })
	return created
}

func percentOfBase(pct string) map[string]any {
	return map[string]any{"type": "percent_of_base", "percent_of_base": pct}
}

func fixedPrice(amount, perUnitID string) map[string]any {
	return map[string]any{"type": "fixed", "unit_price": map[string]any{"amount": amount, "currency": "usd", "per_unit_id": perUnitID}}
}

// createPricedCustomer creates a customer in groupID with the given lists assigned directly, and removes it (and its
// assignments) afterwards.
func createPricedCustomer(t *testing.T, groupID string, listIDs ...string) string {
	t.Helper()
	body := validCustomerBody(uniqueName("e2e-prls-cust"))
	body["group_id"] = groupID
	body["price_list_ids"] = listIDs
	return jsonField(createAndCleanup(t, customersPath, body), "id")
}

func quoteOne(t *testing.T, customerID, productID, qty, unitID string) float64 {
	t.Helper()
	status, body, err := apiClient.Post(salesOrderPriceQuotePath, map[string]any{
		"buyer_account_id": customerID,
		"lines":            []map[string]any{quoteLine(productID, qty, unitID)},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return quoteUnitPriceAt(t, body, 0)
}

// --- Price lists: CRUD ---

func TestPriceLists_CRUD(t *testing.T) {
	t.Parallel()

	resp, err := apiClient.PostFull(priceListsPath, map[string]any{"name": uniqueName("e2e-prls-crud")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	created := parseJSON(resp.Body)
	listID := jsonField(created, "id")
	assertIDFormat(t, listID, id.PriceListIDPrefix)
	assertCreatedLocation(t, resp.Header, listID)

	status, body, err := apiClient.GetListRaw(priceListsPath+"/"+listID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, listID, jsonField(parseJSON(body), "id"))

	status, body, err = apiClient.Patch(priceListsPath+"/"+listID, map[string]any{"name": "Renamed"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, "Renamed", jsonField(parseJSON(body), "name"))

	status, body, err = apiClient.Delete(priceListsPath + "/" + listID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assertDeletedStub(t, body, listID, "price_list")

	status, body, err = apiClient.GetListRaw(priceListsPath+"/"+listID, nil)
	require.NoError(t, err)
	requireStatus(t, 410, status, body)

	status, body, err = apiClient.Delete(priceListsPath + "/" + listID)
	require.NoError(t, err)
	requireStatus(t, 410, status, body)
}

func TestPriceLists_CreateAllFieldsAndDefaults(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-prls-all")
	got := createPriceList(t, map[string]any{"name": name, "description": "Carton pricing", "status": "archived", "currency": "usd"})
	assertObjectField(t, got, "price_list")
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, "Carton pricing", jsonField(got, "description"))
	assert.Equal(t, "archived", jsonField(got, "status"))
	assert.Equal(t, "usd", jsonField(got, "currency"))
	assertNilField(t, got, "rules")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	defaults := createPriceList(t, map[string]any{"name": uniqueName("e2e-prls-def")})
	assert.Equal(t, "active", jsonField(defaults, "status"), "status defaults to active")
	assert.Equal(t, "usd", jsonField(defaults, "currency"), "currency defaults to usd")
	assertNilField(t, defaults, "description")

	status, body, err := apiClient.Patch(priceListsPath+"/"+jsonField(got, "id"), map[string]any{"description": nil, "status": "active"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	updated := parseJSON(body)
	assertNilField(t, updated, "description")
	assert.Equal(t, "active", jsonField(updated, "status"))
}

func TestPriceLists_CreateValidation(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(priceListsPath, map[string]any{"name": uniqueName("e2e-prls-eur"), "currency": "eur"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "currency_not_supported", "invalid_request_error"), "currency")

	status, body, err = apiClient.Post(priceListsPath, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)

	status, body, err = apiClient.Post(priceListsPath, map[string]any{"name": uniqueName("e2e-prls-st"), "status": "inactive"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
}

func TestPriceLists_CreateIdempotent(t *testing.T) {
	t.Parallel()
	key := newIdempotencyKey()
	body := map[string]any{"name": uniqueName("e2e-prls-idem")}

	status1, body1, err := apiClient.Post(priceListsPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	firstID := jsonField(parseJSON(body1), "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(priceListsPath + "/" + firstID) })

	status2, body2, err := apiClient.Post(priceListsPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, firstID, jsonField(parseJSON(body2), "id"))
}

func TestPriceLists_ListFiltersAndSearch(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-prls-find")
	active := createPriceList(t, map[string]any{"name": name})
	archived := createPriceList(t, map[string]any{"name": name + "-arch", "status": "archived"})
	activeID, archivedID := jsonField(active, "id"), jsonField(archived, "id")

	ids := func(params url.Values) []string {
		list, _, err := apiClient.GetList(priceListsPath, params)
		require.NoError(t, err)
		var out []string
		for _, raw := range list.Data {
			out = append(out, DataItemField(raw, "id"))
		}
		return out
	}
	found := ids(url.Values{"q": {name}})
	assert.Contains(t, found, activeID)
	assert.Contains(t, found, archivedID)

	found = ids(url.Values{"q": {name}, "statuses[]": {"archived"}})
	assert.NotContains(t, found, activeID)
	assert.Contains(t, found, archivedID)

	customerID := createPricedCustomer(t, SeedCustomerGroupID, activeID)
	found = ids(url.Values{"customer_ids[]": {customerID}})
	assert.Contains(t, found, activeID, "a list assigned to the customer applies to it")
	assert.NotContains(t, found, archivedID)

	assertCursorPaginationAdvances(t, priceListsPath, nil)
}

// --- Rules ---

func TestPriceListRules_CRUDAndShape(t *testing.T) {
	t.Parallel()
	listID := jsonField(createPriceList(t, nil), "id")

	rule := createPriceListRule(t, listID, map[string]any{
		"scope": map[string]any{"product_line_ids": []string{SeedProductLineID}, "category_ids": []string{SeedItemCategoryID}},
		"price": fixedPrice("18.45", SeedUnitID),
	})
	ruleID := jsonField(rule, "id")
	assertIDFormat(t, ruleID, id.PriceListRuleIDPrefix)
	assertObjectField(t, rule, "price_list_rule")
	scope := jsonObject(rule, "scope")
	require.NotNil(t, scope)
	for _, key := range []string{"items", "product_lines", "categories", "attributes"} {
		assertNilField(t, scope, key)
	}
	price := jsonObject(rule, "price")
	assert.Equal(t, "fixed", jsonField(price, "type"))
	unitPrice := jsonObject(price, "unit_price")
	assert.Equal(t, "18.45", jsonField(unitPrice, "amount"))
	assert.Equal(t, "usd", jsonField(unitPrice, "currency"))
	assertFullyPresentedUnit(t, jsonObject(unitPrice, "per_unit"), "per_unit")
	assertNilField(t, price, "percent_of_base")
	assertNilField(t, price, "unit")
	assert.Empty(t, price["tiers"])

	status, body, err := apiClient.GetListRaw(priceListRulesPath(listID)+"/"+ruleID+"?include[]=scope.product_lines&include[]=scope.categories", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	scope = jsonObject(parseJSON(body), "scope")
	lines := jsonListData(scope, "product_lines")
	require.Len(t, lines, 1)
	assert.Equal(t, SeedProductLineID, jsonField(lines[0].(map[string]any), "id"))
	require.Len(t, jsonListData(scope, "categories"), 1)
	assertNilField(t, scope, "items")

	status, body, err = apiClient.Patch(priceListRulesPath(listID)+"/"+ruleID, map[string]any{
		"scope": map[string]any{"category_ids": []string{}},
		"price": percentOfBase("80"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	price = jsonObject(parseJSON(body), "price")
	assert.Equal(t, "percent_of_base", jsonField(price, "type"))
	assert.Equal(t, "80", jsonField(price, "percent_of_base"))
	assertNilField(t, price, "unit_price")

	status, body, err = apiClient.GetListRaw(priceListRulesPath(listID)+"/"+ruleID+"?include[]=scope.categories&include[]=scope.product_lines", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	scope = jsonObject(parseJSON(body), "scope")
	assert.Empty(t, jsonListData(scope, "categories"), "an empty list clears that dimension")
	assert.Len(t, jsonListData(scope, "product_lines"), 1, "an omitted dimension is unchanged")

	status, body, err = apiClient.Delete(priceListRulesPath(listID) + "/" + ruleID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assertDeletedStub(t, body, ruleID, "price_list_rule")

	status, body, err = apiClient.GetListRaw(priceListRulesPath(listID)+"/"+ruleID, nil)
	require.NoError(t, err)
	requireStatus(t, 410, status, body)
}

func TestPriceListRules_TieredShape(t *testing.T) {
	t.Parallel()
	listID := jsonField(createPriceList(t, nil), "id")

	rule := createPriceListRule(t, listID, map[string]any{
		"price": map[string]any{"type": "tiered", "unit_id": seedDozenUnitID, "tiers": []map[string]any{
			{"min_quantity": "10", "unit_price": map[string]any{"amount": "7", "currency": "usd", "per_unit_id": SeedUnitID}},
			{"min_quantity": "0", "percent_of_base": "100"},
		}},
	})
	price := jsonObject(rule, "price")
	assert.Equal(t, "tiered", jsonField(price, "type"))
	assertFullyPresentedUnit(t, jsonObject(price, "unit"), "unit")
	tiers, _ := price["tiers"].([]any)
	require.Len(t, tiers, 2)
	assert.Equal(t, "0", jsonField(tiers[0].(map[string]any), "min_quantity"), "tiers come back lowest first")
	assert.Equal(t, "100", jsonField(tiers[0].(map[string]any), "percent_of_base"))
	assertNilField(t, tiers[0].(map[string]any), "unit_price")
	assert.Equal(t, "7", jsonField(jsonObject(tiers[1].(map[string]any), "unit_price"), "amount"))
}

func TestPriceListRules_Validation(t *testing.T) {
	t.Parallel()
	listID := jsonField(createPriceList(t, nil), "id")

	cases := []struct {
		name  string
		body  map[string]any
		param string
		code  string
	}{
		{"fixed without unit_price", map[string]any{"price": map[string]any{"type": "fixed"}}, "price.unit_price", ""},
		{"percent of zero", map[string]any{"price": percentOfBase("0")}, "price.percent_of_base", ""},
		{"tiered without tiers", map[string]any{"price": map[string]any{"type": "tiered", "unit_id": SeedUnitID}}, "price.tiers", ""},
		{"duplicate tier", map[string]any{"price": map[string]any{"type": "tiered", "unit_id": SeedUnitID, "tiers": []map[string]any{
			{"min_quantity": "1", "percent_of_base": "90"}, {"min_quantity": "1.0", "percent_of_base": "80"},
		}}}, "price.tiers", ""},
		{"other currency", map[string]any{"price": map[string]any{"type": "fixed", "unit_price": map[string]any{"amount": "1", "currency": "eur", "per_unit_id": SeedUnitID}}}, "price.unit_price.currency", "currency_not_supported"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			status, body, err := apiClient.Post(priceListRulesPath(listID), tc.body, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 422, status, body)
			assertErrorParam(t, requireErrorResponse(t, body, tc.code, "invalid_request_error"), tc.param)
		})
	}

	status, body, err := apiClient.Post(priceListRulesPath(listID), map[string]any{
		"scope": map[string]any{"item_ids": []string{mustGenID(t, id.ItemIDPrefix)}},
		"price": percentOfBase("90"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "resource_not_found", "invalid_request_error"), "scope.item_ids")
}

func TestPriceListRules_ListFiltersAndEmbeddedRules(t *testing.T) {
	t.Parallel()
	listID := jsonField(createPriceList(t, nil), "id")
	lineRule := createPriceListRule(t, listID, map[string]any{"scope": map[string]any{"product_line_ids": []string{SeedProductLineID}}, "price": percentOfBase("95")})
	anyRule := createPriceListRule(t, listID, map[string]any{"price": percentOfBase("99")})

	list, _, err := apiClient.GetList(priceListRulesPath(listID), url.Values{"product_line_ids[]": {SeedProductLineID}})
	require.NoError(t, err)
	require.Len(t, list.Data, 1)
	assert.Equal(t, jsonField(lineRule, "id"), DataItemField(list.Data[0], "id"))

	status, body, err := apiClient.GetListRaw(priceListsPath+"/"+listID+"?include[]=rules", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	rules := jsonListData(parseJSON(body), "rules")
	require.Len(t, rules, 2)
	assert.Equal(t, jsonField(anyRule, "id"), jsonField(rules[0].(map[string]any), "id"), "rules are newest first")
}

// A list's embedded rules hold the first 10, and next_page_url continues at the rules list.
func TestPriceListRules_EmbeddedListPages(t *testing.T) {
	t.Parallel()
	listID := jsonField(createPriceList(t, nil), "id")
	for i := 0; i < 11; i++ {
		createPriceListRule(t, listID, map[string]any{"price": percentOfBase("99")})
	}
	status, body, err := apiClient.GetListRaw(priceListsPath+"/"+listID+"?include[]=rules", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	rules := jsonObject(parseJSON(body), "rules")
	assert.Len(t, jsonListData(parseJSON(body), "rules"), 10)
	next := jsonField(jsonObject(rules, "page_info"), "next_page_url")
	require.NotEmpty(t, next)
	assert.Contains(t, next, priceListRulesPath(listID))
}

// --- Assignment, deletion and in-use rules ---

func TestPriceLists_DeleteBlockedWhileAssigned(t *testing.T) {
	t.Parallel()
	listID := jsonField(createPriceList(t, nil), "id")
	customerID := createPricedCustomer(t, SeedCustomerGroupID, listID)

	status, body, err := apiClient.GetListRaw(customersPath+"/"+customerID+"?include[]=price_lists", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assigned := jsonListData(parseJSON(body), "price_lists")
	require.Len(t, assigned, 1)
	assert.Equal(t, listID, jsonField(assigned[0].(map[string]any), "id"))

	status, body, err = apiClient.GetListRaw(customersPath+"/"+customerID, nil)
	require.NoError(t, err)
	assertNilField(t, parseJSON(body), "price_lists")

	status, body, err = apiClient.Delete(priceListsPath + "/" + listID)
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
	requireErrorResponse(t, body, "resource_in_use", "invalid_request_error")

	status, body, err = apiClient.Patch(customersPath+"/"+customerID, map[string]any{"price_list_ids": []string{}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.Delete(priceListsPath + "/" + listID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

func TestPriceLists_AccountGroupAssignment(t *testing.T) {
	t.Parallel()
	listID := jsonField(createPriceList(t, nil), "id")
	groupID := jsonField(createAndCleanup(t, accountGroupsPath, map[string]any{"name": uniqueName("e2e-prls-grp"), "price_list_ids": []string{listID}}), "id")
	status, body, err := apiClient.GetListRaw(accountGroupsPath+"/"+groupID+"?include[]=price_lists", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	lists := jsonListData(parseJSON(body), "price_lists")
	require.Len(t, lists, 1)
	assert.Equal(t, listID, jsonField(lists[0].(map[string]any), "id"))

	status, body, err = apiClient.Delete(priceListsPath + "/" + listID)
	require.NoError(t, err)
	requireStatus(t, 409, status, body)

	status, body, err = apiClient.Patch(accountGroupsPath+"/"+groupID+"?include[]=price_lists", map[string]any{"price_list_ids": []string{}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Empty(t, jsonListData(parseJSON(body), "price_lists"))
}

func TestPriceLists_ArchivedListCannotBeNewlyAssigned(t *testing.T) {
	t.Parallel()
	listID := jsonField(createPriceList(t, map[string]any{"status": "archived"}), "id")
	body := validCustomerBody(uniqueName("e2e-prls-arch"))
	body["price_list_ids"] = []string{listID}
	status, respBody, err := apiClient.Post(customersPath, forgeCustomerWrite(body), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, respBody)
	assertErrorParam(t, requireErrorResponse(t, respBody, "resource_archived", "invalid_request_error"), "price_list_ids")
}

func TestPriceListRules_BlockScopeDeletes(t *testing.T) {
	t.Parallel()
	listID := jsonField(createPriceList(t, nil), "id")
	category := createAndCleanup(t, itemCategoriesPath, map[string]any{"name": uniqueName("e2e-prls-cat"), "type": "product"})
	categoryID := jsonField(category, "id")
	rule := createPriceListRule(t, listID, map[string]any{"scope": map[string]any{"category_ids": []string{categoryID}}, "price": percentOfBase("90")})

	status, body, err := apiClient.Delete(itemCategoriesPath + "/" + categoryID)
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
	errObj := requireErrorResponse(t, body, "resource_in_use", "invalid_request_error")
	assert.Contains(t, jsonField(errObj, "message"), "price-list rules")

	status, body, err = apiClient.Delete(priceListRulesPath(listID) + "/" + jsonField(rule, "id"))
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	status, body, err = apiClient.Delete(itemCategoriesPath + "/" + categoryID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

// --- Pricing ---

// A fresh customer pays SCK-002's list price (10/pair) until a list applies; then the most specific rule wins, the
// customer's own list beats its group's, and archiving a list stops it applying.
func TestPriceLists_PriceOrderLines(t *testing.T) {
	t.Parallel()
	mirrorPricing(t)

	groupList := jsonField(createPriceList(t, nil), "id")
	createPriceListRule(t, groupList, map[string]any{"scope": map[string]any{"product_line_ids": []string{SeedProductLineID}}, "price": percentOfBase("50")})
	group := createAndCleanup(t, accountGroupsPath, map[string]any{"name": uniqueName("e2e-prls-pgrp"), "price_list_ids": []string{groupList}})

	ownList := jsonField(createPriceList(t, nil), "id")
	customerID := createPricedCustomer(t, jsonField(group, "id"), ownList)

	assert.InDelta(t, 5.0, quoteOne(t, customerID, seedNonContractProductID, "1", SeedUnitID), 0.001, "the group's list applies")

	createPriceListRule(t, ownList, map[string]any{"scope": map[string]any{"product_line_ids": []string{SeedProductLineID}}, "price": percentOfBase("80")})
	assert.InDelta(t, 8.0, quoteOne(t, customerID, seedNonContractProductID, "1", SeedUnitID), 0.001, "the customer's own list beats its group's on an equal scope")

	createPriceListRule(t, groupList, map[string]any{"scope": map[string]any{"category_ids": []string{SeedItemCategoryID}}, "price": fixedPrice("7.25", SeedUnitID)})
	assert.InDelta(t, 7.25, quoteOne(t, customerID, seedNonContractProductID, "1", SeedUnitID), 0.001, "a category scope beats a product line scope")
	assert.InDelta(t, 43.5, quoteOne(t, customerID, seedNonContractProductID, "1", seedDozenUnitID), 0.001, "a fixed price converts to the ordered unit")

	status, body, err := apiClient.Patch(priceListsPath+"/"+groupList, map[string]any{"status": "archived"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.InDelta(t, 8.0, quoteOne(t, customerID, seedNonContractProductID, "1", SeedUnitID), 0.001, "an archived list stops applying")
}

func TestPriceLists_TieredRuleCountsTheOrder(t *testing.T) {
	t.Parallel()
	mirrorPricing(t)

	listID := jsonField(createPriceList(t, nil), "id")
	createPriceListRule(t, listID, map[string]any{
		"scope": map[string]any{"product_line_ids": []string{SeedProductLineID}},
		"price": map[string]any{"type": "tiered", "unit_id": seedDozenUnitID, "tiers": []map[string]any{
			{"min_quantity": "2", "percent_of_base": "90"},
		}},
	})
	customerID := createPricedCustomer(t, SeedCustomerGroupID, listID)

	assert.InDelta(t, 10.0, quoteOne(t, customerID, seedNonContractProductID, "6", SeedUnitID), 0.001, "one dozen reaches no tier")
	assert.InDelta(t, 9.0, quoteOne(t, customerID, seedNonContractProductID, "12", SeedUnitID), 0.001, "two dozen reach the 90% tier")
}

// The internal export accepts a customer and returns a job.
func TestPriceLists_ExportCustomerPrices(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(priceListsPath+"/actions/export-customer-prices", map[string]any{"customer_id": SeedCustomerAccountID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, status, body)
	assert.Equal(t, "async_job", jsonField(parseJSON(body), "object"))
}
