//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Per-item planning overrides: the list of exceptions to the account defaults.
//
// Every test here owns the item it overrides. An override is keyed by item, so
// tests that each bring their own item never contend — but two tests sharing one
// would silently overwrite each other's setting rather than fail.

// putItemSetting writes one item's overrides and returns the raw result, leaving
// the caller to decide what the status should have been.
func putItemSetting(t *testing.T, itemID string, body map[string]any) (int, []byte) {
	t.Helper()

	status, respBody, err := apiClient.Put(itemSettingsPath+"/"+itemID, body)
	require.NoError(t, err)
	require.Less(t, status, 500, "an item setting write must not 5xx: %s", string(respBody))
	return status, respBody
}

// ownedItemSetting writes an override for a fresh item and removes it afterwards.
func ownedItemSetting(t *testing.T, prefix string, body map[string]any) (string, map[string]any) {
	t.Helper()

	itemID := createItemsViaMaterials(t, uniqueName(prefix), 1)[0]
	t.Cleanup(func() { _, _, _ = apiClient.Delete(itemSettingsPath + "/" + itemID) })

	status, respBody := putItemSetting(t, itemID, body)
	requireStatus(t, 200, status, respBody)
	return itemID, parseJSON(respBody)
}

// getItemSetting reads one item's overrides.
func getItemSetting(t *testing.T, itemID string) (int, map[string]any) {
	t.Helper()

	status, body, err := apiClient.GetListRaw(itemSettingsPath+"/"+itemID, nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "reading an item setting must not 5xx: %s", string(body))
	if status != 200 {
		return status, nil
	}
	return status, parseJSON(body)
}

