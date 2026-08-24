//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// covCatalogProductsListContainsID reports whether id appears in a list response's data.
func covCatalogProductsListContainsID(list *ListResponse, id string) bool {
	if list == nil {
		return false
	}
	for _, raw := range list.Data {
		if DataItemField(raw, "id") == id {
			return true
		}
	}
	return false
}

// ──────────────────────────────────────────────
// Create — enum validation, SKU rules, FK-not-found
// ──────────────────────────────────────────────

func TestCovCatalogProducts_Create_DuplicateSKUConflict(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-cov-prod-dupsku")
	created := createAndCleanup(t, productsPath, validProductBody(sku))
	require.NotEmpty(t, jsonField(created, "id"))

	status, body, err := apiClient.Post(productsPath, validProductBody(sku), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, body)

	errObj := requireErrorResponse(t, body, "resource_conflict", "invalid_request_error")
	assertErrorParam(t, errObj, "sku")
}

func TestCovCatalogProducts_Create_SKUTooLong(t *testing.T) {
	t.Parallel()

	longSKU := make([]byte, 256)
	for i := range longSKU {
		longSKU[i] = 'a'
	}
	body := validProductBody(string(longSKU))

	status, respBody, err := apiClient.Post(productsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"sku exceeding max length should return 400/422, got %d: %s", status, string(respBody))

	if status == 400 {
		errObj := requireErrorResponse(t, respBody, "invalid_format", "invalid_request_error")
		assertErrorParam(t, errObj, "sku")
	}
}

func TestCovCatalogProducts_Create_InvalidTypeEnum(t *testing.T) {
	t.Parallel()

	body := validProductBody(uniqueName("e2e-cov-prod-badtype"))
	body["type"] = "bogus_type"

	status, respBody, err := apiClient.Post(productsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)

	errObj := requireErrorResponse(t, respBody, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "type")
}

func TestCovCatalogProducts_Create_InvalidPortalVisibilityEnum(t *testing.T) {
	t.Parallel()

	body := validProductBody(uniqueName("e2e-cov-prod-badpv"))
	body["portal_visibility"] = "public"

	status, respBody, err := apiClient.Post(productsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)

	errObj := requireErrorResponse(t, respBody, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "portal_visibility")
}

func TestCovCatalogProducts_Create_NonexistentCategoryID(t *testing.T) {
	t.Parallel()

	body := validProductBody(uniqueName("e2e-cov-prod-badcat"))
	body["category_id"] = "itcg_00000000000000000000000000"

	status, respBody, err := apiClient.Post(productsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, respBody)
	requireErrorResponse(t, respBody, "resource_not_found", "invalid_request_error")
}

// A well-formed but nonexistent product_line_id is rejected, matching category_id: Vitess enforces no FK on the column, so the existence check has to be explicit.
func TestCovCatalogProducts_Create_NonexistentProductLineID(t *testing.T) {
	t.Parallel()

	body := validProductBody(uniqueName("e2e-cov-prod-badpl"))
	body["product_line_id"] = "pdln_00000000000000000000000000"

	status, respBody, err := apiClient.Post(productsPath, body, newIdempotencyKey())
	require.NoError(t, err)

	// Desired behavior: reject the nonexistent FK. Clean up defensively in case the
	// backend currently (incorrectly) creates the product anyway.
	if status == 201 {
		id := jsonField(parseJSON(respBody), "id")
		if id != "" {
			t.Cleanup(func() { apiClient.Delete(productsPath + "/" + id) })
		}
	}

	assert.True(t, status == 400 || status == 404,
		"nonexistent product_line_id should be rejected with 400/404, got %d: %s", status, string(respBody))
}

// TestCovCatalogProducts_Create_NonexistentAttributeID is the attribute_ids analogue of TestCovCatalogProducts_Create_NonexistentProductLineID. The `_item_attributes` join table has no FK constraint either, so an unchecked id would be stored as a silent orphan join row.
func TestCovCatalogProducts_Create_NonexistentAttributeID(t *testing.T) {
	t.Parallel()

	body := validProductBody(uniqueName("e2e-cov-prod-badattr"))
	body["attribute_ids"] = []string{"at_00000000000000000000000000"}

	status, respBody, err := apiClient.Post(productsPath, body, newIdempotencyKey())
	require.NoError(t, err)

	if status == 201 {
		id := jsonField(parseJSON(respBody), "id")
		if id != "" {
			t.Cleanup(func() { apiClient.Delete(productsPath + "/" + id) })
		}
	}

	assert.True(t, status == 400 || status == 404,
		"nonexistent attribute_ids entry should be rejected with 400/404, got %d: %s", status, string(respBody))
}

