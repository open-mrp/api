//go:build e2e

package api_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const inventoryChangeLogsPath = "/v1/operations/inventory-change-logs"

// ──────────────────────────────────────────────
// InventoryChangeLog — Include Tests
// ──────────────────────────────────────────────

// firstInventoryChangeLogID returns the id of the first inventory change log.
// Fails loudly if list is empty so missing fixtures surface rather than skip.
func firstInventoryChangeLogID(t *testing.T) string {
	t.Helper()
	list, status, err := apiClient.GetList(inventoryChangeLogsPath, nil)
	require.NoError(t, err)
	require.Equal(t, 200, status, "inventory change logs list should return 200")
	require.GreaterOrEqual(t, len(list.Data), 1, "at least one inventory change log must be seeded")
	id := DataItemField(list.Data[0], "id")
	require.NotEmpty(t, id)
	return id
}

func TestInventoryChangeLogs_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()
	id := firstInventoryChangeLogID(t)

	status, body, err := apiClient.GetListRaw(inventoryChangeLogsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["item"], "item should be null without ?include=item")
	assert.Nil(t, got["responsible_user"], "responsible_user should be null without ?include=responsible_user")
	assert.Nil(t, got["responsible_scanning_station"], "responsible_scanning_station should be null without ?include=responsible_scanning_station")

	list, _, err := apiClient.GetList(inventoryChangeLogsPath, nil)
	require.NoError(t, err)
	for _, m := range list.Data {
		mm := parseJSON(m)
		assert.Nil(t, mm["item"], "item should be null on list items without ?include=item")
		assert.Nil(t, mm["responsible_user"], "responsible_user should be null on list items without ?include=responsible_user")
		assert.Nil(t, mm["responsible_scanning_station"], "responsible_scanning_station should be null on list items without ?include=responsible_scanning_station")
	}
}

func TestInventoryChangeLogs_IncludeItem(t *testing.T) {
	t.Parallel()
	id := firstInventoryChangeLogID(t)

	status, body, err := apiClient.GetListRaw(inventoryChangeLogsPath+"/"+id, url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present with ?include=item")
	assert.Equal(t, "item", jsonField(item, "object"))
}

// The amount is part of the entry rather than a relation, so naming it as an include is a caller
// mistake instead of a no-op that would quietly return the row anyway.
func TestInventoryChangeLogs_QuantityIsNotAnInclude(t *testing.T) {
	t.Parallel()
	id := firstInventoryChangeLogID(t)

	status, body, err := apiClient.GetListRaw(inventoryChangeLogsPath+"/"+id, url.Values{"include": {"quantity"}})
	require.NoError(t, err)
	require.Less(t, status, 500, "an unknown include must not 5xx: %s", string(body))
	assert.Equal(t, 400, status, "quantity is not an expandable relation: %s", string(body))
}

func TestInventoryChangeLogs_IncludeResponsibleUser(t *testing.T) {
	t.Parallel()
	id := firstInventoryChangeLogID(t)

	status, body, err := apiClient.GetListRaw(inventoryChangeLogsPath+"/"+id, url.Values{"include": {"responsible_user"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["responsible_user"]
	assert.True(t, ok, "responsible_user key should be present with ?include=responsible_user")
}

func TestInventoryChangeLogs_IncludeResponsibleScanningStation(t *testing.T) {
	t.Parallel()
	id := firstInventoryChangeLogID(t)

	status, body, err := apiClient.GetListRaw(inventoryChangeLogsPath+"/"+id, url.Values{"include": {"responsible_scanning_station"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["responsible_scanning_station"]
	assert.True(t, ok, "responsible_scanning_station key should be present with ?include=responsible_scanning_station")
}

// A change log whose item is gone is left out of the list without cutting it off. The list picks a page of
// change logs and then joins each one's item; a row without one used to vanish from the page and take
// the next-page link with it.
func TestInventoryChangeLogs_ARowWithoutItsItemDoesNotCutTheListOff(t *testing.T) {
	t.Parallel()
	db := authDB(t)
	suffix := uuid.New().String()[:12]
	user := "usr_e2e_" + suffix
	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)
	rows := []struct {
		id, itemID string
		at         time.Time
	}{
		{"icl_e2e_" + suffix + "_c", SeedItemID, base.Add(3 * time.Second)},
		{"icl_e2e_" + suffix + "_b", "it_e2e_gone_" + suffix, base.Add(2 * time.Second)},
		{"icl_e2e_" + suffix + "_a", SeedItemID, base.Add(time.Second)},
	}
	for _, r := range rows {
		quantityID := "qu_e2e_" + r.id
		_, err := db.Exec("INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES (?, 1, ?, NOW(3), NOW(3))", quantityID, unitEach)
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO inventory_change_log (id, item_id, quantity_id, action_type_code, created_at, updated_at, account_id, responsible_user_id)
			VALUES (?, ?, ?, 'system_action', ?, ?, ?, ?)`, r.id, r.itemID, quantityID, r.at, r.at, SeedAccountID, user)
		require.NoError(t, err)
		t.Cleanup(func() {
			_, _ = db.Exec("DELETE FROM inventory_change_log WHERE id = ?", r.id)
			_, _ = db.Exec("DELETE FROM quantity WHERE id = ?", quantityID)
		})
	}

	var listed []string
	params := url.Values{"changed_by_user_ids": {user}, "limit": {"1"}}
	for page := 0; page < len(rows)+1; page++ {
		status, raw, err := apiClient.GetListRaw(inventoryChangeLogsPath, params)
		require.NoError(t, err)
		requireStatus(t, 200, status, raw)
		got := parseJSON(raw)
		for _, row := range jsonArray(got, "data") {
			listed = append(listed, jsonField(row.(map[string]any), "id"))
		}
		next := jsonField(jsonObject(got, "page_info"), "next_page_url")
		if next == "" {
			break
		}
		params = url.Values{"changed_by_user_ids": {user}, "limit": {"1"}, "cursor": {cursorFromURL(t, next)}}
	}
	assert.Equal(t, []string{rows[0].id, rows[2].id}, listed, "both rows with an item are reached; the one without is left out")
}
