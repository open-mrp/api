//go:build e2e

package api_test

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const picksPath = "/v1/operations/picks"

func TestPicks_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(picksPath+"/"+SeedPickID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["sales_order"], "sales_order should be null without include")
	assert.Nil(t, got["lines"], "lines should be null without include")
	assert.Nil(t, got["departments"], "departments should be null without include")
	assert.Nil(t, got["customer"], "customer should be null without include")
	assert.NotNil(t, got["priority"], "priority should always be present")
}

func TestPicks_IncludeSalesOrder(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(picksPath+"/"+SeedPickID, url.Values{"include": {"sales_order"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	so := jsonObject(got, "sales_order")
	require.NotNil(t, so, "sales_order should be present with ?include=sales_order")
	assert.Equal(t, "sales_order", jsonField(so, "object"))
}

func TestPicks_IncludeLines(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(picksPath+"/"+SeedPickID, url.Values{"include": {"lines"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	lines := jsonObject(got, "lines")
	require.NotNil(t, lines, "lines should be present with ?include=lines")
	assert.Equal(t, "list", jsonField(lines, "object"))
}

func TestPicks_IncludeDepartments(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(picksPath+"/"+SeedPickID, url.Values{"include": {"departments"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	depts := jsonObject(got, "departments")
	require.NotNil(t, depts, "departments should be present with ?include=departments")
	assert.Equal(t, "list", jsonField(depts, "object"))
}

func TestPicks_List_SalesOrderNullWithoutInclude(t *testing.T) {
	t.Parallel()
	list, status, err := apiClient.GetList(picksPath, nil)
	require.NoError(t, err)
	require.Equal(t, 200, status, "picks list should return 200")
	require.GreaterOrEqual(t, len(list.Data), 1, "at least one pick must be seeded")

	var first map[string]any
	require.NoError(t, json.Unmarshal(list.Data[0], &first))
	assert.Nil(t, first["sales_order"], "sales_order should be null without include on list")
	assert.Nil(t, first["customer"], "customer should be null without include on list items")
}

func TestPicks_List_IncludeSalesOrder(t *testing.T) {
	t.Parallel()
	list, status, err := apiClient.GetList(picksPath, url.Values{"include": {"sales_order"}})
	require.NoError(t, err)
	require.Equal(t, 200, status, "picks list should return 200")
	require.GreaterOrEqual(t, len(list.Data), 1, "at least one pick must be seeded")

	var first map[string]any
	require.NoError(t, json.Unmarshal(list.Data[0], &first))
	so, ok := first["sales_order"].(map[string]any)
	require.True(t, ok, "sales_order should be an object with ?include=sales_order")
	assert.Equal(t, "sales_order", so["object"])
}
