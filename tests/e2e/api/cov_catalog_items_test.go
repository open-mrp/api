//go:build e2e

package api_test

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file closes e2e gaps for the catalog_items group (/v1/catalog/items)
// documented in the catalog_items coverage task: not-found paths for the
// three action endpoints, the category type-mismatch validation error, the
// supplier_id / starts_at / ends_at list filters, the subassembly_filter
// invalid-enum-value path, a direct attribute_ids filter check, and
// Idempotency-Key header replay on the change-category action endpoint.
//
// Baseline CRUD/response-shape/list/include coverage already lives in
// crud_items_test.go, list_items_test.go, crud_item_inventory_test.go, and
// array_filter_union_test.go — this file only adds what was missing there.

// ──────────────────────────────────────────────
// Change Category — validation & not-found
// ──────────────────────────────────────────────

// TestCovCatalogItems_ChangeCategory_TypeMismatch reassigns a product-type
// item (SeedItemID) to a material_category. This is safe to run directly
// against the seed item: the request is rejected by validation before any
// write happens, so SeedItemID's category is never actually mutated (verified
// below via a follow-up GET).
func TestCovCatalogItems_ChangeCategory_TypeMismatch(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Put(
		itemsPath+"/"+SeedItemID+"/category/"+SeedMaterialCategoryID,
		nil,
	)
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "category_id")

	getStatus, getBody, err := apiClient.GetListRaw(itemsPath+"/"+SeedItemID, url.Values{"include": {"category"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	cat := jsonObject(got, "category")
	require.NotNil(t, cat, "category should still be present with ?include=category")
	assert.Equal(t, SeedItemCategoryID, jsonField(cat, "id"),
		"a rejected type-mismatch change-category request must not mutate the item's category")
}

// TestCovCatalogItems_ChangeCategory_StrandsAttribute covers the attribute guard on a category move: the move keeps the item's attributes, so a target category that doesn't carry their properties is rejected rather than left holding an attribute it can't describe. Shipping carries no properties at all, and the product created here carries the seeded Color attribute.
func TestCovCatalogItems_ChangeCategory_StrandsAttribute(t *testing.T) {
	t.Parallel()

	body := validProductBody(uniqueName("e2e-covitems-movestrand"))
	body["attribute_ids"] = []string{SeedAttributeID}
	created := createAndCleanup(t, productsPath, body)
	productID := jsonField(created, "id")
	require.NotEmpty(t, productID)

	getStatus, getBody, err := apiClient.GetListRaw(productsPath+"/"+productID, url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	itemID := jsonField(jsonObject(parseJSON(getBody), "item"), "id")
	require.NotEmpty(t, itemID)

	// Moved to a category carrying no properties, so the Color attribute already on the item would be stranded. Moving to another Color-carrying category is a legal move and would not exercise the gate.
	status, respBody, err := apiClient.Put(itemsPath+"/"+itemID+"/category/"+SeedPropertylessItemCategoryID, nil)
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)

	errObj := requireErrorResponse(t, respBody, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "category_id")

	// The rejected move must leave the item where it was.
	checkStatus, checkBody, err := apiClient.GetListRaw(itemsPath+"/"+itemID, url.Values{"include": {"category"}})
	require.NoError(t, err)
	requireStatus(t, 200, checkStatus, checkBody)
	cat := jsonObject(parseJSON(checkBody), "category")
	require.NotNil(t, cat)
	assert.Equal(t, SeedItemCategoryID, jsonField(cat, "id"), "a rejected change-category request must not mutate the item's category")
}

// TestCovCatalogItems_Get_NotFound covers GET /v1/catalog/items/{id} for a
// syntactically valid but nonexistent item id.
func TestCovCatalogItems_Get_NotFound(t *testing.T) {
	t.Parallel()
	fakeID := "it_01zzzzzzzzzzzzzzzzzzzzzzz"
	status, body, err := apiClient.GetListRaw(itemsPath+"/"+fakeID, nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// TestCovCatalogItems_ChangeCategory_ItemNotFound covers PUT
// /v1/catalog/items/{id}/category/{category_id} for a nonexistent item id.
func TestCovCatalogItems_ChangeCategory_ItemNotFound(t *testing.T) {
	t.Parallel()
	fakeItemID := "it_01zzzzzzzzzzzzzzzzzzzzzzz"
	status, body, err := apiClient.Put(itemsPath+"/"+fakeItemID+"/category/"+SeedItemCategoryID, nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// TestCovCatalogItems_ChangeCategory_CategoryNotFound covers PUT
// /v1/catalog/items/{id}/category/{category_id} for a nonexistent category
// id. SeedItemID is used directly: the request fails before any write, so
// its category is never mutated.
func TestCovCatalogItems_ChangeCategory_CategoryNotFound(t *testing.T) {
	t.Parallel()
	fakeCategoryID := "itcg_01zzzzzzzzzzzzzzzzzzzzzzz"
	status, body, err := apiClient.Put(itemsPath+"/"+SeedItemID+"/category/"+fakeCategoryID, nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// ──────────────────────────────────────────────
// Attributes — not-found & nonexistent-attribute-id
// ──────────────────────────────────────────────

// TestCovCatalogItems_AddAttribute_ItemNotFound covers PUT
// /v1/catalog/items/{id}/attributes/{attribute_id} for a nonexistent item id.
func TestCovCatalogItems_AddAttribute_ItemNotFound(t *testing.T) {
	t.Parallel()
	fakeItemID := "it_01zzzzzzzzzzzzzzzzzzzzzzz"
	status, body, err := apiClient.Put(itemsPath+"/"+fakeItemID+"/attributes/"+SeedAttributeID, nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// TestCovCatalogItems_RemoveAttribute_ItemNotFound covers DELETE
// /v1/catalog/items/{id}/attributes/{attribute_id} for a nonexistent item id.
func TestCovCatalogItems_RemoveAttribute_ItemNotFound(t *testing.T) {
	t.Parallel()
	fakeItemID := "it_01zzzzzzzzzzzzzzzzzzzzzzz"
	status, body, err := apiClient.Delete(itemsPath + "/" + fakeItemID + "/attributes/" + SeedAttributeID)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// TestCovCatalogItems_AddAttribute_NonexistentAttributeID exercises AddItemAttribute with a syntactically valid but nonexistent attribute id. The `_item_attributes` join table has no FK constraint on attribute_id, so the id is checked in the account scope before the row is written; without that check the request would write an orphaned join row that never surfaces via ?include=attributes.
func TestCovCatalogItems_AddAttribute_NonexistentAttributeID(t *testing.T) {
	t.Parallel()
	_, itemID := newProductItemIDs(t, "e2e-covitems-badattr")
	fakeAttributeID := "at_01zzzzzzzzzzzzzzzzzzzzzzz"

	status, body, err := apiClient.Put(
		itemsPath+"/"+itemID+"/attributes/"+fakeAttributeID+"?include=attributes",
		nil,
	)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// TestCovCatalogItems_AddAttribute_PropertyNotOnCategory covers the category gate: an attribute may only be assigned to an item whose category carries the attribute's property. SeedAttributeID belongs to the Color property, and the category here carries no properties at all — every other product category in the seed carries Color, so pairing against one of those would be a legal assignment rather than the rejection this is testing.
func TestCovCatalogItems_AddAttribute_PropertyNotOnCategory(t *testing.T) {
	t.Parallel()

	body := validProductBody(uniqueName("e2e-covitems-attrcat"))
	body["category_id"] = SeedPropertylessItemCategoryID
	created := createAndCleanup(t, productsPath, body)
	productID := jsonField(created, "id")
	require.NotEmpty(t, productID)

	getStatus, getBody, err := apiClient.GetListRaw(productsPath+"/"+productID, url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	itemID := jsonField(jsonObject(parseJSON(getBody), "item"), "id")
	require.NotEmpty(t, itemID)

	status, respBody, err := apiClient.Put(itemsPath+"/"+itemID+"/attributes/"+SeedAttributeID, nil)
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)

	errObj := requireErrorResponse(t, respBody, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "attribute_id")
}

// ──────────────────────────────────────────────
// List — supplier_id filter
// ──────────────────────────────────────────────

// TestCovCatalogItems_ListItems_FilterBySupplier covers the supplier_id
// filter, matching items via the supplier_material join. SeedMaterialItemID
// (YRN-001) is linked to SeedSupplierAccountID via SeedSupplierMaterialID.
func TestCovCatalogItems_ListItems_FilterBySupplier(t *testing.T) {
	t.Parallel()
	assertListContainsID(t, itemsPath, url.Values{"supplier_id": {SeedSupplierAccountID}}, SeedMaterialItemID)
}

func TestCovCatalogItems_ListItems_FilterBySupplier_NoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(itemsPath, url.Values{
		"supplier_id": {"ac_00000000000000000000000000"},
	})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "Nonsense supplier_id filter should return empty data")
}

// ──────────────────────────────────────────────
// List — starts_at / ends_at filters
// ──────────────────────────────────────────────
//
// ListItemsRequest.StartDate / EndDate are *time.Time (not *string like the
// sales-orders list), so the query decoder requires a full RFC3339 value —
// a bare YYYY-MM-DD ("2000-01-01") fails to parse and 400s. Verified live:
// a date-only value returns 400 parameter_invalid; RFC3339 values are
// required.

func TestCovCatalogItems_ListItems_FilterByEndDate_ExcludesEverything(t *testing.T) {
	t.Parallel()
	// ends_at is inclusive on created_at; nothing predates 2000.
	list, _, err := apiClient.GetList(itemsPath, url.Values{
		"ends_at": {"2000-01-01T00:00:00Z"},
	})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "no items exist before 2000-01-01")
}

func TestCovCatalogItems_ListItems_FilterByStartDate_ExcludesEverything(t *testing.T) {
	t.Parallel()
	// A far-future starts_at (20 years out) is safely beyond every seeded
	// and freshly-created item's created_at, including the +9y filter-coverage
	// seed rows used elsewhere in the suite.
	farFuture := time.Now().AddDate(20, 0, 0).Format(time.RFC3339)
	list, _, err := apiClient.GetList(itemsPath, url.Values{
		"starts_at": {farFuture},
	})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "a 20-years-out starts_at should exclude every item")
}

// ──────────────────────────────────────────────
// List — attribute_ids filter (direct)
// ──────────────────────────────────────────────

// TestCovCatalogItems_ListItems_FilterByAttribute directly exercises
// attribute_ids (previously only covered indirectly via
// TestArrayFilters_UnionExclusion). SeedMaterialItemID carries
// SeedAttributeID at seed time.
func TestCovCatalogItems_ListItems_FilterByAttribute(t *testing.T) {
	t.Parallel()
	assertListContainsID(t, itemsPath, url.Values{"attribute_ids": {SeedAttributeID}}, SeedMaterialItemID)
}

func TestCovCatalogItems_ListItems_FilterByAttribute_NoResults(t *testing.T) {
	t.Parallel()
	// Any id nothing in the suite ever joins against works; this one is kept distinct from the all-zeros sentinel other catalog tests probe with.
	list, _, err := apiClient.GetList(itemsPath, url.Values{
		"attribute_ids": {"at_7h3q9k2m4n6p8r1s5t0v2wxyz3"},
	})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "attribute_ids filter for an attribute id no item is linked to should return empty data")
}

// ──────────────────────────────────────────────
// List — subassembly_filter invalid value
// ──────────────────────────────────────────────

// An invalid subassembly_filter names the query parameter the caller sent, not the Go field behind it.
func TestCovCatalogItems_ListItems_SubassemblyFilterInvalidValue(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(itemsPath, url.Values{"subassembly_filter": {"bogus_value"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "subassembly_filter")
}

// ──────────────────────────────────────────────
// Change Category — Idempotency-Key header replay
// ──────────────────────────────────────────────

// TestCovCatalogItems_ChangeCategory_IdempotencyKeyReplay sends the same
// Idempotency-Key header twice on PUT .../category/{id} via DoFull (not the
// bodyless Put wrapper) to exercise the header-replay path directly.
//
// Discovery:
// the api-gateway's IdempotencyMiddleware only intercepts POST and PATCH
// (see services/api-gateway/internal/middleware/idempotency_middleware.go,
// `r.Method != http.MethodPost && r.Method != http.MethodPatch` short-
// circuits to next.ServeHTTP for every other verb without even stashing the
// key on the request context). Because ChangeItemCategory/AddItemAttribute/
// RemoveItemAttribute are exposed as PUT/DELETE, the client's Idempotency-Key
// header is silently ignored at the gateway layer for these action
// endpoints and the "Idempotent-Replayed" response header never appears,
// even though core-service's item_service.go independently implements
// UpsertIdempotencyKey/CacheSuccessResponse for ChangeItemCategory. This
// contradicts the task's assumption that the header-replay mechanism is
// exercised by these action endpoints; asserted here as the real observed
// behavior (no header on replay) alongside the change-category action's own
// business-level idempotency (re-assigning the same category is a same-body
// no-op), and flagged for follow-up rather than silently accepted as
// "working idempotency".
func TestCovCatalogItems_ChangeCategory_IdempotencyKeyReplay(t *testing.T) {
	t.Parallel()
	_, itemID := newProductItemIDs(t, "e2e-covitems-idemkey")

	path := itemsPath + "/" + itemID + "/category/" + SeedItemCategoryID
	idemKey := newIdempotencyKey()

	resp1, err := apiClient.DoFull(http.MethodPut, path, nil, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, resp1.StatusCode, resp1.Body)
	assert.NotEqual(t, "true", resp1.Header.Get("Idempotent-Replayed"),
		"first request should not have Idempotent-Replayed header")

	resp2, err := apiClient.DoFull(http.MethodPut, path, nil, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, resp2.StatusCode, resp2.Body)

	// BUG (see comment above): PUT bypasses the gateway idempotency
	// middleware entirely, so no Idempotent-Replayed header is ever set for
	// this action endpoint, even on an exact key replay.
	assert.NotEqual(t, "true", resp2.Header.Get("Idempotent-Replayed"),
		"BUG: change-category is a PUT endpoint, so the gateway's IdempotencyMiddleware "+
			"(POST/PATCH only) never marks a replayed request, regardless of a repeated Idempotency-Key")

	// The action's own business-level idempotency still holds regardless:
	// re-assigning the same category is a no-op, so the two responses must
	// be byte-for-byte identical (no duplicate rate-unit side effects).
	assert.JSONEq(t, string(resp1.Body), string(resp2.Body),
		"re-assigning the same category twice must produce identical responses (no duplicate side effects)")
}
