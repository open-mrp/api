//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const partsPath = "/v1/operations/parts"

// firstPartID returns the id of the first part in seed data. Fails if none.
func firstPartID(t *testing.T) string {
	t.Helper()
	list, status, err := apiClient.GetList(partsPath, nil)
	require.NoError(t, err)
	require.Equal(t, 200, status, "parts list endpoint should return 200")
	require.GreaterOrEqual(t, len(list.Data), 1, "at least one part must be seeded")
	id := DataItemField(list.Data[0], "id")
	require.NotEmpty(t, id)
	return id
}

// ──────────────────────────────────────────────
// Part — Include Tests
// ──────────────────────────────────────────────
//
// Part GET endpoint whitelists item and its nested sub-resources:
// item.category, item.unit_value, item.unit_cost, item.burn_rate.

func TestParts_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()
	id := firstPartID(t)

	status, body, err := apiClient.GetListRaw(partsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["item"], "item should be null without ?include=item")
}

func TestParts_IncludeItem(t *testing.T) {
	t.Parallel()
	id := firstPartID(t)

	status, body, err := apiClient.GetListRaw(partsPath+"/"+id, url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present with ?include=item")
	assert.Equal(t, "item", jsonField(item, "object"))
	assert.NotEmpty(t, jsonField(item, "id"))
}

func TestParts_IncludeCategory(t *testing.T) {
	t.Parallel()
	id := firstPartID(t)

	status, body, err := apiClient.GetListRaw(partsPath+"/"+id, url.Values{"include": {"item.category"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present with ?include=item.category")
	cat := jsonObject(item, "category")
	require.NotNil(t, cat, "item.category should be present with ?include=item.category")
	assert.Equal(t, "item_category", jsonField(cat, "object"))
	assert.NotEmpty(t, jsonField(cat, "id"))
}

func TestParts_IncludeUnitValue(t *testing.T) {
	t.Parallel()
	id := firstPartID(t)

	status, body, err := apiClient.GetListRaw(partsPath+"/"+id, url.Values{"include": {"item.unit_value"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present with ?include=item.unit_value")
	_, ok := item["unit_value"]
	assert.True(t, ok, "item.unit_value key should be present with ?include=item.unit_value")
	if uv := jsonObject(item, "unit_value"); uv != nil {
		assert.Equal(t, "rate", jsonField(uv, "object"))
	}
}

func TestParts_IncludeUnitCost(t *testing.T) {
	t.Parallel()
	id := firstPartID(t)

	status, body, err := apiClient.GetListRaw(partsPath+"/"+id, url.Values{"include": {"item.unit_cost"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present with ?include=item.unit_cost")
	_, ok := item["unit_cost"]
	assert.True(t, ok, "item.unit_cost key should be present with ?include=item.unit_cost")
	if uc := jsonObject(item, "unit_cost"); uc != nil {
		assert.Equal(t, "rate", jsonField(uc, "object"))
	}
}

func TestParts_IncludeBurnRate(t *testing.T) {
	t.Parallel()
	id := firstPartID(t)

	status, body, err := apiClient.GetListRaw(partsPath+"/"+id, url.Values{"include": {"item.burn_rate"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present with ?include=item.burn_rate")
	_, ok := item["burn_rate"]
	assert.True(t, ok, "item.burn_rate key should be present with ?include=item.burn_rate")
	if br := jsonObject(item, "burn_rate"); br != nil {
		assert.Equal(t, "rate", jsonField(br, "object"))
	}
}