// TestCovCatalogProducts_Create_AttributeOutsideCategoryProperties covers the category gate on attribute_ids: an attribute may only be linked to a product whose category carries the attribute's property. SeedAttributeID belongs to the Color property, and the category here carries no properties at all.
func TestCovCatalogProducts_Create_AttributeOutsideCategoryProperties(t *testing.T) {
	t.Parallel()

	body := validProductBody(uniqueName("e2e-cov-prod-attrcat"))
	body["category_id"] = SeedPropertylessItemCategoryID
	body["attribute_ids"] = []string{SeedAttributeID}

	status, respBody, err := apiClient.Post(productsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)

	errObj := requireErrorResponse(t, respBody, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "attribute_ids")
}

// ──────────────────────────────────────────────
// Update — duplicate SKU, null-sku rejection, enum validation, clearable fields
// ──────────────────────────────────────────────

func TestCovCatalogProducts_Update_DuplicateSKUConflict(t *testing.T) {
	t.Parallel()

	skuA := uniqueName("e2e-cov-prod-updA")
	createdA := createAndCleanup(t, productsPath, validProductBody(skuA))
	idA := jsonField(createdA, "id")
	require.NotEmpty(t, idA)

	createdB := createAndCleanup(t, productsPath, validProductBody(uniqueName("e2e-cov-prod-updB")))
	idB := jsonField(createdB, "id")
	require.NotEmpty(t, idB)

	status, body, err := apiClient.Patch(productsPath+"/"+idB, map[string]any{"sku": skuA}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, body)

	errObj := requireErrorResponse(t, body, "resource_conflict", "invalid_request_error")
	assertErrorParam(t, errObj, "sku")
}

func TestCovCatalogProducts_Update_SKUNullRejected(t *testing.T) {
	t.Parallel()

	created := createAndCleanup(t, productsPath, validProductBody(uniqueName("e2e-cov-prod-skunull")))
	id := jsonField(created, "id")
	require.NotEmpty(t, id)

	status, body, err := apiClient.Patch(productsPath+"/"+id, map[string]any{"sku": nil}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "sku")
}

