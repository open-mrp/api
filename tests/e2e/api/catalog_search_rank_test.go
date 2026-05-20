//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCatalogList_SearchRankOrder_Table(t *testing.T) {
	t.Parallel()

	skuTopLevel := func(row map[string]any) string {
		return jsonField(row, "sku")
	}
	skuFromIncludedItem := func(row map[string]any) string {
		it := jsonObject(row, "item")
		if it == nil {
			return ""
		}
		return jsonField(it, "sku")
	}

	cases := []struct {
		name         string
		path         string
		query        url.Values
		expectedSKUs []string
		skuFromRow   func(map[string]any) string
	}{
		{
			name:  "items_parts",
			path:  itemsPath,
			query: url.Values{"q": {SeedSearchRankQuery}, "types": {"part"}, "limit": {"50"}},
			expectedSKUs: []string{
				SeedPartSearchRankExactSKU,
				SeedPartSearchRankTokenSKU,
				SeedPartSearchRankPrefixSKU,
				SeedPartSearchRankLooseSKU,
			},
			skuFromRow: skuTopLevel,
		},
		{
			name:  "parts",
			path:  partsPath,
			query: url.Values{"q": {SeedSearchRankQuery}, "include": {"item"}, "limit": {"50"}},
			expectedSKUs: []string{
				SeedPartSearchRankExactSKU,
				SeedPartSearchRankTokenSKU,
				SeedPartSearchRankPrefixSKU,
				SeedPartSearchRankLooseSKU,
			},
			skuFromRow: skuFromIncludedItem,
		},
		{
			name:  "materials",
			path:  materialsPath,
			query: url.Values{"q": {SeedSearchRankQuery}, "include": {"item"}, "limit": {"50"}},
			expectedSKUs: []string{
				SeedMaterialSearchRankTokenSKU,
				SeedMaterialSearchRankPrefixSKU,
				SeedMaterialSearchRankLooseSKU,
			},
			skuFromRow: skuFromIncludedItem,
		},
		{
			name:  "products",
			path:  productsPath,
			query: url.Values{"q": {SeedSearchRankQuery}, "include": {"item"}, "limit": {"50"}},
			expectedSKUs: []string{
				SeedProductSearchRankTokenSKU,
				SeedProductSearchRankPrefixSKU,
				SeedProductSearchRankLooseSKU,
			},
			skuFromRow: skuFromIncludedItem,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			list, _, err := apiClient.GetList(tc.path, tc.query)
			require.NoError(t, err)
			assertSearchRankOrder(t, list.Data, tc.expectedSKUs, tc.skuFromRow)
		})
	}
}
