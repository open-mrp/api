//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The item, product, and customer lists choose their page from one table under a per-filter index hint
// and join the rest after, resolving customer, product-line, and price-group filters to IDs first.
// These pin what those filters and the pages either side of a cursor return, since the plan tests only
// measure how much they read.

// pageBack lists params at limit 1, follows next, then previous, and returns the three pages' ids.
func pageBack(t *testing.T, path string, params url.Values) (first, second, back []string) {
	t.Helper()
	params.Set("limit", "1")
	rows, info := listPayments(t, path, params)
	first = rowIDs(rows)
	require.Equal(t, true, info["has_next_page"], "page_info: %v", info)

	next := jsonField(info, "next_page_url")
	status, body, err := apiClient.GetListRawFromPageURL(&next)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	page := parseJSON(body)
	for _, r := range jsonArray(page, "data") {
		second = append(second, jsonField(r.(map[string]any), "id"))
	}

	pageInfo := jsonObject(page, "page_info")
	require.Equal(t, true, pageInfo["has_prev_page"], "page_info: %v", pageInfo)
	prev := jsonField(pageInfo, "previous_page_url")
	status, body, err = apiClient.GetListRawFromPageURL(&prev)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	for _, r := range jsonArray(parseJSON(body), "data") {
		back = append(back, jsonField(r.(map[string]any), "id"))
	}
	return first, second, back
}

func TestItemsList_SearchPagesBackToTheFirstPage(t *testing.T) {
	t.Parallel()
	prefix := uniqueName("e2e-item-back")
	ids := createItemsViaMaterials(t, prefix, 3)

	first, second, back := pageBack(t, itemsPath, url.Values{"q": {prefix}})
	require.Len(t, first, 1)
	require.Len(t, second, 1)
	assert.Contains(t, ids, first[0])
	assert.Contains(t, ids, second[0])
	assert.NotEqual(t, first, second, "the next page moves past the first")
	assert.Equal(t, first, back, "the previous page is the first page again")
}

func TestItemsList_CustomerAndProductLineBothApply(t *testing.T) {
	t.Parallel()
	both, _ := listPayments(t, itemsPath, url.Values{
		"customer_ids": {SeedCustomerAccountID}, "product_line_ids": {SeedProductLineID},
	})
	assert.NotEmpty(t, both, "a line the customer may buy from lists its items")

	none, _ := listPayments(t, itemsPath, url.Values{
		"customer_ids": {SeedCustomerAccountID}, "product_line_ids": {"pdln_00000000000000000000000000"},
	})
	assert.Empty(t, none, "an item must be on a requested line and one the customer may buy from")
}

func TestItemsList_SupplierAndAttributeWithNoMatches(t *testing.T) {
	t.Parallel()
	rows, _ := listPayments(t, itemsPath, url.Values{"supplier_id": {"ac_00000000000000000000000000"}})
	assert.Empty(t, rows, "an unknown supplier supplies nothing")
	rows, _ = listPayments(t, itemsPath, url.Values{"attribute_ids": {"attr_00000000000000000000000000"}})
	assert.Empty(t, rows, "an unknown attribute is on nothing")
}

func TestProductsList_CustomerOrProductLine(t *testing.T) {
	t.Parallel()
	onLine, _ := listPayments(t, productsPath, url.Values{"product_line_ids": {SeedProductLineID}})
	require.NotEmpty(t, onLine)

	// A product is listed when it is on a requested line or on one the customers may buy from.
	either, _ := listPayments(t, productsPath, url.Values{
		"product_line_ids": {SeedProductLineID}, "customer_ids": {"ac_00000000000000000000000000"},
	})
	assert.ElementsMatch(t, rowIDs(onLine), rowIDs(either), "a customer with no lines adds none and removes none")

	none, _ := listPayments(t, productsPath, url.Values{"customer_ids": {"ac_00000000000000000000000000"}})
	assert.Empty(t, none, "a customer with no lines lists nothing")
}

func TestProductsList_SearchPagesBackToTheFirstPage(t *testing.T) {
	t.Parallel()
	prefix := uniqueName("e2e-prod-back")
	var ids []string
	for _, suffix := range []string{"-0", "-1", "-2"} {
		created := createAndCleanup(t, productsPath, validProductBody(prefix+suffix))
		ids = append(ids, jsonField(created, "id"))
	}

	first, second, back := pageBack(t, productsPath, url.Values{"q": {prefix}})
	require.Len(t, first, 1)
	require.Len(t, second, 1)
	assert.Contains(t, ids, first[0])
	assert.Contains(t, ids, second[0])
	assert.NotEqual(t, first, second, "the next page moves past the first")
	assert.Equal(t, first, back, "the previous page is the first page again")
}

func TestCustomersList_PriceGroup(t *testing.T) {
	t.Parallel()
	pg := createAndCleanup(t, accountGroupsPath, map[string]any{
		"name": uniqueName("e2e-cust-list-pg"),
		"type": "pricing_group",
	})
	pgID := jsonField(pg, "id")
	older := jsonField(covSalesCustomersCreate(t, map[string]any{"customer_price_group_ids": []string{pgID}}), "id")
	newer := jsonField(covSalesCustomersCreate(t, map[string]any{"customer_price_group_ids": []string{pgID}}), "id")
	covSalesCustomersCreate(t, nil)

	rows, _ := listPayments(t, customersPath, url.Values{"pricing_group_ids": {pgID}})
	assert.Equal(t, []string{newer, older}, rowIDs(rows), "the group's customers, newest first")

	first, second, back := pageBack(t, customersPath, url.Values{"pricing_group_ids": {pgID}})
	assert.Equal(t, []string{newer}, first)
	assert.Equal(t, []string{older}, second)
	assert.Equal(t, []string{newer}, back, "the previous page is the first page again")

	empty := createAndCleanup(t, accountGroupsPath, map[string]any{
		"name": uniqueName("e2e-cust-list-pg-empty"),
		"type": "pricing_group",
	})
	rows, _ = listPayments(t, customersPath, url.Values{"pricing_group_ids": {jsonField(empty, "id")}})
	assert.Empty(t, rows, "a price group with no customers lists nothing")
}
