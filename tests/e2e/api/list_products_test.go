//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const productsPath = "/v1/catalog/products"

func TestListProducts_ReturnsSeededData(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(productsPath, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Should have at least 1 seeded product")
}

func TestListProducts_FilterByProductLine(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(productsPath, url.Values{
		"product_line_ids": {SeedProductLineID},
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Filter by Socks product line should return at least 1 result")
}

func TestListProducts_SearchBySKU(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(productsPath, url.Values{"q": {"SCK"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Search for 'SCK' should return at least 1 result")
}

func TestListProducts_SearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(productsPath, url.Values{"q": {"zzzznotaproduct99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "Nonsense search should return empty data")
}

func TestListProducts_FilterByProductLine_NoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(productsPath, url.Values{
		"product_line_ids": {"pdln_00000000000000000000000000"},
	})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "Nonsense product line filter should return empty data")
}

func TestListProducts_Pagination(t *testing.T) {
	t.Parallel()
	page1, _, err := apiClient.GetList(productsPath, url.Values{"limit": {"2"}})
	require.NoError(t, err)
	assert.LessOrEqual(t, len(page1.Data), 2, "Limit=2 should return at most 2 items")

	if !page1.PageInfo.HasNextPage {
		t.Skip("Not enough products for pagination test")
		return
	}

	page2, _, err := apiClient.GetList(productsPath, url.Values{
		"limit":  {"2"},
		"cursor": {*page1.PageInfo.NextCursor},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, page2.Data, "Page 2 should have data")
	assert.True(t, page2.PageInfo.HasPrevPage, "Page 2 should have prev_page=true")
}
