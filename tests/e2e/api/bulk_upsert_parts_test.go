//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/open-mrp/api/shared/id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const partsBulkUpsertPath = partsPath + "/actions/bulk-upsert"

func bulkUpsertParts(t *testing.T, parts ...map[string]any) (int, []byte) {
	t.Helper()
	rows := make([]any, len(parts))
	for i, p := range parts {
		rows[i] = p
	}
	status, body, err := apiClient.Post(partsBulkUpsertPath, map[string]any{"parts": rows}, newIdempotencyKey())
	require.NoError(t, err)
	return status, body
}

func cleanupPartIDs(ids []string) {
	for _, partID := range ids {
		if partID != "" {
			apiClient.Delete(partsPath + "/" + partID)
		}
	}
}

// posts a bulk upsert, requires the 202 job acknowledgment, and returns the completed job
func bulkUpsertPartsJob(t *testing.T, parts ...map[string]any) map[string]any {
	t.Helper()
	status, body := bulkUpsertParts(t, parts...)
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
func bulkUpsertPartIDs(t *testing.T, parts ...map[string]any) (createdIDs, updatedIDs []string) {
	t.Helper()
	job := bulkUpsertPartsJob(t, parts...)
	require.NotEmpty(t, jobResults(job), "a completed job must carry results")
	return jobResultIDs(job)
}

// --- create paths ---

func TestParts_BulkUpsert_AllCreates(t *testing.T) {
	t.Parallel()

	created, updated := bulkUpsertPartIDs(t,
		map[string]any{"sku": uniqueName("e2e-bup-part-a"), "category": map[string]any{"id": SeedItemCategoryID}},
		map[string]any{"sku": uniqueName("e2e-bup-part-b"), "category": map[string]any{"id": SeedItemCategoryID}},
	)
	defer cleanupPartIDs(created)

	require.Len(t, created, 2)
	for _, createdID := range created {
		assertIDFormat(t, createdID, id.PartIDPrefix)
	}
	assert.Empty(t, updated)
}

// TestParts_BulkUpsert_CreateWithAllFields exercises the full create branch:
// description, notes, unit_price, unit_cost, and find-or-create properties.
func TestParts_BulkUpsert_CreateWithAllFields(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-bup-part-full")
	propName := uniqueName("e2e-bup-prop")
	// Attribute values are unique per account, so use a per-run value.
	propValue := uniqueName("red")
	created, _ := bulkUpsertPartIDs(t, map[string]any{
		"sku":         sku,
		"category":    map[string]any{"id": SeedItemCategoryID},
		"description": "full part description",
		"notes":       "full part notes",
		"unit_price":  bulkCurrencyRate("5.00"),
		"unit_cost":   bulkCurrencyRate("2.50"),
		"properties":  []any{bulkProperty(propName, propValue)},
	})
	defer cleanupPartIDs(created)
	require.Len(t, created, 1)
	id := created[0]

	item := catalogItem(t, partsPath, id)
	assert.Equal(t, "full part description", jsonField(item, "description"))
	assert.Equal(t, "full part notes", jsonField(item, "notes"))
	assert.Equal(t, "5.00", catalogRateValue(t, partsPath, id, "item.unit_value"))
	assert.Equal(t, "2.50", catalogRateValue(t, partsPath, id, "item.unit_cost"))
	assert.Contains(t, catalogItemAttributeValues(t, jsonField(item, "id")), propValue)
}

// checks the rate-unit rule, which the write applies: the job completes and the row lands
// in `errors` rather than failing the request
func TestParts_BulkUpsert_RejectsInvalidRateUnit(t *testing.T) {
	t.Parallel()

	job := bulkUpsertPartsJob(t, map[string]any{
		"sku":      uniqueName("e2e-bup-part-badrate"),
		"category": map[string]any{"id": SeedItemCategoryID},
		"unit_cost": map[string]any{
			"value":               "1.00",
			"numerator_unit_id":   nonCurrencyUnitID, // must be currency
			"denominator_unit_id": nonCurrencyUnitID,
		},
	})
	assert.Empty(t, jobWrittenResults(job), "a bad rate unit must not be written")
	rowErrs := jobErrors(job)
	require.Len(t, rowErrs, 1, "the rejected row is recorded in errors")
}

func TestParts_BulkUpsert_RejectsDuplicateSKUInRequest(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-bup-part-dup")
	status, body := bulkUpsertParts(t,
		map[string]any{"sku": sku, "category": map[string]any{"id": SeedItemCategoryID}},
		map[string]any{"sku": sku, "category": map[string]any{"id": SeedItemCategoryID}},
	)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "parts[1].sku")
	assert.Contains(t, errObj["message"], "duplicate SKU")
}

