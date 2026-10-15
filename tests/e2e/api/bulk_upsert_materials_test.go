//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/open-mrp/api/shared/id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The materials bulk upsert answers preview.9 callers only; forge upserts every item type through
// /v1/catalog/items/actions/bulk-upsert (bulk_upsert_items_test.go).
const materialsBulkUpsertPath = materialsPath + "/actions/bulk-upsert"

func bulkUpsertMaterials(t *testing.T, materials ...map[string]any) (int, []byte) {
	t.Helper()
	rows := make([]any, len(materials))
	for i, m := range materials {
		rows[i] = m
	}
	status, body, err := materialsP9().Post(materialsBulkUpsertPath, map[string]any{"materials": rows}, newIdempotencyKey())
	require.NoError(t, err)
	return status, body
}

// bulkQuantity builds an order_point / lead_time quantity input.
func bulkQuantity(value string) map[string]any {
	return map[string]any{"value": value, "unit_id": SeedUnitID}
}

func cleanupMaterialIDs(ids []string) {
	for _, materialID := range ids {
		if materialID != "" {
			materialsP9().Delete(materialsPath + "/" + materialID)
		}
	}
}

// posts a bulk upsert, requires the 202 job acknowledgment, and returns the completed job
func bulkUpsertMaterialsJob(t *testing.T, materials ...map[string]any) map[string]any {
	t.Helper()
	status, body := bulkUpsertMaterials(t, materials...)
	requireStatus(t, 202, status, body)

	m := parseJSON(body)
	assert.Equal(t, "job", jsonField(m, "object"), "202 returns the job, under its preview.9 name")
	jobID := jsonField(m, "id")
	require.NotEmpty(t, jobID, "202 must name the job to poll")

	job := pollJobUntilTerminal(t, jobID)
	require.Equal(t, "completed", jsonField(job, "status"), "job should complete: %v", job)
	return job
}

// posts a bulk upsert, follows the job to completion, and returns the created/updated ids
func bulkUpsertMaterialIDs(t *testing.T, materials ...map[string]any) (createdIDs, updatedIDs []string) {
	t.Helper()
	job := bulkUpsertMaterialsJob(t, materials...)
	require.NotEmpty(t, jobResults(job), "a completed job must carry results")
	return jobResultIDs(job)
}

// --- writes ---

// Every field a create row carries lands; notes are accepted but no longer stored.
func TestMaterials_BulkUpsert_CreateWithAllFields(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-bup-mat-full")
	propName := uniqueName("e2e-bup-mat-prop")
	// Attribute values are unique per account, so use a per-run value.
	propValue := uniqueName("a36")
	created, updated := bulkUpsertMaterialIDs(t, map[string]any{
		"sku":         sku,
		"category":    map[string]any{"id": SeedMaterialCategoryID},
		"description": "full material description",
		"notes":       "not stored",
		"order_point": bulkQuantity("10"),
		"lead_time":   bulkQuantity("5"),
		"unit_price":  bulkCurrencyRate("1.50"),
		"unit_cost":   bulkCurrencyRate("0.75"),
		"properties":  []any{bulkProperty(propName, propValue)},
	})
	defer cleanupMaterialIDs(created)
	require.Len(t, created, 1)
	assert.Empty(t, updated)
	assertIDFormat(t, created[0], id.MaterialIDPrefix)

	got := getMaterialP9(t, created[0], "item")
	assertMaterialP9Envelope(t, got)
	assertMaterialP9Quantity(t, jsonObject(got, "order_point"), "10", SeedUnitID, "order_point")
	assertMaterialP9Quantity(t, jsonObject(got, "lead_time"), "5", SeedUnitID, "lead_time")

	item := jsonObject(got, "item")
	require.NotNil(t, item)
	assertPreview9ItemShape(t, item)
	assert.Equal(t, sku, jsonField(item, "sku"))
	assert.Equal(t, "full material description", jsonField(item, "description"))
	assertMaterialP9Rate(t, jsonObject(item, "unit_value"), "1.50", "", nonCurrencyUnitID, "item.unit_value")
	assertMaterialP9Rate(t, jsonObject(item, "unit_cost"), "0.75", "", nonCurrencyUnitID, "item.unit_cost")
	assert.Contains(t, catalogItemAttributeValues(t, jsonField(item, "id")), propValue)
}

