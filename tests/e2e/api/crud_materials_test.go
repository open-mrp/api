//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const materialsPath = "/v1/catalog/materials"

// materialFullCreateBody returns a create payload with all optional fields set (no attribute_ids).
func materialFullCreateBody(sku string) map[string]any {
	body := validMaterialBody(sku)
	body["description"] = "All-fields material description"
	body["notes"] = "All-fields material notes"
	body["order_point"] = map[string]any{"value": "10", "unit_id": SeedUnitID}
	body["lead_time"] = map[string]any{"value": "5", "unit_id": SeedUnitID}
	body["unit_price"] = map[string]any{
		"value":               "4.00",
		"numerator_unit_id":   currencyUnitID,
		"denominator_unit_id": nonCurrencyUnitID,
	}
	body["unit_cost"] = map[string]any{
		"value":               "2.50",
		"numerator_unit_id":   currencyUnitID,
		"denominator_unit_id": nonCurrencyUnitID,
	}
	body["burn_rate"] = map[string]any{
		"value":               "0.15",
		"numerator_unit_id":   nonCurrencyUnitID,
		"denominator_unit_id": nonCurrencyUnitID,
	}
	return body
}

func assertMaterialQuantityInfo(t *testing.T, doc map[string]any, field string, expectUnitID string) {
	t.Helper()
	q := jsonObject(doc, field)
	require.NotNil(t, q, "%s should be set", field)
	u := jsonObject(q, "unit")
	require.NotNil(t, u, "%s.unit should be set", field)
	assert.Equal(t, expectUnitID, jsonField(u, "id"))
	assert.Equal(t, "unit", jsonField(u, "object"))
	assert.NotEmpty(t, jsonField(q, "value"), "%s.value should be non-empty", field)
}

// ──────────────────────────────────────────────
// --- CRUD ---
// ──────────────────────────────────────────────

