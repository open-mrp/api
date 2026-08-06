//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/augno/api/shared/id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const materialsBulkUpsertPath = materialsPath + "/actions/bulk-upsert"

func bulkUpsertMaterials(t *testing.T, materials ...map[string]any) (int, []byte) {
	t.Helper()
	rows := make([]any, len(materials))
	for i, m := range materials {
		rows[i] = m
	}
	status, body, err := apiClient.Post(materialsBulkUpsertPath, map[string]any{"materials": rows}, newIdempotencyKey())
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
			apiClient.Delete(materialsPath + "/" + materialID)
		}
	}
}

// posts a bulk upsert, requires the 202 job acknowledgment, and returns the completed job
func bulkUpsertMaterialsJob(t *testing.T, materials ...map[string]any) map[string]any {
	t.Helper()
	status, body := bulkUpsertMaterials(t, materials...)
	requireStatus(t, 202, status, body)

	m := parseJSON(body)
	assert.Equal(t, "job", jsonField(m, "object"), "202 returns the canonical job resource")
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
	require.NotEmpty(t, jsonArray(job, "results"), "a completed job must carry results")
	return jobResultIDs(job)
}

// --- create paths ---

func TestMaterials_BulkUpsert_AllCreates(t *testing.T) {
	t.Parallel()

	created, updated := bulkUpsertMaterialIDs(t,
		map[string]any{"sku": uniqueName("e2e-bup-mat-a"), "category": map[string]any{"id": SeedMaterialCategoryID}},
		map[string]any{"sku": uniqueName("e2e-bup-mat-b"), "category": map[string]any{"id": SeedMaterialCategoryID}},
	)
	defer cleanupMaterialIDs(created)

	require.Len(t, created, 2)
	for _, createdID := range created {
		assertIDFormat(t, createdID, id.MaterialIDPrefix)
	}
	assert.Empty(t, updated)
}

// TestMaterials_BulkUpsert_CreateWithAllFields exercises the full create branch:
// description, notes, order_point, lead_time, unit_price, unit_cost, and properties.
func TestMaterials_BulkUpsert_CreateWithAllFields(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-bup-mat-full")
	propName := uniqueName("e2e-bup-mat-prop")
	// Attribute values are unique per account, so use a per-run value.
	propValue := uniqueName("a36")
	created, _ := bulkUpsertMaterialIDs(t, map[string]any{
		"sku":         sku,
		"category":    map[string]any{"id": SeedMaterialCategoryID},
		"description": "full material description",
		"notes":       "full material notes",
		"order_point": bulkQuantity("10"),
		"lead_time":   bulkQuantity("5"),
		"unit_price":  bulkCurrencyRate("1.50"),
		"unit_cost":   bulkCurrencyRate("0.75"),
		"properties":  []any{bulkProperty(propName, propValue)},
	})
	defer cleanupMaterialIDs(created)
	require.Len(t, created, 1)
	id := created[0]

	// Top-level material quantities.
	getStatus, getBody, err := apiClient.GetListRaw(materialsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assert.Equal(t, "10", jsonField(jsonObject(got, "order_point"), "value"))
	assert.Equal(t, "5", jsonField(jsonObject(got, "lead_time"), "value"))

	// Item fields + rates + attributes.
	item := catalogItem(t, materialsPath, id)
	assert.Equal(t, "full material description", jsonField(item, "description"))
	assert.Equal(t, "full material notes", jsonField(item, "notes"))
	assert.Equal(t, "1.50", catalogRateValue(t, materialsPath, id, "item.unit_value"))
	assert.Equal(t, "0.75", catalogRateValue(t, materialsPath, id, "item.unit_cost"))
	assert.Contains(t, catalogItemAttributeValues(t, jsonField(item, "id")), propValue)
}

// checks the rate-unit rule, which the write applies: the job completes and the row lands
// in `errors` rather than failing the request
func TestMaterials_BulkUpsert_RejectsInvalidRateUnit(t *testing.T) {
	t.Parallel()

	job := bulkUpsertMaterialsJob(t, map[string]any{
		"sku":      uniqueName("e2e-bup-mat-badrate"),
		"category": map[string]any{"id": SeedMaterialCategoryID},
		"unit_price": map[string]any{
			"value":               "1.00",
			"numerator_unit_id":   currencyUnitID,
			"denominator_unit_id": currencyUnitID, // must not be currency
		},
	})
	assert.Empty(t, jobResults(job), "a bad rate unit must not be written")
	require.Len(t, jsonArray(job, "errors"), 1, "the rejected row is recorded in errors")
}

func TestMaterials_BulkUpsert_RejectsDuplicateSKUInRequest(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-bup-mat-dup")
	status, body := bulkUpsertMaterials(t,
		map[string]any{"sku": sku, "category": map[string]any{"id": SeedMaterialCategoryID}},
		map[string]any{"sku": sku, "category": map[string]any{"id": SeedMaterialCategoryID}},
	)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "materials[1].sku")
	assert.Contains(t, errObj["message"], "duplicate SKU")
}