// A row whose SKU exists updates every field it carries; a new SKU in the same batch is created.
func TestMaterials_BulkUpsert_UpdatesEveryField(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-bup-mat-upd")
	createdIDs, _ := bulkUpsertMaterialIDs(t, map[string]any{
		"sku":         sku,
		"category":    map[string]any{"id": SeedMaterialCategoryID},
		"description": "original description",
		"order_point": bulkQuantity("10"),
		"lead_time":   bulkQuantity("5"),
		"unit_price":  bulkCurrencyRate("1.50"),
		"unit_cost":   bulkCurrencyRate("0.75"),
	})
	defer cleanupMaterialIDs(createdIDs)
	require.Len(t, createdIDs, 1)
	materialID := createdIDs[0]

	propName := uniqueName("e2e-bup-mat-upd-prop")
	propValue := uniqueName("316l")
	created, updated := bulkUpsertMaterialIDs(t,
		map[string]any{
			"sku":         sku,
			"category":    map[string]any{"id": SeedMaterialCategoryID},
			"description": "updated description",
			"order_point": bulkQuantity("20"),
			"lead_time":   bulkQuantity("8"),
			"unit_price":  bulkCurrencyRate("2.22"),
			"unit_cost":   bulkCurrencyRate("1.11"),
			"properties":  []any{bulkProperty(propName, propValue)},
		},
		map[string]any{"sku": uniqueName("e2e-bup-mat-upd-new"), "category": map[string]any{"id": SeedMaterialCategoryID}},
	)
	defer cleanupMaterialIDs(created)
	assert.Len(t, created, 1, "the new SKU is created")
	assert.Equal(t, []string{materialID}, updated, "the existing SKU is updated, not created")

	got := getMaterialP9(t, materialID, "item")
	assertMaterialP9Quantity(t, jsonObject(got, "order_point"), "20", SeedUnitID, "order_point")
	assertMaterialP9Quantity(t, jsonObject(got, "lead_time"), "8", SeedUnitID, "lead_time")
	item := jsonObject(got, "item")
	require.NotNil(t, item)
	assert.Equal(t, sku, jsonField(item, "sku"), "a case-insensitive match keeps the stored SKU")
	assert.Equal(t, "updated description", jsonField(item, "description"))
	assertMaterialP9Rate(t, jsonObject(item, "unit_value"), "2.22", "", nonCurrencyUnitID, "item.unit_value")
	assertMaterialP9Rate(t, jsonObject(item, "unit_cost"), "1.11", "", nonCurrencyUnitID, "item.unit_cost")
	assert.Contains(t, catalogItemAttributeValues(t, jsonField(item, "id")), propValue)
}

// The category is a fuzzy reference: it resolves by name, not only by id.
func TestMaterials_BulkUpsert_ResolvesCategoryByName(t *testing.T) {
	t.Parallel()

	created, _ := bulkUpsertMaterialIDs(t, map[string]any{
		"sku":      uniqueName("e2e-bup-mat-catname"),
		"category": map[string]any{"name": "yarn"}, // by name, case-insensitive
	})
	defer cleanupMaterialIDs(created)
	require.Len(t, created, 1)

	item := jsonObject(getMaterialP9(t, created[0], "item.category"), "item")
	require.NotNil(t, item)
	assert.Equal(t, SeedMaterialCategoryID, jsonField(jsonObject(item, "category"), "id"))
}

// --- rejections ---

// Rules the write applies fail their row: the job completes and the row lands in errors.
func TestMaterials_BulkUpsert_RowRejections(t *testing.T) {
	t.Parallel()

	t.Run("price per a currency", func(t *testing.T) {
		job := bulkUpsertMaterialsJob(t, map[string]any{
			"sku":        uniqueName("e2e-bup-mat-badrate"),
			"category":   map[string]any{"id": SeedMaterialCategoryID},
			"unit_price": map[string]any{"amount": "1.00", "currency": "usd", "per_unit_id": e2eCurrencyUnitID},
		})
		assert.Empty(t, jobWrittenResults(job), "a bad rate unit must not be written")
		require.Len(t, jobErrors(job), 1, "the rejected row is recorded in errors")
	})

	t.Run("sku of another item type", func(t *testing.T) {
		sku := uniqueName("e2e-bup-mat-xtype")
		createForgeItem(t, itemsPath, forgeItemBody("product", sku))
		job := bulkUpsertMaterialsJob(t, map[string]any{"sku": sku, "category": map[string]any{"id": SeedMaterialCategoryID}})
		assert.Empty(t, jobWrittenResults(job), "a conflicting SKU must not be written")
		require.Len(t, jobErrors(job), 1, "the conflicting row is recorded in errors")
	})
}