func TestMaterials_CRUD(t *testing.T) {
	t.Parallel()
	sku := uniqueName("e2e-mat-crud")
	body := validMaterialBody(sku)
	body["order_point"] = map[string]any{"value": "3", "unit_id": SeedUnitID}
	body["lead_time"] = map[string]any{"value": "2", "unit_id": SeedUnitID}

	createStatus, createBody, err := apiClient.Post(materialsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	created := parseJSON(createBody)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	assert.Equal(t, "material", jsonField(created, "object"))
	assertMaterialQuantityInfo(t, created, "order_point", SeedUnitID)
	assertMaterialQuantityInfo(t, created, "lead_time", SeedUnitID)

	getStatus, getBody, err := apiClient.GetListRaw(materialsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assert.Equal(t, id, jsonField(got, "id"))

	newSKU := uniqueName("e2e-mat-crud-upd")
	patchStatus, patchBody, err := apiClient.Patch(materialsPath+"/"+id, map[string]any{"sku": newSKU}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	st, itemBody, err := apiClient.GetListRaw(materialsPath+"/"+id, url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, st, itemBody)
	item := jsonObject(parseJSON(itemBody), "item")
	require.NotNil(t, item)
	assert.Equal(t, newSKU, jsonField(item, "sku"))

	delStatus, delBody, err := apiClient.Delete(materialsPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	getStatus2, _, err := apiClient.GetListRaw(materialsPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus2)
}

// ──────────────────────────────────────────────
// --- Create and Update All Fields ---
// ──────────────────────────────────────────────

func TestMaterials_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()
	sku := uniqueName("e2e-mat-allf")
	body := materialFullCreateBody(sku)

	createStatus, createBody, err := apiClient.Post(materialsPath+"?include=item", body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	got := parseJSON(createBody)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(materialsPath + "/" + id)

	assertObjectField(t, got, "material")
	assertMaterialQuantityInfo(t, got, "order_point", SeedUnitID)
	assertMaterialQuantityInfo(t, got, "lead_time", SeedUnitID)
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	item := jsonObject(got, "item")
	require.NotNil(t, item, "item must be present with ?include=item")
	assert.Equal(t, "item", jsonField(item, "object"))
	assert.NotEmpty(t, jsonField(item, "id"))
	assert.Equal(t, sku, jsonField(item, "sku"))
	assert.Equal(t, "All-fields material description", jsonField(item, "description"))
	assert.Equal(t, "All-fields material notes", jsonField(item, "notes"))

	origCreatedAt := jsonField(got, "created_at")

	newSKU := uniqueName("e2e-mat-allf-u")
	patchStatus, patchBody, err := apiClient.Patch(materialsPath+"/"+id+"?include=item", map[string]any{
		"sku":         newSKU,
		"description": "Updated all-fields description",
		"notes":       "Updated all-fields notes",
		"order_point": map[string]any{"value": "20", "unit_id": SeedUnitID},
		"lead_time":   map[string]any{"value": "8", "unit_id": SeedUnitID},
		"unit_cost": map[string]any{
			"value":               "3.00",
			"numerator_unit_id":   currencyUnitID,
			"denominator_unit_id": nonCurrencyUnitID,
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	assert.Equal(t, id, jsonField(updated, "id"))
	assert.Equal(t, origCreatedAt, jsonField(updated, "created_at"), "created_at must not change")
	assertValidTimestamp(t, jsonField(updated, "updated_at"), "updated_at")

	assertMaterialQuantityInfo(t, updated, "order_point", SeedUnitID)
	assertMaterialQuantityInfo(t, updated, "lead_time", SeedUnitID)

	upItem := jsonObject(updated, "item")
	require.NotNil(t, upItem)
	assert.Equal(t, newSKU, jsonField(upItem, "sku"))
	assert.Equal(t, "Updated all-fields description", jsonField(upItem, "description"))
	assert.Equal(t, "Updated all-fields notes", jsonField(upItem, "notes"))
}

// ──────────────────────────────────────────────
// --- Omitted Fields ---
// ──────────────────────────────────────────────

func TestMaterials_OmittedFields(t *testing.T) {
	t.Parallel()

	t.Run("CreateWithOnlyRequiredFields", func(t *testing.T) {
		sku := uniqueName("e2e-mat-omit")
		status, body, err := apiClient.Post(materialsPath, validMaterialBody(sku), newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)

		got := parseJSON(body)
		id := jsonField(got, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(materialsPath + "/" + id)

		assertObjectField(t, got, "material")
		assertNilField(t, got, "order_point")
		assertNilField(t, got, "lead_time")
		assertNilField(t, got, "item")
		assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	})

	t.Run("CreateMissingRequiredFields", func(t *testing.T) {
		status, body, err := apiClient.Post(materialsPath, map[string]any{
			"category_id": SeedItemCategoryID,
		}, newIdempotencyKey())
		require.NoError(t, err)
		assert.True(t, status == 400 || status == 422,
			"missing sku should return 400 or 422, got %d: %s", status, string(body))

		status2, body2, err := apiClient.Post(materialsPath, map[string]any{
			"sku": uniqueName("e2e-mat-miss-cat"),
		}, newIdempotencyKey())
		require.NoError(t, err)
		assert.True(t, status2 == 400 || status2 == 422,
			"missing category_id should return 400 or 422, got %d: %s", status2, string(body2))
	})

	t.Run("UpdatePreservesOmittedFields", func(t *testing.T) {
		sku := uniqueName("e2e-mat-pres")
		createStatus, createBody, err := apiClient.Post(materialsPath+"?include=item", materialFullCreateBody(sku), newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, createStatus, createBody)

		created := parseJSON(createBody)
		id := jsonField(created, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(materialsPath + "/" + id)

		origOPVal := jsonField(jsonObject(created, "order_point"), "value")
		origLTVal := jsonField(jsonObject(created, "lead_time"), "value")
		item0 := jsonObject(created, "item")
		require.NotNil(t, item0)
		origDesc := jsonField(item0, "description")

		newSKU := uniqueName("e2e-mat-pres-u")
		patchStatus, patchBody, err := apiClient.Patch(materialsPath+"/"+id, map[string]any{"sku": newSKU}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, patchStatus, patchBody)

		patched := parseJSON(patchBody)
		assert.Equal(t, origOPVal, jsonField(jsonObject(patched, "order_point"), "value"), "order_point should be preserved")
		assert.Equal(t, origLTVal, jsonField(jsonObject(patched, "lead_time"), "value"), "lead_time should be preserved")

		st, body, err := apiClient.GetListRaw(materialsPath+"/"+id, url.Values{"include": {"item"}})
		require.NoError(t, err)
		requireStatus(t, 200, st, body)
		item := jsonObject(parseJSON(body), "item")
		require.NotNil(t, item)
		assert.Equal(t, origDesc, jsonField(item, "description"), "description should be preserved")
		assert.Equal(t, newSKU, jsonField(item, "sku"), "sku should reflect PATCH")
	})
}

// ──────────────────────────────────────────────
// --- Response Shape ---
// ──────────────────────────────────────────────

func TestMaterials_CreateResponseShape(t *testing.T) {
	t.Parallel()
	sku := uniqueName("e2e-mat-shape")
	status, body, err := apiClient.Post(materialsPath, validMaterialBody(sku), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	got := parseJSON(body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(materialsPath + "/" + id)

	assertIDFormat(t, id, "ml")
	assertObjectField(t, got, "material")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
}

// ──────────────────────────────────────────────
// --- List ---
// ──────────────────────────────────────────────

func TestMaterials_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(materialsPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 1, "should have at least one seeded material")
}

func TestMaterials_ListPagination(t *testing.T) {
	t.Parallel()
	page1, _, err := apiClient.GetList(materialsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	require.Len(t, page1.Data, 1)

	if !page1.PageInfo.HasNextPage {
		t.Skip("not enough materials for pagination test")
		return
	}
	require.NotNil(t, page1.PageInfo.NextPageURL)

	page2, _, err := apiClient.GetListFromPageURL(page1.PageInfo.NextPageURL)
	require.NoError(t, err)
	require.Len(t, page2.Data, 1)

	id1 := DataItemField(page1.Data[0], "id")
	id2 := DataItemField(page2.Data[0], "id")
	assert.NotEqual(t, id1, id2, "pages should return different items")
}

func TestMaterials_ListSearch(t *testing.T) {
	t.Parallel()
	sku := uniqueName("e2e-mat-q")
	body := validMaterialBody(sku)
	createStatus, createBody, err := apiClient.Post(materialsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(materialsPath + "/" + id)

	list, _, err := apiClient.GetList(materialsPath, url.Values{"q": {sku}})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1, "search by SKU should return the created material")

	found := false
	for _, raw := range list.Data {
		if DataItemField(raw, "id") == id {
			found = true
			break
		}
	}
	assert.True(t, found, "created material id %s should appear in search results", id)
}

func TestMaterials_ListSearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(materialsPath, url.Values{"q": {"zzzznotamaterial99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

// ──────────────────────────────────────────────
// --- List Filters ---
// ──────────────────────────────────────────────

func TestMaterials_ListFilterByCategoryID(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(materialsPath, url.Values{"category_ids": {SeedMaterialCategoryID}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "filter by seeded material category should return at least one material")
}

func TestMaterials_ListFilterByAttributeID(t *testing.T) {
	t.Parallel()
	sku := uniqueName("e2e-mat-fattr")
	body := validMaterialBody(sku)
	body["attribute_ids"] = []string{SeedAttributeID}
	createStatus, createBody, err := apiClient.Post(materialsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(materialsPath + "/" + id)

	list, _, err := apiClient.GetList(materialsPath, url.Values{"attribute_ids": {SeedAttributeID}})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	found := false
	for _, raw := range list.Data {
		if DataItemField(raw, "id") == id {
			found = true
			break
		}
	}
	assert.True(t, found, "material with linked attribute should appear when filtering by attribute_ids")
}

// ──────────────────────────────────────────────
// --- Idempotency ---
// ──────────────────────────────────────────────

func TestMaterials_CreateIdempotent(t *testing.T) {
	t.Parallel()
	sku := uniqueName("e2e-idem-mat")
	body := validMaterialBody(sku)
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Post(materialsPath, body, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")

	status2, body2, err := apiClient.Post(materialsPath, body, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id1, jsonField(parseJSON(body2), "id"))

	apiClient.Delete(materialsPath + "/" + id1)
}

func TestMaterials_UpdateIdempotent(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, materialsPath, validMaterialBody(uniqueName("e2e-idem-mat-upd")))
	id := jsonField(created, "id")

	idemKey := newIdempotencyKey()
	newSKU := uniqueName("e2e-idem-mat-upd2")
	payload := map[string]any{"sku": newSKU}

	status1, body1, err := apiClient.Patch(materialsPath+"/"+id, payload, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)

	status2, body2, err := apiClient.Patch(materialsPath+"/"+id, payload, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)

	assert.JSONEq(t, string(body1), string(body2))
}

// ──────────────────────────────────────────────
// --- Material Create — Include Tests ---
// ──────────────────────────────────────────────

func TestMaterials_Create_IncludeItem(t *testing.T) {
	t.Parallel()

	resp, err := apiClient.PostFull(materialsPath+"?include=item", validMaterialBody(uniqueName("e2e-mat-inc-item")), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	got := parseJSON(resp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { apiClient.Delete(materialsPath + "/" + id) })

	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be expanded in create response with ?include=item")
	assert.Equal(t, "item", jsonField(item, "object"))
	assert.NotEmpty(t, jsonField(item, "id"))
}

func TestMaterials_Create_IncludeItemCategory(t *testing.T) {
	t.Parallel()

	resp, err := apiClient.PostFull(materialsPath+"?include=item.category", validMaterialBody(uniqueName("e2e-mat-inc-cat")), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	got := parseJSON(resp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { apiClient.Delete(materialsPath + "/" + id) })

	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present with ?include=item.category")
	cat := jsonObject(item, "category")
	require.NotNil(t, cat, "item.category should be expanded in create response")
	assert.Equal(t, "item_category", jsonField(cat, "object"))
	assert.NotEmpty(t, jsonField(cat, "id"))
}

func TestMaterials_Create_IncludeItemAttributes(t *testing.T) {
	t.Parallel()

	body := validMaterialBody(uniqueName("e2e-mat-inc-attrs-create"))
	body["attribute_ids"] = []string{SeedAttributeID}

	resp, err := apiClient.PostFull(materialsPath+"?include=item.attributes", body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	got := parseJSON(resp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { apiClient.Delete(materialsPath + "/" + id) })

	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present with ?include=item.attributes")
	attrs, ok := item["attributes"]
	assert.True(t, ok, "item.attributes key should be present in create response")
	if ok && attrs != nil {
		attrsObj, isObj := attrs.(map[string]any)
		require.True(t, isObj, "item.attributes should be a list object")
		assert.Equal(t, "list", jsonField(attrsObj, "object"))
		data := jsonArray(attrsObj, "data")
		require.NotEmpty(t, data, "linked attribute should appear in item.attributes")
		found := false
		for _, raw := range data {
			attr, ok := raw.(map[string]any)
			require.True(t, ok)
			if jsonField(attr, "id") == SeedAttributeID {
				found = true
			}
		}
		assert.True(t, found, "SeedAttributeID %s should appear in item.attributes", SeedAttributeID)
	}
}

// ──────────────────────────────────────────────
// --- Material Update — Include Tests ---
// ──────────────────────────────────────────────

func TestMaterials_Update_IncludeItem(t *testing.T) {
	t.Parallel()

	created := createAndCleanup(t, materialsPath, validMaterialBody(uniqueName("e2e-mat-upd-inc")))
	id := jsonField(created, "id")

	newSKU := uniqueName("e2e-mat-upd-sku")
	patchStatus, patchBody, err := apiClient.Patch(
		materialsPath+"/"+id+"?include=item",
		map[string]any{"sku": newSKU},
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	got := parseJSON(patchBody)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be expanded in update response with ?include=item")
	assert.Equal(t, "item", jsonField(item, "object"))
	assert.NotEmpty(t, jsonField(item, "id"))
	assert.Equal(t, newSKU, jsonField(item, "sku"))
}

func TestMaterials_Update_IncludeItemCategory(t *testing.T) {
	t.Parallel()

	created := createAndCleanup(t, materialsPath, validMaterialBody(uniqueName("e2e-mat-upd-cat")))
	id := jsonField(created, "id")

	patchStatus, patchBody, err := apiClient.Patch(
		materialsPath+"/"+id+"?include=item.category",
		map[string]any{"sku": uniqueName("e2e-mat-upd-cat-sku")},
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	got := parseJSON(patchBody)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present with ?include=item.category")
	cat := jsonObject(item, "category")
	require.NotNil(t, cat, "item.category should be expanded in update response")
	assert.Equal(t, "item_category", jsonField(cat, "object"))
}

func TestMaterials_Update_IncludeItemAttributes(t *testing.T) {
	t.Parallel()

	body := validMaterialBody(uniqueName("e2e-mat-upd-attrs"))
	body["attribute_ids"] = []string{SeedAttributeID}
	created := createAndCleanup(t, materialsPath, body)
	id := jsonField(created, "id")

	patchStatus, patchBody, err := apiClient.Patch(
		materialsPath+"/"+id+"?include=item.attributes",
		map[string]any{"sku": uniqueName("e2e-mat-upd-attrs-sku")},
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	got := parseJSON(patchBody)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present with ?include=item.attributes")
	attrs, ok := item["attributes"]
	assert.True(t, ok, "item.attributes key should be present in update response")
	if ok && attrs != nil {
		attrsObj, isObj := attrs.(map[string]any)
		require.True(t, isObj, "item.attributes should be a list object")
		assert.Equal(t, "list", jsonField(attrsObj, "object"))
	}
}

// ──────────────────────────────────────────────
// --- Item.Attributes include (retrieve) ---
// ──────────────────────────────────────────────

func TestMaterials_IncludeItemAttributes(t *testing.T) {
	t.Parallel()
	id := SeedMaterialID

	status, body, err := apiClient.GetListRaw(materialsPath+"/"+id, url.Values{"include": {"item.attributes"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present with ?include=item.attributes")
	attrs, ok := item["attributes"]
	assert.True(t, ok, "item.attributes key should be present with ?include=item.attributes")
	if ok && attrs != nil {
		attrsObj, isObj := attrs.(map[string]any)
		require.True(t, isObj, "item.attributes should be a list object")
		assert.Equal(t, "list", jsonField(attrsObj, "object"))
		data := jsonArray(attrsObj, "data")
		for _, raw := range data {
			attr, ok := raw.(map[string]any)
			require.True(t, ok, "each attribute should be an object")
			assert.Equal(t, "attribute", jsonField(attr, "object"))
			assert.NotEmpty(t, jsonField(attr, "id"))
		}
	}
}

// ──────────────────────────────────────────────
// Material — Include Tests
// ──────────────────────────────────────────────

func TestMaterials_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()
	id := SeedMaterialID

	status, body, err := apiClient.GetListRaw(materialsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["item"], "item should be null without ?include=item")

	list, _, err := apiClient.GetList(materialsPath, nil)
	require.NoError(t, err)
	for _, m := range list.Data {
		mm := parseJSON(m)
		assert.Nil(t, mm["item"], "item should be null on list items without ?include=item")
	}
}

func TestMaterials_IncludeItem(t *testing.T) {
	t.Parallel()
	id := SeedMaterialID

	status, body, err := apiClient.GetListRaw(materialsPath+"/"+id, url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present with ?include=item")
	assert.Equal(t, "item", jsonField(item, "object"))
	assert.NotEmpty(t, jsonField(item, "id"))

	// Nested expandable fields on item should remain null when only ?include=item is used.
	assertNilField(t, item, "category")
	assertNilField(t, item, "unit_value")
	assertNilField(t, item, "unit_cost")
	assertNilField(t, item, "burn_rate")
	assertNilField(t, item, "attributes")
}

// materialWithRatesID creates a material with unit_price, unit_cost, and burn_rate,
// registers cleanup, and returns its ID. Used by include tests that need rate data.
func materialWithRatesID(t *testing.T) string {
	t.Helper()
	body := validMaterialBody(uniqueName("e2e-mat-inc"))
	body["unit_price"] = map[string]any{
		"value":               "2.00",
		"numerator_unit_id":   currencyUnitID,
		"denominator_unit_id": nonCurrencyUnitID,
	}
	body["unit_cost"] = map[string]any{
		"value":               "1.00",
		"numerator_unit_id":   currencyUnitID,
		"denominator_unit_id": nonCurrencyUnitID,
	}
	body["burn_rate"] = map[string]any{
		"value":               "0.05",
		"numerator_unit_id":   nonCurrencyUnitID,
		"denominator_unit_id": nonCurrencyUnitID,
	}
	created := createAndCleanup(t, materialsPath, body)
	return jsonField(created, "id")
}

// ──────────────────────────────────────────────
// Material — Nested Include Tests (single endpoint)
// ──────────────────────────────────────────────

func TestMaterials_IncludeItemCategory(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(materialsPath+"/"+SeedMaterialID, url.Values{"include": {"item.category"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present when include=item.category is requested")
	assert.Equal(t, "item", jsonField(item, "object"))
	assert.NotEmpty(t, jsonField(item, "id"))

	category := jsonObject(item, "category")
	require.NotNil(t, category, "item.category should be present with include=item.category")
	assert.Equal(t, "item_category", jsonField(category, "object"))
	assert.NotEmpty(t, jsonField(category, "id"))
	assert.NotEmpty(t, jsonField(category, "name"))

	// Non-requested nested expandable fields should remain null.
	assertNilField(t, item, "unit_value")
	assertNilField(t, item, "unit_cost")
	assertNilField(t, item, "burn_rate")
	assertNilField(t, item, "attributes")
}

func TestMaterials_IncludeItemUnitValue(t *testing.T) {
	t.Parallel()
	id := materialWithRatesID(t)

	status, body, err := apiClient.GetListRaw(materialsPath+"/"+id, url.Values{"include": {"item.unit_value"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present when include=item.unit_value is requested")

	unitValue := jsonObject(item, "unit_value")
	require.NotNil(t, unitValue, "item.unit_value should be present with include=item.unit_value")
	assert.Equal(t, "rate", jsonField(unitValue, "object"))
	assert.NotEmpty(t, jsonField(unitValue, "id"))
	assert.NotEmpty(t, jsonField(unitValue, "value"))
	assert.NotEmpty(t, jsonField(unitValue, "display_value"))

	assertNilField(t, item, "category")
	assertNilField(t, item, "unit_cost")
	assertNilField(t, item, "burn_rate")
	assertNilField(t, item, "attributes")
}

func TestMaterials_IncludeItemUnitCost(t *testing.T) {
	t.Parallel()
	id := materialWithRatesID(t)

	status, body, err := apiClient.GetListRaw(materialsPath+"/"+id, url.Values{"include": {"item.unit_cost"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present when include=item.unit_cost is requested")

	unitCost := jsonObject(item, "unit_cost")
	require.NotNil(t, unitCost, "item.unit_cost should be present with include=item.unit_cost")
	assert.Equal(t, "rate", jsonField(unitCost, "object"))
	assert.NotEmpty(t, jsonField(unitCost, "id"))
	assert.NotEmpty(t, jsonField(unitCost, "value"))
	assert.NotEmpty(t, jsonField(unitCost, "display_value"))

	assertNilField(t, item, "category")
	assertNilField(t, item, "unit_value")
	assertNilField(t, item, "burn_rate")
	assertNilField(t, item, "attributes")
}

func TestMaterials_IncludeItemBurnRate(t *testing.T) {
	t.Parallel()
	id := materialWithRatesID(t)

	status, body, err := apiClient.GetListRaw(materialsPath+"/"+id, url.Values{"include": {"item.burn_rate"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present when include=item.burn_rate is requested")

	burnRate := jsonObject(item, "burn_rate")
	require.NotNil(t, burnRate, "item.burn_rate should be present with include=item.burn_rate")
	assert.Equal(t, "rate", jsonField(burnRate, "object"))
	assert.NotEmpty(t, jsonField(burnRate, "id"))
	assert.NotEmpty(t, jsonField(burnRate, "value"))
	assert.NotEmpty(t, jsonField(burnRate, "display_value"))

	assertNilField(t, item, "category")
	assertNilField(t, item, "unit_value")
	assertNilField(t, item, "unit_cost")
	assertNilField(t, item, "attributes")
}

func TestMaterials_IncludeAllFields(t *testing.T) {
	t.Parallel()
	id := materialWithRatesID(t)

	params := url.Values{"include": {"item,item.category,item.unit_value,item.unit_cost,item.burn_rate"}}
	status, body, err := apiClient.GetListRaw(materialsPath+"/"+id, params)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present")
	assert.Equal(t, "item", jsonField(item, "object"))
	assert.NotEmpty(t, jsonField(item, "id"))

	category := jsonObject(item, "category")
	require.NotNil(t, category, "item.category should be present")
	assert.Equal(t, "item_category", jsonField(category, "object"))
	assert.NotEmpty(t, jsonField(category, "id"))

	unitValue := jsonObject(item, "unit_value")
	require.NotNil(t, unitValue, "item.unit_value should be present")
	assert.Equal(t, "rate", jsonField(unitValue, "object"))
	assert.NotEmpty(t, jsonField(unitValue, "id"))

	unitCost := jsonObject(item, "unit_cost")
	require.NotNil(t, unitCost, "item.unit_cost should be present")
	assert.Equal(t, "rate", jsonField(unitCost, "object"))
	assert.NotEmpty(t, jsonField(unitCost, "id"))

	burnRate := jsonObject(item, "burn_rate")
	require.NotNil(t, burnRate, "item.burn_rate should be present")
	assert.Equal(t, "rate", jsonField(burnRate, "object"))
	assert.NotEmpty(t, jsonField(burnRate, "id"))
}

// ──────────────────────────────────────────────
// Material List — Include Tests
// ──────────────────────────────────────────────

func TestMaterials_List_IncludeItem(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(materialsPath, url.Values{"include": {"item"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.GreaterOrEqual(t, len(list.Data), 1, "at least one material must be seeded")

	for _, raw := range list.Data {
		m := parseJSON(raw)
		item := jsonObject(m, "item")
		require.NotNil(t, item, "item should be present on every list entry with include=item")
		assert.Equal(t, "item", jsonField(item, "object"))
		assert.NotEmpty(t, jsonField(item, "id"))
		// Nested expandable fields on item should remain null.
		assertNilField(t, item, "category")
		assertNilField(t, item, "unit_value")
		assertNilField(t, item, "unit_cost")
		assertNilField(t, item, "burn_rate")
	}
}

func TestMaterials_List_IncludeItemCategory(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(materialsPath, url.Values{"include": {"item.category"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.GreaterOrEqual(t, len(list.Data), 1, "at least one material must be seeded")

	for _, raw := range list.Data {
		m := parseJSON(raw)
		item := jsonObject(m, "item")
		require.NotNil(t, item, "item should be present on every list entry with include=item.category")
		category := jsonObject(item, "category")
		require.NotNil(t, category, "item.category should be present on every list entry with include=item.category")
		assert.Equal(t, "item_category", jsonField(category, "object"))
		assert.NotEmpty(t, jsonField(category, "id"))
	}
}

func TestMaterials_List_IncludeItemUnitValue(t *testing.T) {
	t.Parallel()
	sku := uniqueName("e2e-mat-luv")
	body := validMaterialBody(sku)
	body["unit_price"] = map[string]any{
		"value":               "3.00",
		"numerator_unit_id":   currencyUnitID,
		"denominator_unit_id": nonCurrencyUnitID,
	}
	created := createAndCleanup(t, materialsPath, body)
	createdID := jsonField(created, "id")

	list, status, err := apiClient.GetList(materialsPath, url.Values{
		"include": {"item.unit_value"},
		"q":       {sku},
	})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.GreaterOrEqual(t, len(list.Data), 1, "search by SKU should return the created material")

	var found map[string]any
	for _, raw := range list.Data {
		m := parseJSON(raw)
		if jsonField(m, "id") == createdID {
			found = m
			break
		}
	}
	require.NotNil(t, found, "created material should appear in list results")

	item := jsonObject(found, "item")
	require.NotNil(t, item, "item should be present with include=item.unit_value")
	unitValue := jsonObject(item, "unit_value")
	require.NotNil(t, unitValue, "item.unit_value should be present with include=item.unit_value")
	assert.Equal(t, "rate", jsonField(unitValue, "object"))
	assert.NotEmpty(t, jsonField(unitValue, "id"))
}

func TestMaterials_List_IncludeItemUnitCost(t *testing.T) {
	t.Parallel()
	sku := uniqueName("e2e-mat-luc")
	body := validMaterialBody(sku)
	body["unit_cost"] = map[string]any{
		"value":               "1.50",
		"numerator_unit_id":   currencyUnitID,
		"denominator_unit_id": nonCurrencyUnitID,
	}
	created := createAndCleanup(t, materialsPath, body)
	createdID := jsonField(created, "id")

	list, status, err := apiClient.GetList(materialsPath, url.Values{
		"include": {"item.unit_cost"},
		"q":       {sku},
	})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.GreaterOrEqual(t, len(list.Data), 1, "search by SKU should return the created material")

	var found map[string]any
	for _, raw := range list.Data {
		m := parseJSON(raw)
		if jsonField(m, "id") == createdID {
			found = m
			break
		}
	}
	require.NotNil(t, found, "created material should appear in list results")

	item := jsonObject(found, "item")
	require.NotNil(t, item, "item should be present with include=item.unit_cost")
	unitCost := jsonObject(item, "unit_cost")
	require.NotNil(t, unitCost, "item.unit_cost should be present with include=item.unit_cost")
	assert.Equal(t, "rate", jsonField(unitCost, "object"))
	assert.NotEmpty(t, jsonField(unitCost, "id"))
}

func TestMaterials_List_IncludeItemBurnRate(t *testing.T) {
	t.Parallel()
	sku := uniqueName("e2e-mat-lbr")
	body := validMaterialBody(sku)
	body["burn_rate"] = map[string]any{
		"value":               "0.20",
		"numerator_unit_id":   nonCurrencyUnitID,
		"denominator_unit_id": nonCurrencyUnitID,
	}
	created := createAndCleanup(t, materialsPath, body)
	createdID := jsonField(created, "id")

	list, status, err := apiClient.GetList(materialsPath, url.Values{
		"include": {"item.burn_rate"},
		"q":       {sku},
	})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.GreaterOrEqual(t, len(list.Data), 1, "search by SKU should return the created material")

	var found map[string]any
	for _, raw := range list.Data {
		m := parseJSON(raw)
		if jsonField(m, "id") == createdID {
			found = m
			break
		}
	}
	require.NotNil(t, found, "created material should appear in list results")

	item := jsonObject(found, "item")
	require.NotNil(t, item, "item should be present with include=item.burn_rate")
	burnRate := jsonObject(item, "burn_rate")
	require.NotNil(t, burnRate, "item.burn_rate should be present with include=item.burn_rate")
	assert.Equal(t, "rate", jsonField(burnRate, "object"))
	assert.NotEmpty(t, jsonField(burnRate, "id"))
}
