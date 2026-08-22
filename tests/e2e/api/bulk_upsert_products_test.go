//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/open-mrp/api/shared/id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const productsBulkUpsertPath = productsPath + "/actions/bulk-upsert"

func bulkUpsertProducts(t *testing.T, products ...map[string]any) (int, []byte) {
	t.Helper()
	rows := make([]any, len(products))
	for i, p := range products {
		rows[i] = p
	}
	status, body, err := apiClient.Post(productsBulkUpsertPath, map[string]any{"products": rows}, newIdempotencyKey())
	require.NoError(t, err)
	return status, body
}

func cleanupProductIDs(ids []string) {
	for _, productID := range ids {
		if productID != "" {
			apiClient.Delete(productsPath + "/" + productID)
		}
	}
}

// posts a bulk upsert, requires the 202 job acknowledgment, and returns the completed job
func bulkUpsertProductsJob(t *testing.T, products ...map[string]any) map[string]any {
	t.Helper()
	status, body := bulkUpsertProducts(t, products...)
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
func bulkUpsertProductIDs(t *testing.T, products ...map[string]any) (createdIDs, updatedIDs []string) {
	t.Helper()
	job := bulkUpsertProductsJob(t, products...)
	require.NotEmpty(t, jobResults(job), "a completed job must carry results")
	return jobResultIDs(job)
}

// --- create paths ---

// TestProducts_BulkUpsert_AllCreates also covers the type default ("sale" when omitted).
func TestProducts_BulkUpsert_AllCreates(t *testing.T) {
	t.Parallel()

	created, updated := bulkUpsertProductIDs(t,
		map[string]any{"sku": uniqueName("e2e-bup-prod-a"), "category": map[string]any{"id": SeedItemCategoryID}},
		map[string]any{"sku": uniqueName("e2e-bup-prod-b"), "category": map[string]any{"id": SeedItemCategoryID}},
	)
	defer cleanupProductIDs(created)

	require.Len(t, created, 2)
	for _, createdID := range created {
		assertIDFormat(t, createdID, id.ProductIDPrefix)
	}
	assert.Empty(t, updated)
}

// TestProducts_BulkUpsert_CreateWithAllFields exercises the full create branch:
// type, product_line, portal visibility, description, notes, rates, and properties.
func TestProducts_BulkUpsert_CreateWithAllFields(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-bup-prod-full")
	propName := uniqueName("e2e-bup-prod-prop")
	// Attribute values are unique per account, so use a per-run value.
	propValue := uniqueName("green")
	created, _ := bulkUpsertProductIDs(t, map[string]any{
		"sku":               sku,
		"type":              "sale",
		"category":          map[string]any{"id": SeedItemCategoryID},
		"product_line":      map[string]any{"id": SeedProductLineID},
		"portal_visibility": "visible",
		"description":       "full product description",
		"notes":             "full product notes",
		"unit_price":        bulkCurrencyRate("9.99"),
		"unit_cost":         bulkCurrencyRate("4.50"),
		"properties":        []any{bulkProperty(propName, propValue)},
	})
	defer cleanupProductIDs(created)
	require.Len(t, created, 1)
	id := created[0]

	// Top-level product fields.
	getStatus, getBody, err := apiClient.GetListRaw(productsPath+"/"+id, url.Values{"include": {"product_line"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assert.Equal(t, "visible", jsonField(got, "portal_visibility"))
	assert.Equal(t, SeedProductLineID, jsonField(jsonObject(got, "product_line"), "id"))

	// Item fields + rates + attributes.
	item := catalogItem(t, productsPath, id)
	assert.Equal(t, "full product description", jsonField(item, "description"))
	assert.Equal(t, "full product notes", jsonField(item, "notes"))
	assert.Equal(t, "9.99", catalogRateValue(t, productsPath, id, "item.unit_value"))
	assert.Equal(t, "4.50", catalogRateValue(t, productsPath, id, "item.unit_cost"))
	assert.Contains(t, catalogItemAttributeValues(t, jsonField(item, "id")), propValue)
}

// checks the rate-unit rule, which the write applies: the job completes and the row lands
// in `errors` rather than failing the request
func TestProducts_BulkUpsert_RejectsInvalidRateUnit(t *testing.T) {
	t.Parallel()

	job := bulkUpsertProductsJob(t, map[string]any{
		"sku":      uniqueName("e2e-bup-prod-badrate"),
		"category": map[string]any{"id": SeedItemCategoryID},
		"unit_price": map[string]any{
			"value":               "1.00",
			"numerator_unit_id":   currencyUnitID,
			"denominator_unit_id": currencyUnitID, // must not be currency
		},
	})
	assert.Empty(t, jobWrittenResults(job), "a bad rate unit must not be written")
	require.Len(t, jobErrors(job), 1, "the rejected row is recorded in errors")
}

func TestProducts_BulkUpsert_RejectsDuplicateSKUInRequest(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-bup-prod-dup")
	status, body := bulkUpsertProducts(t,
		map[string]any{"sku": sku, "category": map[string]any{"id": SeedItemCategoryID}},
		map[string]any{"sku": sku, "category": map[string]any{"id": SeedItemCategoryID}},
	)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "products[1].sku")
	assert.Contains(t, errObj["message"], "duplicate SKU")
}

// TestProducts_BulkUpsert_RejectsCrossTypeSKUConflict: a SKU already used by a part
// fails the whole product batch.
func TestProducts_BulkUpsert_RejectsCrossTypeSKUConflict(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-bup-prod-xtype")

	resp, err := apiClient.PostFull(partsPath, map[string]any{
		"sku": sku, "category_id": SeedItemCategoryID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	partID := jsonField(parseJSON(resp.Body), "id")
	require.NotEmpty(t, partID)
	t.Cleanup(func() { apiClient.Delete(partsPath + "/" + partID) })

	job := bulkUpsertProductsJob(t, map[string]any{"sku": sku, "type": "sale", "category": map[string]any{"id": SeedItemCategoryID}})
	assert.Empty(t, jobWrittenResults(job), "a conflicting SKU must not be written")
	require.Len(t, jobErrors(job), 1, "the conflicting row is recorded in errors")
}

func TestProducts_BulkUpsert_EmptyRejected(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(productsBulkUpsertPath, map[string]any{"products": []any{}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

// TestProducts_BulkUpsert_RejectsUnknownCategory: a create row whose category does not
// resolve fails as a clear validation error naming the offending row, not an opaque
// 404, and the whole batch is atomic.
func TestProducts_BulkUpsert_RejectsUnknownCategory(t *testing.T) {
	t.Parallel()

	status, body := bulkUpsertProducts(t,
		map[string]any{"sku": uniqueName("e2e-bup-prod-ok"), "category": map[string]any{"id": SeedItemCategoryID}},
		map[string]any{"sku": uniqueName("e2e-bup-prod-badcat"), "category": map[string]any{"id": "ic_does_not_exist_000000000"}},
	)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "products[1].category")
}

// TestProducts_BulkUpsert_RejectsWrongCategoryType: products may only be created in
// product categories; a material category fails as a row-indexed validation error.
func TestProducts_BulkUpsert_RejectsWrongCategoryType(t *testing.T) {
	t.Parallel()

	status, body := bulkUpsertProducts(t,
		map[string]any{"sku": uniqueName("e2e-bup-prod-wrongcat"), "category": map[string]any{"id": SeedMaterialCategoryID}},
	)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "products[0].category")
}

// TestProducts_BulkUpsert_ResolvesRefsByName: category and product_line are both fuzzy
// references — they resolve by name, not only by id.
func TestProducts_BulkUpsert_ResolvesRefsByName(t *testing.T) {
	t.Parallel()

	created, _ := bulkUpsertProductIDs(t, map[string]any{
		"sku":          uniqueName("e2e-bup-prod-refname"),
		"category":     map[string]any{"name": "socks"}, // by name, case-insensitive
		"product_line": map[string]any{"name": "socks"},
	})
	defer cleanupProductIDs(created)
	require.Len(t, created, 1)
	id := created[0]

	getStatus, getBody, err := apiClient.GetListRaw(productsPath+"/"+id, url.Values{"include": {"product_line", "item.category"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assert.Equal(t, SeedProductLineID, jsonField(jsonObject(got, "product_line"), "id"))
	item := jsonObject(got, "item")
	require.NotNil(t, item)
	assert.Equal(t, SeedItemCategoryID, jsonField(jsonObject(item, "category"), "id"))
}

// TestProducts_BulkUpsert_RejectsUnknownProductLine: a create row whose product line does
// not resolve fails as a row-indexed validation error rather than a foreign-key failure,
// and the whole batch is atomic.
func TestProducts_BulkUpsert_RejectsUnknownProductLine(t *testing.T) {
	t.Parallel()

	status, body := bulkUpsertProducts(t,
		map[string]any{
			"sku":      uniqueName("e2e-bup-prod-plok"),
			"category": map[string]any{"id": SeedItemCategoryID},
		},
		map[string]any{
			"sku":          uniqueName("e2e-bup-prod-plbad"),
			"category":     map[string]any{"id": SeedItemCategoryID},
			"product_line": map[string]any{"id": "pdln_does_not_exist_00000"},
		},
	)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "products[1].product_line")
}

// --- update paths: every updatable portion ---

// TestProducts_BulkUpsert_UpdatesEveryField creates a product, then upserts the same
// SKU changing every field the update path supports: description, notes,
// portal_visibility, unit_price, unit_cost, and properties.
func TestProducts_BulkUpsert_UpdatesEveryField(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-bup-prod-upd")

	createdIDs, _ := bulkUpsertProductIDs(t, map[string]any{
		"sku":               sku,
		"type":              "sale",
		"category":          map[string]any{"id": SeedItemCategoryID},
		"portal_visibility": "hidden",
		"description":       "original description",
		"notes":             "original notes",
		"unit_price":        bulkCurrencyRate("9.99"),
		"unit_cost":         bulkCurrencyRate("4.50"),
	})
	defer cleanupProductIDs(createdIDs)
	require.Len(t, createdIDs, 1)
	productID := createdIDs[0]

	propName := uniqueName("e2e-bup-prod-upd-prop")
	propValue := uniqueName("yellow")
	created, updated := bulkUpsertProductIDs(t, map[string]any{
		"sku":               sku,
		"category":          map[string]any{"id": SeedItemCategoryID},
		"portal_visibility": "visible",
		"description":       "updated description",
		"notes":             "updated notes",
		"unit_price":        bulkCurrencyRate("12.34"),
		"unit_cost":         bulkCurrencyRate("6.66"),
		"properties":        []any{bulkProperty(propName, propValue)},
	})

	assert.Empty(t, created, "existing SKU must update, not create")
	require.Len(t, updated, 1)
	assert.Equal(t, productID, updated[0])

	// Verify every field changed.
	getStatus, getBody, err := apiClient.GetListRaw(productsPath+"/"+productID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, "visible", jsonField(parseJSON(getBody), "portal_visibility"))

	item := catalogItem(t, productsPath, productID)
	assert.Equal(t, "updated description", jsonField(item, "description"))
	assert.Equal(t, "updated notes", jsonField(item, "notes"))
	assert.Equal(t, "12.34", catalogRateValue(t, productsPath, productID, "item.unit_value"))
	assert.Equal(t, "6.66", catalogRateValue(t, productsPath, productID, "item.unit_cost"))
	assert.Contains(t, catalogItemAttributeValues(t, jsonField(item, "id")), propValue)
}

func TestProducts_BulkUpsert_MixedCreateAndUpdate(t *testing.T) {
	t.Parallel()

	existingSKU := uniqueName("e2e-bup-prod-mix-exist")
	newSKU := uniqueName("e2e-bup-prod-mix-new")

	seeded, _ := bulkUpsertProductIDs(t, map[string]any{"sku": existingSKU, "category": map[string]any{"id": SeedItemCategoryID}})
	defer cleanupProductIDs(seeded)

	created, updated := bulkUpsertProductIDs(t,
		map[string]any{"sku": existingSKU, "category": map[string]any{"id": SeedItemCategoryID}, "notes": "touched"},
		map[string]any{"sku": newSKU, "category": map[string]any{"id": SeedItemCategoryID}},
	)
	defer cleanupProductIDs(created)

	assert.Len(t, created, 1)
	assert.Len(t, updated, 1)
}
