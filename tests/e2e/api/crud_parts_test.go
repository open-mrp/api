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
// Part GET endpoint whitelists: category, unit_value, unit_cost, burn_rate.

func TestParts_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()
	id := firstPartID(t)

	status, body, err := apiClient.GetListRaw(partsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["category"], "category should be null without ?include=category")
	assert.Nil(t, got["unit_value"], "unit_value should be null without ?include=unit_value")
	assert.Nil(t, got["unit_cost"], "unit_cost should be null without ?include=unit_cost")
	assert.Nil(t, got["burn_rate"], "burn_rate should be null without ?include=burn_rate")
}

func TestParts_IncludeCategory(t *testing.T) {
	t.Parallel()
	id := firstPartID(t)

	status, body, err := apiClient.GetListRaw(partsPath+"/"+id, url.Values{"include": {"category"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	cat := jsonObject(got, "category")
	require.NotNil(t, cat, "category should be present with ?include=category")
	assert.Equal(t, "item_category", jsonField(cat, "object"))
	assert.NotEmpty(t, jsonField(cat, "id"))
}

func TestParts_IncludeUnitValue(t *testing.T) {
	t.Parallel()
	id := firstPartID(t)

	status, body, err := apiClient.GetListRaw(partsPath+"/"+id, url.Values{"include": {"unit_value"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["unit_value"]
	assert.True(t, ok, "unit_value key should be present with ?include=unit_value")
	if uv := jsonObject(got, "unit_value"); uv != nil {
		assert.Equal(t, "rate", jsonField(uv, "object"))
	}
}

func TestParts_IncludeUnitCost(t *testing.T) {
	t.Parallel()
	id := firstPartID(t)

	status, body, err := apiClient.GetListRaw(partsPath+"/"+id, url.Values{"include": {"unit_cost"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["unit_cost"]
	assert.True(t, ok, "unit_cost key should be present with ?include=unit_cost")
	if uc := jsonObject(got, "unit_cost"); uc != nil {
		assert.Equal(t, "rate", jsonField(uc, "object"))
	}
}

func TestParts_IncludeBurnRate(t *testing.T) {
	t.Parallel()
	id := firstPartID(t)

	status, body, err := apiClient.GetListRaw(partsPath+"/"+id, url.Values{"include": {"burn_rate"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["burn_rate"]
	assert.True(t, ok, "burn_rate key should be present with ?include=burn_rate")
	if br := jsonObject(got, "burn_rate"); br != nil {
		assert.Equal(t, "rate", jsonField(br, "object"))
	}
}