// listItemSettingFor finds one item's row in the account-wide list, or nil.
func listItemSettingFor(t *testing.T, itemID string) map[string]any {
	t.Helper()

	status, body, err := apiClient.GetListRaw(itemSettingsPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	parsed := parseJSON(body)
	assert.Equal(t, "list", jsonField(parsed, "object"), "item settings must serialize as a list resource")

	var found map[string]any
	matches := 0
	for _, raw := range jsonArray(parsed, "data") {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if item, ok := row["item"].(map[string]any); ok && jsonField(item, "id") == itemID {
			matches++
			found = row
		}
	}
	// An item has at most one set of overrides. A second row would make which one the
	// solver reads arbitrary, so the count is asserted here rather than only the find.
	assert.LessOrEqual(t, matches, 1, "an item must never have two override rows")
	return found
}

// A written override has to come back whole: what was set, on which item, and when.
func TestItemSettings_RetrieveDescribesTheOverride(t *testing.T) {
	t.Parallel()

	itemID, written := ownedItemSetting(t, "e2e-itemset-read", map[string]any{
		"participation_status": "included",
		"lot_multiple_units":   120,
		"fulfillment_policy":   "make_to_order",
	})

	status, got := getItemSetting(t, itemID)
	requireStatus(t, 200, status, nil)

	assert.Equal(t, "production_schedule_item_setting", jsonField(got, "object"))
	assert.Equal(t, jsonField(written, "id"), jsonField(got, "id"),
		"the read must return the row the write created")
	assert.NotEmpty(t, jsonField(got, "id"))
	assert.Equal(t, "included", jsonField(got, "participation_status"))
	assert.Equal(t, "make_to_order", jsonField(got, "fulfillment_policy"))
	assert.Equal(t, "120", jsonField(got, "lot_multiple_units"))
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	item, ok := got["item"].(map[string]any)
	require.True(t, ok, "the override must name the item it applies to: %v", got)
	assert.Equal(t, itemID, jsonField(item, "id"))
	assert.Equal(t, "item", jsonField(item, "type"))
	assert.NotEmpty(t, jsonField(item, "handle"),
		"the item reference carries its SKU as the handle, so a row is readable without a second lookup")
}

// An item with no override is planned on the defaults, which is not a resource
// anyone can edit — so this is a 404 rather than an empty set of overrides.
func TestItemSettings_RetrieveIsNotFoundWithoutAnOverride(t *testing.T) {
	t.Parallel()

	itemID := createItemsViaMaterials(t, uniqueName("e2e-itemset-none"), 1)[0]

	status, _ := getItemSetting(t, itemID)
	assert.Equal(t, 404, status, "an item with no override has nothing to return")

	assert.Nil(t, listItemSettingFor(t, itemID),
		"the list is the list of exceptions, so an item with no override must not appear")

	// A mistyped item id is the same answer, not a crash.
	unknownStatus, _ := getItemSetting(t, "it_doesnotexist0000")
	assert.Equal(t, 404, unknownStatus)
}

// An item has at most one set of overrides, so a second write replaces the first
// and keeps its identity rather than adding a row the solver would have to choose
// between.
func TestItemSettings_UpsertReplacesRatherThanDuplicates(t *testing.T) {
	t.Parallel()

	itemID, first := ownedItemSetting(t, "e2e-itemset-upsert", map[string]any{
		"participation_status": "included",
		"lot_multiple_units":   60,
		"fulfillment_policy":   "make_to_order",
	})
	settingID := jsonField(first, "id")
	require.NotEmpty(t, settingID)

	status, respBody := putItemSetting(t, itemID, map[string]any{
		"participation_status": "excluded",
		"lot_multiple_units":   90,
		"fulfillment_policy":   "make_to_stock",
	})
	requireStatus(t, 200, status, respBody)

	second := parseJSON(respBody)
	assert.Equal(t, settingID, jsonField(second, "id"), "the override must keep the id it already had")
	assert.Equal(t, "excluded", jsonField(second, "participation_status"))
	assert.Equal(t, "make_to_stock", jsonField(second, "fulfillment_policy"))
	assert.Equal(t, "90", jsonField(second, "lot_multiple_units"))

	row := listItemSettingFor(t, itemID)
	require.NotNil(t, row, "the override should still be listed after being replaced")
	assert.Equal(t, settingID, jsonField(row, "id"))
	assert.Equal(t, "excluded", jsonField(row, "participation_status"))
}

// The write replaces the whole override, so a field left out of the request is
// removed rather than carried over. Anything else would make it impossible to take
// a lot override off an item without deleting the whole row.
func TestItemSettings_OmittedFieldsAreReset(t *testing.T) {
	t.Parallel()

	itemID, _ := ownedItemSetting(t, "e2e-itemset-reset", map[string]any{
		"participation_status": "included",
		"lot_multiple_units":   150,
		"fulfillment_policy":   "make_to_order",
	})

	status, respBody := putItemSetting(t, itemID, map[string]any{"participation_status": "included"})
	requireStatus(t, 200, status, respBody)

	after := parseJSON(respBody)
	assert.Nil(t, after["lot_multiple_units"],
		"a lot override left out of a replacing write must be removed, not kept")
	assert.Nil(t, after["fulfillment_policy"],
		"a policy left out must return the item to its product line and the account default")

	_, reread := getItemSetting(t, itemID)
	assert.Nil(t, reread["lot_multiple_units"], "and the reset must be durable, not just echoed")
	assert.Nil(t, reread["fulfillment_policy"])
}

// An excluded item is left out of the plan entirely, so the status has to survive
// the round trip: reading it back as included would quietly put it back in.
func TestItemSettings_ExclusionRoundTrips(t *testing.T) {
	t.Parallel()

	itemID, written := ownedItemSetting(t, "e2e-itemset-excluded", map[string]any{
		"participation_status": "excluded",
	})
	assert.Equal(t, "excluded", jsonField(written, "participation_status"))

	_, got := getItemSetting(t, itemID)
	assert.Equal(t, "excluded", jsonField(got, "participation_status"))

	row := listItemSettingFor(t, itemID)
	require.NotNil(t, row)
	assert.Equal(t, "excluded", jsonField(row, "participation_status"))
}

func TestItemSettings_RejectsUnworkableOverrides(t *testing.T) {
	t.Parallel()

	itemID := createItemsViaMaterials(t, uniqueName("e2e-itemset-invalid"), 1)[0]

	for _, tc := range []struct {
		name string
		body map[string]any
	}{
		{"zero lot multiple", map[string]any{"participation_status": "included", "lot_multiple_units": 0}},
		{"negative lot multiple", map[string]any{"participation_status": "included", "lot_multiple_units": -10}},
		{"missing participation status", map[string]any{"lot_multiple_units": 60}},
		{"unknown participation status", map[string]any{"participation_status": "maybe"}},
		{"unknown policy", map[string]any{"participation_status": "included", "fulfillment_policy": "make_to_whatever"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, body := putItemSetting(t, itemID, tc.body)
			assert.Equal(t, 400, status, "body: %s", string(body))
		})
	}

	// A rejected write leaves nothing behind: a 400 that still created a row would be
	// worse than either outcome on its own.
	status, _ := getItemSetting(t, itemID)
	assert.Equal(t, 404, status, "no rejected write should have created an override")
}

// An override for an item that does not exist would be invisible to the solver and
// unexplainable to whoever wrote it.
func TestItemSettings_RejectsUnknownItem(t *testing.T) {
	t.Parallel()

	status, body := putItemSetting(t, "it_doesnotexist0000", map[string]any{
		"participation_status": "included",
	})
	assert.Contains(t, []int{400, 404}, status,
		"an override for an unknown item must be rejected, got %d: %s", status, string(body))
}

// Deleting returns the item to the defaults, and deleting again is a 404 — saying
// "removed" about an override that never existed hides a mistyped id.
func TestItemSettings_DeleteReturnsItemToDefaults(t *testing.T) {
	t.Parallel()

	itemID, _ := ownedItemSetting(t, "e2e-itemset-delete", map[string]any{
		"participation_status": "excluded",
		"fulfillment_policy":   "make_to_order",
	})

	status, body, err := apiClient.Delete(itemSettingsPath + "/" + itemID)
	require.NoError(t, err)
	require.Less(t, status, 500, "delete must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	getStatus, _ := getItemSetting(t, itemID)
	assert.Equal(t, 404, getStatus, "a deleted override must stop being readable")
	assert.Nil(t, listItemSettingFor(t, itemID), "and must stop being listed")

	repeatStatus, repeatBody, err := apiClient.Delete(itemSettingsPath + "/" + itemID)
	require.NoError(t, err)
	require.Less(t, repeatStatus, 500, "repeat delete must not 5xx: %s", string(repeatBody))
	assert.Equal(t, 404, repeatStatus)

	unknownStatus, unknownBody, err := apiClient.Delete(itemSettingsPath + "/it_doesnotexist0000")
	require.NoError(t, err)
	require.Less(t, unknownStatus, 500, "deleting an unknown item must not 5xx: %s", string(unknownBody))
	assert.Equal(t, 404, unknownStatus)
}

// Planning overrides decide what a factory builds, so another tenant must not be
// able to read one, replace one, or remove one.
func TestItemSettings_TenantIsolation(t *testing.T) {
	t.Parallel()
	clientB := getTenantBClient()

	itemID, written := ownedItemSetting(t, "e2e-itemset-tenant", map[string]any{
		"participation_status": "included",
		"lot_multiple_units":   75,
		"fulfillment_policy":   "make_to_order",
	})

	readStatus, readBody, err := clientB.GetListRaw(itemSettingsPath+"/"+itemID, nil)
	require.NoError(t, err)
	require.Less(t, readStatus, 500, "cross-tenant read must not 5xx: %s", string(readBody))
	assert.Equal(t, 404, readStatus, "tenant B must not read tenant A's override: %s", string(readBody))

	// The item belongs to tenant A, so from B's side it does not exist at all.
	writeStatus, writeBody, err := clientB.Put(itemSettingsPath+"/"+itemID, map[string]any{
		"participation_status": "excluded",
	})
	require.NoError(t, err)
	require.Less(t, writeStatus, 500, "cross-tenant write must not 5xx: %s", string(writeBody))
	assert.Contains(t, []int{400, 404}, writeStatus,
		"tenant B must not write against tenant A's item, got %d: %s", writeStatus, string(writeBody))

	deleteStatus, deleteBody, err := clientB.Delete(itemSettingsPath + "/" + itemID)
	require.NoError(t, err)
	require.Less(t, deleteStatus, 500, "cross-tenant delete must not 5xx: %s", string(deleteBody))
	assert.Equal(t, 404, deleteStatus, "tenant B must not delete tenant A's override")

	// The whole point: none of that touched the override.
	_, after := getItemSetting(t, itemID)
	assert.Equal(t, jsonField(written, "id"), jsonField(after, "id"))
	assert.Equal(t, "included", jsonField(after, "participation_status"))
	assert.Equal(t, "make_to_order", jsonField(after, "fulfillment_policy"))
	assert.Equal(t, "75", jsonField(after, "lot_multiple_units"))
}
