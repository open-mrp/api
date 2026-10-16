//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertQuantityInfoShape(t *testing.T, qi map[string]any, label string) {
	t.Helper()
	require.NotNil(t, qi, "%s should be present", label)
	assert.NotEmpty(t, jsonField(qi, "value"), "%s.value should be a non-empty decimal string", label)
	assertValueShapeBare(t, qi, label)
	// The unit travels with the figure instead of sitting behind an include of its own.
	assertUnitHydrated(t, jsonObject(qi, "unit"), label+".unit")
}

// ──────────────────────────────────────────────
// Item Inventory — Get
// ──────────────────────────────────────────────

func TestItemInventory_Get_ResponseShape(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(itemsPath+"/"+SeedItemID+"/inventory", url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assertObjectField(t, got, "inventory_level")
	assert.Equal(t, SeedItemID, jsonField(jsonObject(got, "item"), "id"))
	assertNilField(t, got, "location")
	assertQuantityInfoShape(t, jsonObject(got, "on_hand"), "on_hand")
	assertQuantityInfoShape(t, jsonObject(got, "reserved"), "reserved")
	assertQuantityInfoShape(t, jsonObject(got, "available_to_promise"), "available_to_promise")
	assertQuantityInfoShape(t, jsonObject(got, "short"), "short")
}

func TestItemInventory_Get_NotFound(t *testing.T) {
	t.Parallel()
	fakeID := "it_01zzzzzzzzzzzzzzzzzzzzzzz"
	status, _, err := apiClient.GetListRaw(itemsPath+"/"+fakeID+"/inventory", nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status, "unknown item id should return 404 for inventory")
}
