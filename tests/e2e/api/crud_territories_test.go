//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func territoriesPath() string {
	return "/v1/sales/accounts/" + SeedAccountID + "/territories"
}

// ──────────────────────────────────────────────
// Territory — Include Tests
// ──────────────────────────────────────────────

func TestTerritories_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(territoriesPath()+"/"+SeedTerritoryID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["sales_rep"], "sales_rep should be null without ?include=sales_rep")
	assert.Nil(t, got["product_line"], "product_line should be null without ?include=product_line")

	list, _, err := apiClient.GetList(territoriesPath(), nil)
	require.NoError(t, err)
	for _, item := range list.Data {
		m := parseJSON(item)
		assert.Nil(t, m["sales_rep"], "sales_rep should be null on list items without ?include=sales_rep")
		assert.Nil(t, m["product_line"], "product_line should be null on list items without ?include=product_line")
	}
}

func TestTerritories_IncludeSalesRep(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(territoriesPath()+"/"+SeedTerritoryID, url.Values{"include": {"sales_rep"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["sales_rep"]
	assert.True(t, ok, "sales_rep key should be present with ?include=sales_rep")
	if sr := jsonObject(got, "sales_rep"); sr != nil {
		assert.Equal(t, "account_user", jsonField(sr, "object"))
	}
}

func TestTerritories_IncludeProductLine(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(territoriesPath()+"/"+SeedTerritoryID, url.Values{"include": {"product_line"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	pl := jsonObject(got, "product_line")
	require.NotNil(t, pl, "product_line should be present with ?include=product_line")
	assert.Equal(t, "product_line", jsonField(pl, "object"))
}
