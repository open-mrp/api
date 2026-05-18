//go:build e2e

package api_test

import (
	"net/url"
	"strings"
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
	// The product itself has no top-level SKU; the search matches on item.sku or
	// item.description. Use include=item so we can assert on the matched field.
	list, _, err := apiClient.GetList(productsPath, url.Values{"q": {"SCK"}, "include": {"item"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Search for 'SCK' should return at least 1 result")

	for _, item := range list.Data {
		m := parseJSON(item)
		itemObj := jsonObject(m, "item")
		require.NotNil(t, itemObj, "item should be present with ?include=item")
		sku := strings.ToUpper(jsonField(itemObj, "sku"))
		desc := strings.ToLower(jsonField(itemObj, "description"))
		assert.True(t,
			strings.Contains(sku, "SCK") || strings.Contains(desc, "sck"),
			"Search result (sku=%q, description=%q) should match 'SCK' in item.sku or item.description",
			jsonField(itemObj, "sku"), jsonField(itemObj, "description"),
		)
	}
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

func TestListProducts_SearchRanksExactSKUBeforeSubstring(t *testing.T) {
	t.Parallel()
	exactSKU := uniqueName("e2e-rank-ex")
	longSKU := "Z" + exactSKU + "Z"

	respLong, err := apiClient.PostFull(productsPath, validProductBody(longSKU), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, respLong.StatusCode, respLong.Body)
	longID := jsonField(parseJSON(respLong.Body), "id")
	require.NotEmpty(t, longID)
	t.Cleanup(func() { apiClient.Delete(productsPath + "/" + longID) })

	respExact, err := apiClient.PostFull(productsPath, validProductBody(exactSKU), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, respExact.StatusCode, respExact.Body)
	exactID := jsonField(parseJSON(respExact.Body), "id")
	require.NotEmpty(t, exactID)
	t.Cleanup(func() { apiClient.Delete(productsPath + "/" + exactID) })

	list, _, err := apiClient.GetList(productsPath, url.Values{"q": {exactSKU}, "include": {"item"}, "limit": {"10"}})
	require.NoError(t, err)
	require.NotEmpty(t, list.Data, "search should return both products")

	first := parseJSON(list.Data[0])
	itemObj := jsonObject(first, "item")
	require.NotNil(t, itemObj)
	assert.Equal(t, exactSKU, jsonField(itemObj, "sku"), "exact SKU match should sort before substring-only match")
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

	require.NotNil(t, page1.PageInfo.NextPageURL)

	page2, _, err := apiClient.GetListFromPageURL(page1.PageInfo.NextPageURL)
	require.NoError(t, err)
	assert.NotEmpty(t, page2.Data, "Page 2 should have data")
	assert.True(t, page2.PageInfo.HasPrevPage, "Page 2 should have prev_page=true")
}
