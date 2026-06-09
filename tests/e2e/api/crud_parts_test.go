//go:build e2e

package api_test

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const partsPath = "/v1/catalog/parts"

// firstPartID returns a stable seeded part id.
func firstPartID(t *testing.T) string {
	t.Helper()
	status, body, err := apiClient.GetListRaw(partsPath+"/"+SeedPartID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return SeedPartID
}

// ──────────────────────────────────────────────
// Part — CRUD Lifecycle
// ──────────────────────────────────────────────

func TestParts_CRUD(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-part-crud")
	desc := "CRUD lifecycle description"
	body := validPartBody(sku)
	body["description"] = desc

	createStatus, createBody, err := apiClient.Post(partsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	created := parseJSON(createBody)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	assert.Equal(t, "part", jsonField(created, "object"))

	get1Status, get1Body, err := apiClient.GetListRaw(partsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, get1Status, get1Body)
	assert.Equal(t, id, jsonField(parseJSON(get1Body), "id"))

	getIncStatus, getIncBody, err := apiClient.GetListRaw(partsPath+"/"+id, url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, getIncStatus, getIncBody)
	item1 := jsonObject(parseJSON(getIncBody), "item")
	require.NotNil(t, item1)
	assert.Equal(t, sku, jsonField(item1, "sku"))
	assert.Equal(t, desc, jsonField(item1, "description"))

	newSKU := uniqueName("e2e-part-crud-upd")
	newDesc := "Updated CRUD description"
	patchStatus, patchBody, err := apiClient.Patch(partsPath+"/"+id, map[string]any{
		"sku":         newSKU,
		"description": newDesc,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	get2Status, get2Body, err := apiClient.GetListRaw(partsPath+"/"+id, url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, get2Status, get2Body)
	item2 := jsonObject(parseJSON(get2Body), "item")
	require.NotNil(t, item2)
	assert.Equal(t, newSKU, jsonField(item2, "sku"))
	assert.Equal(t, newDesc, jsonField(item2, "description"))

	delStatus, delBody, err := apiClient.Delete(partsPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	get404Status, _, err := apiClient.GetListRaw(partsPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, get404Status, "GET deleted part should return 404")
}

// ──────────────────────────────────────────────
// Part — Create / Update All Fields
// ──────────────────────────────────────────────

func TestParts_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-part-allf")
	desc := "All-fields part description"
	createBody := validPartBody(sku)
	createBody["description"] = desc

	resp, err := apiClient.PostFull(partsPath+"?include=item,item.category", createBody, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	got := parseJSON(resp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { apiClient.Delete(partsPath + "/" + id) })

	assertObjectField(t, got, "part")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	item := jsonObject(got, "item")
	require.NotNil(t, item)
	assert.Equal(t, "item", jsonField(item, "object"))
	assert.NotEmpty(t, jsonField(item, "id"))
	assert.Equal(t, sku, jsonField(item, "sku"))
	assert.Equal(t, desc, jsonField(item, "description"))
	assert.Equal(t, "part", jsonField(item, "type"))
	assertValidTimestamp(t, jsonField(item, "created_at"), "item.created_at")
	assertValidTimestamp(t, jsonField(item, "updated_at"), "item.updated_at")

	cat := jsonObject(item, "category")
	require.NotNil(t, cat)
	assert.Equal(t, SeedItemCategoryID, jsonField(cat, "id"))

	assertNilField(t, item, "unit_value")
	assertNilField(t, item, "unit_cost")
	assertNilField(t, item, "burn_rate")
	assertNilField(t, item, "attributes")

	newSKU := uniqueName("e2e-part-allf-u")
	patchStatus, patchBody, err := apiClient.Patch(partsPath+"/"+id+"?include=item,item.category", map[string]any{
		"sku": newSKU,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	assert.Equal(t, id, jsonField(updated, "id"))
	assertObjectField(t, updated, "part")
	assertValidTimestamp(t, jsonField(updated, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(updated, "updated_at"), "updated_at")

	itemU := jsonObject(updated, "item")
	require.NotNil(t, itemU)
	assert.Equal(t, newSKU, jsonField(itemU, "sku"))
	assert.Equal(t, desc, jsonField(itemU, "description"))
	assert.Equal(t, jsonField(item, "id"), jsonField(itemU, "id"))
	catU := jsonObject(itemU, "category")
	require.NotNil(t, catU)
	assert.Equal(t, SeedItemCategoryID, jsonField(catU, "id"))
}

// ──────────────────────────────────────────────
// Part — Omitted Fields
// ──────────────────────────────────────────────

func TestParts_OmittedFields(t *testing.T) {
	t.Parallel()

	t.Run("CreateWithOnlyRequiredFields", func(t *testing.T) {
		sku := uniqueName("e2e-part-omit")
		status, body, err := apiClient.Post(partsPath, validPartBody(sku), newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)

		got := parseJSON(body)
		id := jsonField(got, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(partsPath + "/" + id)

		assertObjectField(t, got, "part")
		getStatus, getBody, err := apiClient.GetListRaw(partsPath+"/"+id, url.Values{"include": {"item"}})
		require.NoError(t, err)
		requireStatus(t, 200, getStatus, getBody)
		item := jsonObject(parseJSON(getBody), "item")
		require.NotNil(t, item)
		assert.Equal(t, sku, jsonField(item, "sku"))
		assert.Nil(t, item["description"])
		assert.Nil(t, item["notes"])
	})

	t.Run("CreateMissingRequiredFields", func(t *testing.T) {
		status, body, err := apiClient.Post(partsPath, map[string]any{
			"category_id": SeedItemCategoryID,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
	})

	t.Run("UpdatePreservesOmittedFields", func(t *testing.T) {
		sku := uniqueName("e2e-part-pres")
		desc := "Preserve-me description"
		createBody := validPartBody(sku)
		createBody["description"] = desc

		createStatus, createResp, err := apiClient.Post(partsPath, createBody, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, createStatus, createResp)

		created := parseJSON(createResp)
		id := jsonField(created, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(partsPath + "/" + id)

		newSKU := uniqueName("e2e-part-pres-u")
		patchStatus, patchBody, err := apiClient.Patch(partsPath+"/"+id+"?include=item", map[string]any{
			"sku": newSKU,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, patchStatus, patchBody)

		item := jsonObject(parseJSON(patchBody), "item")
		require.NotNil(t, item)
		assert.Equal(t, newSKU, jsonField(item, "sku"))
		assert.Equal(t, desc, jsonField(item, "description"))
	})
}

// ──────────────────────────────────────────────
// Part — Response Shape
// ──────────────────────────────────────────────

func TestParts_CreateResponseShape(t *testing.T) {
	t.Parallel()

	resp, err := apiClient.PostFull(partsPath, validPartBody(uniqueName("e2e-part-shape")), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	got := parseJSON(resp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(partsPath + "/" + id)

	assertIDFormat(t, id, "pt")
	assertObjectField(t, got, "part")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	assertCreatedLocation(t, resp.Header, id)
}

// ──────────────────────────────────────────────
// Part — List
// ──────────────────────────────────────────────

func TestParts_List(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(partsPath, nil)
	require.NoError(t, err)
	require.Equal(t, 200, status)
	assert.Equal(t, "list", list.Object)
	require.GreaterOrEqual(t, len(list.Data), 1, "at least one seeded part")
}

func TestParts_ListPagination(t *testing.T) {
	t.Parallel()

	page1, _, err := apiClient.GetList(partsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	require.LessOrEqual(t, len(page1.Data), 1)

	if !page1.PageInfo.HasNextPage || page1.PageInfo.NextPageURL == nil {
		t.Fatal("Not enough parts for pagination test")
	}

	page2, _, err := apiClient.GetListFromPageURL(page1.PageInfo.NextPageURL)
	require.NoError(t, err)
	if len(page2.Data) == 0 {
		t.Fatal("Pagination page returned empty; likely parallel test interference")
	}

	id1 := DataItemField(page1.Data[0], "id")
	id2 := DataItemField(page2.Data[0], "id")
	assert.NotEqual(t, id1, id2, "pages should return different parts")
}

func TestParts_ListSearch(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-part-q")
	created := createAndCleanup(t, partsPath, validPartBody(sku))
	createdID := jsonField(created, "id")

	list, status, err := apiClient.GetList(partsPath, url.Values{"q": {sku}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.GreaterOrEqual(t, len(list.Data), 1)

	var found bool
	for _, raw := range list.Data {
		if DataItemField(raw, "id") == createdID {
			found = true
			break
		}
	}
	assert.True(t, found, "search by SKU should return the created part")
}

func TestParts_ListSearchNoResults(t *testing.T) {
	t.Parallel()

	list, _, err := apiClient.GetList(partsPath, url.Values{"q": {"zzzznotapart99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

func TestParts_ListFilterByCategoryID(t *testing.T) {
	t.Parallel()

	// Create a part we own (in SeedItemCategoryID) and assert the filter returns it
	// with item.category populated. We deliberately do NOT iterate over every part in
	// the shared filtered list: other parallel tests create and then delete parts in
	// this same category, so a part can be present in the list snapshot but have its
	// item soft-deleted by the time the include loader runs, leaving item nil.
	created := createAndCleanup(t, partsPath, validPartBody(uniqueName("e2e-part-catfilt")))
	createdID := jsonField(created, "id")
	require.NotEmpty(t, createdID)

	list, status, err := apiClient.GetList(partsPath, url.Values{
		"category_ids": {SeedItemCategoryID},
		"include":      {"item.category"},
	})
	require.NoError(t, err)
	require.Equal(t, 200, status)

	var found map[string]any
	for _, raw := range list.Data {
		part := parseJSON(raw)
		if jsonField(part, "id") == createdID {
			found = part
			break
		}
	}
	require.NotNil(t, found, "category filter should include the part we created in SeedItemCategoryID")

	item := jsonObject(found, "item")
	require.NotNil(t, item, "item should be populated via include=item.category")
	cat := jsonObject(item, "category")
	require.NotNil(t, cat, "item.category should be populated via include=item.category")
	assert.Equal(t, SeedItemCategoryID, jsonField(cat, "id"))
}

func TestParts_ListFilterByAttributeID(t *testing.T) {
	t.Parallel()

	body := validPartBody(uniqueName("e2e-part-attrfilt"))
	body["attribute_ids"] = []string{SeedAttributeID}
	created := createAndCleanup(t, partsPath, body)
	createdID := jsonField(created, "id")

	list, status, err := apiClient.GetList(partsPath, url.Values{
		"attribute_ids": {SeedAttributeID},
	})
	require.NoError(t, err)
	require.Equal(t, 200, status)

	var found bool
	for _, raw := range list.Data {
		if DataItemField(raw, "id") == createdID {
			found = true
			break
		}
	}
	assert.True(t, found, "attribute filter should include the part linked to SeedAttributeID")
}

// ──────────────────────────────────────────────
// Part — Idempotency
// ──────────────────────────────────────────────

func TestParts_CreateIdempotent(t *testing.T) {
	t.Parallel()

	idem := newIdempotencyKey()
	body := validPartBody(uniqueName("e2e-part-idem"))

	status1, body1, err := apiClient.Post(partsPath, body, idem)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id := jsonField(parseJSON(body1), "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(partsPath + "/" + id)

	status2, body2, err := apiClient.Post(partsPath, body, idem)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id, jsonField(parseJSON(body2), "id"))
}

func TestParts_UpdateIdempotent(t *testing.T) {
	t.Parallel()

	created := createAndCleanup(t, partsPath, validPartBody(uniqueName("e2e-part-patch-idem")))
	id := jsonField(created, "id")

	idem := newIdempotencyKey()
	patch := map[string]any{"sku": uniqueName("e2e-patch-idem-sku")}

	status1, body1, err := apiClient.Patch(partsPath+"/"+id, patch, idem)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)

	status2, body2, err := apiClient.Patch(partsPath+"/"+id, patch, idem)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)

	p1 := parseJSON(body1)
	p2 := parseJSON(body2)
	assert.Equal(t, jsonField(p1, "id"), jsonField(p2, "id"))
	assert.Equal(t, jsonField(p1, "object"), jsonField(p2, "object"))
	assert.Equal(t, jsonField(p1, "created_at"), jsonField(p2, "created_at"))
	assert.Equal(t, jsonField(p1, "updated_at"), jsonField(p2, "updated_at"))
}

// ──────────────────────────────────────────────
// Part — Include Tests
// ──────────────────────────────────────────────
//
// Part endpoints (retrieve, list, create, update) all whitelist:
// item, item.category, item.unit_value, item.unit_cost, item.burn_rate, item.attributes.

func TestParts_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()
	id := firstPartID(t)

	status, body, err := apiClient.GetListRaw(partsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["item"], "item should be null without ?include=item")
}

func TestParts_IncludeItem(t *testing.T) {
	t.Parallel()
	id := firstPartID(t)

	status, body, err := apiClient.GetListRaw(partsPath+"/"+id, url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present with ?include=item")
	assert.Equal(t, "item", jsonField(item, "object"))
	assert.NotEmpty(t, jsonField(item, "id"))

	assertNilField(t, item, "category")
	assertNilField(t, item, "unit_value")
	assertNilField(t, item, "unit_cost")
	assertNilField(t, item, "burn_rate")
	assertNilField(t, item, "attributes")
}

func TestParts_IncludeCategory(t *testing.T) {
	t.Parallel()
	id := firstPartID(t)

	status, body, err := apiClient.GetListRaw(partsPath+"/"+id, url.Values{"include": {"item.category"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present with ?include=item.category")
	cat := jsonObject(item, "category")
	require.NotNil(t, cat, "item.category should be present with ?include=item.category")
	assert.Equal(t, "item_category", jsonField(cat, "object"))
	assert.NotEmpty(t, jsonField(cat, "id"))
}

func TestParts_IncludeUnitValue(t *testing.T) {
	t.Parallel()
	id := firstPartID(t)

	status, body, err := apiClient.GetListRaw(partsPath+"/"+id, url.Values{"include": {"item.unit_value"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present with ?include=item.unit_value")
	_, ok := item["unit_value"]
	assert.True(t, ok, "item.unit_value key should be present with ?include=item.unit_value")
	if uv := jsonObject(item, "unit_value"); uv != nil {
		assert.Equal(t, "rate", jsonField(uv, "object"))
	}
}

func TestParts_IncludeUnitCost(t *testing.T) {
	t.Parallel()
	id := firstPartID(t)

	status, body, err := apiClient.GetListRaw(partsPath+"/"+id, url.Values{"include": {"item.unit_cost"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present with ?include=item.unit_cost")
	_, ok := item["unit_cost"]
	assert.True(t, ok, "item.unit_cost key should be present with ?include=item.unit_cost")
	if uc := jsonObject(item, "unit_cost"); uc != nil {
		assert.Equal(t, "rate", jsonField(uc, "object"))
	}
}

func TestParts_IncludeBurnRate(t *testing.T) {
	t.Parallel()
	id := firstPartID(t)

	status, body, err := apiClient.GetListRaw(partsPath+"/"+id, url.Values{"include": {"item.burn_rate"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present with ?include=item.burn_rate")
	_, ok := item["burn_rate"]
	assert.True(t, ok, "item.burn_rate key should be present with ?include=item.burn_rate")
	if br := jsonObject(item, "burn_rate"); br != nil {
		assert.Equal(t, "rate", jsonField(br, "object"))
	}
}

func TestParts_IncludeAttributes(t *testing.T) {
	t.Parallel()
	id := firstPartID(t)

	status, body, err := apiClient.GetListRaw(partsPath+"/"+id, url.Values{"include": {"item.attributes"}})
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

func TestParts_IncludeMultiple(t *testing.T) {
	t.Parallel()
	id := firstPartID(t)

	status, body, err := apiClient.GetListRaw(partsPath+"/"+id, url.Values{
		"include": {"item,item.category,item.attributes"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present with multiple includes")
	assert.Equal(t, "item", jsonField(item, "object"))

	cat := jsonObject(item, "category")
	require.NotNil(t, cat, "item.category should be present with ?include=item,item.category,item.attributes")
	assert.Equal(t, "item_category", jsonField(cat, "object"))

	attrs, ok := item["attributes"]
	assert.True(t, ok, "item.attributes key should be present with ?include=item,item.category,item.attributes")
	if ok && attrs != nil {
		attrsObj, isObj := attrs.(map[string]any)
		require.True(t, isObj, "item.attributes should be a list object")
		assert.Equal(t, "list", jsonField(attrsObj, "object"))
	}
}

// ──────────────────────────────────────────────
// Part List — Include Tests
// ──────────────────────────────────────────────

func TestParts_List_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(partsPath, nil)
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.GreaterOrEqual(t, len(list.Data), 1)

	var first map[string]any
	require.NoError(t, json.Unmarshal(list.Data[0], &first))
	assert.Nil(t, first["item"], "item should be null in list results without ?include=item")
}

func TestParts_List_IncludeItem(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(partsPath, url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	require.Equal(t, "list", jsonField(got, "object"))
	data := jsonArray(got, "data")
	require.NotEmpty(t, data, "list should have at least one part")

	verified := 0
	for _, raw := range data {
		part, ok := raw.(map[string]any)
		require.True(t, ok)
		item := jsonObject(part, "item")
		if item == nil {
			continue
		}
		assert.Equal(t, "item", jsonField(item, "object"))
		assert.NotEmpty(t, jsonField(item, "id"))
		verified++
	}
	assert.GreaterOrEqual(t, verified, 1, "at least one part should have item expanded")
}

func TestParts_List_IncludeCategory(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(partsPath, url.Values{"include": {"item.category"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	data := jsonArray(got, "data")
	require.NotEmpty(t, data)

	verified := 0
	for _, raw := range data {
		part, ok := raw.(map[string]any)
		require.True(t, ok)
		item := jsonObject(part, "item")
		if item == nil {
			continue
		}
		cat := jsonObject(item, "category")
		if cat == nil {
			continue
		}
		assert.Equal(t, "item_category", jsonField(cat, "object"))
		verified++
	}
	assert.GreaterOrEqual(t, verified, 1, "at least one part should have item.category populated")
}

func TestParts_List_IncludeAttributes(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(partsPath, url.Values{"include": {"item.attributes"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	data := jsonArray(got, "data")
	require.NotEmpty(t, data)

	verified := 0
	for _, raw := range data {
		part, ok := raw.(map[string]any)
		require.True(t, ok)
		item := jsonObject(part, "item")
		if item == nil {
			continue
		}
		attrs, ok := item["attributes"]
		if !ok || attrs == nil {
			continue
		}
		attrsObj, isObj := attrs.(map[string]any)
		require.True(t, isObj, "item.attributes should be a list object")
		assert.Equal(t, "list", jsonField(attrsObj, "object"))
		verified++
	}
	assert.GreaterOrEqual(t, verified, 1, "at least one part should have item.attributes populated")
}

// ──────────────────────────────────────────────
// Part Create — Include Tests
// ──────────────────────────────────────────────

func TestParts_Create_IncludeItem(t *testing.T) {
	t.Parallel()

	resp, err := apiClient.PostFull(partsPath+"?include=item", validPartBody(uniqueName("e2e-part-inc-item")), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	got := parseJSON(resp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { apiClient.Delete(partsPath + "/" + id) })

	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be expanded in create response with ?include=item")
	assert.Equal(t, "item", jsonField(item, "object"))
	assert.NotEmpty(t, jsonField(item, "id"))
}

func TestParts_Create_IncludeItemCategory(t *testing.T) {
	t.Parallel()

	resp, err := apiClient.PostFull(partsPath+"?include=item.category", validPartBody(uniqueName("e2e-part-inc-cat")), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	got := parseJSON(resp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { apiClient.Delete(partsPath + "/" + id) })

	item := jsonObject(got, "item")
	require.NotNil(t, item, "item should be present with ?include=item.category")
	cat := jsonObject(item, "category")
	require.NotNil(t, cat, "item.category should be expanded in create response")
	assert.Equal(t, "item_category", jsonField(cat, "object"))
	assert.NotEmpty(t, jsonField(cat, "id"))
}

func TestParts_Create_IncludeItemAttributes(t *testing.T) {
	t.Parallel()

	body := validPartBody(uniqueName("e2e-part-inc-attrs"))
	body["attribute_ids"] = []string{SeedAttributeID}

	resp, err := apiClient.PostFull(partsPath+"?include=item.attributes", body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	got := parseJSON(resp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { apiClient.Delete(partsPath + "/" + id) })

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
// Part Update — Include Tests
// ──────────────────────────────────────────────

func TestParts_Update_IncludeItem(t *testing.T) {
	t.Parallel()

	created := createAndCleanup(t, partsPath, validPartBody(uniqueName("e2e-part-upd-inc")))
	id := jsonField(created, "id")

	newSKU := uniqueName("upd-sku")
	patchStatus, patchBody, err := apiClient.Patch(
		partsPath+"/"+id+"?include=item",
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

func TestParts_Update_IncludeItemCategory(t *testing.T) {
	t.Parallel()

	created := createAndCleanup(t, partsPath, validPartBody(uniqueName("e2e-part-upd-cat")))
	id := jsonField(created, "id")

	patchStatus, patchBody, err := apiClient.Patch(
		partsPath+"/"+id+"?include=item.category",
		map[string]any{"sku": uniqueName("upd-sku")},
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

func TestParts_Update_IncludeAttributes(t *testing.T) {
	t.Parallel()

	body := validPartBody(uniqueName("e2e-part-upd-attrs"))
	body["attribute_ids"] = []string{SeedAttributeID}
	created := createAndCleanup(t, partsPath, body)
	id := jsonField(created, "id")

	patchStatus, patchBody, err := apiClient.Patch(
		partsPath+"/"+id+"?include=item.attributes",
		map[string]any{"sku": uniqueName("upd-sku")},
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
