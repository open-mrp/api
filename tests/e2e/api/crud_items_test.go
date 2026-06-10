//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newProductItemIDs creates a product and returns its id and the linked item id.
// Registers cleanup to delete the product.
func newProductItemIDs(t *testing.T, namePrefix string) (productID, itemID string) {
	t.Helper()
	sku := uniqueName(namePrefix)
	resp, err := apiClient.PostFull(productsPath, validProductBody(sku), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	productID = jsonField(parseJSON(resp.Body), "id")
	require.NotEmpty(t, productID)
	t.Cleanup(func() { apiClient.Delete(productsPath + "/" + productID) })

	getStatus, getBody, err := apiClient.GetListRaw(productsPath+"/"+productID, url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	itemID = jsonField(jsonObject(parseJSON(getBody), "item"), "id")
	require.NotEmpty(t, itemID)
	return productID, itemID
}

func assertItemCoreFields(t *testing.T, got map[string]any) {
	t.Helper()
	assertIDFormat(t, jsonField(got, "id"), "it")
	assertObjectField(t, got, "item")
	assert.NotEmpty(t, jsonField(got, "sku"))
	assert.NotEmpty(t, jsonField(got, "type"))
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	assertNilField(t, got, "category")
	assertNilField(t, got, "unit_value")
	assertNilField(t, got, "unit_cost")
	assertNilField(t, got, "burn_rate")
	assertNilField(t, got, "attributes")
}

// ──────────────────────────────────────────────
// Item — CRUD
// ──────────────────────────────────────────────

func TestItems_CRUD(t *testing.T) {
	t.Parallel()
	sku := uniqueName("e2e-item-crud")
	resp, err := apiClient.PostFull(productsPath, validProductBody(sku), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	productID := jsonField(parseJSON(resp.Body), "id")
	require.NotEmpty(t, productID)
	deleted := false
	t.Cleanup(func() {
		if !deleted {
			apiClient.Delete(productsPath + "/" + productID)
		}
	})

	getStatus, getBody, err := apiClient.GetListRaw(productsPath+"/"+productID, url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	itemID := jsonField(jsonObject(parseJSON(getBody), "item"), "id")
	require.NotEmpty(t, itemID)

	get1Status, get1Body, err := apiClient.GetListRaw(itemsPath+"/"+itemID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, get1Status, get1Body)

	newSKU := uniqueName("e2e-crud-sku")
	desc := "e2e crud description"
	notes := "e2e crud notes"
	patchStatus, patchBody, err := apiClient.Patch(productsPath+"/"+productID, map[string]any{
		"sku":         newSKU,
		"description": desc,
		"notes":       notes,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	afterPatch := parseJSON(patchBody)
	assert.Equal(t, productID, jsonField(afterPatch, "id"))

	get2Status, get2Body, err := apiClient.GetListRaw(itemsPath+"/"+itemID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, get2Status, get2Body)
	got := parseJSON(get2Body)
	assert.Equal(t, newSKU, jsonField(got, "sku"))
	assert.Equal(t, desc, jsonField(got, "description"))
	assert.Equal(t, notes, jsonField(got, "notes"))

	delStatus, delBody, err := apiClient.Delete(productsPath + "/" + productID)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)
	deleted = true

	finalStatus, _, err := apiClient.GetListRaw(itemsPath+"/"+itemID, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, finalStatus, "item GET should 404 after parent product is deleted")
}

// ──────────────────────────────────────────────
// Item — Response Shape
// ──────────────────────────────────────────────

func TestItems_RetrieveResponseShape(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(itemsPath+"/"+SeedItemID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, SeedItemID, jsonField(got, "id"))
	assertObjectField(t, got, "item")
	assert.Equal(t, SeedItemSKU, jsonField(got, "sku"))
	assert.Equal(t, SeedItemDescription, jsonField(got, "description"))
	assertNilField(t, got, "notes")
	assert.Equal(t, "product", jsonField(got, "type"))
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	assertNilField(t, got, "category")
	assertNilField(t, got, "unit_value")
	assertNilField(t, got, "unit_cost")
	assertNilField(t, got, "burn_rate")
	assertNilField(t, got, "attributes")
}

// ──────────────────────────────────────────────
// Item — Update
// ──────────────────────────────────────────────

func TestItems_UpdateAllFields(t *testing.T) {
	t.Parallel()
	productID, itemID := newProductItemIDs(t, "e2e-item-allf")

	newSKU := uniqueName("e2e-allf-sku")
	desc := "all fields description"
	notes := "all fields notes"
	patchStatus, patchBody, err := apiClient.Patch(productsPath+"/"+productID, map[string]any{
		"sku":         newSKU,
		"description": desc,
		"notes":       notes,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	got := parseJSON(patchBody)
	assert.Equal(t, productID, jsonField(got, "id"))
	assertObjectField(t, got, "product")
	assert.Equal(t, "sale", jsonField(got, "type"))
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	getStatus, getBody, err := apiClient.GetListRaw(itemsPath+"/"+itemID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	item := parseJSON(getBody)
	assert.Equal(t, itemID, jsonField(item, "id"))
	assertObjectField(t, item, "item")
	assert.Equal(t, newSKU, jsonField(item, "sku"))
	assert.Equal(t, desc, jsonField(item, "description"))
	assert.Equal(t, notes, jsonField(item, "notes"))
	assertNilField(t, item, "category")
	assertNilField(t, item, "unit_value")
	assertNilField(t, item, "unit_cost")
	assertNilField(t, item, "burn_rate")
	assertNilField(t, item, "attributes")
}

func TestItems_UpdatePreservesOmittedFields(t *testing.T) {
	t.Parallel()
	productID, itemID := newProductItemIDs(t, "e2e-item-omit")

	firstDesc := "first description patch"
	firstNotes := "first notes patch"
	s0, b0, err := apiClient.Patch(productsPath+"/"+productID, map[string]any{
		"description": firstDesc,
		"notes":       firstNotes,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, s0, b0)

	getStatus, getBody, err := apiClient.GetListRaw(itemsPath+"/"+itemID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	baseline := parseJSON(getBody)
	origSKU := jsonField(baseline, "sku")
	origType := jsonField(baseline, "type")
	origCreated := jsonField(baseline, "created_at")

	secondDesc := "second description only"
	patchStatus, patchBody, err := apiClient.Patch(productsPath+"/"+productID, map[string]any{
		"description": secondDesc,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	require.Equal(t, productID, jsonField(parseJSON(patchBody), "id"))

	get2Status, get2Body, err := apiClient.GetListRaw(itemsPath+"/"+itemID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, get2Status, get2Body)
	got := parseJSON(get2Body)
	assert.Equal(t, secondDesc, jsonField(got, "description"))
	assert.Equal(t, firstNotes, jsonField(got, "notes"))
	assert.Equal(t, origSKU, jsonField(got, "sku"))
	assert.Equal(t, origType, jsonField(got, "type"))
	assert.Equal(t, origCreated, jsonField(got, "created_at"))
}

func TestItems_UpdateNullDescription(t *testing.T) {
	t.Parallel()
	productID, itemID := newProductItemIDs(t, "e2e-item-nulldesc")

	s0, b0, err := apiClient.Patch(productsPath+"/"+productID, map[string]any{
		"description": "temp then clear",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, s0, b0)

	patchStatus, patchBody, err := apiClient.Patch(productsPath+"/"+productID, map[string]any{
		"description": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	require.Equal(t, productID, jsonField(parseJSON(patchBody), "id"))

	getStatus, getBody, err := apiClient.GetListRaw(itemsPath+"/"+itemID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assertNilField(t, got, "description")
}

func TestItems_UpdateNullNotes(t *testing.T) {
	t.Parallel()
	productID, itemID := newProductItemIDs(t, "e2e-item-nullnotes")

	s0, b0, err := apiClient.Patch(productsPath+"/"+productID, map[string]any{
		"notes": "temp note",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, s0, b0)

	patchStatus, patchBody, err := apiClient.Patch(productsPath+"/"+productID, map[string]any{
		"notes": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	require.Equal(t, productID, jsonField(parseJSON(patchBody), "id"))

	getStatus, getBody, err := apiClient.GetListRaw(itemsPath+"/"+itemID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assertNilField(t, got, "notes")
}

func TestItems_UpdateIdempotent(t *testing.T) {
	t.Parallel()
	productID, itemID := newProductItemIDs(t, "e2e-item-idem")

	newSKU := uniqueName("e2e-idem-sku")
	body := map[string]any{"sku": newSKU}
	idem := newIdempotencyKey()

	s1, b1, err := apiClient.Patch(productsPath+"/"+productID, body, idem)
	require.NoError(t, err)
	requireStatus(t, 200, s1, b1)
	got1 := parseJSON(b1)

	s2, b2, err := apiClient.Patch(productsPath+"/"+productID, body, idem)
	require.NoError(t, err)
	requireStatus(t, 200, s2, b2)
	got2 := parseJSON(b2)

	assert.Equal(t, jsonField(got1, "id"), jsonField(got2, "id"))
	assert.Equal(t, productID, jsonField(got1, "id"))

	getStatus, getBody, err := apiClient.GetListRaw(itemsPath+"/"+itemID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, newSKU, jsonField(parseJSON(getBody), "sku"))
}

// ──────────────────────────────────────────────
// Item — Category
// ──────────────────────────────────────────────

func TestItems_ChangeCategory_ResponseShape(t *testing.T) {
	t.Parallel()
	_, itemID := newProductItemIDs(t, "e2e-item-chcat-shape")

	putStatus, putBody, err := apiClient.Put(
		itemsPath+"/"+itemID+"/category/"+SeedItemCategoryID,
		nil,
	)
	require.NoError(t, err)
	requireStatus(t, 200, putStatus, putBody)

	got := parseJSON(putBody)
	assertItemCoreFields(t, got)
	assert.Equal(t, itemID, jsonField(got, "id"))
}

// ──────────────────────────────────────────────
// Item — Attributes
// ──────────────────────────────────────────────

func TestItems_AddAttribute_Idempotent(t *testing.T) {
	t.Parallel()
	_, itemID := newProductItemIDs(t, "e2e-item-addattr-idem")

	path := itemsPath + "/" + itemID + "/attributes/" + SeedAttributeID + "?include=attributes"

	put1Status, put1Body, err := apiClient.Put(path, nil)
	require.NoError(t, err)
	requireStatus(t, 200, put1Status, put1Body)
	n1 := len(jsonArray(jsonObject(parseJSON(put1Body), "attributes"), "data"))

	put2Status, put2Body, err := apiClient.Put(path, nil)
	require.NoError(t, err)
	requireStatus(t, 200, put2Status, put2Body)
	n2 := len(jsonArray(jsonObject(parseJSON(put2Body), "attributes"), "data"))

	assert.Equal(t, n1, n2, "second PUT add-attribute should not duplicate rows")
	assert.GreaterOrEqual(t, n1, 1)
}

// ──────────────────────────────────────────────
// Item — Validation
// ──────────────────────────────────────────────

func TestItems_UpdateValidation_EmptySKU(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Patch(productsPath+"/"+SeedProductID, map[string]any{
		"sku": "",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"empty sku should return 400 or 422, got %d: %s", status, string(body))
}

func TestItems_UpdateValidation_SKUTooLong(t *testing.T) {
	t.Parallel()
	longSKU := strings.Repeat("a", 256)
	status, body, err := apiClient.Patch(productsPath+"/"+SeedProductID, map[string]any{
		"sku": longSKU,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"sku over max length should return 400 or 422, got %d: %s", status, string(body))
}

// Item — Include Tests
// ──────────────────────────────────────────────
//
// Item GET endpoint whitelists: category, unit_value, unit_cost, burn_rate
// (attributes is a registered include but is served at a different endpoint).

func TestItems_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(itemsPath+"/"+SeedItemID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["category"], "category should be null without ?include=category")
	assert.Nil(t, got["unit_value"], "unit_value should be null without ?include=unit_value")
	assert.Nil(t, got["unit_cost"], "unit_cost should be null without ?include=unit_cost")
	assert.Nil(t, got["burn_rate"], "burn_rate should be null without ?include=burn_rate")
	assert.Nil(t, got["attributes"], "attributes should be null without ?include=attributes")

	list, _, err := apiClient.GetList(itemsPath, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	for _, item := range list.Data {
		m := parseJSON(item)
		assert.Nil(t, m["category"], "category should be null on list items without ?include=category")
		assert.Nil(t, m["unit_value"], "unit_value should be null on list items without ?include=unit_value")
		assert.Nil(t, m["unit_cost"], "unit_cost should be null on list items without ?include=unit_cost")
		assert.Nil(t, m["burn_rate"], "burn_rate should be null on list items without ?include=burn_rate")
		assert.Nil(t, m["attributes"], "attributes should be null on list items without ?include=attributes")
	}
}

func TestItems_IncludeCategory(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(itemsPath+"/"+SeedItemID, url.Values{"include": {"category"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	cat := jsonObject(got, "category")
	require.NotNil(t, cat, "category should be present with ?include=category")
	assert.Equal(t, "item_category", jsonField(cat, "object"))
	assert.NotEmpty(t, jsonField(cat, "id"))
}

func TestItems_IncludeUnitValue(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(itemsPath+"/"+SeedItemID, url.Values{"include": {"unit_value"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["unit_value"]
	assert.True(t, ok, "unit_value key should be present with ?include=unit_value")
	if uv := jsonObject(got, "unit_value"); uv != nil {
		assert.Equal(t, "rate", jsonField(uv, "object"))
		assert.NotEmpty(t, jsonField(uv, "id"))
	}
}

func TestItems_IncludeUnitCost(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(itemsPath+"/"+SeedItemID, url.Values{"include": {"unit_cost"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["unit_cost"]
	assert.True(t, ok, "unit_cost key should be present with ?include=unit_cost")
	if uc := jsonObject(got, "unit_cost"); uc != nil {
		assert.Equal(t, "rate", jsonField(uc, "object"))
		assert.NotEmpty(t, jsonField(uc, "id"))
	}
}

func TestItems_IncludeBurnRate(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(itemsPath+"/"+SeedItemID, url.Values{"include": {"burn_rate"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["burn_rate"]
	assert.True(t, ok, "burn_rate key should be present with ?include=burn_rate")
	if br := jsonObject(got, "burn_rate"); br != nil {
		assert.Equal(t, "rate", jsonField(br, "object"))
		assert.NotEmpty(t, jsonField(br, "id"))
	}
}

func TestItems_IncludeAttributes(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(itemsPath+"/"+SeedItemID, url.Values{"include": {"attributes"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	attrs := jsonObject(got, "attributes")
	require.NotNil(t, attrs, "attributes should be present with ?include=attributes")
	_, ok := attrs["data"].([]any)
	require.True(t, ok, "attributes should include a data array")
}

func TestItems_ListIncludeCategory(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(itemsPath, url.Values{"include": {"category"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	data, ok := got["data"].([]any)
	require.True(t, ok, "response should have data array")
	require.NotEmpty(t, data, "should have at least one item")

	first, ok := data[0].(map[string]any)
	require.True(t, ok)
	cat := jsonObject(first, "category")
	require.NotNil(t, cat, "category should be present on list items with ?include=category")
	assert.Equal(t, "item_category", jsonField(cat, "object"))
}

func TestItems_ListIncludeAttributes(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(itemsPath, url.Values{"include": {"attributes"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	data, ok := got["data"].([]any)
	require.True(t, ok, "response should have data array")
	require.NotEmpty(t, data, "should have at least one item")

	first, ok := data[0].(map[string]any)
	require.True(t, ok)
	attrs := jsonObject(first, "attributes")
	require.NotNil(t, attrs, "attributes should be present on list items with ?include=attributes")
	_, ok = attrs["data"].([]any)
	require.True(t, ok, "attributes should include a data array")
}

func TestItems_ListIncludeUnitValue(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(itemsPath, url.Values{"include": {"unit_value"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	data, ok := got["data"].([]any)
	require.True(t, ok, "response should have data array")
	require.NotEmpty(t, data, "should have at least one item")

	first, ok := data[0].(map[string]any)
	require.True(t, ok)
	_, ok = first["unit_value"]
	assert.True(t, ok, "unit_value key should be present on list items with ?include=unit_value")
	if uv := jsonObject(first, "unit_value"); uv != nil {
		assert.Equal(t, "rate", jsonField(uv, "object"))
		assert.NotEmpty(t, jsonField(uv, "id"))
	}
}

func TestItems_ListIncludeUnitCost(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(itemsPath, url.Values{"include": {"unit_cost"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	data, ok := got["data"].([]any)
	require.True(t, ok, "response should have data array")
	require.NotEmpty(t, data, "should have at least one item")

	first, ok := data[0].(map[string]any)
	require.True(t, ok)
	_, ok = first["unit_cost"]
	assert.True(t, ok, "unit_cost key should be present on list items with ?include=unit_cost")
	if uc := jsonObject(first, "unit_cost"); uc != nil {
		assert.Equal(t, "rate", jsonField(uc, "object"))
		assert.NotEmpty(t, jsonField(uc, "id"))
	}
}

func TestItems_ListIncludeBurnRate(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(itemsPath, url.Values{"include": {"burn_rate"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	data, ok := got["data"].([]any)
	require.True(t, ok, "response should have data array")
	require.NotEmpty(t, data, "should have at least one item")

	first, ok := data[0].(map[string]any)
	require.True(t, ok)
	_, ok = first["burn_rate"]
	assert.True(t, ok, "burn_rate key should be present on list items with ?include=burn_rate")
	if br := jsonObject(first, "burn_rate"); br != nil {
		assert.Equal(t, "rate", jsonField(br, "object"))
		assert.NotEmpty(t, jsonField(br, "id"))
	}
}

// ──────────────────────────────────────────────
// Item — Include Tests on Mutation Endpoints
// ──────────────────────────────────────────────

func TestItems_UpdateIncludesCategory(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Patch(
		productsPath+"/"+SeedProductID+"?include=item.category",
		map[string]any{"description": SeedItemDescription},
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present on PATCH product with ?include=item.category")
	cat := jsonObject(item, "category")
	require.NotNil(t, cat, "item.category should be present on PATCH response with ?include=item.category")
	assert.Equal(t, "item_category", jsonField(cat, "object"))
	assert.NotEmpty(t, jsonField(cat, "id"))
}

func TestItems_UpdateIncludesAttributes(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Patch(
		productsPath+"/"+SeedProductID+"?include=item.attributes",
		map[string]any{"description": SeedItemDescription},
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present on PATCH product with ?include=item.attributes")
	attrs := jsonObject(item, "attributes")
	require.NotNil(t, attrs, "item.attributes should be present on PATCH response with ?include=item.attributes")
	assert.Equal(t, "list", jsonField(attrs, "object"))
	_, ok := attrs["data"].([]any)
	require.True(t, ok, "attributes should include a data array")
}

func TestItems_AddAttributeIncludesAttributes(t *testing.T) {
	t.Parallel()
	// PUT is idempotent — adding an already-associated attribute is a no-op.
	putStatus, putBody, err := apiClient.Put(
		itemsPath+"/"+SeedItemID+"/attributes/"+SeedAttributeID+"?include=attributes",
		nil,
	)
	require.NoError(t, err)
	requireStatus(t, 200, putStatus, putBody)

	got := parseJSON(putBody)
	attrs := jsonObject(got, "attributes")
	require.NotNil(t, attrs, "attributes should be present on add-attribute response with ?include=attributes")
	assert.Equal(t, "list", jsonField(attrs, "object"))
	data, ok := attrs["data"].([]any)
	require.True(t, ok, "attributes should include a data array")
	assert.GreaterOrEqual(t, len(data), 1, "item should have at least one attribute after add")
}

func TestItems_RemoveAttributeIncludesAttributes(t *testing.T) {
	t.Parallel()

	// Create a fresh product so we get an item we can safely mutate without
	// affecting shared seed data used by other parallel tests.
	resp, err := apiClient.PostFull(productsPath, validProductBody(uniqueName("e2e-item-rmattr")), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	productID := jsonField(parseJSON(resp.Body), "id")
	require.NotEmpty(t, productID)
	t.Cleanup(func() { apiClient.Delete(productsPath + "/" + productID) })

	// Resolve the item ID from the product.
	getStatus, getBody, err := apiClient.GetListRaw(productsPath+"/"+productID, url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	itemID := jsonField(jsonObject(parseJSON(getBody), "item"), "id")
	require.NotEmpty(t, itemID)

	// Add the attribute first so there is something to remove.
	addStatus, addBody, err := apiClient.Put(itemsPath+"/"+itemID+"/attributes/"+SeedAttributeID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, addStatus, addBody)

	// Remove the attribute and verify the include works.
	delStatus, delBody, err := apiClient.Delete(
		itemsPath + "/" + itemID + "/attributes/" + SeedAttributeID + "?include=attributes",
	)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	got := parseJSON(delBody)
	attrs := jsonObject(got, "attributes")
	require.NotNil(t, attrs, "attributes should be present on remove-attribute response with ?include=attributes")
	assert.Equal(t, "list", jsonField(attrs, "object"))
	_, ok := attrs["data"].([]any)
	require.True(t, ok, "attributes should include a data array")
}

func TestItems_ChangeCategoryIncludesCategory(t *testing.T) {
	t.Parallel()

	// Create a fresh product so we can safely change its item's category.
	resp, err := apiClient.PostFull(productsPath, validProductBody(uniqueName("e2e-item-chcat")), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	productID := jsonField(parseJSON(resp.Body), "id")
	require.NotEmpty(t, productID)
	t.Cleanup(func() { apiClient.Delete(productsPath + "/" + productID) })

	// Resolve the item ID.
	getStatus, getBody, err := apiClient.GetListRaw(productsPath+"/"+productID, url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	itemID := jsonField(jsonObject(parseJSON(getBody), "item"), "id")
	require.NotEmpty(t, itemID)

	// Change to the same category (safe no-op) with ?include=category.
	putStatus, putBody, err := apiClient.Put(
		itemsPath+"/"+itemID+"/category/"+SeedItemCategoryID+"?include=category",
		nil,
	)
	require.NoError(t, err)
	requireStatus(t, 200, putStatus, putBody)

	got := parseJSON(putBody)
	cat := jsonObject(got, "category")
	require.NotNil(t, cat, "category should be present on change-category response with ?include=category")
	assert.Equal(t, "item_category", jsonField(cat, "object"))
	assert.Equal(t, SeedItemCategoryID, jsonField(cat, "id"))
}
