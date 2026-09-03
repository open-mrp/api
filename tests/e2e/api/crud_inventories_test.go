//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// List Inventories: every item alongside the on-hand figure derived from the ledger.
//
// The dashboard's inventory export walks this endpoint to exhaustion and joins it against the
// catalog on item id, so the contract that matters here is that every row carries both an item
// and a quantity — an item the ledger has never touched must still appear, reporting zero, or
// the export silently loses it.

const inventoriesPath = "/v1/operations/inventories"

// inventoryRowForItem finds an item's row in the inventory list, scoping the search by the SKU so
// parallel tests creating items cannot shift the page window.
func inventoryRowForItem(t *testing.T, sku string, params url.Values) map[string]any {
	t.Helper()

	query := url.Values{"q": {sku}}
	for k, vs := range params {
		for _, v := range vs {
			query.Add(k, v)
		}
	}

	list, status, err := apiClient.GetList(inventoriesPath, query)
	require.NoError(t, err)
	require.Less(t, status, 500, "inventories list must not 5xx")
	require.Equal(t, 200, status)
	require.Len(t, list.Data, 1, "exactly one inventory row should match the unique SKU %q", sku)

	return parseJSON(list.Data[0])
}

// --- Response shape ---

func TestInventories_ListResponseShape(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(inventoriesPath, url.Values{"limit": {"3"}})
	require.NoError(t, err)
	require.Less(t, status, 500, "inventories list must not 5xx")
	requireStatus(t, 200, status, nil)
	require.NotEmpty(t, list.Data, "the seeded account has items, so the list cannot be empty")

	for _, raw := range list.Data {
		row := parseJSON(raw)
		assertObjectField(t, row, "inventory_item")

		item := jsonObject(row, "item")
		require.NotNil(t, item, "every row names the item it reports on: %s", string(raw))
		assert.NotEmpty(t, jsonField(item, "id"))
		assertObjectField(t, item, "item")

		quantity := jsonObject(row, "quantity")
		require.NotNil(t, quantity, "every row carries a quantity, even at zero: %s", string(raw))
		assert.NotEmpty(t, jsonField(quantity, "value"), "the on-hand figure is a decimal string")
	}
}

// --- Expandable fields ---

func TestInventories_QuantityUnitIsAlwaysNull(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(inventoriesPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.NotEmpty(t, list.Data)

	quantity := jsonObject(parseJSON(list.Data[0]), "quantity")
	require.NotNil(t, quantity)
	assertNilField(t, quantity, "unit")
}

// The on-hand figure is computed per request and the core service returns only the unit's
// abbreviation and type with it, never an id, so there is nothing to resolve a Unit from. The
// endpoint used to advertise `quantity.unit` anyway and panicked when asked for it; it now says no.
func TestInventories_RejectsTheQuantityUnitInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(inventoriesPath, url.Values{
		"limit":   {"1"},
		"include": {"quantity.unit"},
	})
	require.NoError(t, err)
	require.Less(t, status, 500, "an unresolvable include must be refused, not panic: %s", string(body))
	assert.Equal(t, 400, status, "quantity.unit is not offered on this endpoint: %s", string(body))
}

// The unit still reaches the caller, formatted into display_value, which is the only place it is
// available here.
func TestInventories_QuantityCarriesItsUnitInTheDisplayValue(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(inventoriesPath, url.Values{"q": {SeedItemSKU}})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.NotEmpty(t, list.Data)

	quantity := jsonObject(parseJSON(list.Data[0]), "quantity")
	require.NotNil(t, quantity)
	assertObjectField(t, quantity, "computed_quantity")
	assert.NotEmpty(t, jsonField(quantity, "display_value"),
		"display_value carries the abbreviation the raw value lacks: %v", quantity)
}

// --- Ledger-less items ---

// An item nothing has ever been received or issued against has no ledger rows at all. It still has
// to appear, reporting zero: the export treats an absent row as "never moved" and substitutes a
// zero itself, so a missing row and a zero row must not be the same thing on the wire.
func TestInventories_ItemWithNoLedgerHistoryReportsZero(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-inv-zero")
	createStatus, createBody, err := apiClient.Post(materialsPath, validMaterialBody(sku), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	materialID := jsonField(parseJSON(createBody), "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(materialsPath + "/" + materialID) })

	row := inventoryRowForItem(t, sku, nil)

	quantity := jsonObject(row, "quantity")
	require.NotNil(t, quantity)
	assertDecimalEqual(t, "0", jsonField(quantity, "value"),
		"a brand-new item reports zero rather than being omitted: %v", row)
}

// --- Search ---

func TestInventories_SearchMatchesSKU(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(inventoriesPath, url.Values{"q": {SeedItemSKU}})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.NotEmpty(t, list.Data, "searching a known SKU must match it")

	found := false
	for _, raw := range list.Data {
		if item := jsonObject(parseJSON(raw), "item"); item != nil && jsonField(item, "sku") == SeedItemSKU {
			found = true
			break
		}
	}
	assert.True(t, found, "the seeded SKU %q must appear in its own search results", SeedItemSKU)
}

func TestInventories_SearchNoResults(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(inventoriesPath, url.Values{"q": {"zzz-no-such-inventory-sku-zzz"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	assertEmptyListData(t, list.Data, "a nonsense search returns an empty page, not an error")
}

// --- Pagination ---

// An inventory row is a projection rather than an entity: it carries no id of its own, so the
// cursor is checked against the item each page reports on.
func TestInventories_PaginationAdvances(t *testing.T) {
	t.Parallel()

	page1, status, err := apiClient.GetList(inventoriesPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.Len(t, page1.Data, 1, "the seeded account has more than one item")
	require.True(t, page1.PageInfo.HasNextPage, "one row per page leaves a next page")
	require.NotNil(t, page1.PageInfo.NextPageURL)

	page2, _, err := apiClient.GetListFromPageURL(page1.PageInfo.NextPageURL)
	require.NoError(t, err)
	require.Len(t, page2.Data, 1)

	first := jsonField(jsonObject(parseJSON(page1.Data[0]), "item"), "id")
	second := jsonField(jsonObject(parseJSON(page2.Data[0]), "item"), "id")
	require.NotEmpty(t, first)
	assert.NotEqual(t, first, second, "consecutive pages must report on different items")
}

// --- Validation ---

func TestInventories_RejectsUnknownQueryParam(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(inventoriesPath, url.Values{bogusE2EQueryParam: {"1"}})
	require.NoError(t, err)
	assertUnknownQueryParamRejected(t, inventoriesPath, status, body)
}

func TestInventories_RejectsAnOutOfRangeLimit(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(inventoriesPath, url.Values{"limit": {"100000"}})
	require.NoError(t, err)
	require.Less(t, status, 500, "an out-of-range limit is a client error, not a crash: %s", string(body))
	require.Equal(t, 400, status, "limit is bounded: %s", string(body))
}

func TestInventories_RejectsAMalformedCursor(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(inventoriesPath, url.Values{"cursor": {"not-a-real-cursor"}})
	require.NoError(t, err)
	require.Less(t, status, 500, "a malformed cursor must not 5xx: %s", string(body))
	assert.Equal(t, 400, status, "a malformed cursor is rejected: %s", string(body))
}
