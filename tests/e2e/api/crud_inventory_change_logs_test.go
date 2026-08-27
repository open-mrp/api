//go:build e2e

package api_test

import (
	"net/url"
	"testing"

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
