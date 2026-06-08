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
	assert.NotEmpty(t, jsonField(qi, "display_value"), "%s.display_value should convey the unit", label)
	// unit is an expandable sub-resource of a quantity: null unless explicitly
	// requested. The unit abbreviation is still conveyed via display_value.
	assert.Nil(t, qi["unit"], "%s.unit should be null without an explicit unit include", label)
}

// ──────────────────────────────────────────────
// Item Inventory — Get
// ──────────────────────────────────────────────

func TestItemInventory_Get_ResponseShape(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(itemsPath+"/"+SeedItemID+"/inventory", url.Values{
		"include": {"on_hand", "reserved", "available_to_promise", "short"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assertObjectField(t, got, "item_inventory")
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
