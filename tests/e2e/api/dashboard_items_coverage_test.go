//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The items and inventory surface the dashboard reads and writes: the catalog item list and record, the stock
// position and its corrections, bulk reconcile, attribute and category changes, the inventories list, and the
// inventory change log.
//
// Every test that moves stock does it to a material it created (yarn, stocked by the pound), and proves a side
// effect through the change log filtered to that item: the log is written in the same transaction as the movement,
// so a count of user_correction rows is a count of corrections that moved stock.

const dashItemsReceiptsAnalyticsPath = "/v1/core/analytics/inventory-receipts"

var dashItemsDashboardIncludes = []string{
	"category", "category.unit_group", "category.unit_group.base_unit", "category.unit_group.associated_units",
	"category.unit_group.associated_units.unit", "category.properties", "unit_value", "unit_cost", "burn_rate",
	"attributes",
}

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

// dashItemsMaterial creates a yarn material with this SKU and returns the catalog item behind it.
func dashItemsMaterial(t *testing.T, sku string) string {
	t.Helper()
	return dashItemsMaterialFrom(t, validMaterialBody(sku))
}

func dashItemsMaterialFrom(t *testing.T, body map[string]any) string {
	t.Helper()
	status, raw, err := apiClient.Post(materialsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	materialID := jsonField(parseJSON(raw), "id")
	require.NotEmpty(t, materialID)
	t.Cleanup(func() { _, _, _ = apiClient.Delete(materialsPath + "/" + materialID) })
	return dashItemsItemBySKU(t, body["sku"].(string))
}

// dashItemsPart creates a part in the socks category and returns the catalog item behind it.
func dashItemsPart(t *testing.T, sku string) string {
	t.Helper()
	status, body, err := apiClient.Post(partsPath, validPartBody(sku), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	partID := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, partID)
	t.Cleanup(func() { _, _, _ = apiClient.Delete(partsPath + "/" + partID) })
	return dashItemsItemBySKU(t, sku)
}

func dashItemsItemBySKU(t *testing.T, sku string) string {
	t.Helper()
	list, status, err := apiClient.GetList(itemsPath, url.Values{"q": {sku}})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	for _, raw := range list.Data {
		if DataItemField(raw, "sku") == sku {
			return DataItemField(raw, "id")
		}
	}
	t.Fatalf("no item carries the SKU %q", sku)
	return ""
}

// dashItemsColorAttribute creates an attribute of the seeded Color property, which the yarn category carries.
func dashItemsColorAttribute(t *testing.T) string {
	t.Helper()
	status, body, err := apiClient.Post(attributesPath(SeedPropertyID), map[string]any{"value": uniqueName("e2e-dashitems-attr")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	attributeID := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, attributeID)
	t.Cleanup(func() { _, _, _ = apiClient.Delete(attributePath(SeedPropertyID, attributeID)) })
	return attributeID
}

// dashItemsCreateAs posts with client and deletes the created row through the same client when the test ends.
func dashItemsCreateAs(t *testing.T, client *Client, path string, body map[string]any) string {
	t.Helper()
	status, raw, err := client.Post(path, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	id := jsonField(parseJSON(raw), "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { _, _, _ = client.Delete(path + "/" + id) })
	return id
}

// dashItemsMaterialCategory creates a material category counted in eaches, so moving yarn into it changes its base unit.
func dashItemsMaterialCategory(t *testing.T, client *Client) string {
	t.Helper()
	unitGroupID := dashItemsCreateAs(t, client, unitGroupsPath, map[string]any{
		"name": uniqueName("e2e-dashitems-ug"), "type": "quantity", "base_unit_id": "each",
	})
	return dashItemsCreateAs(t, client, itemCategoriesPath, map[string]any{
		"name": uniqueName("e2e-dashitems-cat"), "type": "material_category", "unit_group_id": unitGroupID,
	})
}

type dashItemsForeignRefs struct {
	locationID, attributeID, categoryID string
}

// dashItemsTenantBRefs creates a location, an attribute and a material category that tenant B owns.
func dashItemsTenantBRefs(t *testing.T) dashItemsForeignRefs {
	t.Helper()
	tenantB := getTenantBClient()
	propertyID := dashItemsCreateAs(t, tenantB, propertiesPath, map[string]any{"name": uniqueName("e2e-dashitems-bprop")})
	return dashItemsForeignRefs{
		locationID:  dashItemsCreateAs(t, tenantB, locationsPath, map[string]any{"name": uniqueName("e2e-dashitems-bloc"), "type": "building"}),
		attributeID: dashItemsCreateAs(t, tenantB, attributesPath(propertyID), map[string]any{"value": uniqueName("e2e-dashitems-battr")}),
		categoryID:  dashItemsMaterialCategory(t, tenantB),
	}
}

// ---------------------------------------------------------------------------
// Reading back what happened
// ---------------------------------------------------------------------------

func dashItemsPound(value string) map[string]any {
	return map[string]any{"value": value, "unit_id": poundUnitID}
}

func dashItemsPatchInventory(t *testing.T, client *Client, itemID string, body map[string]any) (int, []byte) {
	t.Helper()
	status, raw, err := client.Patch(itemsPath+"/"+itemID+"/inventory", body, newIdempotencyKey())
	require.NoError(t, err)
	return status, raw
}

func dashItemsMustAdjust(t *testing.T, itemID string, body map[string]any) {
	t.Helper()
	status, raw := dashItemsPatchInventory(t, apiClient, itemID, body)
	requireStatus(t, 200, status, raw)
}

func dashItemsLevel(t *testing.T, itemID string) string {
	t.Helper()
	return readInventory(t, itemID).level().String()
}

// dashItemsWalk follows next_page_url from the first page to the last and returns every row.
func dashItemsWalk(t *testing.T, client *Client, path string, params url.Values) []map[string]any {
	t.Helper()
	list, status, err := client.GetList(path, params)
	require.NoError(t, err, "listing %s", path)
	requireStatus(t, 200, status, nil)

	var rows []map[string]any
	for page := 1; ; page++ {
		require.LessOrEqual(t, page, 200, "%s should reach its last page", path)
		for _, raw := range list.Data {
			rows = append(rows, parseJSON(raw))
		}
		if !list.PageInfo.HasNextPage {
			return rows
		}
		require.NotNil(t, list.PageInfo.NextPageURL, "a page with a next page links to it")
		list, status, err = client.GetListFromPageURL(list.PageInfo.NextPageURL)
		require.NoError(t, err, "paging %s", path)
		requireStatus(t, 200, status, nil)
	}
}

// dashItemsCorrections returns the item's user_correction change logs, newest first.
func dashItemsCorrections(t *testing.T, itemID string) []map[string]any {
	t.Helper()
	return dashItemsWalk(t, apiClient, inventoryChangeLogsPath, url.Values{
		"item_ids": {itemID}, "action_types": {"user_correction"}, "limit": {"100"},
	})
}

// dashItemsReceiptGroups reads the item's stock grouped by location, lot and owner.
func dashItemsReceiptGroups(t *testing.T, itemID string) []map[string]any {
	t.Helper()
	status, body, err := apiClient.Put(dashItemsReceiptsAnalyticsPath, map[string]any{"item_ids": []string{itemID}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	var groups []map[string]any
	for _, raw := range jsonArray(parseJSON(body), "data") {
		group, ok := raw.(map[string]any)
		require.True(t, ok, "%s", string(body))
		groups = append(groups, group)
	}
	return groups
}

func dashItemsIDs(rows []map[string]any, idOf func(map[string]any) string) map[string]int {
	seen := map[string]int{}
	for _, row := range rows {
		seen[idOf(row)]++
	}
	return seen
}

func dashItemsRowID(row map[string]any) string { return jsonField(row, "id") }

func dashItemsInventoryRowItemID(row map[string]any) string {
	return jsonField(jsonObject(row, "item"), "id")
}

// dashItemsRequest is a bodiless write sent by a particular caller.
type dashItemsRequest struct {
	name         string
	client       *Client
	method, path string
}

func (r dashItemsRequest) do(t *testing.T) (int, []byte) {
	t.Helper()
	status, body, err := r.client.Do(r.method, r.path, nil, "")
	require.NoError(t, err)
	return status, body
}

// dashItemsAssertUnchanged asserts an item still reads this level and carries exactly this many corrections.
func dashItemsAssertUnchanged(t *testing.T, itemID, level string, corrections int) {
	t.Helper()
	assert.Equal(t, level, dashItemsLevel(t, itemID), "the level must not move")
	assert.Len(t, dashItemsCorrections(t, itemID), corrections, "no further correction may be logged")
}

// ---------------------------------------------------------------------------
// Items — list and retrieve
// ---------------------------------------------------------------------------

// The dashboard's item lists ask for this whole include set, nested unit-group keys included; the record takes the same set.
func TestDashItems_DashboardIncludeSetHydratesEveryKey(t *testing.T) {
	t.Parallel()
	sku := uniqueName("e2e-dashitems-inc")
	itemID := dashItemsMaterial(t, sku)
	include := url.Values{"include": {strings.Join(dashItemsDashboardIncludes, ",")}}

	list, status, err := apiClient.GetList(itemsPath, url.Values{"q": {sku}, "include": include["include"]})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.Len(t, list.Data, 1)

	status, body, err := apiClient.GetListRaw(itemsPath+"/"+itemID, include)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	for where, item := range map[string]map[string]any{"list": parseJSON(list.Data[0]), "retrieve": parseJSON(body)} {
		assert.Equal(t, itemID, jsonField(item, "id"), where)
		assertObjectField(t, item, "item")
		assert.Equal(t, sku, jsonField(item, "sku"), where)
		assert.Equal(t, "material", jsonField(item, "type"), where)

		category := jsonObject(item, "category")
		require.NotNil(t, category, "%s: category", where)
		assert.Equal(t, SeedMaterialCategoryID, jsonField(category, "id"), where)
		require.NotNil(t, jsonObject(category, "properties"), "%s: category.properties", where)

		unitGroup := jsonObject(category, "unit_group")
		require.NotNil(t, unitGroup, "%s: category.unit_group", where)
		assertUnitHydrated(t, jsonObject(unitGroup, "base_unit"), where+": category.unit_group.base_unit")
		members := jsonListData(unitGroup, "associated_units")
		require.NotEmpty(t, members, "%s: category.unit_group.associated_units", where)
		for _, raw := range members {
			member, ok := raw.(map[string]any)
			require.True(t, ok)
			assertUnitHydrated(t, jsonObject(member, "unit"), where+": associated_units[].unit")
		}

		for _, rate := range []string{"unit_value", "unit_cost", "burn_rate"} {
			got := jsonObject(item, rate)
			require.NotNil(t, got, "%s: %s", where, rate)
			assertObjectField(t, got, "rate")
			assertUnitHydrated(t, jsonObject(got, "numerator_unit"), where+": "+rate+".numerator_unit")
			assertUnitHydrated(t, jsonObject(got, "denominator_unit"), where+": "+rate+".denominator_unit")
		}
		attributes := jsonObject(item, "attributes")
		require.NotNil(t, attributes, "%s: attributes", where)
		assert.Empty(t, jsonArray(attributes, "data"), "%s: a new item carries no attributes", where)
	}
}

func TestDashItems_ListFiltersByCreatedWindow(t *testing.T) {
	t.Parallel()
	sku := uniqueName("e2e-dashitems-win")
	itemID := dashItemsMaterial(t, sku)

	status, body, err := apiClient.GetListRaw(itemsPath+"/"+itemID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	createdAt, err := time.Parse(time.RFC3339, jsonField(parseJSON(body), "created_at"))
	require.NoError(t, err)

	window := func(start, end time.Time) []string {
		return listIDs(t, itemsPath, url.Values{
			"q": {sku}, "starts_at": {start.Format(time.RFC3339)}, "ends_at": {end.Format(time.RFC3339)},
		})
	}
	assert.Equal(t, []string{itemID}, window(createdAt.Add(-time.Minute), createdAt.Add(time.Minute)),
		"a window around the creation time holds the item")
	assert.Empty(t, window(createdAt.Add(-2*time.Hour), createdAt.Add(-time.Hour)), "a window closing before it leaves it out")
	assert.Empty(t, window(createdAt.Add(time.Hour), createdAt.Add(2*time.Hour)), "a window opening after it leaves it out")
}

// Values within types are an OR; the filter combines with q as an AND.
func TestDashItems_ListTypesUnionTheirValues(t *testing.T) {
	t.Parallel()
	token := searchToken("dashitemstypes")
	materialID := dashItemsMaterial(t, token+"-mat")
	partID := dashItemsPart(t, token+"-part")

	assert.ElementsMatch(t, []string{materialID, partID},
		listIDs(t, itemsPath, url.Values{"q": {token}, "types": {"material", "part"}}))
	assert.Equal(t, []string{partID}, listIDs(t, itemsPath, url.Values{"q": {token}, "types": {"part"}}))
	assert.Empty(t, listIDs(t, itemsPath, url.Values{"q": {token}, "types": {"product"}}))
}

// types has three documented values; anything else must be refused rather than silently match nothing.
func TestDashItems_ListRejectsAnUnknownType(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(itemsPath, url.Values{"types": {"bogus_e2e_type"}})
	require.NoError(t, err)
	require.Less(t, status, 500, string(body))
	assert.Equal(t, 400, status, "an unknown item type is a client error: %s", string(body))
}

func TestDashItems_ListRejectsAMalformedDate(t *testing.T) {
	t.Parallel()
	for _, param := range []string{"starts_at", "ends_at"} {
		status, body, err := apiClient.GetListRaw(itemsPath, url.Values{param: {"2026-13-01"}})
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	}
}

// ---------------------------------------------------------------------------
// The export: items, inventories and categories, each paged to the end
// ---------------------------------------------------------------------------

// The dashboard builds its item and inventory exports by walking these lists to the last page with its include
// sets. Every page must carry the includes forward, and no row may repeat.
func TestDashItems_ExportWalksReachEveryRowOnce(t *testing.T) {
	t.Parallel()
	token := searchToken("dashitemsexport")
	var fixtures []string
	for i := 0; i < 3; i++ {
		fixtures = append(fixtures, dashItemsMaterial(t, fmt.Sprintf("%s-%d", token, i)))
	}
	dashItemsMustAdjust(t, fixtures[0], map[string]any{"quantity": dashItemsPound("4.5")})

	t.Run("items", func(t *testing.T) {
		rows := dashItemsWalk(t, apiClient, itemsPath, url.Values{
			"limit": {"7"}, "include": {strings.Join(dashItemsDashboardIncludes, ",")},
		})
		seen := dashItemsIDs(rows, dashItemsRowID)
		for id, n := range seen {
			assert.Equal(t, 1, n, "item %s appears %d times", id, n)
		}
		for _, id := range fixtures {
			assert.Equal(t, 1, seen[id], "fixture %s appears once", id)
		}
		for _, row := range rows {
			assert.NotNil(t, jsonObject(row, "category"), "every page carries the include set: %v", row["id"])
			assert.NotNil(t, jsonObject(row, "unit_cost"), "every page carries the include set: %v", row["id"])
		}
	})

	t.Run("inventories", func(t *testing.T) {
		rows := dashItemsWalk(t, apiClient, inventoriesPath, url.Values{"limit": {"7"}, "include": {"product_line"}})
		seen := dashItemsIDs(rows, dashItemsInventoryRowItemID)
		for id, n := range seen {
			assert.Equal(t, 1, n, "item %s appears %d times", id, n)
		}
		for _, row := range rows {
			assertObjectField(t, row, "inventory_item")
			quantity := jsonObject(row, "quantity")
			require.NotNil(t, quantity, "every row carries a quantity")
			require.NotNil(t, jsonObject(quantity, "unit"), "and its unit")
			switch dashItemsInventoryRowItemID(row) {
			case fixtures[0]:
				assertDecimalEqual(t, "4.5", jsonField(quantity, "value"))
			case fixtures[1], fixtures[2]:
				assertDecimalEqual(t, "0", jsonField(quantity, "value"), "an item never stocked reports zero")
			case SeedItemID:
				// A seed row sits on a late page, so its product line shows the include was carried there.
				assert.Equal(t, SeedProductLineID, jsonField(jsonObject(row, "product_line"), "id"))
			}
		}
		for _, id := range append(fixtures, SeedItemID) {
			assert.Equal(t, 1, seen[id], "%s appears once", id)
		}
	})

	t.Run("categories", func(t *testing.T) {
		rows := dashItemsWalk(t, apiClient, itemCategoriesPath, url.Values{"limit": {"1000"}})
		seen := dashItemsIDs(rows, dashItemsRowID)
		for id, n := range seen {
			assert.Equal(t, 1, n, "category %s appears %d times", id, n)
		}
		assert.Equal(t, 1, seen[SeedMaterialCategoryID], "the fixtures' category is reachable")
	})

	t.Run("scoped one row per page", func(t *testing.T) {
		assertScopedCursorPagination(t, itemsPath, url.Values{"q": {token}}, fixtures)

		rows := dashItemsWalk(t, apiClient, inventoriesPath, url.Values{"q": {token}, "limit": {"1"}})
		var reached []string
		for _, row := range rows {
			reached = append(reached, dashItemsInventoryRowItemID(row))
		}
		assert.ElementsMatch(t, fixtures, reached, "the inventories walk visits each row exactly once")
	})
}

// ---------------------------------------------------------------------------
// Who may read
// ---------------------------------------------------------------------------

func TestDashItems_TenantBCannotReadOrListTenantAItems(t *testing.T) {
	t.Parallel()
	sku := uniqueName("e2e-dashitems-tenant")
	itemID := dashItemsMaterial(t, sku)
	dashItemsMustAdjust(t, itemID, map[string]any{"quantity": dashItemsPound("3")})
	corrections := dashItemsCorrections(t, itemID)
	require.Len(t, corrections, 1)
	changeLogID := jsonField(corrections[0], "id")
	tenantB := getTenantBClient()

	for _, path := range []string{
		itemsPath + "/" + itemID,
		itemsPath + "/" + itemID + "/inventory",
		itemsPath + "/" + itemID + "/trends?trend_type=inventory",
		itemsPath + "/" + itemID + "/lot-default",
		itemsPath + "/" + SeedItemID + "/costs",
		inventoryChangeLogsPath + "/" + changeLogID,
	} {
		status, body, err := tenantB.GetListRaw(path, nil)
		require.NoError(t, err)
		assert.Equal(t, 404, status, "GET %s from another tenant: %s", path, string(body))
	}

	for _, tc := range []struct {
		path   string
		params url.Values
	}{
		{itemsPath, url.Values{"q": {sku}}},
		{itemsPath, url.Values{"category_ids": {SeedMaterialCategoryID}}},
		{inventoriesPath, url.Values{"q": {sku}}},
		{inventoryChangeLogsPath, url.Values{"item_ids": {itemID}}},
	} {
		list, status, err := tenantB.GetList(tc.path, tc.params)
		require.NoError(t, err)
		requireStatus(t, 200, status, nil)
		assert.Empty(t, list.Data, "%s %v must not leak tenant A rows", tc.path, tc.params)
	}
}

// The customer portal is refused the seller's item records, stock and inventory log.
func TestDashItems_CustomerPortalCannotReadSellerItems(t *testing.T) {
	t.Parallel()
	itemID := dashItemsMaterial(t, uniqueName("e2e-dashitems-portal"))
	portal := getCustomerPortalClient()

	for _, path := range []string{
		itemsPath,
		itemsPath + "/" + itemID,
		itemsPath + "/" + itemID + "/inventory",
		itemsPath + "/" + itemID + "/trends?trend_type=inventory",
		itemsPath + "/" + itemID + "/lot-default",
		itemsPath + "/" + SeedItemID + "/costs",
		inventoriesPath,
		inventoryChangeLogsPath,
		inventoryChangeLogsPath + "/" + SeedInventoryChangeLogID,
	} {
		status, body, err := portal.GetListRaw(path, nil)
		require.NoError(t, err)
		assert.Equal(t, 403, status, "portal GET %s: %s", path, string(body))
	}
}

// A seller targeting a customer account through OpenMRP-Account is not an internal user there.
func TestDashItems_SellerTargetingACustomerAccountIsRefused(t *testing.T) {
	t.Parallel()
	itemID := dashItemsMaterial(t, uniqueName("e2e-dashitems-target"))
	targeting := apiClient.WithAccountID(SeedCustomerAccountID)

	for _, path := range []string{itemsPath, inventoriesPath, inventoryChangeLogsPath} {
		status, body, err := targeting.GetListRaw(path, nil)
		require.NoError(t, err)
		assert.Equal(t, 403, status, "GET %s against a customer account: %s", path, string(body))
	}
	status, body := dashItemsPatchInventory(t, targeting, itemID, map[string]any{"quantity": dashItemsPound("1")})
	assert.Equal(t, 403, status, string(body))
	dashItemsAssertUnchanged(t, itemID, "0", 0)
}

// ---------------------------------------------------------------------------
// Item inventory — read
// ---------------------------------------------------------------------------

func TestDashItems_InventoryFiguresAreNullWithoutInclude(t *testing.T) {
	t.Parallel()
	itemID := dashItemsMaterial(t, uniqueName("e2e-dashitems-invnull"))

	status, body, err := apiClient.GetListRaw(itemsPath+"/"+itemID+"/inventory", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	assertObjectField(t, got, "item_inventory")
	for _, field := range []string{"on_hand", "reserved", "available_to_promise", "short"} {
		assertNilField(t, got, field)
	}

	status, body, err = apiClient.GetListRaw(itemsPath+"/"+itemID+"/inventory", url.Values{"include": {"on_hand,bogus"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

// ---------------------------------------------------------------------------
// Item inventory — update
// ---------------------------------------------------------------------------

func TestDashItems_InventoryOperationDefaultsToAdjust(t *testing.T) {
	t.Parallel()
	itemID := dashItemsMaterial(t, uniqueName("e2e-dashitems-default"))

	dashItemsMustAdjust(t, itemID, map[string]any{"quantity": dashItemsPound("4")})
	dashItemsMustAdjust(t, itemID, map[string]any{"quantity": dashItemsPound("4")})

	assert.Equal(t, "8", dashItemsLevel(t, itemID), "two unlabelled corrections add, not replace")
}

// The log records the movement a correction made, not the figure it was set to; a correction that moves nothing logs nothing.
func TestDashItems_InventoryCorrectionsLogTheDelta(t *testing.T) {
	t.Parallel()
	itemID := dashItemsMaterial(t, uniqueName("e2e-dashitems-delta"))

	dashItemsMustAdjust(t, itemID, map[string]any{"quantity": dashItemsPound("40"), "operation": "adjust"})
	dashItemsMustAdjust(t, itemID, map[string]any{"quantity": dashItemsPound("13.5"), "operation": "reconcile"})

	corrections := dashItemsCorrections(t, itemID)
	require.Len(t, corrections, 2)
	for i, want := range []string{"-26.5", "40"} {
		quantity := jsonObject(corrections[i], "quantity")
		assertDecimalEqual(t, want, jsonField(quantity, "value"), "correction %d", i)
		assert.Equal(t, poundUnitID, jsonField(jsonObject(quantity, "unit"), "id"))
		assert.Equal(t, "user_correction", jsonField(corrections[i], "action_type"))
	}

	dashItemsMustAdjust(t, itemID, map[string]any{"quantity": dashItemsPound("13.5"), "operation": "reconcile"})
	dashItemsMustAdjust(t, itemID, map[string]any{"quantity": dashItemsPound("0"), "operation": "adjust"})
	dashItemsAssertUnchanged(t, itemID, "13.5", 2)
}

// Stock held for a customer is owned by the customer and held by the seller, and counts toward the seller's on hand.
func TestDashItems_InventoryCustomerOwnedStockIsHeldOnHand(t *testing.T) {
	t.Parallel()
	itemID := dashItemsMaterial(t, uniqueName("e2e-dashitems-owner"))

	dashItemsMustAdjust(t, itemID, map[string]any{"quantity": dashItemsPound("2"), "customer_id": SeedCustomerAccountID})

	assert.Equal(t, "2", readInventory(t, itemID).onHand.String())
	groups := dashItemsReceiptGroups(t, itemID)
	require.Len(t, groups, 1, "%v", groups)
	assert.Equal(t, SeedCustomerAccountID, jsonField(jsonObject(groups[0], "owner_account"), "id"))
	assert.Equal(t, SeedAccountID, jsonField(jsonObject(groups[0], "holder_account"), "id"))
	assertDecimalEqual(t, "2", jsonField(jsonObject(groups[0], "remaining_quantity"), "value"))
}

// A lot number names a lot that is created once and reused by every later correction carrying the same number.
func TestDashItems_InventoryRecordsTheLocationAndLot(t *testing.T) {
	t.Parallel()
	itemID := dashItemsMaterial(t, uniqueName("e2e-dashitems-lot"))
	lotNumber := "E2E-LOT-" + uuid.New().String()[:8]

	dashItemsMustAdjust(t, itemID, map[string]any{"quantity": dashItemsPound("3"), "location_id": SeedLocationID})
	dashItemsMustAdjust(t, itemID, map[string]any{"quantity": dashItemsPound("4"), "lot_number": lotNumber})
	dashItemsMustAdjust(t, itemID, map[string]any{"quantity": dashItemsPound("5"), "lot_number": lotNumber})

	byLocation, byLot := map[string]string{}, map[string]string{}
	for _, group := range dashItemsReceiptGroups(t, itemID) {
		remaining := jsonField(jsonObject(group, "remaining_quantity"), "value")
		if location := jsonObject(group, "location"); location != nil {
			byLocation[jsonField(location, "id")] = remaining
		}
		if lot := jsonObject(group, "lot"); lot != nil {
			byLot[jsonField(lot, "number")] = remaining
		}
	}
	require.Contains(t, byLocation, SeedLocationID)
	assertDecimalEqual(t, "3", byLocation[SeedLocationID])
	require.Len(t, byLot, 1, "both corrections land in one lot: %v", byLot)
	require.Contains(t, byLot, lotNumber)
	assertDecimalEqual(t, "9", byLot[lotNumber])
}

func TestDashItems_InventoryRejectsInvalidFields(t *testing.T) {
	t.Parallel()
	itemID := dashItemsMaterial(t, uniqueName("e2e-dashitems-inval"))
	quantity := dashItemsPound("1")

	for _, tc := range []struct {
		name, code string
		body       map[string]any
	}{
		{"unknown operation", "parameter_invalid", map[string]any{"quantity": quantity, "operation": "replace"}},
		{"null operation", "invalid_format", map[string]any{"quantity": quantity, "operation": nil}},
		{"null customer", "invalid_format", map[string]any{"quantity": quantity, "customer_id": nil}},
		{"blank location", "invalid_format", map[string]any{"quantity": quantity, "location_id": ""}},
		{"blank lot", "invalid_format", map[string]any{"quantity": quantity, "lot_number": ""}},
		{"lot over 255", "invalid_format", map[string]any{"quantity": quantity, "lot_number": strings.Repeat("L", 256)}},
		{"numeric value", "invalid_format", map[string]any{"quantity": map[string]any{"value": 1, "unit_id": poundUnitID}}},
		{"blank value", "missing_field", map[string]any{"quantity": map[string]any{"value": "", "unit_id": poundUnitID}}},
		{"missing unit", "missing_field", map[string]any{"quantity": map[string]any{"value": "1"}}},
	} {
		status, body := dashItemsPatchInventory(t, apiClient, itemID, tc.body)
		assert.Equal(t, 400, status, "%s: %s", tc.name, string(body))
		if status == 400 {
			requireErrorResponse(t, body, tc.code, "invalid_request_error")
		}
	}
	dashItemsAssertUnchanged(t, itemID, "0", 0)
}

// Holding stock for an account needs a relationship with it; another tenant and a made-up id read the same.
func TestDashItems_InventoryCustomerMustBeOneOfYours(t *testing.T) {
	t.Parallel()
	itemID := dashItemsMaterial(t, uniqueName("e2e-dashitems-cust"))

	for _, customerID := range []string{SeedTenantBAccountID, "ac_01zzzzzzzzzzzzzzzzzzzzzz"} {
		status, body := dashItemsPatchInventory(t, apiClient, itemID, map[string]any{"quantity": dashItemsPound("1"), "customer_id": customerID})
		requireStatus(t, 403, status, body)
		requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
	}
	dashItemsAssertUnchanged(t, itemID, "0", 0)
}

func TestDashItems_InventoryOtherTenantsLocationIsNotFound(t *testing.T) {
	t.Parallel()
	itemID := dashItemsMaterial(t, uniqueName("e2e-dashitems-bloc"))
	refs := dashItemsTenantBRefs(t)

	status, body := dashItemsPatchInventory(t, apiClient, itemID, map[string]any{"quantity": dashItemsPound("1"), "location_id": refs.locationID})
	requireStatus(t, 404, status, body)
	dashItemsAssertUnchanged(t, itemID, "0", 0)
}

// A unit id that names no unit must be refused, not booked against nothing.
func TestDashItems_InventoryRejectsAnUnknownUnit(t *testing.T) {
	t.Parallel()
	itemID := dashItemsMaterial(t, uniqueName("e2e-dashitems-nounit"))

	status, body := dashItemsPatchInventory(t, apiClient, itemID, map[string]any{
		"quantity": map[string]any{"value": "1", "unit_id": "un_01zzzzzzzzzzzzzzzzzzzz"},
	})
	assert.Contains(t, []int{400, 404}, status, "an unknown unit is a client error: %s", string(body))
	dashItemsAssertUnchanged(t, itemID, "0", 0)
}

// Pairs are not a weight. Bulk reconcile refuses a unit outside the item's group, and the single correction must too.
func TestDashItems_InventoryRejectsAUnitOutsideTheItemsGroup(t *testing.T) {
	t.Parallel()
	itemID := dashItemsMaterial(t, uniqueName("e2e-dashitems-pairs"))

	status, body := dashItemsPatchInventory(t, apiClient, itemID, map[string]any{
		"quantity": map[string]any{"value": "1", "unit_id": SeedUnitID},
	})
	assert.Equal(t, 400, status, "a pair of yarn has no weight to convert: %s", string(body))
	dashItemsAssertUnchanged(t, itemID, "0", 0)
}

func TestDashItems_InventoryReplayAdjustsOnce(t *testing.T) {
	t.Parallel()
	itemID := dashItemsMaterial(t, uniqueName("e2e-dashitems-replay"))
	path := itemsPath + "/" + itemID + "/inventory"
	key := newIdempotencyKey()
	body := map[string]any{"quantity": dashItemsPound("3"), "operation": "adjust"}

	first, err := apiClient.PatchFull(path, body, key)
	require.NoError(t, err)
	requireStatus(t, 200, first.StatusCode, first.Body)
	second, err := apiClient.PatchFull(path, body, key)
	require.NoError(t, err)
	requireStatus(t, 200, second.StatusCode, second.Body)

	assert.JSONEq(t, string(first.Body), string(second.Body))
	assert.Equal(t, "true", second.Header.Get("Idempotent-Replayed"))
	dashItemsAssertUnchanged(t, itemID, "3", 1)

	status, raw, err := apiClient.Patch(path, map[string]any{"quantity": dashItemsPound("4"), "operation": "adjust"}, key)
	require.NoError(t, err)
	requireStatus(t, 400, status, raw)
	requireErrorResponse(t, raw, "validation_failed", "idempotency_error")
	dashItemsAssertUnchanged(t, itemID, "3", 1)
}

// A key belongs to one request; sent for a different item it must not answer with the first item's result.
func TestDashItems_InventoryKeyReusedOnAnotherItemIsNotReplayed(t *testing.T) {
	t.Parallel()
	first := dashItemsMaterial(t, uniqueName("e2e-dashitems-key1"))
	second := dashItemsMaterial(t, uniqueName("e2e-dashitems-key2"))
	key := newIdempotencyKey()
	body := map[string]any{"quantity": dashItemsPound("1")}

	resp, err := apiClient.PatchFull(itemsPath+"/"+first+"/inventory", body, key)
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)

	resp, err = apiClient.PatchFull(itemsPath+"/"+second+"/inventory", body, key)
	require.NoError(t, err)
	assert.NotEqual(t, "true", resp.Header.Get("Idempotent-Replayed"), "the second item's request is not a replay of the first's")
	switch resp.StatusCode {
	case 200:
		assert.Equal(t, "1", dashItemsLevel(t, second), "a 200 for the second item must have adjusted it")
	case 400:
		requireErrorResponse(t, resp.Body, "validation_failed", "idempotency_error")
		dashItemsAssertUnchanged(t, second, "0", 0)
	default:
		t.Errorf("unexpected status %d: %s", resp.StatusCode, string(resp.Body))
	}
	assert.Equal(t, "1", dashItemsLevel(t, first))
}

func TestDashItems_InventoryCannotBeAdjustedFromOutside(t *testing.T) {
	t.Parallel()
	itemID := dashItemsMaterial(t, uniqueName("e2e-dashitems-outside"))
	body := map[string]any{"quantity": dashItemsPound("5")}

	status, raw := dashItemsPatchInventory(t, getTenantBClient(), itemID, body)
	assert.Equal(t, 404, status, "another tenant: %s", string(raw))

	status, raw = dashItemsPatchInventory(t, getCustomerPortalClient(), itemID, body)
	assert.Equal(t, 403, status, "the customer portal: %s", string(raw))

	dashItemsAssertUnchanged(t, itemID, "0", 0)
}

// items:read lets a role see items and their stock; changing stock takes items:update.
func TestDashItems_InventoryReadOnlyRoleCannotAdjust(t *testing.T) {
	t.Parallel()
	sku := uniqueName("e2e-dashitems-ro")
	itemID := dashItemsMaterial(t, sku)
	reader := customRoleClient(t, "items:read")

	for path, params := range map[string]url.Values{itemsPath: {"q": {sku}}, itemsPath + "/" + itemID: nil} {
		status, body, err := reader.GetListRaw(path, params)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		assert.Contains(t, string(body), itemID, "GET %s", path)
	}
	status, body, err := reader.GetListRaw(itemsPath+"/"+itemID+"/inventory", url.Values{"include": {"on_hand"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body = dashItemsPatchInventory(t, reader, itemID, map[string]any{"quantity": dashItemsPound("5")})
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
	dashItemsAssertUnchanged(t, itemID, "0", 0)
}

// The dashboard signs people in, and the inventory log's "changed by" filter finds their corrections.
func TestDashItems_InventoryCorrectionIsAttributedToTheSignedInUser(t *testing.T) {
	t.Parallel()
	itemID := dashItemsMaterial(t, uniqueName("e2e-dashitems-user"))

	status, body := dashItemsPatchInventory(t, loginAsSeedUser(t), itemID, map[string]any{"quantity": dashItemsPound("2.5")})
	requireStatus(t, 200, status, body)

	include := "item,responsible_user,responsible_scanning_station"
	list, status, err := apiClient.GetList(inventoryChangeLogsPath, url.Values{
		"item_ids": {itemID}, "changed_by_user_ids": {SeedUserID}, "include": {include},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.Len(t, list.Data, 1)
	listed := parseJSON(list.Data[0])
	logID := jsonField(listed, "id")

	status, body, err = apiClient.GetListRaw(inventoryChangeLogsPath+"/"+logID, url.Values{"include": {include}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	for where, got := range map[string]map[string]any{"list": listed, "retrieve": parseJSON(body)} {
		assert.Equal(t, logID, jsonField(got, "id"), where)
		assertObjectField(t, got, "inventory_change_log")
		assert.Equal(t, "user_correction", jsonField(got, "action_type"), where)
		quantity := jsonObject(got, "quantity")
		assertObjectField(t, quantity, "quantity")
		assertDecimalEqual(t, "2.5", jsonField(quantity, "value"), where)
		assert.Equal(t, poundUnitID, jsonField(jsonObject(quantity, "unit"), "id"), where)
		assert.Equal(t, itemID, jsonField(jsonObject(got, "item"), "id"), where)
		assert.Equal(t, SeedUserID, jsonField(jsonObject(got, "responsible_user"), "id"), where)
		assertNilField(t, got, "responsible_scanning_station")
		assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	}

	status, body, err = apiClient.GetListRaw(inventoryChangeLogsPath+"/"+logID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	plain := parseJSON(body)
	assertNilField(t, plain, "item")
	assertNilField(t, plain, "responsible_user")
}

// ---------------------------------------------------------------------------
// Inventory change logs
// ---------------------------------------------------------------------------

// Creating a material logs its opening level, and each correction adds one more.
func TestDashItems_ChangeLogsPageThroughAnItemsHistory(t *testing.T) {
	t.Parallel()
	itemID := dashItemsMaterial(t, uniqueName("e2e-dashitems-hist"))
	for _, value := range []string{"1", "2", "3"} {
		dashItemsMustAdjust(t, itemID, map[string]any{"quantity": dashItemsPound(value)})
	}

	history := listIDs(t, inventoryChangeLogsPath, url.Values{"item_ids": {itemID}, "limit": {"100"}})
	require.Len(t, history, 4, "the opening entry and three corrections")
	assertScopedCursorPagination(t, inventoryChangeLogsPath, url.Values{"item_ids": {itemID}}, history)

	corrections := dashItemsCorrections(t, itemID)
	require.Len(t, corrections, 3)
	for i, want := range []string{"3", "2", "1"} {
		assert.Equal(t, history[i], jsonField(corrections[i], "id"), "newest first")
		assertDecimalEqual(t, want, jsonField(jsonObject(corrections[i], "quantity"), "value"))
	}
	assert.Empty(t, listIDs(t, inventoryChangeLogsPath, url.Values{"item_ids": {itemID}, "action_types": {"scan"}}))
}

// ---------------------------------------------------------------------------
// Bulk reconcile
// ---------------------------------------------------------------------------

func TestDashItems_BulkReconcileReplayAppliesOnce(t *testing.T) {
	t.Parallel()
	sku, itemID := newReconcilableItem(t)
	key := newIdempotencyKey()
	body := map[string]any{"data": []map[string]any{{"sku": sku, "unit": "lbs", "quantity": "25"}}, "reconcile_type": "addition"}

	first, err := apiClient.PostFull(bulkReconcilePath, body, key)
	require.NoError(t, err)
	requireStatus(t, 200, first.StatusCode, first.Body)
	second, err := apiClient.PostFull(bulkReconcilePath, body, key)
	require.NoError(t, err)
	requireStatus(t, 200, second.StatusCode, second.Body)

	assert.JSONEq(t, string(first.Body), string(second.Body), "a replay answers with the first result")
	assert.Equal(t, "true", second.Header.Get("Idempotent-Replayed"))
	assert.Equal(t, "25", dashItemsLevel(t, itemID), "the replay adds nothing")
	corrections := dashItemsCorrections(t, itemID)
	require.Len(t, corrections, 1, "one correction for one applied request")
	assertDecimalEqual(t, "25", jsonField(jsonObject(corrections[0], "quantity"), "value"))

	other := map[string]any{"data": []map[string]any{{"sku": sku, "unit": "lbs", "quantity": "7"}}, "reconcile_type": "addition"}
	status, raw, err := apiClient.Post(bulkReconcilePath, other, key)
	require.NoError(t, err)
	requireStatus(t, 400, status, raw)
	requireErrorResponse(t, raw, "validation_failed", "idempotency_error")
	dashItemsAssertUnchanged(t, itemID, "25", 1)
}

// Units are matched by abbreviation without regard to case, and an addition may take stock off.
func TestDashItems_BulkReconcileUnitCaseAndNegativeAddition(t *testing.T) {
	t.Parallel()
	sku, itemID := newReconcilableItem(t)

	resp := bulkReconcile(t, []map[string]any{{"sku": sku, "unit": "LBS", "quantity": "5"}}, "addition")
	require.Len(t, jsonListData(resp, "reconciled_items"), 1, "%v", resp)
	resp = bulkReconcile(t, []map[string]any{{"sku": sku, "unit": "lbs", "quantity": "-1.5"}}, "addition")
	require.Len(t, jsonListData(resp, "reconciled_items"), 1, "%v", resp)

	assert.Equal(t, "3.5", dashItemsLevel(t, itemID))
}

func TestDashItems_BulkReconcileRejectsANumericQuantity(t *testing.T) {
	t.Parallel()
	sku, itemID := newReconcilableItem(t)

	status, body, err := apiClient.Post(bulkReconcilePath, map[string]any{
		"data": []map[string]any{{"sku": sku, "unit": "lbs", "quantity": 5}}, "reconcile_type": "force",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	dashItemsAssertUnchanged(t, itemID, "0", 0)
}

// SKUs resolve inside the caller's own account: another tenant naming one of ours finds nothing.
func TestDashItems_BulkReconcileCannotReachAnotherTenantsSKU(t *testing.T) {
	t.Parallel()
	sku, itemID := newReconcilableItem(t)

	status, body, err := getTenantBClient().Post(bulkReconcilePath, map[string]any{
		"data": []map[string]any{{"sku": sku, "unit": "lbs", "quantity": "50"}}, "reconcile_type": "force",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	resp := parseJSON(body)
	assert.Empty(t, jsonListData(resp, "reconciled_items"))
	skipped := jsonListData(resp, "skipped_items")
	require.Len(t, skipped, 1, "%v", resp)
	assert.Equal(t, sku, jsonField(skipped[0].(map[string]any), "sku"))
	dashItemsAssertUnchanged(t, itemID, "0", 0)
}

func TestDashItems_BulkReconcileRefusesThePortalAndAnUpdateOnlyRole(t *testing.T) {
	t.Parallel()
	sku, itemID := newReconcilableItem(t)
	body := map[string]any{"data": []map[string]any{{"sku": sku, "unit": "lbs", "quantity": "50"}}, "reconcile_type": "force"}

	status, raw, err := getCustomerPortalClient().Post(bulkReconcilePath, body, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 403, status, "the customer portal: %s", string(raw))

	// The single correction takes items:update; the bulk one takes items:create.
	updater := customRoleClient(t, "items:read", "items:update")
	status, raw, err = updater.Post(bulkReconcilePath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, raw)
	requireErrorResponse(t, raw, "insufficient_permissions", "invalid_request_error")
	dashItemsAssertUnchanged(t, itemID, "0", 0)

	status, raw = dashItemsPatchInventory(t, updater, itemID, map[string]any{"quantity": dashItemsPound("2")})
	requireStatus(t, 200, status, raw)
	assert.Equal(t, "2", dashItemsLevel(t, itemID))
}

// ---------------------------------------------------------------------------
// Attributes and category
// ---------------------------------------------------------------------------

func TestDashItems_AttributeAssignAndRemoveRoundTrip(t *testing.T) {
	t.Parallel()
	attributeID := dashItemsColorAttribute(t)
	sku := uniqueName("e2e-dashitems-attr")
	itemID := dashItemsMaterial(t, sku)
	path := itemsPath + "/" + itemID + "/attributes/" + attributeID
	withAttributes := url.Values{"include": {"attributes"}}
	attributeIDs := func(item map[string]any) []string {
		var ids []string
		for _, raw := range jsonListData(item, "attributes") {
			ids = append(ids, jsonField(raw.(map[string]any), "id"))
		}
		return ids
	}

	status, body, err := apiClient.PutRaw(path, withAttributes, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, []string{attributeID}, attributeIDs(parseJSON(body)))
	assert.Equal(t, []string{itemID}, listIDs(t, itemsPath, url.Values{"attribute_ids": {attributeID}}))
	assert.Equal(t, []string{itemID}, listIDs(t, itemsPath, url.Values{
		"attribute_ids": {attributeID}, "category_ids": {SeedMaterialCategoryID}, "types": {"material"},
	}), "attribute, category and type filters intersect")
	assert.Empty(t, listIDs(t, itemsPath, url.Values{"attribute_ids": {attributeID}, "types": {"part"}}))

	resp, err := apiClient.DoFull("DELETE", path+"?include=attributes", nil, "")
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	assert.Empty(t, attributeIDs(parseJSON(resp.Body)))
	assert.Empty(t, listIDs(t, itemsPath, url.Values{"attribute_ids": {attributeID}}))

	status, body, err = apiClient.Delete(path)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// Moving yarn into a category counted in eaches moves its rates onto eaches and keeps their figures.
func TestDashItems_ChangeCategoryMovesTheRatesOntoTheNewBaseUnit(t *testing.T) {
	t.Parallel()
	categoryID := dashItemsMaterialCategory(t, apiClient)
	sku := uniqueName("e2e-dashitems-move")
	material := validMaterialBody(sku)
	material["unit_price"] = map[string]any{"value": "1.50", "numerator_unit_id": "dollar", "denominator_unit_id": poundUnitID}
	material["unit_cost"] = map[string]any{"value": "0.75", "numerator_unit_id": "dollar", "denominator_unit_id": poundUnitID}
	itemID := dashItemsMaterialFrom(t, material)
	include := url.Values{"include": {"category,unit_value,unit_cost,burn_rate"}}

	status, body, err := apiClient.GetListRaw(itemsPath+"/"+itemID, include)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	before := parseJSON(body)

	status, body, err = apiClient.PutRaw(itemsPath+"/"+itemID+"/category/"+categoryID, include, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	after := parseJSON(body)

	assert.Equal(t, categoryID, jsonField(jsonObject(after, "category"), "id"))
	for rate, value := range map[string]string{"unit_value": "1.5", "unit_cost": "0.75"} {
		assertDecimalEqual(t, value, jsonField(jsonObject(before, rate), "value"), "%s before the move", rate)
		assertDecimalEqual(t, value, jsonField(jsonObject(after, rate), "value"), "%s keeps its figure", rate)
		assert.Equal(t, poundUnitID, jsonField(jsonObject(jsonObject(before, rate), "denominator_unit"), "id"), "%s was per pound", rate)
		assert.Equal(t, "each", jsonField(jsonObject(jsonObject(after, rate), "denominator_unit"), "id"), "%s is now per each", rate)
		assert.Equal(t, "dollar", jsonField(jsonObject(jsonObject(after, rate), "numerator_unit"), "id"), "%s stays in dollars", rate)
	}
	assert.Equal(t, "each", jsonField(jsonObject(jsonObject(after, "burn_rate"), "numerator_unit"), "id"), "burn rate counts eaches")

	assert.Equal(t, []string{itemID}, listIDs(t, itemsPath, url.Values{"q": {sku}, "category_ids": {categoryID}}))
	assert.Empty(t, listIDs(t, itemsPath, url.Values{"q": {sku}, "category_ids": {SeedMaterialCategoryID}}))

	// Re-assigning the current category succeeds and leaves the item where it is.
	status, body, err = apiClient.PutRaw(itemsPath+"/"+itemID+"/category/"+categoryID, include, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	again := parseJSON(body)
	assert.Equal(t, categoryID, jsonField(jsonObject(again, "category"), "id"))
	assert.Equal(t, jsonField(jsonObject(after, "unit_cost"), "value"), jsonField(jsonObject(again, "unit_cost"), "value"))
}

func TestDashItems_AttributeAndCategoryWritesStayInTheirTenant(t *testing.T) {
	t.Parallel()
	attributeID := dashItemsColorAttribute(t)
	itemID := dashItemsMaterial(t, uniqueName("e2e-dashitems-xattr"))
	refs := dashItemsTenantBRefs(t)
	tenantB := getTenantBClient()
	item := itemsPath + "/" + itemID

	for _, req := range []dashItemsRequest{
		{"tenant B assigns our attribute", tenantB, "PUT", item + "/attributes/" + attributeID},
		{"tenant B removes an attribute", tenantB, "DELETE", item + "/attributes/" + attributeID},
		{"tenant B moves our item", tenantB, "PUT", item + "/category/" + SeedMaterialCategoryID},
		{"we assign tenant B's attribute", apiClient, "PUT", item + "/attributes/" + refs.attributeID},
		{"we remove tenant B's attribute", apiClient, "DELETE", item + "/attributes/" + refs.attributeID},
		{"we move into tenant B's category", apiClient, "PUT", item + "/category/" + refs.categoryID},
	} {
		status, body := req.do(t)
		assert.Equal(t, 404, status, "%s: %s", req.name, string(body))
	}
	assert.Empty(t, listIDs(t, itemsPath, url.Values{"category_ids": {refs.categoryID}}))
	assert.Empty(t, listIDs(t, itemsPath, url.Values{"attribute_ids": {refs.attributeID}}))

	status, body, err := apiClient.GetListRaw(item, url.Values{"include": {"category,attributes"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	assert.Equal(t, SeedMaterialCategoryID, jsonField(jsonObject(got, "category"), "id"), "the item stays where it was")
	assert.Empty(t, jsonListData(got, "attributes"), "and carries no attribute")
}

func TestDashItems_AttributeAndCategoryWritesRefuseReadersAndThePortal(t *testing.T) {
	t.Parallel()
	attributeID := dashItemsColorAttribute(t)
	itemID := dashItemsMaterial(t, uniqueName("e2e-dashitems-roattr"))
	categoryID := dashItemsMaterialCategory(t, apiClient)
	item := itemsPath + "/" + itemID

	// Assigned up front so removing it is a request that would succeed with permission.
	status, body, err := apiClient.Put(item+"/attributes/"+attributeID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	t.Cleanup(func() { _, _, _ = apiClient.Delete(item + "/attributes/" + attributeID) })

	for who, client := range map[string]*Client{"items:read role": customRoleClient(t, "items:read"), "customer portal": getCustomerPortalClient()} {
		for _, req := range []dashItemsRequest{
			{"assign an attribute", client, "PUT", item + "/attributes/" + attributeID},
			{"remove an attribute", client, "DELETE", item + "/attributes/" + attributeID},
			{"change the category", client, "PUT", item + "/category/" + categoryID},
		} {
			status, body := req.do(t)
			assert.Equal(t, 403, status, "%s may not %s: %s", who, req.name, string(body))
		}
	}

	status, body, err = apiClient.GetListRaw(item, url.Values{"include": {"category,attributes"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	assert.Equal(t, SeedMaterialCategoryID, jsonField(jsonObject(got, "category"), "id"))
	attributes := jsonListData(got, "attributes")
	require.Len(t, attributes, 1)
	assert.Equal(t, attributeID, jsonField(attributes[0].(map[string]any), "id"))
}
