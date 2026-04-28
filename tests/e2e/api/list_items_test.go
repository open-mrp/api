//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const itemsPath = "/v1/catalog/items"

func TestListItems_ReturnsSeededData(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(itemsPath, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Should have at least 1 seeded item")
}

func TestListItems_SearchBySKU(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(itemsPath, url.Values{"q": {SeedItemSKU}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Search for SKU %q should return at least 1 result", SeedItemSKU)

	found := false
	for _, item := range list.Data {
		sku := DataItemField(item, "sku")
		if sku == SeedItemSKU {
			found = true
			break
		}
	}
	assert.True(t, found, "Seeded item with SKU %q not found in search results", SeedItemSKU)
}

func TestListItems_SearchByDescription(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(itemsPath, url.Values{"q": {"sock"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Search for 'sock' should return at least 1 result")

	for _, item := range list.Data {
		desc := DataItemField(item, "description")
		assert.True(t,
			strings.Contains(strings.ToLower(desc), "sock"),
			"Search result description %q should contain 'sock'", desc,
		)
	}
}

func TestListItems_FilterByCategory(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(itemsPath, url.Values{
		"category_ids": {SeedItemCategoryID},
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Filter by Socks category should return at least 1 result")
}

func TestListItems_FilterByTypeCode(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(itemsPath, url.Values{
		"types": {"product"},
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Filter by type_code=product should return at least 1 result")

	for _, item := range list.Data {
		m := parseJSON(item)
		assert.Equal(t, "product", jsonField(m, "type"),
			"All results should have type=product")
	}
}

func TestListItems_FilterByCategory_NoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(itemsPath, url.Values{
		"category_ids": {"itcg_00000000000000000000000000"},
	})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "Nonsense category filter should return empty data")
}

func TestListItems_FilterByTypeCode_NoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(itemsPath, url.Values{
		"types": {"zzzznotatypecode99999"},
	})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "Nonsense type code filter should return empty data")
}

func TestListItems_SearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(itemsPath, url.Values{"q": {"zzzznotanitem99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "Nonsense search should return empty data")
}