// TestMaterials_BulkUpsert_RejectsCrossTypeSKUConflict: a SKU already used by a product
// fails the whole material batch.
func TestMaterials_BulkUpsert_RejectsCrossTypeSKUConflict(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-bup-mat-xtype")

	resp, err := apiClient.PostFull(productsPath, map[string]any{
		"sku": sku, "type": "sale", "category_id": SeedItemCategoryID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	productID := jsonField(parseJSON(resp.Body), "id")
	require.NotEmpty(t, productID)
	t.Cleanup(func() { apiClient.Delete(productsPath + "/" + productID) })

	job := bulkUpsertMaterialsJob(t, map[string]any{"sku": sku, "category": map[string]any{"id": SeedMaterialCategoryID}})
	assert.Empty(t, jobResults(job), "a conflicting SKU must not be written")
	require.Len(t, jsonArray(job, "errors"), 1, "the conflicting row is recorded in errors")
}

func TestMaterials_BulkUpsert_EmptyRejected(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(materialsBulkUpsertPath, map[string]any{"materials": []any{}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

// TestMaterials_BulkUpsert_ResolvesCategoryByName: the category is a fuzzy reference —
// it resolves by name, not only by id.
func TestMaterials_BulkUpsert_ResolvesCategoryByName(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-bup-mat-catname")
	created, _ := bulkUpsertMaterialIDs(t, map[string]any{
		"sku":      sku,
		"category": map[string]any{"name": "yarn"}, // by name, case-insensitive
	})
	defer cleanupMaterialIDs(created)
	require.Len(t, created, 1)

	getStatus, getBody, err := apiClient.GetListRaw(materialsPath+"/"+created[0], url.Values{"include": {"item.category"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	item := jsonObject(parseJSON(getBody), "item")
	require.NotNil(t, item)
	cat := jsonObject(item, "category")
	require.NotNil(t, cat, "category should be populated with ?include=item.category")
	assert.Equal(t, SeedMaterialCategoryID, jsonField(cat, "id"))
}

// TestMaterials_BulkUpsert_RejectsUnknownCategory: a create row whose category does not
// resolve (missing, or — same path — a real category whose unit group has no base unit)
// fails as a clear validation error naming the offending row, not an opaque 404.
func TestMaterials_BulkUpsert_RejectsUnknownCategory(t *testing.T) {
	t.Parallel()

	status, body := bulkUpsertMaterials(t,
		map[string]any{"sku": uniqueName("e2e-bup-mat-ok"), "category": map[string]any{"id": SeedMaterialCategoryID}},
		map[string]any{"sku": uniqueName("e2e-bup-mat-badcat"), "category": map[string]any{"id": "ic_does_not_exist_000000000"}},
	)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "materials[1].category")
}

// TestMaterials_BulkUpsert_RejectsWrongCategoryType: materials may only be created in
// material categories; a product category fails as a row-indexed validation error.
func TestMaterials_BulkUpsert_RejectsWrongCategoryType(t *testing.T) {
	t.Parallel()

	status, body := bulkUpsertMaterials(t,
		map[string]any{"sku": uniqueName("e2e-bup-mat-wrongcat"), "category": map[string]any{"id": SeedItemCategoryID}},
	)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "materials[0].category")
}

// TestMaterials_BulkUpsert_RejectsDuplicateAttributeValueAcrossProperties: attribute
// values are unique per account (mirroring the single attribute-create endpoint), so a
// value used under two different properties — within one request or against an
// existing attribute — fails as a validation error and creates nothing.
func TestMaterials_BulkUpsert_RejectsDuplicateAttributeValueAcrossProperties(t *testing.T) {
	t.Parallel()

	// Same value under two properties within one request. Attribute resolution runs before
	// the row loop, so the conflict sinks the whole job rather than one row.
	dupValue := uniqueName("e2e-bup-mat-dupval")
	status, body := bulkUpsertMaterials(t, map[string]any{
		"sku":      uniqueName("e2e-bup-mat-dupattr"),
		"category": map[string]any{"id": SeedMaterialCategoryID},
		"properties": []any{
			bulkProperty(uniqueName("e2e-bup-mat-dv-a"), dupValue),
			bulkProperty(uniqueName("e2e-bup-mat-dv-b"), dupValue),
		},
	})
	requireStatus(t, 202, status, body)
	job := pollJobUntilTerminal(t, jsonField(parseJSON(body), "id"))
	assert.Equal(t, "failed", jsonField(job, "status"))
	assert.Empty(t, jobResults(job))

	// Value already existing under a different property from a prior import.
	existingValue := uniqueName("e2e-bup-mat-exval")
	seeded, _ := bulkUpsertMaterialIDs(t, map[string]any{
		"sku":        uniqueName("e2e-bup-mat-dupattr-seed"),
		"category":   map[string]any{"id": SeedMaterialCategoryID},
		"properties": []any{bulkProperty(uniqueName("e2e-bup-mat-dv-c"), existingValue)},
	})
	defer cleanupMaterialIDs(seeded)

	conflictStatus, conflictBody := bulkUpsertMaterials(t, map[string]any{
		"sku":        uniqueName("e2e-bup-mat-dupattr2"),
		"category":   map[string]any{"id": SeedMaterialCategoryID},
		"properties": []any{bulkProperty(uniqueName("e2e-bup-mat-dv-d"), existingValue)},
	})
	requireStatus(t, 202, conflictStatus, conflictBody)
	conflictJob := pollJobUntilTerminal(t, jsonField(parseJSON(conflictBody), "id"))
	assert.Equal(t, "failed", jsonField(conflictJob, "status"))
	assert.Empty(t, jobResults(conflictJob))
}

// --- update paths: every updatable portion ---

// TestMaterials_BulkUpsert_UpdatesEveryField creates a material, then upserts the same
// SKU changing every field the update path supports: description, notes, order_point,
// lead_time, unit_price, unit_cost, and properties.
func TestMaterials_BulkUpsert_UpdatesEveryField(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-bup-mat-upd")

	createdIDs, _ := bulkUpsertMaterialIDs(t, map[string]any{
		"sku":         sku,
		"category":    map[string]any{"id": SeedMaterialCategoryID},
		"description": "original description",
		"notes":       "original notes",
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
	created, updated := bulkUpsertMaterialIDs(t, map[string]any{
		"sku":         sku,
		"category":    map[string]any{"id": SeedMaterialCategoryID},
		"description": "updated description",
		"notes":       "updated notes",
		"order_point": bulkQuantity("20"),
		"lead_time":   bulkQuantity("8"),
		"unit_price":  bulkCurrencyRate("2.22"),
		"unit_cost":   bulkCurrencyRate("1.11"),
		"properties":  []any{bulkProperty(propName, propValue)},
	})

	assert.Empty(t, created, "existing SKU must update, not create")
	require.Len(t, updated, 1)
	assert.Equal(t, materialID, updated[0])

	// Verify every field changed.
	getStatus, getBody, err := apiClient.GetListRaw(materialsPath+"/"+materialID, url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assert.Equal(t, "20", jsonField(jsonObject(got, "order_point"), "value"))
	assert.Equal(t, "8", jsonField(jsonObject(got, "lead_time"), "value"))

	item := catalogItem(t, materialsPath, materialID)
	assert.Equal(t, "updated description", jsonField(item, "description"))
	assert.Equal(t, "updated notes", jsonField(item, "notes"))
	assert.Equal(t, "2.22", catalogRateValue(t, materialsPath, materialID, "item.unit_value"))
	assert.Equal(t, "1.11", catalogRateValue(t, materialsPath, materialID, "item.unit_cost"))
	assert.Contains(t, catalogItemAttributeValues(t, jsonField(item, "id")), propValue)
}

func TestMaterials_BulkUpsert_MixedCreateAndUpdate(t *testing.T) {
	t.Parallel()

	existingSKU := uniqueName("e2e-bup-mat-mix-exist")
	newSKU := uniqueName("e2e-bup-mat-mix-new")

	seeded, _ := bulkUpsertMaterialIDs(t, map[string]any{"sku": existingSKU, "category": map[string]any{"id": SeedMaterialCategoryID}})
	defer cleanupMaterialIDs(seeded)

	created, updated := bulkUpsertMaterialIDs(t,
		map[string]any{"sku": existingSKU, "category": map[string]any{"id": SeedMaterialCategoryID}, "notes": "touched"},
		map[string]any{"sku": newSKU, "category": map[string]any{"id": SeedMaterialCategoryID}},
	)
	defer cleanupMaterialIDs(created)

	assert.Len(t, created, 1)
	assert.Len(t, updated, 1)
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

// TestMaterials_BulkUpsert_ReusesPropertyAndAttributeAcrossCasingAndWhitespace: a
// second import whose property name and value differ only by casing and surrounding
// whitespace must reuse the existing property and attribute — no duplicates.
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
	item := catalogItem(t, materialsPath, second[0])
	assert.Contains(t, catalogItemAttributeValues(t, jsonField(item, "id")), propValue)

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

	// Exactly one attribute under it, with a valid named color and 1-based order.
	attrs := propertyAttributes(t, propName)
	require.Len(t, attrs, 1)
	assert.Equal(t, propValue, jsonField(attrs[0], "value"))
	color := jsonField(attrs[0], "color")
	assert.NotEmpty(t, color)
	assert.NotEqual(t, "default", color)
	order, ok := attrs[0]["sort_order"].(float64)
	require.True(t, ok, "sort_order should be numeric")
	assert.GreaterOrEqual(t, order, float64(1))
}

// TestMaterials_BulkUpsert_SharedValueAcrossRowsAndReimport: the same (property, value)
// pair on multiple rows in one batch resolves to a single attribute attached to every
// row, and re-importing the batch creates nothing new.
func TestMaterials_BulkUpsert_SharedValueAcrossRowsAndReimport(t *testing.T) {
	t.Parallel()

	propName := uniqueName("e2e-bup-mat-shared")
	propValue := uniqueName("sharedval")
	skuA := uniqueName("e2e-bup-mat-shared-a")
	skuB := uniqueName("e2e-bup-mat-shared-b")
	rows := []map[string]any{
		{"sku": skuA, "category": map[string]any{"id": SeedMaterialCategoryID}, "properties": []any{bulkProperty(propName, propValue)}},
		{"sku": skuB, "category": map[string]any{"id": SeedMaterialCategoryID}, "properties": []any{bulkProperty(propName, propValue)}},
	}

	created, _ := bulkUpsertMaterialIDs(t, rows...)
	defer cleanupMaterialIDs(created)
	require.Len(t, created, 2)

	// Both items carry the value; the property holds exactly one attribute.
	for _, materialID := range created {
		item := catalogItem(t, materialsPath, materialID)
		assert.Contains(t, catalogItemAttributeValues(t, jsonField(item, "id")), propValue)
	}
	require.Len(t, propertyAttributes(t, propName), 1)

	// Re-import the identical batch: pure updates, still exactly one attribute.
	reCreated, reUpdated := bulkUpsertMaterialIDs(t, rows...)
	assert.Empty(t, reCreated)
	assert.Len(t, reUpdated, 2)
	assert.Len(t, propertyAttributes(t, propName), 1)
}

// TestMaterials_BulkUpsert_SkipsBlankProperties: property pairs whose name or value is
// blank after trimming are skipped rather than creating junk rows — the current
// contract for spreadsheet imports with stray whitespace cells.
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
	item := catalogItem(t, materialsPath, created[0])
	assert.Empty(t, catalogItemAttributeValues(t, jsonField(item, "id")))
}
