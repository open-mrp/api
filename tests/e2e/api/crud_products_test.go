//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// Product — Include Tests
// ──────────────────────────────────────────────

func TestProducts_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(productsPath+"/"+SeedProductID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["product_line"], "product_line should be null without ?include=product_line")
	assert.Nil(t, got["item"], "item should be null without ?include=item")

	list, _, err := apiClient.GetList(productsPath, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	for _, item := range list.Data {
		m := parseJSON(item)
		assert.Nil(t, m["product_line"], "product_line should be null on list items without ?include=product_line")
		assert.Nil(t, m["item"], "item should be null on list items without ?include=item")
	}
}

func TestProducts_IncludeProductLine(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(productsPath+"/"+SeedProductID, url.Values{"include": {"product_line"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	pl := jsonObject(got, "product_line")
	require.NotNil(t, pl, "product_line should be present with ?include=product_line")
	assert.Equal(t, "product_line", jsonField(pl, "object"))
}

func TestProducts_IncludeItem(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(productsPath+"/"+SeedProductID, url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present with ?include=item")
	assert.Equal(t, "item", jsonField(item, "object"))
	assert.NotEmpty(t, jsonField(item, "id"))
}

func TestProducts_ListIncludeProductLine(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(productsPath, url.Values{
		"include":          {"product_line"},
		"product_line_ids": {SeedProductLineID},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	data, ok := got["data"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, data)

	first, ok := data[0].(map[string]any)
	require.True(t, ok)
	pl := jsonObject(first, "product_line")
	require.NotNil(t, pl, "product_line should be present on list items with ?include=product_line")
	assert.Equal(t, "product_line", jsonField(pl, "object"))
}

func TestProducts_IncludeProductLineUnitGroup(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(productsPath+"/"+SeedProductID, url.Values{"include": {"product_line.unit_group"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	pl := jsonObject(got, "product_line")
	require.NotNil(t, pl, "product_line should be present with ?include=product_line.unit_group")
	assert.Equal(t, "product_line", jsonField(pl, "object"))

	ug := jsonObject(pl, "unit_group")
	require.NotNil(t, ug, "unit_group should be present inside product_line with ?include=product_line.unit_group")
	assert.Equal(t, "unit_group", jsonField(ug, "object"))
	assert.NotEmpty(t, jsonField(ug, "id"))
}

func TestProducts_IncludeItemCategory(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(productsPath+"/"+SeedProductID, url.Values{"include": {"item.category"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present with ?include=item.category")
	assert.Equal(t, "item", jsonField(item, "object"))

	cat := jsonObject(item, "category")
	require.NotNil(t, cat, "category should be present inside item with ?include=item.category")
	assert.Equal(t, "item_category", jsonField(cat, "object"))
	assert.NotEmpty(t, jsonField(cat, "id"))
}

func TestProducts_IncludeItemUnitValue(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(productsPath+"/"+SeedProductID, url.Values{"include": {"item.unit_value"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	AssertResponseBodyValid(t, body)

	got := parseJSON(body)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present with ?include=item.unit_value")

	_, ok := item["unit_value"]
	assert.True(t, ok, "unit_value key should be present on item with ?include=item.unit_value")
	if uv := jsonObject(item, "unit_value"); uv != nil {
		assert.Equal(t, "rate", jsonField(uv, "object"))
		assert.NotEmpty(t, jsonField(uv, "id"))
	}
}

func TestProducts_IncludeItemUnitCost(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(productsPath+"/"+SeedProductID, url.Values{"include": {"item.unit_cost"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	AssertResponseBodyValid(t, body)

	got := parseJSON(body)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present with ?include=item.unit_cost")

	_, ok := item["unit_cost"]
	assert.True(t, ok, "unit_cost key should be present on item with ?include=item.unit_cost")
	if uc := jsonObject(item, "unit_cost"); uc != nil {
		assert.Equal(t, "rate", jsonField(uc, "object"))
		assert.NotEmpty(t, jsonField(uc, "id"))
	}
}

func TestProducts_IncludeItemBurnRate(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(productsPath+"/"+SeedProductID, url.Values{"include": {"item.burn_rate"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	AssertResponseBodyValid(t, body)

	got := parseJSON(body)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present with ?include=item.burn_rate")

	_, ok := item["burn_rate"]
	assert.True(t, ok, "burn_rate key should be present on item with ?include=item.burn_rate")
	if br := jsonObject(item, "burn_rate"); br != nil {
		assert.Equal(t, "rate", jsonField(br, "object"))
		assert.NotEmpty(t, jsonField(br, "id"))
	}
}

func TestProducts_IncludeItemAttributes(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(productsPath+"/"+SeedProductID, url.Values{"include": {"item.attributes"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present with ?include=item.attributes")

	attrs := jsonObject(item, "attributes")
	require.NotNil(t, attrs, "attributes should be present inside item with ?include=item.attributes")
	assert.Equal(t, "list", jsonField(attrs, "object"))
	data, _ := attrs["data"].([]any)
	assert.GreaterOrEqual(t, len(data), 1, "seeded item should have at least one attribute")
}

func TestProducts_ListIncludeItem(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(productsPath, url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	data, ok := got["data"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, data)

	first, ok := data[0].(map[string]any)
	require.True(t, ok)
	item := jsonObject(first, "item")
	require.NotNil(t, item, "item should be present on list items with ?include=item")
	assert.Equal(t, "item", jsonField(item, "object"))
	assert.NotEmpty(t, jsonField(item, "id"))
}

func TestProducts_ListIncludeProductLineUnitGroup(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(productsPath, url.Values{
		"include":          {"product_line.unit_group"},
		"product_line_ids": {SeedProductLineID},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	data, ok := got["data"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, data)

	first, ok := data[0].(map[string]any)
	require.True(t, ok)
	pl := jsonObject(first, "product_line")
	require.NotNil(t, pl, "product_line should be present on list items with ?include=product_line.unit_group")
	assert.Equal(t, "product_line", jsonField(pl, "object"))
	ug := jsonObject(pl, "unit_group")
	require.NotNil(t, ug, "unit_group should be present inside product_line on list items with ?include=product_line.unit_group")
	assert.Equal(t, "unit_group", jsonField(ug, "object"))
}

// ──────────────────────────────────────────────
// Product — CRUD & lifecycle
// ──────────────────────────────────────────────

func TestProducts_CRUD(t *testing.T) {
	t.Parallel()
	sku := uniqueName("e2e-prod-crud")
	createStatus, createBody, err := apiClient.Post(productsPath, validProductBody(sku), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	created := parseJSON(createBody)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	assertIDFormat(t, id, "pd")
	assertObjectField(t, created, "product")

	getStatus, getBody, err := apiClient.GetListRaw(productsPath+"/"+id, url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assert.Equal(t, id, jsonField(got, "id"))
	item := jsonObject(got, "item")
	require.NotNil(t, item)
	assert.Equal(t, sku, jsonField(item, "sku"))

	desc := "e2e prod crud description"
	notes := "e2e prod crud notes"
	newSKU := uniqueName("e2e-prod-crud-upd")
	patchStatus, patchBody, err := apiClient.Patch(productsPath+"/"+id+"?include=item", map[string]any{
		"sku":               newSKU,
		"description":       desc,
		"notes":             notes,
		"portal_visibility": "visible",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	updated := parseJSON(patchBody)
	assert.Equal(t, id, jsonField(updated, "id"))
	assert.Equal(t, "visible", jsonField(updated, "portal_visibility"))
	item = jsonObject(updated, "item")
	require.NotNil(t, item)
	assert.Equal(t, newSKU, jsonField(item, "sku"))
	assert.Equal(t, desc, jsonField(item, "description"))
	assert.Equal(t, notes, jsonField(item, "notes"))

	delStatus, delBody, err := apiClient.Delete(productsPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	getAfterStatus, _, err := apiClient.GetListRaw(productsPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getAfterStatus, "GET after delete should return 404")
}

func TestProducts_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()
	sku := uniqueName("e2e-prod-allf")
	desc := "All-fields create description"
	notes := "All-fields create notes"
	body := map[string]any{
		"sku":               sku,
		"type":              "sale",
		"category_id":       SeedItemCategoryID,
		"product_line_id":   SeedProductLineID,
		"portal_visibility": "visible",
		"description":       desc,
		"notes":             notes,
		"unit_price": map[string]any{
			"value":               "12.34",
			"numerator_unit_id":   currencyUnitID,
			"denominator_unit_id": nonCurrencyUnitID,
		},
	}

	createStatus, createBody, err := apiClient.Post(productsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	created := parseJSON(createBody)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { apiClient.Delete(productsPath + "/" + id) })

	assertIDFormat(t, id, "pd")
	assertObjectField(t, created, "product")
	assert.Equal(t, "sale", jsonField(created, "type"))
	assert.Equal(t, "visible", jsonField(created, "portal_visibility"))
	assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")
	origCreatedAt := jsonField(created, "created_at")

	getStatus, getBody, err := apiClient.GetListRaw(productsPath+"/"+id, url.Values{
		"include": {"product_line", "item", "item.unit_value"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)

	pl := jsonObject(got, "product_line")
	require.NotNil(t, pl)
	assert.Equal(t, "product_line", jsonField(pl, "object"))
	assert.Equal(t, SeedProductLineID, jsonField(pl, "id"))

	item := jsonObject(got, "item")
	require.NotNil(t, item)
	assert.Equal(t, "item", jsonField(item, "object"))
	assert.NotEmpty(t, jsonField(item, "id"))
	assert.Equal(t, sku, jsonField(item, "sku"))
	assert.Equal(t, desc, jsonField(item, "description"))
	assert.Equal(t, notes, jsonField(item, "notes"))
	uv := jsonObject(item, "unit_value")
	require.NotNil(t, uv)
	assert.Equal(t, "rate", jsonField(uv, "object"))
	assert.NotEmpty(t, jsonField(uv, "id"))

	newSKU := uniqueName("e2e-prod-allf-u")
	newDesc := "All-fields updated description"
	newNotes := "All-fields updated notes"
	patchQ := url.Values{"include": {"product_line", "item", "item.unit_value"}}
	patchStatus, patchBody, err := apiClient.Patch(productsPath+"/"+id+"?"+patchQ.Encode(), map[string]any{
		"sku":               newSKU,
		"description":       newDesc,
		"notes":             newNotes,
		"portal_visibility": "hidden",
		"unit_price": map[string]any{
			"value":               "99.01",
			"numerator_unit_id":   currencyUnitID,
			"denominator_unit_id": nonCurrencyUnitID,
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	upd := parseJSON(patchBody)
	assert.Equal(t, id, jsonField(upd, "id"))
	assert.Equal(t, "sale", jsonField(upd, "type"))
	assert.Equal(t, "hidden", jsonField(upd, "portal_visibility"))
	assert.Equal(t, origCreatedAt, jsonField(upd, "created_at"), "created_at must not change")
	assertValidTimestamp(t, jsonField(upd, "updated_at"), "updated_at")

	getUpdStatus, getUpdBody, err := apiClient.GetListRaw(productsPath+"/"+id, url.Values{
		"include": {"product_line", "item", "item.unit_value"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, getUpdStatus, getUpdBody)
	updInc := parseJSON(getUpdBody)

	pl = jsonObject(updInc, "product_line")
	require.NotNil(t, pl)
	assert.Equal(t, SeedProductLineID, jsonField(pl, "id"))

	item = jsonObject(updInc, "item")
	require.NotNil(t, item)
	assert.Equal(t, newSKU, jsonField(item, "sku"))
	assert.Equal(t, newDesc, jsonField(item, "description"))
	assert.Equal(t, newNotes, jsonField(item, "notes"))
	uv = jsonObject(item, "unit_value")
	require.NotNil(t, uv)
	assert.Equal(t, "rate", jsonField(uv, "object"))
}

func TestProducts_OmittedFields(t *testing.T) {
	t.Parallel()

	t.Run("CreateWithOnlyRequiredFields", func(t *testing.T) {
		sku := uniqueName("e2e-prod-omit")
		status, body, err := apiClient.Post(productsPath, validProductBody(sku), newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)

		got := parseJSON(body)
		id := jsonField(got, "id")
		require.NotEmpty(t, id)
		t.Cleanup(func() { apiClient.Delete(productsPath + "/" + id) })

		assertObjectField(t, got, "product")
		assert.Equal(t, "hidden", jsonField(got, "portal_visibility"))
		assertNilField(t, got, "product_line")

		item := jsonObject(got, "item")
		require.NotNil(t, item, "create response embeds item")
		assert.Equal(t, "item", jsonField(item, "object"))
		assert.Equal(t, sku, jsonField(item, "sku"))
	})

	t.Run("CreateMissingRequiredFields", func(t *testing.T) {
		base := validProductBody(uniqueName("e2e-prod-miss"))

		t.Run("missing_sku", func(t *testing.T) {
			status, body, err := apiClient.Post(productsPath, map[string]any{
				"type":        base["type"],
				"category_id": base["category_id"],
			}, newIdempotencyKey())
			require.NoError(t, err)
			assert.True(t, status == 400 || status == 422,
				"missing sku should return 400 or 422, got %d: %s", status, string(body))
		})

		t.Run("missing_type", func(t *testing.T) {
			status, body, err := apiClient.Post(productsPath, map[string]any{
				"sku":         base["sku"],
				"category_id": base["category_id"],
			}, newIdempotencyKey())
			require.NoError(t, err)
			assert.True(t, status == 400 || status == 422,
				"missing type should return 400 or 422, got %d: %s", status, string(body))
		})

		t.Run("missing_category_id", func(t *testing.T) {
			status, body, err := apiClient.Post(productsPath, map[string]any{
				"sku":  base["sku"],
				"type": base["type"],
			}, newIdempotencyKey())
			require.NoError(t, err)
			assert.True(t, status == 400 || status == 422,
				"missing category_id should return 400 or 422, got %d: %s", status, string(body))
		})
	})

	t.Run("UpdatePreservesOmittedFields", func(t *testing.T) {
		sku := uniqueName("e2e-prod-pres")
		createStatus, createBody, err := apiClient.Post(productsPath, map[string]any{
			"sku":         sku,
			"type":        "sale",
			"category_id": SeedItemCategoryID,
			"description": "Original description",
			"notes":       "Original notes",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, createStatus, createBody)

		id := jsonField(parseJSON(createBody), "id")
		require.NotEmpty(t, id)
		t.Cleanup(func() { apiClient.Delete(productsPath + "/" + id) })

		newSKU := uniqueName("e2e-prod-pres-u")
		patchStatus, patchBody, err := apiClient.Patch(productsPath+"/"+id, map[string]any{
			"sku": newSKU,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, patchStatus, patchBody)

		getStatus, getBody, err := apiClient.GetListRaw(productsPath+"/"+id, url.Values{"include": {"item"}})
		require.NoError(t, err)
		requireStatus(t, 200, getStatus, getBody)
		item := jsonObject(parseJSON(getBody), "item")
		require.NotNil(t, item)
		assert.Equal(t, newSKU, jsonField(item, "sku"))
		assert.Equal(t, "Original description", jsonField(item, "description"))
		assert.Equal(t, "Original notes", jsonField(item, "notes"))
	})
}

func TestProducts_CreateResponseShape(t *testing.T) {
	t.Parallel()
	sku := uniqueName("e2e-prod-shape")
	status, body, err := apiClient.Post(productsPath, validProductBody(sku), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	got := parseJSON(body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { apiClient.Delete(productsPath + "/" + id) })

	assertIDFormat(t, id, "pd")
	assertObjectField(t, got, "product")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
}

func TestProducts_CreateIdempotent(t *testing.T) {
	t.Parallel()
	sku := uniqueName("e2e-prod-idem")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Post(productsPath, validProductBody(sku), idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")
	require.NotEmpty(t, id1)
	t.Cleanup(func() { apiClient.Delete(productsPath + "/" + id1) })

	status2, body2, err := apiClient.Post(productsPath, validProductBody(sku), idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id1, jsonField(parseJSON(body2), "id"))
}

func TestProducts_UpdateIdempotent(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, productsPath, validProductBody(uniqueName("e2e-prod-idem-upd")))
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	newSKU := uniqueName("e2e-prod-idem-upd2")
	idemKey := newIdempotencyKey()

	inc := url.Values{"include": {"item"}}
	pathWithInc := productsPath + "/" + id + "?" + inc.Encode()

	status1, body1, err := apiClient.Patch(pathWithInc, map[string]any{"sku": newSKU}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)

	status2, body2, err := apiClient.Patch(pathWithInc, map[string]any{"sku": newSKU}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)

	assert.Equal(t, jsonField(parseJSON(body1), "id"), jsonField(parseJSON(body2), "id"))
	item1 := jsonObject(parseJSON(body1), "item")
	item2 := jsonObject(parseJSON(body2), "item")
	require.NotNil(t, item1)
	require.NotNil(t, item2)
	assert.Equal(t, newSKU, jsonField(item1, "sku"))
	assert.Equal(t, newSKU, jsonField(item2, "sku"))
}

func TestProducts_ChangeProductLine(t *testing.T) {
	t.Parallel()

	plStatus, plBody, err := apiClient.Post(productLinesPath, map[string]any{
		"name":              uniqueName("e2e-pdln-for-product"),
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "commission_exempt",
		"freight_policy":    "billed_freight",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, plStatus, plBody)
	newLineID := jsonField(parseJSON(plBody), "id")
	require.NotEmpty(t, newLineID)
	t.Cleanup(func() { apiClient.Delete(productLinesPath + "/" + newLineID) })

	sku := uniqueName("e2e-prod-chgpl")
	createBody := map[string]any{
		"sku":               sku,
		"type":              "sale",
		"category_id":       SeedItemCategoryID,
		"product_line_id":   SeedProductLineID,
		"portal_visibility": "hidden",
	}
	createStatus, prodBody, err := apiClient.Post(productsPath, createBody, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, prodBody)
	productID := jsonField(parseJSON(prodBody), "id")
	require.NotEmpty(t, productID)
	t.Cleanup(func() { apiClient.Delete(productsPath + "/" + productID) })

	putStatus, putBody, err := apiClient.Put(productsPath+"/"+productID+"/product-line/"+newLineID, map[string]any{})
	require.NoError(t, err)
	requireStatus(t, 200, putStatus, putBody)

	getStatus, getBody, err := apiClient.GetListRaw(productsPath+"/"+productID, url.Values{"include": {"product_line"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	pl := jsonObject(parseJSON(getBody), "product_line")
	require.NotNil(t, pl)
	assert.Equal(t, newLineID, jsonField(pl, "id"))
}