// A malformed batch is refused whole, with the preview.9 error naming the offending row.
func TestMaterials_BulkUpsert_RejectedAtAccept(t *testing.T) {
	t.Parallel()

	dupSKU := uniqueName("e2e-bup-mat-dup")
	cases := []struct {
		name  string
		rows  []map[string]any
		param string
	}{
		{"no rows", nil, "materials"},
		{"duplicate sku", []map[string]any{
			{"sku": dupSKU, "category": map[string]any{"id": SeedMaterialCategoryID}},
			{"sku": dupSKU, "category": map[string]any{"id": SeedMaterialCategoryID}},
		}, "materials[1].sku"},
		{"unknown category", []map[string]any{
			{"sku": uniqueName("e2e-bup-mat-ok"), "category": map[string]any{"id": SeedMaterialCategoryID}},
			{"sku": uniqueName("e2e-bup-mat-badcat"), "category": map[string]any{"id": "ic_does_not_exist_000000000"}},
		}, "materials[1].category"},
		{"product category", []map[string]any{
			{"sku": uniqueName("e2e-bup-mat-wrongcat"), "category": map[string]any{"id": SeedItemCategoryID}},
		}, "materials[0].category"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body := bulkUpsertMaterials(t, tc.rows...)
			requireStatus(t, 400, status, body)
			assertErrorParam(t, requirePreview9Error(t, body, "invalid_format"), tc.param)
		})
	}
}

// --- property / attribute edge cases (shared resolver; pinned here once) ---