func TestCovCatalogProducts_Update_InvalidPortalVisibilityEnum(t *testing.T) {
	t.Parallel()

	created := createAndCleanup(t, productsPath, validProductBody(uniqueName("e2e-cov-prod-updpv")))
	id := jsonField(created, "id")
	require.NotEmpty(t, id)

	status, body, err := apiClient.Patch(productsPath+"/"+id, map[string]any{"portal_visibility": "public"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "portal_visibility")
}

func TestCovCatalogProducts_Update_ClearDescriptionAndNotes(t *testing.T) {
	t.Parallel()

	body := validProductBody(uniqueName("e2e-cov-prod-clear"))
	body["description"] = "has a description"
	body["notes"] = "has notes"
	created := createAndCleanup(t, productsPath, body)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)

	status, respBody, err := apiClient.Patch(productsPath+"/"+id+"?include=item", map[string]any{
		"description": nil,
		"notes":       nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, respBody)

	updated := parseJSON(respBody)
	item := jsonObject(updated, "item")
	require.NotNil(t, item)
	assertNilField(t, item, "description")
	assertNilField(t, item, "notes")

	// Re-fetch to confirm the clear persisted, not just reflected in the response.
	getStatus, getBody, err := apiClient.GetListRaw(productsPath+"/"+id, url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	item = jsonObject(parseJSON(getBody), "item")
	require.NotNil(t, item)
	assertNilField(t, item, "description")
	assertNilField(t, item, "notes")
}

func TestCovCatalogProducts_Update_BlankStringDoesNotClear(t *testing.T) {
	t.Parallel()

	body := validProductBody(uniqueName("e2e-cov-prod-blank"))
	body["description"] = "has a description"
	created := createAndCleanup(t, productsPath, body)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)

	status, respBody, err := apiClient.Patch(productsPath+"/"+id+"?include=item", map[string]any{
		"description": "",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, respBody)

	item := jsonObject(parseJSON(respBody), "item")
	require.NotNil(t, item)
	// A blank string is a real value, not a clear: field.Clearable semantics only clear on
	// JSON null. The description key must still be present and equal to "" (not absent/null).
	desc, ok := item["description"]
	require.True(t, ok, "description key should be present after blank-string update")
	assert.Equal(t, "", desc, "blank string update should set description to empty string, not clear it")
}

// ──────────────────────────────────────────────
// Not-found — GET/PATCH/DELETE/PUT-action on a well-formed-but-nonexistent product id
// ──────────────────────────────────────────────

const covCatalogProductsNonexistentID = "pd_00000000000000000000000000"

func TestCovCatalogProducts_NotFound_Get(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(productsPath+"/"+covCatalogProductsNonexistentID, nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

func TestCovCatalogProducts_NotFound_Patch(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Patch(productsPath+"/"+covCatalogProductsNonexistentID, map[string]any{
		"sku": uniqueName("e2e-cov-prod-notfound-patch"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

func TestCovCatalogProducts_NotFound_Delete(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Delete(productsPath + "/" + covCatalogProductsNonexistentID)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

// ──────────────────────────────────────────────
// Action: change-product-line — negative paths + idempotence
// ──────────────────────────────────────────────

func TestCovCatalogProducts_ChangeProductLine_NotFoundProductID(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Put(
		productsPath+"/"+covCatalogProductsNonexistentID+"/product-line/"+SeedProductLineID,
		map[string]any{},
	)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

// TestCovCatalogProducts_ChangeProductLine_NotFoundProductLineID asserts the desired/correct
// behavior for a well-formed-but-nonexistent product_line_id path param: it must be rejected
// (400/404). the live stack currently returns 200 and silently leaves
// the product's product_line association unset/unchanged rather than validating the FK
// (services/core-service/internal/infrastructure/repository/product_repo.go ChangeProductLine
// writes product_line_id with no existence check, matching the create-time gap above). Expected
// to be RED until fixed.
func TestCovCatalogProducts_ChangeProductLine_NotFoundProductLineID(t *testing.T) {
	t.Parallel()

	created := createAndCleanup(t, productsPath, validProductBody(uniqueName("e2e-cov-prod-chgplnf")))
	id := jsonField(created, "id")
	require.NotEmpty(t, id)

	status, body, err := apiClient.Put(
		productsPath+"/"+id+"/product-line/pdln_00000000000000000000000000",
		map[string]any{},
	)
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 404,
		"nonexistent product_line_id on change-product-line should return 400/404, got %d: %s", status, string(body))
}

func TestCovCatalogProducts_ChangeProductLine_Idempotent(t *testing.T) {
	t.Parallel()

	created := createAndCleanup(t, productsPath, validProductBody(uniqueName("e2e-cov-prod-chgplidem")))
	id := jsonField(created, "id")
	require.NotEmpty(t, id)

	// Associate against SeedIncludePutProductLineChangeTargetID rather than
	// SeedProductLineID. Both are valid seed product lines, but this test also
	// re-fetches the product with ?include=product_line, which forces the
	// gateway/core to resolve the product line's unit_group. SeedProductLineID
	// ("Socks") is a resolvable target for the PUTs, but its unit_group must
	// itself resolve for the include walk to succeed; this alternate line has a
	// live unit_group, keeping the idempotency check hermetic and independent of
	// unrelated product-line fixture state.
	targetProductLineID := SeedIncludePutProductLineChangeTargetID
	putPath := productsPath + "/" + id + "/product-line/" + targetProductLineID

	status1, body1, err := apiClient.Put(putPath, map[string]any{})
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)

	status2, body2, err := apiClient.Put(putPath, map[string]any{})
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)

	// product_line is not included in the PUT response by default (only on GET with
	// ?include=product_line), so confirm the association via a follow-up GET instead.
	getStatus, getBody, err := apiClient.GetListRaw(productsPath+"/"+id, url.Values{"include": {"product_line"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	pl := jsonObject(parseJSON(getBody), "product_line")
	require.NotNil(t, pl)
	assert.Equal(t, targetProductLineID, jsonField(pl, "id"), "repeat change-product-line calls should be idempotent")
}

// ──────────────────────────────────────────────
// List — dedicated filter tests: category_ids, attribute_ids, customer_ids,
// starts_at/ends_at, portal_visibility
// ──────────────────────────────────────────────

func TestCovCatalogProducts_List_FilterByCategoryIDs(t *testing.T) {
	t.Parallel()

	created := createAndCleanup(t, productsPath, validProductBody(uniqueName("e2e-cov-prod-listcat")))
	id := jsonField(created, "id")
	require.NotEmpty(t, id)

	list, _, err := apiClient.GetList(productsPath, url.Values{
		"category_ids": {SeedItemCategoryID},
		"limit":        {"1000"},
	})
	require.NoError(t, err)
	assert.True(t, covCatalogProductsListContainsID(list, id), "category_ids filter should include the newly created product in the seeded category")
}

func TestCovCatalogProducts_List_FilterByCategoryIDs_NoResults(t *testing.T) {
	t.Parallel()

	list, _, err := apiClient.GetList(productsPath, url.Values{
		"category_ids": {"itcg_00000000000000000000000000"},
	})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "Nonexistent category_ids filter should return empty data")
}

func TestCovCatalogProducts_List_FilterByAttributeIDs(t *testing.T) {
	t.Parallel()

	body := validProductBody(uniqueName("e2e-cov-prod-listattr"))
	body["attribute_ids"] = []string{SeedAttributeID}
	created := createAndCleanup(t, productsPath, body)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)

	list, _, err := apiClient.GetList(productsPath, url.Values{
		"attribute_ids": {SeedAttributeID},
		"limit":         {"1000"},
	})
	require.NoError(t, err)
	assert.True(t, covCatalogProductsListContainsID(list, id), "attribute_ids filter should include the newly created product linked to the seeded attribute")
}

func TestCovCatalogProducts_List_FilterByAttributeIDs_NoResults(t *testing.T) {
	t.Parallel()

	// Any id nothing in the suite ever joins against works as a no-results probe.
	list, _, err := apiClient.GetList(productsPath, url.Values{
		"attribute_ids": {"at_prod9noattr7filter0match55"},
	})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "attribute_ids filter for an attribute id no product is linked to should return empty data")
}

func TestCovCatalogProducts_List_FilterByCustomerIDs(t *testing.T) {
	t.Parallel()

	// The seeded customer has product-line access to SeedProductLineID (SeedProductLineAccessID),
	// so a product on that line should be returned when filtering by the customer's account id.
	body := validProductBody(uniqueName("e2e-cov-prod-listcust"))
	body["product_line_id"] = SeedProductLineID
	created := createAndCleanup(t, productsPath, body)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)

	list, _, err := apiClient.GetList(productsPath, url.Values{
		"customer_ids": {SeedCustomerAccountID},
		"limit":        {"1000"},
	})
	require.NoError(t, err)
	assert.True(t, covCatalogProductsListContainsID(list, id), "customer_ids filter should include the newly created product on the customer-accessible product line")
}

func TestCovCatalogProducts_List_FilterByCustomerIDs_NoResults(t *testing.T) {
	t.Parallel()

	list, _, err := apiClient.GetList(productsPath, url.Values{
		"customer_ids": {"ac_00000000000000000000000000"},
	})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "Nonexistent customer_ids filter should return empty data")
}

func TestCovCatalogProducts_List_FilterByPortalVisibility(t *testing.T) {
	t.Parallel()

	visibleBody := validProductBody(uniqueName("e2e-cov-prod-listpv-vis"))
	visibleBody["portal_visibility"] = "visible"
	visible := createAndCleanup(t, productsPath, visibleBody)
	visibleID := jsonField(visible, "id")
	require.NotEmpty(t, visibleID)

	hiddenBody := validProductBody(uniqueName("e2e-cov-prod-listpv-hid"))
	hiddenBody["portal_visibility"] = "hidden"
	hidden := createAndCleanup(t, productsPath, hiddenBody)
	hiddenID := jsonField(hidden, "id")
	require.NotEmpty(t, hiddenID)

	visList, _, err := apiClient.GetList(productsPath, url.Values{"portal_visibility": {"visible"}, "limit": {"1000"}})
	require.NoError(t, err)
	assert.True(t, covCatalogProductsListContainsID(visList, visibleID), "portal_visibility=visible filter should include the visible product")
	assert.False(t, covCatalogProductsListContainsID(visList, hiddenID), "portal_visibility=visible filter should exclude the hidden product")

	hidList, _, err := apiClient.GetList(productsPath, url.Values{"portal_visibility": {"hidden"}, "limit": {"1000"}})
	require.NoError(t, err)
	assert.True(t, covCatalogProductsListContainsID(hidList, hiddenID), "portal_visibility=hidden filter should include the hidden product")
	assert.False(t, covCatalogProductsListContainsID(hidList, visibleID), "portal_visibility=hidden filter should exclude the visible product")
}

func TestCovCatalogProducts_List_FilterByPortalVisibility_InvalidEnum(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(productsPath, url.Values{"portal_visibility": {"bogus"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

func TestCovCatalogProducts_List_FilterByDateRange(t *testing.T) {
	t.Parallel()

	created := createAndCleanup(t, productsPath, validProductBody(uniqueName("e2e-cov-prod-listdate")))
	id := jsonField(created, "id")
	require.NotEmpty(t, id)

	inRange, _, err := apiClient.GetList(productsPath, url.Values{
		"starts_at": {"2020-01-01T00:00:00Z"},
		"ends_at":   {"2030-01-01T00:00:00Z"},
		"limit":     {"1000"},
	})
	require.NoError(t, err)
	assert.True(t, covCatalogProductsListContainsID(inRange, id), "date range bracketing created_at should include the newly created product")

	outOfRange, _, err := apiClient.GetList(productsPath, url.Values{
		"starts_at": {"2020-01-01T00:00:00Z"},
		"ends_at":   {"2020-06-01T00:00:00Z"},
	})
	require.NoError(t, err)
	assert.False(t, covCatalogProductsListContainsID(outOfRange, id), "date range before created_at should exclude the newly created product")
}

func TestCovCatalogProducts_List_MalformedStartDate(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(productsPath, url.Values{"starts_at": {"not-a-date"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}
