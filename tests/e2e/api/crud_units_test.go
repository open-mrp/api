//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const unitsPath = "/v1/catalog/units"
const unitsValidatePath = "/v1/catalog/units/actions/validate"

// --- List ---

func TestUnits_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(unitsPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Should have at least 1 seeded unit")

	found := false
	for _, item := range list.Data {
		if DataItemField(item, "id") == SeedUnitID {
			found = true
			break
		}
	}
	assert.True(t, found, "Seeded unit should appear in list")
}

func TestUnits_ListResponseShape(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(unitsPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	AssertResponseBodyValid(t, body)

	list := parseJSON(body)
	data, ok := list["data"].([]any)
	require.True(t, ok, "units list data should be an array")

	for _, item := range data {
		m, ok := item.(map[string]any)
		require.True(t, ok, "unit list item should be an object")
		assert.Equal(t, "unit", jsonField(m, "object"))
		assert.NotEmpty(t, jsonField(m, "id"))
		assert.NotEmpty(t, jsonField(m, "name"))
		assert.NotEmpty(t, jsonField(m, "abbreviation"))
		assert.NotEmpty(t, jsonField(m, "type"))
		assert.NotEmpty(t, jsonField(m, "created_at"))
		assert.NotEmpty(t, jsonField(m, "updated_at"))
	}
}

func TestUnits_ListPagination(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(unitsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	assert.Len(t, list.Data, 1)
}

func TestUnits_ListCursorPagination(t *testing.T) {
	t.Parallel()
	page1, _, err := apiClient.GetList(unitsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	require.Len(t, page1.Data, 1)

	if !page1.PageInfo.HasNextPage {
		t.Skip("Not enough units for pagination test")
		return
	}
	require.NotNil(t, page1.PageInfo.NextCursor, "next_cursor should be set when has_next_page is true")

	page1ID := DataItemField(page1.Data[0], "id")
	assert.NotEmpty(t, page1ID)

	page2, _, err := apiClient.GetList(unitsPath, url.Values{
		"limit":  {"1"},
		"cursor": {*page1.PageInfo.NextCursor},
	})
	require.NoError(t, err)
	require.Len(t, page2.Data, 1)

	page2ID := DataItemField(page2.Data[0], "id")
	assert.NotEqual(t, page1ID, page2ID, "Page 2 should return a different item than page 1")
}

func TestUnits_ListSearchByName(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(unitsPath, url.Values{"q": {"Pair"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Search for 'Pair' should return at least 1 result")

	for _, item := range list.Data {
		m := parseJSON(item)
		name := strings.ToLower(jsonField(m, "name"))
		abbr := strings.ToLower(jsonField(m, "abbreviation"))
		assert.True(t,
			strings.Contains(name, "pair") || strings.Contains(abbr, "pair"),
			"Search result (name=%q, abbreviation=%q) should contain 'pair'",
			jsonField(m, "name"), jsonField(m, "abbreviation"),
		)
	}
}

func TestUnits_ListSearchByAbbreviation(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(unitsPath, url.Values{"q": {"pr"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Search for 'pr' should return at least 1 result")

	for _, item := range list.Data {
		m := parseJSON(item)
		name := strings.ToLower(jsonField(m, "name"))
		abbr := strings.ToLower(jsonField(m, "abbreviation"))
		assert.True(t,
			strings.Contains(name, "pr") || strings.Contains(abbr, "pr"),
			"Search result (name=%q, abbreviation=%q) should contain 'pr'",
			jsonField(m, "name"), jsonField(m, "abbreviation"),
		)
	}
}

func TestUnits_ListSearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(unitsPath, url.Values{"q": {"zzzznotaunit99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

func TestUnits_ListFilterByType(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(unitsPath, url.Values{"type": {"quantity"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Should have at least 1 quantity unit")

	for _, item := range list.Data {
		m := parseJSON(item)
		assert.Equal(t, "quantity", jsonField(m, "type"), "All results should have type=quantity")
	}
}

// --- Validate ---

func TestUnits_ValidateKnown(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Put(unitsValidatePath, map[string]any{
		"unit_map": map[string]string{
			"quantity": "ea",
			"weight":   "dz",
		},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	m := parseJSON(body)
	assert.Equal(t, "map", jsonField(m, "object"))

	units, ok := m["units"].(map[string]any)
	require.True(t, ok, "units should be a map")
	assert.Len(t, units, 2)

	for key, val := range units {
		unit, ok := val.(map[string]any)
		require.True(t, ok, "unit %s should be an object", key)
		assert.Equal(t, "unit", jsonField(unit, "object"))
		assert.NotEmpty(t, jsonField(unit, "id"))
		assert.NotEmpty(t, jsonField(unit, "name"))
		assert.NotEmpty(t, jsonField(unit, "abbreviation"))
	}
}

func TestUnits_ValidateUnknown(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Put(unitsValidatePath, map[string]any{
		"unit_map": map[string]string{
			"thing": "zzz_nonexistent",
		},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	m := parseJSON(body)
	units, ok := m["units"].(map[string]any)
	require.True(t, ok, "units should be a map")
	assert.Nil(t, units["thing"], "Unknown abbreviation should map to null")
}

func TestUnits_ValidateMixed(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Put(unitsValidatePath, map[string]any{
		"unit_map": map[string]string{
			"known":   "ea",
			"unknown": "zzz_fake",
		},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	m := parseJSON(body)
	units, ok := m["units"].(map[string]any)
	require.True(t, ok, "units should be a map")

	known, ok := units["known"].(map[string]any)
	assert.True(t, ok, "known abbreviation should resolve to a unit object")
	assert.Equal(t, "unit", jsonField(known, "object"))

	assert.Nil(t, units["unknown"], "Unknown abbreviation should be null")
}
