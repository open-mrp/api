//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

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

// The search reaches a description: the word is in this test's own item's description and nowhere else.
func TestListItems_SearchByDescription(t *testing.T) {
	t.Parallel()
	word := searchToken("e2edesc")
	body := forgeItemBody("part", uniqueName("e2e-desc-search"))
	body["description"] = "Ribbed " + word + " cuff"
	itemID := jsonField(createForgeItem(t, itemsPath, body), "id")

	list, _, err := apiClient.GetList(itemsPath, url.Values{"q": {word}})
	require.NoError(t, err)
	require.Len(t, list.Data, 1, "searching a word only the item's description holds")
	assert.Equal(t, itemID, DataItemField(list.Data[0], "id"))
	assert.True(t, strings.Contains(DataItemField(list.Data[0], "description"), word))
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

func TestListItems_FilterByTypeCode_RejectsAnUnknownType(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(itemsPath, url.Values{"types": {"zzzznotatypecode99999"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

func TestListItems_SearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(itemsPath, url.Values{"q": {"zzzznotanitem99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "Nonsense search should return empty data")
}

func TestListItems_Pagination(t *testing.T) {
	t.Parallel()
	// Paginate over items this test owns (created via materials with a unique
	// SKU prefix), scoped by q, so parallel tests creating or deleting items
	// cannot shift the cursor window.
	prefix := uniqueName("e2e-item-pg")
	ids := createItemsViaMaterials(t, prefix, 2)

	assertScopedCursorPagination(t, itemsPath, url.Values{"q": {prefix}}, ids)
}

// createItemsViaMaterials creates n materials with SKUs sharing the given
// unique prefix (registering cleanup) and returns the ids of the catalog
// items they produce.
func createItemsViaMaterials(t *testing.T, prefix string, n int) []string {
	t.Helper()
	var ids []string
	for i := 0; i < n; i++ {
		sku := fmt.Sprintf("%s-%d", prefix, i)
		createStatus, createBody, err := apiClient.Post(materialsPath, validMaterialBody(sku), newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, createStatus, createBody)
		materialID := jsonField(parseJSON(createBody), "id")
		require.NotEmpty(t, materialID)
		t.Cleanup(func() { apiClient.Delete(materialsPath + "/" + materialID) })

		list, _, err := apiClient.GetList(itemsPath, url.Values{"q": {sku}})
		require.NoError(t, err)
		require.Len(t, list.Data, 1, "exactly one item should match the unique SKU %q", sku)
		ids = append(ids, DataItemField(list.Data[0], "id"))
	}
	return ids
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

	// Paginate over items this test owns (validMaterialBody puts them in
	// SeedMaterialCategoryID — materials may only use material categories), scoped
	// by both the category filter and a unique q, verifying filters persist across
	// cursor pages without interference from parallel tests mutating the shared
	// category.
	prefix := uniqueName("e2e-item-catpg")
	ids := createItemsViaMaterials(t, prefix, 2)

	assertScopedCursorPagination(t, itemsPath, url.Values{
		"category_ids": {SeedMaterialCategoryID},
		"q":            {prefix},
	}, ids)
}

// fetchProductionStepInStepIDs returns parent step IDs from GET /production-steps/{id}?include=in_steps.
// _parent_child_production_steps stores (A=downstream child, B=upstream parent); in_steps are rows where this step is A.
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
		if !list.PageInfo.HasNextPage() || list.PageInfo.NextPageURL == nil {
			break
		}
		nextPageURL = list.PageInfo.NextPageURL
	}
	return skus
}

func TestListItems_SubassemblyFilterInitialOnly_ReturnsInitialPartsOnly(t *testing.T) {
	t.Parallel()

	// Prove seeded edges match API semantics (A=downstream child, B=upstream parent). If rows were swapped so tests on SKUs
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
		"limit":              {"100"},
	})
	require.Contains(t, allSKUs, SeedLknItemSKU, "seed baseline (all) must include LKN")
	require.Contains(t, allSKUs, SeedSknItemSKU, "seed baseline (all) must include SKN")
	require.Contains(t, allSKUs, SeedLsnItemSKU, "seed baseline (all) must include downstream part LSN")

	initialSKUs := collectPagedItemSKUs(t, url.Values{
		"types":              {"part"},
		"subassembly_filter": {"initial_only"},
		"limit":              {"100"},
	})

	assert.Contains(t, initialSKUs, SeedLknItemSKU, "LKN (Large Knitted Sock) must appear in initial_only results")
	assert.Contains(t, initialSKUs, SeedSknItemSKU, "SKN (Small Knitted Sock) must appear in initial_only results")
	assert.NotContains(t, initialSKUs, SeedLsnItemSKU, "LSN (Large Sewn Sock) must NOT appear: produced downstream of root knit step")

	for sku := range initialSKUs {
		assert.Contains(t, allSKUs, sku, "initial_only result SKU %q must exist in unfiltered part list", sku)
	}
	assert.Less(t, len(initialSKUs), len(allSKUs), "initial_only must return strictly fewer parts than subassembly_filter=all when downstream parts exist")
}

// --- Forge filters ---

// listItemIDs lists items with params, scoped by the caller, and returns their ids.
func listItemIDs(t *testing.T, params url.Values) []string {
	t.Helper()
	list, _, err := apiClient.GetList(itemsPath, params)
	require.NoError(t, err)
	ids := make([]string, 0, len(list.Data))
	for _, raw := range list.Data {
		ids = append(ids, DataItemField(raw, "id"))
	}
	return ids
}

// skus matches whole SKUs regardless of case, never a substring.
func TestListItems_FilterBySKUs(t *testing.T) {
	t.Parallel()
	sku := uniqueName("e2e-item-skus")
	itemID := jsonField(createForgeItem(t, itemsPath, forgeItemBody("part", sku)), "id")
	otherID := jsonField(createForgeItem(t, itemsPath, forgeItemBody("part", sku+"-x")), "id")

	assert.Equal(t, []string{itemID}, listItemIDs(t, url.Values{"skus": {strings.ToUpper(sku)}}))
	assert.ElementsMatch(t, []string{itemID, otherID}, listItemIDs(t, url.Values{"skus": {sku, sku + "-x"}}))
	assert.Empty(t, listItemIDs(t, url.Values{"skus": {sku[:len(sku)-2]}}), "no substring match")
}

func TestListItems_FilterBySKUs_RejectsMoreThan100(t *testing.T) {
	t.Parallel()
	skus := make([]string, 101)
	for i := range skus {
		skus[i] = fmt.Sprintf("e2e-many-%d", i)
	}
	status, body, err := apiClient.GetListRaw(itemsPath, url.Values{"skus": skus})
	require.NoError(t, err)
	requireClientRejection(t, status, body)
}

// Archived items are listed by default; statuses narrows to one.
func TestListItems_FilterByStatuses(t *testing.T) {
	t.Parallel()
	prefix := uniqueName("e2e-item-status")
	activeID := jsonField(createForgeItem(t, itemsPath, forgeItemBody("part", prefix+"-a")), "id")
	archived := forgeItemBody("part", prefix+"-b")
	archived["status"] = "archived"
	archivedID := jsonField(createForgeItem(t, itemsPath, archived), "id")

	assert.ElementsMatch(t, []string{activeID, archivedID}, listItemIDs(t, url.Values{"q": {prefix}}))
	assert.Equal(t, []string{archivedID}, listItemIDs(t, url.Values{"q": {prefix}, "statuses": {"archived"}}))
	assert.Equal(t, []string{activeID}, listItemIDs(t, url.Values{"q": {prefix}, "statuses": {"active"}}))
	assert.ElementsMatch(t, []string{activeID, archivedID}, listItemIDs(t, url.Values{"q": {prefix}, "statuses": {"active", "archived"}}))

	status, body, err := apiClient.GetListRaw(itemsPath, url.Values{"statuses": {"deleted"}})
	require.NoError(t, err)
	requireClientRejection(t, status, body)
}

func TestListItems_FilterByTypes(t *testing.T) {
	t.Parallel()
	prefix := uniqueName("e2e-item-types")
	ids := map[string]string{}
	for _, itemType := range []string{"material", "part", "product", "service"} {
		ids[itemType] = jsonField(createForgeItem(t, itemsPath, forgeItemBody(itemType, prefix+"-"+itemType)), "id")
	}

	for itemType, itemID := range ids {
		assert.Equal(t, []string{itemID}, listItemIDs(t, url.Values{"q": {prefix}, "types": {itemType}}), itemType)
	}
	assert.ElementsMatch(t, []string{ids["product"], ids["service"]},
		listItemIDs(t, url.Values{"q": {prefix}, "types": {"product", "service"}}))
}

func TestListItems_FilterByPortalVisibilities(t *testing.T) {
	t.Parallel()
	prefix := uniqueName("e2e-item-vis")
	visible := forgeItemBody("product", prefix+"-v")
	visible["sales"] = map[string]any{"portal_visibility": "visible"}
	visibleID := jsonField(createForgeItem(t, itemsPath, visible), "id")
	hiddenID := jsonField(createForgeItem(t, itemsPath, forgeItemBody("product", prefix+"-h")), "id")
	// A part that isn't sold has no visibility, so neither value matches it.
	createForgeItem(t, itemsPath, forgeItemBody("part", prefix+"-p"))

	assert.Equal(t, []string{visibleID}, listItemIDs(t, url.Values{"q": {prefix}, "portal_visibilities": {"visible"}}))
	assert.Equal(t, []string{hiddenID}, listItemIDs(t, url.Values{"q": {prefix}, "portal_visibilities": {"hidden"}}))
}

func TestListItems_FilterByAttributeIDs(t *testing.T) {
	t.Parallel()
	attrID := newColorAttribute(t, "active")
	prefix := uniqueName("e2e-item-attrf")
	tagged := forgeItemBody("part", prefix+"-t")
	tagged["attribute_ids"] = []string{attrID}
	taggedID := jsonField(createForgeItem(t, itemsPath, tagged), "id")
	createForgeItem(t, itemsPath, forgeItemBody("part", prefix+"-u"))

	assert.Equal(t, []string{taggedID}, listItemIDs(t, url.Values{"q": {prefix}, "attribute_ids": {attrID}}))
}

// The seeded supplier supplies the seeded yarn.
func TestListItems_FilterBySupplierIDs(t *testing.T) {
	t.Parallel()
	assertListContainsID(t, itemsPath, url.Values{"supplier_ids": {SeedSupplierAccountID}}, SeedMaterialItemID)
	assert.Empty(t, listItemIDs(t, url.Values{"supplier_ids": {"ac_00000000000000000000000000"}}),
		"an unknown supplier supplies nothing")
}

func TestListItems_FilterByCreatedRange(t *testing.T) {
	t.Parallel()
	prefix := uniqueName("e2e-item-created")
	created := createForgeItem(t, itemsPath, forgeItemBody("part", prefix))
	itemID := jsonField(created, "id")
	createdAt, err := time.Parse(time.RFC3339Nano, jsonField(created, "created_at"))
	require.NoError(t, err)
	at := func(d time.Duration) string { return createdAt.Add(d).UTC().Format(time.RFC3339) }

	assert.Equal(t, []string{itemID}, listItemIDs(t, url.Values{"q": {prefix}, "created_after": {at(-time.Minute)}, "created_before": {at(time.Minute)}}))
	assert.Empty(t, listItemIDs(t, url.Values{"q": {prefix}, "created_after": {at(time.Hour)}}))
	assert.Empty(t, listItemIDs(t, url.Values{"q": {prefix}, "created_before": {at(-time.Hour)}}))

	status, body, err := apiClient.GetListRaw(itemsPath, url.Values{"created_after": {at(time.Hour)}, "created_before": {at(-time.Hour)}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error"), "created_before")
}

func TestListItems_LimitMax100(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(itemsPath, url.Values{"limit": {"100"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.GetListRaw(itemsPath, url.Values{"limit": {"101"}})
	require.NoError(t, err)
	requireClientRejection(t, status, body)
}

// starts_at, ends_at and supplier_id are preview.9's names; forge callers use created_after, created_before and supplier_ids.
func TestListItems_Preview9FiltersRejected(t *testing.T) {
	t.Parallel()
	for param, value := range map[string]string{
		"starts_at":   "2024-01-01T00:00:00Z",
		"ends_at":     "2024-01-01T00:00:00Z",
		"supplier_id": SeedSupplierAccountID,
	} {
		status, body, err := apiClient.GetListRaw(itemsPath, url.Values{param: {value}})
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		assertErrorParam(t, requireErrorResponse(t, body, "parameter_unknown", "invalid_request_error"), param)
	}
}