// checks that a SKU already used by a product is rejected; the conflict is found by the
// write, so it lands on the job as a row error
func TestParts_BulkUpsert_RejectsCrossTypeSKUConflict(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-bup-part-xtype")

	// Seed a product with the SKU.
	resp, err := apiClient.PostFull(productsPath, map[string]any{
		"sku": sku, "type": "sale", "category_id": SeedItemCategoryID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	productID := jsonField(parseJSON(resp.Body), "id")
	require.NotEmpty(t, productID)
	t.Cleanup(func() { apiClient.Delete(productsPath + "/" + productID) })

	job := bulkUpsertPartsJob(t, map[string]any{"sku": sku, "category": map[string]any{"id": SeedItemCategoryID}})
	assert.Empty(t, jobWrittenResults(job), "a conflicting SKU must not be written")
	require.Len(t, jobErrors(job), 1, "the conflicting row is recorded in errors")
}

func TestParts_BulkUpsert_EmptyRejected(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(partsBulkUpsertPath, map[string]any{"parts": []any{}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

// TestParts_BulkUpsert_RejectsUnknownCategory: a create row whose category does not
// resolve fails as a clear validation error naming the offending row, not an opaque
// 404, and the whole batch is atomic.
func TestParts_BulkUpsert_RejectsUnknownCategory(t *testing.T) {
	t.Parallel()

	status, body := bulkUpsertParts(t,
		map[string]any{"sku": uniqueName("e2e-bup-part-ok"), "category": map[string]any{"id": SeedItemCategoryID}},
		map[string]any{"sku": uniqueName("e2e-bup-part-badcat"), "category": map[string]any{"id": "ic_does_not_exist_000000000"}},
	)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "parts[1].category")
}

// TestParts_BulkUpsert_ResolvesCategoryByName: the category is a fuzzy reference —
// it resolves by name, not only by id.
func TestParts_BulkUpsert_ResolvesCategoryByName(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-bup-part-catname")
	created, _ := bulkUpsertPartIDs(t, map[string]any{
		"sku":      sku,
		"category": map[string]any{"name": "socks"}, // by name, case-insensitive
	})
	defer cleanupPartIDs(created)
	require.Len(t, created, 1)

	getStatus, getBody, err := apiClient.GetListRaw(partsPath+"/"+created[0], url.Values{"include": {"item.category"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	item := jsonObject(parseJSON(getBody), "item")
	require.NotNil(t, item)
	cat := jsonObject(item, "category")
	require.NotNil(t, cat, "category should be populated with ?include=item.category")
	assert.Equal(t, SeedItemCategoryID, jsonField(cat, "id"))
}

// TestParts_BulkUpsert_RejectsWrongCategoryType: parts may only be created in product
// categories; a material category fails as a row-indexed validation error.
func TestParts_BulkUpsert_RejectsWrongCategoryType(t *testing.T) {
	t.Parallel()

	status, body := bulkUpsertParts(t,
		map[string]any{"sku": uniqueName("e2e-bup-part-wrongcat"), "category": map[string]any{"id": SeedMaterialCategoryID}},
	)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "parts[0].category")
}

// --- update paths: every updatable portion ---

// TestParts_BulkUpsert_UpdatesEveryField creates a part, then upserts the same SKU
// changing description, notes, unit_price, unit_cost, and attaching a property —
// asserting each change landed.
func TestParts_BulkUpsert_UpdatesEveryField(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-bup-part-upd")

	// Create the part with initial values.
	createdIDs, _ := bulkUpsertPartIDs(t, map[string]any{
		"sku":         sku,
		"category":    map[string]any{"id": SeedItemCategoryID},
		"description": "original description",
		"notes":       "original notes",
		"unit_price":  bulkCurrencyRate("5.00"),
		"unit_cost":   bulkCurrencyRate("2.50"),
	})
	defer cleanupPartIDs(createdIDs)
	require.Len(t, createdIDs, 1)
	partID := createdIDs[0]

	// Upsert the same SKU with all new values + a property.
	propName := uniqueName("e2e-bup-upd-prop")
	propValue := uniqueName("blue")
	created, updated := bulkUpsertPartIDs(t, map[string]any{
		"sku":         sku,
		"category":    map[string]any{"id": SeedItemCategoryID},
		"description": "updated description",
		"notes":       "updated notes",
		"unit_price":  bulkCurrencyRate("7.77"),
		"unit_cost":   bulkCurrencyRate("3.33"),
		"properties":  []any{bulkProperty(propName, propValue)},
	})

	assert.Empty(t, created, "existing SKU must update, not create")
	require.Len(t, updated, 1)
	assert.Equal(t, partID, updated[0])

	// Verify every field was updated.
	item := catalogItem(t, partsPath, partID)
	assert.Equal(t, "updated description", jsonField(item, "description"))
	assert.Equal(t, "updated notes", jsonField(item, "notes"))
	assert.Equal(t, "7.77", catalogRateValue(t, partsPath, partID, "item.unit_value"))
	assert.Equal(t, "3.33", catalogRateValue(t, partsPath, partID, "item.unit_cost"))
	assert.Contains(t, catalogItemAttributeValues(t, jsonField(item, "id")), propValue)
}

func TestParts_BulkUpsert_MixedCreateAndUpdate(t *testing.T) {
	t.Parallel()

	existingSKU := uniqueName("e2e-bup-part-mix-exist")
	newSKU := uniqueName("e2e-bup-part-mix-new")

	seeded, _ := bulkUpsertPartIDs(t, map[string]any{"sku": existingSKU, "category": map[string]any{"id": SeedItemCategoryID}})
	defer cleanupPartIDs(seeded)

	created, updated := bulkUpsertPartIDs(t,
		map[string]any{"sku": existingSKU, "category": map[string]any{"id": SeedItemCategoryID}, "notes": "touched"},
		map[string]any{"sku": newSKU, "category": map[string]any{"id": SeedItemCategoryID}},
	)
	defer cleanupPartIDs(created)

	assert.Len(t, created, 1)
	assert.Len(t, updated, 1)
}