// propertyAttributes fetches a property's attribute rows (as raw maps) by property name.
func propertyAttributes(t *testing.T, propertyName string) []map[string]any {
	t.Helper()
	item := listFindByField(t, propertiesPath, url.Values{"q": {propertyName}}, "name", propertyName)
	require.NotNil(t, item, "property %q should exist", propertyName)
	id := DataItemField(item, "id")
	require.NotEmpty(t, id)
	status, body, err := apiClient.GetListRaw(propertiesPath+"/"+id, url.Values{"include": {"attributes"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	var out []map[string]any
	for _, raw := range jsonListData(parseJSON(body), "attributes") {
		if obj, ok := raw.(map[string]any); ok {
			out = append(out, obj)
		}
	}
	return out
}

// materialItemAttributeValues reads the attribute values on a material's item.
func materialItemAttributeValues(t *testing.T, materialID string) []string {
	t.Helper()
	item := jsonObject(getMaterialP9(t, materialID, "item"), "item")
	require.NotNil(t, item)
	return catalogItemAttributeValues(t, jsonField(item, "id"))
}

// Attribute values are unique within their property, so one value under two properties resolves to two attributes,
// one per property.
func TestMaterials_BulkUpsert_SameAttributeValueUnderTwoProperties(t *testing.T) {
	t.Parallel()

	value := uniqueName("e2e-bup-mat-dupval")
	propA := uniqueName("e2e-bup-mat-dv-a")
	propB := uniqueName("e2e-bup-mat-dv-b")
	created, _ := bulkUpsertMaterialIDs(t, map[string]any{
		"sku":        uniqueName("e2e-bup-mat-dupattr"),
		"category":   map[string]any{"id": SeedMaterialCategoryID},
		"properties": []any{bulkProperty(propA, value), bulkProperty(propB, value)},
	})
	defer cleanupMaterialIDs(created)
	require.Len(t, created, 1)

	for _, prop := range []string{propA, propB} {
		attrs := propertyAttributes(t, prop)
		require.Len(t, attrs, 1, "property %s has its own attribute", prop)
		assert.Equal(t, value, jsonField(attrs[0], "value"))
	}
}

// A second import whose property name and value differ only by casing and surrounding whitespace reuses the existing
// property and attribute — no duplicates.
func TestMaterials_BulkUpsert_ReusesPropertyAndAttributeAcrossCasingAndWhitespace(t *testing.T) {
	t.Parallel()

	propName := uniqueName("e2e-bup-mat-reuse")
	propValue := uniqueName("reuseval")

	first, _ := bulkUpsertMaterialIDs(t, map[string]any{
		"sku":        uniqueName("e2e-bup-mat-reuse-a"),
		"category":   map[string]any{"id": SeedMaterialCategoryID},
		"properties": []any{bulkProperty(propName, propValue)},
	})
	defer cleanupMaterialIDs(first)

	second, _ := bulkUpsertMaterialIDs(t, map[string]any{
		"sku":        uniqueName("e2e-bup-mat-reuse-b"),
		"category":   map[string]any{"id": SeedMaterialCategoryID},
		"properties": []any{bulkProperty("  "+strings.ToUpper(propName)+" ", " "+strings.ToUpper(propValue)+"  ")},
	})
	defer cleanupMaterialIDs(second)

	// The second item carries the ORIGINAL attribute (original casing), proving reuse.
	require.Len(t, second, 1)
	assert.Contains(t, materialItemAttributeValues(t, second[0]), propValue)

	// Exactly one property with this name, keeping the first-seen casing.
	list, _, err := apiClient.GetList(propertiesPath, url.Values{"q": {propName}})
	require.NoError(t, err)
	matches := 0
	for _, li := range list.Data {
		if strings.EqualFold(DataItemField(li, "name"), propName) {
			matches++
			assert.Equal(t, propName, DataItemField(li, "name"))
		}
	}
	assert.Equal(t, 1, matches)

	// Exactly one attribute under it, with the default color and 1-based order.
	attrs := propertyAttributes(t, propName)
	require.Len(t, attrs, 1)
	assert.Equal(t, propValue, jsonField(attrs[0], "value"))
	assert.Equal(t, "default", jsonField(attrs[0], "color"))
	order, ok := attrs[0]["sort_order"].(float64)
	require.True(t, ok, "sort_order should be numeric")
	assert.GreaterOrEqual(t, order, float64(1))
}

// The same (property, value) pair on multiple rows in one batch resolves to a single attribute attached to every row,
// and re-importing the batch creates nothing new.
func TestMaterials_BulkUpsert_SharedValueAcrossRowsAndReimport(t *testing.T) {
	t.Parallel()

	propName := uniqueName("e2e-bup-mat-shared")
	propValue := uniqueName("sharedval")
	rows := []map[string]any{
		{"sku": uniqueName("e2e-bup-mat-shared-a"), "category": map[string]any{"id": SeedMaterialCategoryID}, "properties": []any{bulkProperty(propName, propValue)}},
		{"sku": uniqueName("e2e-bup-mat-shared-b"), "category": map[string]any{"id": SeedMaterialCategoryID}, "properties": []any{bulkProperty(propName, propValue)}},
	}

	created, _ := bulkUpsertMaterialIDs(t, rows...)
	defer cleanupMaterialIDs(created)
	require.Len(t, created, 2)

	for _, materialID := range created {
		assert.Contains(t, materialItemAttributeValues(t, materialID), propValue)
	}
	require.Len(t, propertyAttributes(t, propName), 1)

	reCreated, reUpdated := bulkUpsertMaterialIDs(t, rows...)
	assert.Empty(t, reCreated)
	assert.Len(t, reUpdated, 2)
	assert.Len(t, propertyAttributes(t, propName), 1)
}

// Property pairs whose name or value is blank after trimming are skipped rather than creating junk rows.
func TestMaterials_BulkUpsert_SkipsBlankProperties(t *testing.T) {
	t.Parallel()

	created, _ := bulkUpsertMaterialIDs(t, map[string]any{
		"sku":      uniqueName("e2e-bup-mat-blankprop"),
		"category": map[string]any{"id": SeedMaterialCategoryID},
		"properties": []any{
			bulkProperty("   ", "some-value"),
			bulkProperty(uniqueName("e2e-bup-mat-blankval"), "   "),
		},
	})
	defer cleanupMaterialIDs(created)
	require.Len(t, created, 1)
	assert.Empty(t, materialItemAttributeValues(t, created[0]))
}
