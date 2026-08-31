//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Covers the read surface of /v1/operations/inventory-change-logs: the resource shape, every list
// filter, keyset paging, and the export.
//
// The log is account-wide and other tests write to it concurrently, so nothing here asserts a count
// or a list length. The two enriched fixtures are dated into 2099 (0014_e2e_extras.sql), which is
// what makes their position at the head of the list stable; everything else is a containment check.
//
// A filter the server silently ignores still answers 200 with a full page, so every filter case
// pairs a positive match against a value that must narrow the list to nothing.

const inventoryChangeLogsExportPath = inventoryChangeLogsPath + "/actions/export"

// Collects the ids the change-log list returns under the given filters.
func inventoryChangeLogIDsFiltered(t *testing.T, params url.Values) []string {
	t.Helper()
	return listIDs(t, inventoryChangeLogsPath, params)
}

// ──────────────────────────────────────────────
// Resource shape
// ──────────────────────────────────────────────

// Every field on the resource is asserted here, so a field that stops being populated fails loudly
// rather than quietly reading null through the include tests.
func TestInventoryChangeLogs_RetrieveReportsEveryField(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(inventoryChangeLogsPath+"/"+SeedInventoryChangeLogID,
		url.Values{"include": {"item,responsible_user,responsible_scanning_station"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, SeedInventoryChangeLogID, jsonField(got, "id"))
	assertObjectField(t, got, "inventory_change_log")
	assert.Equal(t, "user_action", jsonField(got, "action_type"))
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	// The amount is what the entry is, so it rides inline rather than behind an include.
	quantity := jsonObject(got, "quantity")
	require.NotNil(t, quantity, "quantity is always present: %s", string(body))
	assertObjectField(t, quantity, "quantity")
	assert.NotEmpty(t, jsonField(quantity, "value"), "the quantity reports a value")
	unit := jsonObject(quantity, "unit")
	require.NotNil(t, unit, "the quantity carries its unit: %s", string(body))
	assertObjectField(t, unit, "unit")

	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present with ?include=item")
	assert.Equal(t, SeedInventoryChangeLogItemID, jsonField(item, "id"))
	assert.Equal(t, SeedItemSKU, jsonField(item, "sku"))
	// The include must carry the full item, not a stub built from the change-log join: a stub that
	// drops the item's own base fields reads null here even though the item has a description.
	assert.Equal(t, SeedItemDescription, jsonField(item, "description"),
		"the included item carries its description, not a partial stub")

	user := jsonObject(got, "responsible_user")
	require.NotNil(t, user, "responsible_user should be present with ?include=responsible_user")
	assert.Equal(t, SeedUserID, jsonField(user, "id"))

	station := jsonObject(got, "responsible_scanning_station")
	require.NotNil(t, station, "responsible_scanning_station should be present with ?include=responsible_scanning_station")
	assert.Equal(t, SeedScanningStationID, jsonField(station, "id"))
}

// Listing with ?include=item must hydrate each row's own item — fully, and matched to that row.
//
// The two 2099-dated fixtures head the list newest-first and point at distinct items (SCK-001,
// SCK-002), so a bug that stitched the same item onto every row, or built a partial stub that drops
// the item's base fields, fails here. The retrieve test above cannot catch a per-row mismatch because
// it only ever reads a single row.
func TestInventoryChangeLogs_ListIncludeItemHydratesEachRow(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(inventoryChangeLogsPath,
		url.Values{"limit": {"2"}, "include": {"item"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, list.Data, 2, "the two 2099 fixtures head the list")

	first := parseJSON(list.Data[0])
	assert.Equal(t, SeedInventoryChangeLogID, jsonField(first, "id"))
	firstItem := jsonObject(first, "item")
	require.NotNil(t, firstItem, "the first row's item is hydrated: %s", string(list.Data[0]))
	assert.Equal(t, SeedInventoryChangeLogItemID, jsonField(firstItem, "id"), "the first row carries its own item")
	assert.Equal(t, SeedItemSKU, jsonField(firstItem, "sku"))
	assert.Equal(t, SeedItemDescription, jsonField(firstItem, "description"),
		"a list row's included item carries its base fields, not a stub")

	second := parseJSON(list.Data[1])
	assert.Equal(t, SeedInventoryChangeLog2ID, jsonField(second, "id"))
	secondItem := jsonObject(second, "item")
	require.NotNil(t, secondItem, "the second row's item is hydrated: %s", string(list.Data[1]))
	assert.Equal(t, SeedInventoryChangeLog2ItemID, jsonField(secondItem, "id"),
		"the second row carries a different item than the first")
	assert.Equal(t, "SCK-002", jsonField(secondItem, "sku"))
	assert.NotEqual(t, jsonField(firstItem, "id"), jsonField(secondItem, "id"),
		"distinct rows must not share one hydrated item")
}

// The quantity is not expandable, so it must arrive on a list row that asked for no includes at all
// — a row reporting null for the amount it recorded says nothing.
func TestInventoryChangeLogs_ListCarriesQuantityWithoutInclude(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(inventoryChangeLogsPath, url.Values{"limit": {"5"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.GreaterOrEqual(t, len(list.Data), 1, "at least one change log must be seeded")

	for _, raw := range list.Data {
		row := parseJSON(raw)
		quantity := jsonObject(row, "quantity")
		require.NotNil(t, quantity, "every row carries its quantity: %s", string(raw))
		assert.NotEmpty(t, jsonField(quantity, "value"), "the quantity reports a value: %s", string(raw))
		require.NotNil(t, jsonObject(quantity, "unit"), "the quantity carries its unit: %s", string(raw))
		assert.NotEmpty(t, jsonField(row, "action_type"), "every row reports its action type: %s", string(raw))
	}
}

func TestInventoryChangeLogs_RetrieveUnknownIDIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(inventoryChangeLogsPath+"/ivcl_doesnotexist0000", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "an unknown change log must not 5xx: %s", string(body))
	assert.Equal(t, 404, status, "retrieving an unknown change log must 404: %s", string(body))
}

// ──────────────────────────────────────────────
// List — ordering and paging
// ──────────────────────────────────────────────

// Newest first is what makes the log readable as a feed; the two 2099-dated fixtures pin the order
// of the head regardless of what the rest of the suite writes.
func TestInventoryChangeLogs_ListIsNewestFirst(t *testing.T) {
	t.Parallel()

	ids := inventoryChangeLogIDsFiltered(t, url.Values{"limit": {"2"}})
	require.Len(t, ids, 2, "the account has at least two change logs")
	assert.Equal(t, []string{SeedInventoryChangeLogID, SeedInventoryChangeLog2ID}, ids,
		"the two newest fixtures head the list, newest first")
}

// Paging forward then back must land on the page it started from: the keyset carries created_at and
// id together, and dropping either would repeat or skip rows at the boundary.
func TestInventoryChangeLogs_ListPagesForwardAndBack(t *testing.T) {
	t.Parallel()

	page1, status, err := apiClient.GetList(inventoryChangeLogsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, page1.Data, 1)
	require.True(t, page1.PageInfo.HasNextPage, "the account has more than one change log")

	page2, status, err := apiClient.GetListFromPageURL(page1.PageInfo.NextPageURL)
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, page2.Data, 1)

	first, second := DataItemField(page1.Data[0], "id"), DataItemField(page2.Data[0], "id")
	assert.NotEqual(t, first, second, "the second page holds a different row")

	back, status, err := apiClient.GetListFromPageURL(page2.PageInfo.PreviousPageURL)
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, back.Data, 1)
	assert.Equal(t, first, DataItemField(back.Data[0], "id"), "paging back returns the page it came from")
}

// ──────────────────────────────────────────────
// List — filters
// ──────────────────────────────────────────────

func TestInventoryChangeLogs_FiltersByItem(t *testing.T) {
	t.Parallel()

	matched := inventoryChangeLogIDsFiltered(t, url.Values{"item_ids": {SeedInventoryChangeLogItemID}})
	assert.Contains(t, matched, SeedInventoryChangeLogID, "SCK-001's change log is filed under SCK-001")
	assert.NotContains(t, matched, SeedInventoryChangeLog2ID, "SCK-002's change log is not")

	assert.Empty(t, inventoryChangeLogIDsFiltered(t, url.Values{"item_ids": {"it_01nosuchitem000000"}}),
		"an unknown item must narrow the list to nothing rather than be ignored")
}

// The values within one filter are an OR, so naming both items returns both rows.
func TestInventoryChangeLogs_ItemFilterUnionsItsValues(t *testing.T) {
	t.Parallel()

	both := inventoryChangeLogIDsFiltered(t, url.Values{
		"item_ids": {SeedInventoryChangeLogItemID, SeedInventoryChangeLog2ItemID},
	})
	assert.Contains(t, both, SeedInventoryChangeLogID)
	assert.Contains(t, both, SeedInventoryChangeLog2ID)
}

func TestInventoryChangeLogs_FiltersByActionType(t *testing.T) {
	t.Parallel()

	matched := inventoryChangeLogIDsFiltered(t, url.Values{"action_types": {"system_action"}})
	assert.Contains(t, matched, SeedInventoryChangeLog2ID, "the second fixture is a system_action")
	assert.NotContains(t, matched, SeedInventoryChangeLogID, "the first is a user_action")

	both := inventoryChangeLogIDsFiltered(t, url.Values{"action_types": {"system_action", "user_action"}})
	assert.Contains(t, both, SeedInventoryChangeLogID)
	assert.Contains(t, both, SeedInventoryChangeLog2ID)
}

func TestInventoryChangeLogs_ListRejectsAnUnknownActionType(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(inventoryChangeLogsPath,
		url.Values{"action_types": {"bogus_e2e_action"}})
	require.NoError(t, err)
	require.Less(t, status, 500, "an unknown action type must not 5xx: %s", string(body))
	require.Equal(t, 400, status, "action_types only accepts the documented values: %s", string(body))
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

func TestInventoryChangeLogs_FiltersByChangedByUser(t *testing.T) {
	t.Parallel()

	matched := inventoryChangeLogIDsFiltered(t, url.Values{"changed_by_user_ids": {SeedUserID}})
	assert.Contains(t, matched, SeedInventoryChangeLogID, "John Doe recorded the first fixture")
	assert.NotContains(t, matched, SeedInventoryChangeLog2ID, "Sarah Martinez recorded the second")

	assert.Empty(t, inventoryChangeLogIDsFiltered(t, url.Values{"changed_by_user_ids": {"us_nosuchuser00"}}),
		"an unknown user must narrow the list to nothing rather than be ignored")
}

// Most of the log is system-recorded and carries no responsible user. Naming a user has to exclude
// those rows: a NULL responsible_user_id is not a match for anybody.
func TestInventoryChangeLogs_ChangedByUserExcludesUnattributedChanges(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(inventoryChangeLogsPath, url.Values{
		"changed_by_user_ids": {SeedUserID},
		"include":             {"responsible_user"},
		"limit":               {"20"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	for _, raw := range jsonArray(parseJSON(body), "data") {
		row, ok := raw.(map[string]any)
		require.True(t, ok, "the list holds objects: %s", string(body))
		user := jsonObject(row, "responsible_user")
		require.NotNil(t, user, "a row filtered by user must carry that user: %s", string(body))
		assert.Equal(t, SeedUserID, jsonField(user, "id"))
	}
}

// The window filters on when the change was recorded. The fixtures sit in 2099, so a window that
// closes before then must leave them out while a window reaching past them lets them in.
func TestInventoryChangeLogs_FiltersByCreatedWindow(t *testing.T) {
	t.Parallel()

	inWindow := inventoryChangeLogIDsFiltered(t, url.Values{"starts_at": {"2099-01-01T00:00:00Z"}})
	assert.Contains(t, inWindow, SeedInventoryChangeLogID, "a window opening in 2099 keeps the 2099 fixtures")

	excluded := inventoryChangeLogIDsFiltered(t, url.Values{"ends_at": {"2098-12-31T23:59:59Z"}})
	assert.NotContains(t, excluded, SeedInventoryChangeLogID, "a window closing in 2098 leaves them behind")
	assert.NotContains(t, excluded, SeedInventoryChangeLog2ID)

	assert.Empty(t, inventoryChangeLogIDsFiltered(t, url.Values{
		"starts_at": {"2000-01-01T00:00:00Z"},
		"ends_at":   {"2000-01-02T00:00:00Z"},
	}), "a window that closed decades ago must exclude every change log")
}

// Filters combine with AND, so a pairing that matches nothing returns nothing even though each half
// matches on its own.
func TestInventoryChangeLogs_FiltersCombineWithAnd(t *testing.T) {
	t.Parallel()

	// The first fixture's item paired with the second fixture's user: neither row satisfies both.
	assert.Empty(t, inventoryChangeLogIDsFiltered(t, url.Values{
		"item_ids":            {SeedInventoryChangeLogItemID},
		"changed_by_user_ids": {SeedInventoryChangeLog2UserID},
	}), "filters intersect rather than union across dimensions")
}

func TestInventoryChangeLogs_ListRejectsAnUnknownQueryParam(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(inventoryChangeLogsPath, url.Values{bogusE2EQueryParam: {"1"}})
	require.NoError(t, err)
	assertUnknownQueryParamRejected(t, inventoryChangeLogsPath, status, body)
}

// `q` searches the affected item's SKU, so a fixture's exact SKU returns that fixture's log and
// excludes the other.
func TestInventoryChangeLogs_SearchesByItemSKU(t *testing.T) {
	t.Parallel()

	matched := inventoryChangeLogIDsFiltered(t, url.Values{"q": {SeedItemSKU}})
	assert.Contains(t, matched, SeedInventoryChangeLogID, "SCK-001's log matches its own SKU")
	assert.NotContains(t, matched, SeedInventoryChangeLog2ID, "SCK-002's log does not match SCK-001")
}

// The match is a substring anywhere in the SKU, not a prefix: a fragment that sits at the tail of one
// fixture's SKU and nowhere in the other still separates them.
func TestInventoryChangeLogs_SearchMatchesSKUSubstring(t *testing.T) {
	t.Parallel()

	matched := inventoryChangeLogIDsFiltered(t, url.Values{"q": {"001"}})
	assert.Contains(t, matched, SeedInventoryChangeLogID, `"001" is a substring of SCK-001`)
	assert.NotContains(t, matched, SeedInventoryChangeLog2ID, `SCK-002 does not contain "001"`)
}

// A SKU no item carries narrows the list to nothing rather than being ignored.
func TestInventoryChangeLogs_SearchWithNoSKUMatchIsEmpty(t *testing.T) {
	t.Parallel()

	assert.Empty(t, inventoryChangeLogIDsFiltered(t, url.Values{"q": {"zzz-no-such-sku-e2e"}}),
		"an unmatched SKU search returns no rows")
}

// SKU search intersects with the other filters rather than widening them: a SKU paired with an item
// it does not belong to matches nothing even though each half matches on its own.
func TestInventoryChangeLogs_SearchCombinesWithItemFilter(t *testing.T) {
	t.Parallel()

	assert.Empty(t, inventoryChangeLogIDsFiltered(t, url.Values{
		"q":        {SeedItemSKU},
		"item_ids": {SeedInventoryChangeLog2ItemID},
	}), "SCK-001's SKU paired with SCK-002's item satisfies neither row")

	both := inventoryChangeLogIDsFiltered(t, url.Values{
		"q":        {SeedItemSKU},
		"item_ids": {SeedInventoryChangeLogItemID},
	})
	assert.Contains(t, both, SeedInventoryChangeLogID, "the SKU and the item agree on SCK-001's log")
}

func TestInventoryChangeLogs_ListRejectsAnOutOfRangeLimit(t *testing.T) {
	t.Parallel()

	for _, limit := range []string{"0", "-1", "abc"} {
		status, body, err := apiClient.GetListRaw(inventoryChangeLogsPath, url.Values{"limit": {limit}})
		require.NoError(t, err)
		require.Less(t, status, 500, "limit=%s must not 5xx: %s", limit, string(body))
		assert.Equal(t, 400, status, "limit=%s should 400: %s", limit, string(body))
	}
}

// ──────────────────────────────────────────────
// Read-only
// ──────────────────────────────────────────────

// The log is an audit trail: it is written by the movements it records, never by a client.
func TestInventoryChangeLogs_AreNotWritable(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(inventoryChangeLogsPath, map[string]any{
		"action_type": "user_action",
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "POST must not 5xx: %s", string(body))
	assert.Equal(t, 405, status, "the collection accepts no writes: %s", string(body))

	path := inventoryChangeLogsPath + "/" + SeedInventoryChangeLogID
	for method, do := range map[string]func() (int, []byte, error){
		"PATCH":  func() (int, []byte, error) { return apiClient.Patch(path, map[string]any{}, newIdempotencyKey()) },
		"DELETE": func() (int, []byte, error) { return apiClient.Delete(path) },
	} {
		status, body, err := do()
		require.NoError(t, err)
		require.Less(t, status, 500, "%s must not 5xx: %s", method, string(body))
		assert.Equal(t, 405, status, "%s on a change log is not allowed: %s", method, string(body))
	}
}

// ──────────────────────────────────────────────
// Export
// ──────────────────────────────────────────────

// The export answers with the workbook itself, and names the file for the window it was asked for
// so a user downloading two ranges does not overwrite one with the other.
func TestInventoryChangeLogs_ExportNamesTheFileForItsWindow(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		params   url.Values
		filename string
	}{
		"no window": {nil, "inventory-change-logs-all-all.xlsx"},
		"both bounds": {url.Values{
			"starts_at": {"2099-01-01T00:00:00Z"},
			"ends_at":   {"2099-12-31T00:00:00Z"},
		}, "inventory-change-logs-2099-01-01-2099-12-31.xlsx"},
		// An open bound reads as `all` rather than being dropped, so the name still says what was asked for.
		"open end": {url.Values{"starts_at": {"2099-01-01T00:00:00Z"}}, "inventory-change-logs-2099-01-01-all.xlsx"},
	} {
		resp, err := apiClient.GetFull(inventoryChangeLogsExportPath, tc.params)
		require.NoError(t, err)
		require.Less(t, resp.StatusCode, 500, "%s must not 5xx: %s", name, string(resp.Body))
		requireStatus(t, 200, resp.StatusCode, resp.Body)

		assert.Contains(t, resp.Header.Get("Content-Disposition"), tc.filename,
			"%s should be named for its window", name)
		assert.NotEmpty(t, resp.Body, "%s returns the workbook itself", name)
	}
}

// The export takes the same filters as the list, so a filter it ignored would hand back a file
// covering rows the caller excluded.
func TestInventoryChangeLogs_ExportAcceptsTheListFilters(t *testing.T) {
	t.Parallel()

	resp, err := apiClient.GetFull(inventoryChangeLogsExportPath, url.Values{
		"item_ids":            {SeedInventoryChangeLogItemID},
		"action_types":        {"user_action"},
		"changed_by_user_ids": {SeedUserID},
		"starts_at":           {"2099-01-01T00:00:00Z"},
	})
	require.NoError(t, err)
	require.Less(t, resp.StatusCode, 500, "a filtered export must not 5xx: %s", string(resp.Body))
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	assert.NotEmpty(t, resp.Body)
}

func TestInventoryChangeLogs_ExportRejectsAnUnknownActionType(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(inventoryChangeLogsExportPath,
		url.Values{"action_types": {"bogus_e2e_action"}})
	require.NoError(t, err)
	require.Less(t, status, 500, "an unknown action type must not 5xx: %s", string(body))
	assert.Equal(t, 400, status, "the export validates its filters like the list does: %s", string(body))
}
