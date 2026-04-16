//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// Item — Include Tests
// ──────────────────────────────────────────────
//
// Item GET endpoint whitelists: category, unit_value, unit_cost, burn_rate
// (attributes is a registered include but is served at a different endpoint).

func TestItems_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(itemsPath+"/"+SeedItemID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["category"], "category should be null without ?include=category")
	assert.Nil(t, got["unit_value"], "unit_value should be null without ?include=unit_value")
	assert.Nil(t, got["unit_cost"], "unit_cost should be null without ?include=unit_cost")
	assert.Nil(t, got["burn_rate"], "burn_rate should be null without ?include=burn_rate")

	list, _, err := apiClient.GetList(itemsPath, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	for _, item := range list.Data {
		m := parseJSON(item)
		assert.Nil(t, m["category"], "category should be null on list items without ?include=category")
		assert.Nil(t, m["unit_value"], "unit_value should be null on list items without ?include=unit_value")
		assert.Nil(t, m["unit_cost"], "unit_cost should be null on list items without ?include=unit_cost")
		assert.Nil(t, m["burn_rate"], "burn_rate should be null on list items without ?include=burn_rate")
	}
}

func TestItems_IncludeCategory(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(itemsPath+"/"+SeedItemID, url.Values{"include": {"category"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	cat := jsonObject(got, "category")
	require.NotNil(t, cat, "category should be present with ?include=category")
	assert.Equal(t, "item_category", jsonField(cat, "object"))
	assert.NotEmpty(t, jsonField(cat, "id"))
}

func TestItems_IncludeUnitValue(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(itemsPath+"/"+SeedItemID, url.Values{"include": {"unit_value"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["unit_value"]
	assert.True(t, ok, "unit_value key should be present with ?include=unit_value")
	if uv := jsonObject(got, "unit_value"); uv != nil {
		assert.Equal(t, "rate", jsonField(uv, "object"))
		assert.NotEmpty(t, jsonField(uv, "id"))
	}
}

func TestItems_IncludeUnitCost(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(itemsPath+"/"+SeedItemID, url.Values{"include": {"unit_cost"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["unit_cost"]
	assert.True(t, ok, "unit_cost key should be present with ?include=unit_cost")
	if uc := jsonObject(got, "unit_cost"); uc != nil {
		assert.Equal(t, "rate", jsonField(uc, "object"))
		assert.NotEmpty(t, jsonField(uc, "id"))
	}
}

func TestItems_IncludeBurnRate(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(itemsPath+"/"+SeedItemID, url.Values{"include": {"burn_rate"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["burn_rate"]
	assert.True(t, ok, "burn_rate key should be present with ?include=burn_rate")
	if br := jsonObject(got, "burn_rate"); br != nil {
		assert.Equal(t, "rate", jsonField(br, "object"))
		assert.NotEmpty(t, jsonField(br, "id"))
	}
}

func TestItems_ListIncludeCategory(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(itemsPath, url.Values{"include": {"category"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	data, ok := got["data"].([]any)
	require.True(t, ok, "response should have data array")
	require.NotEmpty(t, data, "should have at least one item")

	first, ok := data[0].(map[string]any)
	require.True(t, ok)
	cat := jsonObject(first, "category")
	require.NotNil(t, cat, "category should be present on list items with ?include=category")
	assert.Equal(t, "item_category", jsonField(cat, "object"))
}
