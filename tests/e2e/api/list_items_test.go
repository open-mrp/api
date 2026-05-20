//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestListItems_Pagination(t *testing.T) {
	t.Parallel()
	page1, _, err := apiClient.GetList(itemsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	require.Len(t, page1.Data, 1)
	require.True(t, page1.PageInfo.HasNextPage, "seeded catalog should have more than one item for pagination")
	require.NotNil(t, page1.PageInfo.NextPageURL)

	page2, _, err := apiClient.GetListFromPageURL(page1.PageInfo.NextPageURL)
	require.NoError(t, err)
	require.Len(t, page2.Data, 1)

	id1 := DataItemField(page1.Data[0], "id")
	id2 := DataItemField(page2.Data[0], "id")
	assert.NotEqual(t, id1, id2, "cursor pages should return different items")
}

func TestListItems_FilterByProductLine(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(itemsPath, url.Values{
		"product_line_ids": {SeedProductLineID},
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Filter by Socks product line should return at least 1 result")
}

func TestListItems_FilterByProductLine_NoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(itemsPath, url.Values{
		"product_line_ids": {"pdln_00000000000000000000000000"},
	})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "Nonsense product line filter should return empty data")
}

func TestListItems_FilterByCustomer(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(itemsPath, url.Values{
		"customer_ids": {SeedCustomerAccountID},
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Filter by seeded customer should return at least 1 result (customer has Socks product line access)")
}

func TestListItems_FilterByCustomer_NoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(itemsPath, url.Values{
		"customer_ids": {"ac_00000000000000000000000000"},
	})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "Nonsense customer filter should return empty data")
}

func TestListItems_FilterByCategoryWithPagination(t *testing.T) {
	t.Parallel()

	// Page 1: limit=1 with category filter — verifies filters apply on first page.
	page1, _, err := apiClient.GetList(itemsPath, url.Values{
		"category_ids": {SeedItemCategoryID},
		"limit":        {"1"},
	})
	require.NoError(t, err)
	require.Len(t, page1.Data, 1, "First page with category filter should have exactly 1 item")

	if !page1.PageInfo.HasNextPage || page1.PageInfo.NextPageURL == nil {
		// Only one item in the category — still confirms the filter worked.
		return
	}

	// Page 2: follows next_page_url — verifies filters persist across pages.
	page2, _, err := apiClient.GetListFromPageURL(page1.PageInfo.NextPageURL)
	require.NoError(t, err)
	require.Len(t, page2.Data, 1, "Second page with category filter should have exactly 1 item")

	id1 := DataItemField(page1.Data[0], "id")
	id2 := DataItemField(page2.Data[0], "id")
	assert.NotEqual(t, id1, id2, "Cursor pages should return different items")
}

// fetchProductionStepInStepIDs returns parent step IDs from GET /production-steps/{id}?include=in_steps.
// _parent_child_production_steps stores parent→child as (A,B); in_steps are rows where this step is B.
func fetchProductionStepInStepIDs(t *testing.T, productionStepID string) []string {
	t.Helper()
	path := "/v1/operations/production-steps/" + productionStepID
	status, body, err := apiClient.GetListRaw(path, url.Values{"include": {"in_steps"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	root := parseJSON(body)
	inSteps := jsonObject(root, "in_steps")
	require.NotNil(t, inSteps, "expected in_steps when include=in_steps")
	rawData, ok := inSteps["data"]
	if !ok || rawData == nil {
		return nil
	}
	arr, ok := rawData.([]any)
	if !ok {
		return nil
	}
	var ids []string
	for _, el := range arr {
		obj, ok := el.(map[string]any)
		if !ok {
			continue
		}
		if id := jsonField(obj, "id"); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func collectPagedItemSKUs(t *testing.T, baseParams url.Values) map[string]struct{} {
	t.Helper()
	skus := map[string]struct{}{}
	var nextPageURL *string
	for {
		var list *ListResponse
		var err error
		if nextPageURL == nil {
			list, _, err = apiClient.GetList(itemsPath, baseParams)
		} else {
			list, _, err = apiClient.GetListFromPageURL(nextPageURL)
		}
		require.NoError(t, err)
		for _, item := range list.Data {
			skus[DataItemField(item, "sku")] = struct{}{}
		}
		if !list.PageInfo.HasNextPage || list.PageInfo.NextPageURL == nil {
			break
		}
		nextPageURL = list.PageInfo.NextPageURL
	}
	return skus
}

func TestListItems_SubassemblyFilterInitialOnly_ReturnsInitialPartsOnly(t *testing.T) {
	t.Parallel()

	// Prove seeded edges match API semantics (A=parent, B=child). If rows were swapped so tests on SKUs
	// still accidentally lined up, parent/child via in_steps would disagree with shared/db/seed/0009_production.sql comments.
	sewParents := fetchProductionStepInStepIDs(t, SeedSewLargeProductionStepID)
	assert.Contains(t, sewParents, SeedProductionStepID, "Sew Large Sock must list Knit Large Sock as in_step")
	knitParents := fetchProductionStepInStepIDs(t, SeedProductionStepID)
	assert.Empty(t, knitParents, "Knit Large Sock must have no in_steps (graph root)")

	washSmParents := fetchProductionStepInStepIDs(t, SeedWashSmallProductionStepID)
	assert.Contains(t, washSmParents, SeedKnitSmallProductionStepID, "Wash Small Sock must list Knit Small Sock as in_step")
	knitSmParents := fetchProductionStepInStepIDs(t, SeedKnitSmallProductionStepID)
	assert.Empty(t, knitSmParents, "Knit Small Sock must have no in_steps (graph root)")

	allSKUs := collectPagedItemSKUs(t, url.Values{
		"types":              {"part"},
		"subassembly_filter": {"all"},
		"limit":              {"500"},
	})
	require.Contains(t, allSKUs, SeedLknItemSKU, "seed baseline (all) must include LKN")
	require.Contains(t, allSKUs, SeedSknItemSKU, "seed baseline (all) must include SKN")
	require.Contains(t, allSKUs, SeedLsnItemSKU, "seed baseline (all) must include downstream part LSN")

	initialSKUs := collectPagedItemSKUs(t, url.Values{
		"types":              {"part"},
		"subassembly_filter": {"initial_only"},
		"limit":              {"500"},
	})

	assert.Contains(t, initialSKUs, SeedLknItemSKU, "LKN (Large Knitted Sock) must appear in initial_only results")
	assert.Contains(t, initialSKUs, SeedSknItemSKU, "SKN (Small Knitted Sock) must appear in initial_only results")
	assert.NotContains(t, initialSKUs, SeedLsnItemSKU, "LSN (Large Sewn Sock) must NOT appear: produced downstream of root knit step")

	for sku := range initialSKUs {
		assert.Contains(t, allSKUs, sku, "initial_only result SKU %q must exist in unfiltered part list", sku)
	}
	assert.Less(t, len(initialSKUs), len(allSKUs), "initial_only must return strictly fewer parts than subassembly_filter=all when downstream parts exist")
}
