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
	assert.Nil(t, got["related"], "related should be null until one of its members is included")
	assert.Nil(t, got["lines"], "lines should be null without include")
	assert.Nil(t, got["customer"], "customer should be null without include")
	assert.NotNil(t, got["priority"], "priority should always be present")
	assert.NotNil(t, got["totals"], "totals is a base scalar, always present")
	assert.NotNil(t, got["ship_to"], "ship_to is carried inline from the order")
}

func TestPicks_IncludeRelatedSalesOrder(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(picksPath+"/"+SeedPickID, url.Values{"include": {"related.sales_order"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	related := jsonObject(got, "related")
	require.NotNil(t, related, "related should be present with ?include=related.sales_order")
	so := jsonObject(related, "sales_order")
	require.NotNil(t, so, "related.sales_order should be present")
	assert.Equal(t, "record", jsonField(so, "object"), "related members are Record references")
	assert.Equal(t, "sales_order", jsonField(so, "type"))
	assert.NotEmpty(t, jsonField(so, "number"), "the record carries the order number for its link")
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

func TestPicks_List_SalesOrderNullWithoutInclude(t *testing.T) {
	t.Parallel()
	list, status, err := apiClient.GetList(picksPath, nil)
	require.NoError(t, err)
	require.Equal(t, 200, status, "picks list should return 200")
	require.GreaterOrEqual(t, len(list.Data), 1, "at least one pick must be seeded")

	var first map[string]any
	require.NoError(t, json.Unmarshal(list.Data[0], &first))
	assert.Nil(t, first["related"], "related should be null without include on list")
	assert.Nil(t, first["customer"], "customer should be null without include on list items")
}

func TestPicks_List_IncludeRelatedSalesOrder(t *testing.T) {
	t.Parallel()
	list, status, err := apiClient.GetList(picksPath, url.Values{"include": {"related.sales_order"}})
	require.NoError(t, err)
	require.Equal(t, 200, status, "picks list should return 200")
	require.GreaterOrEqual(t, len(list.Data), 1, "at least one pick must be seeded")

	var first map[string]any
	require.NoError(t, json.Unmarshal(list.Data[0], &first))
	related, ok := first["related"].(map[string]any)
	require.True(t, ok, "related should be an object with ?include=related.sales_order")
	so, ok := related["sales_order"].(map[string]any)
	require.True(t, ok, "related.sales_order should be an object")
	assert.Equal(t, "record", so["object"])
}
