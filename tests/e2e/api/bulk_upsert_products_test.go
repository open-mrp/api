//go:build e2e

package api_test

import (
	"testing"

	"github.com/open-mrp/api/shared/id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The products bulk upsert answers only preview.9 callers; forge upserts items (bulk_upsert_items_test.go).

const productsBulkUpsertPath = productsPath + "/actions/bulk-upsert"

func bulkUpsertProducts(t *testing.T, products ...map[string]any) (int, []byte) {
	t.Helper()
	rows := make([]any, len(products))
	for i, p := range products {
		rows[i] = p
	}
	status, body, err := apiClient.WithAPIVersion(preview9Version).Post(productsBulkUpsertPath, map[string]any{"products": rows}, newIdempotencyKey())
	require.NoError(t, err)
	return status, body
}

func cleanupProductIDs(ids []string) {
	preview := apiClient.WithAPIVersion(preview9Version)
	for _, productID := range ids {
		if productID != "" {
			preview.Delete(productsPath + "/" + productID)
		}
	}
}

// posts a bulk upsert, requires the 202 job acknowledgment, and returns the completed job
func bulkUpsertProductsJob(t *testing.T, products ...map[string]any) map[string]any {
	t.Helper()
	status, body := bulkUpsertProducts(t, products...)
	requireStatus(t, 202, status, body)

	m := parseJSON(body)
	assert.Equal(t, "job", jsonField(m, "object"), "202 returns the job, under its preview.9 object name")
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

// Creates with every field, then updates the same SKU by every field alongside a new row. Notes are accepted but not
// stored.
func TestProductsPreview9_BulkUpsert_CreatesAndUpdates(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-bup-p9-prod")
	createValue := uniqueName("green")
	created, updated := bulkUpsertProductIDs(t,
		map[string]any{
			"sku":               sku,
			"type":              "sale",
			"category":          map[string]any{"id": SeedItemCategoryID},
			"product_line":      map[string]any{"id": SeedProductLineID},
			"portal_visibility": "visible",
			"description":       "full product description",
			"notes":             "full product notes",
			"unit_price":        bulkCurrencyRate("9.99"),
			"unit_cost":         bulkCurrencyRate("4.50"),
			"properties":        []any{bulkProperty(uniqueName("e2e-bup-p9-prop"), createValue)},
		},
		// The type defaults to sale.
		map[string]any{"sku": uniqueName("e2e-bup-p9-prod-min"), "category": map[string]any{"id": SeedItemCategoryID}},
	)
	defer cleanupProductIDs(created)
	require.Len(t, created, 2)
	assert.Empty(t, updated)
	for _, createdID := range created {
		assertIDFormat(t, createdID, id.ProductIDPrefix)
	}
	productID := created[0]

	got := getPreview9Product(t, productID, "product_line", "item")
	assert.Equal(t, "visible", jsonField(got, "portal_visibility"))
	assert.Equal(t, SeedProductLineID, jsonField(jsonObject(got, "product_line"), "id"))
	item := jsonObject(got, "item")
	assertPreview9ItemShape(t, item)
	assert.Equal(t, "full product description", jsonField(item, "description"))
	assertProductsPreview9Price(t, jsonObject(item, "unit_value"), "9.99", nonCurrencyUnitID, "item.unit_value")
	assertProductsPreview9Price(t, jsonObject(item, "unit_cost"), "4.50", nonCurrencyUnitID, "item.unit_cost")
	assert.Contains(t, catalogItemAttributeValues(t, jsonField(item, "id")), createValue)
	assert.Equal(t, "sale", jsonField(getPreview9Product(t, created[1]), "type"))

	updateValue := uniqueName("yellow")
	newSKU := uniqueName("e2e-bup-p9-prod-new")
	created2, updated2 := bulkUpsertProductIDs(t,
		map[string]any{
			"sku":               sku,
			"category":          map[string]any{"id": SeedItemCategoryID},
			"portal_visibility": "hidden",
			"description":       "updated description",
			"notes":             "updated notes",
			"unit_price":        bulkCurrencyRate("12.34"),
			"unit_cost":         bulkCurrencyRate("6.66"),
			"properties":        []any{bulkProperty(uniqueName("e2e-bup-p9-prop-upd"), updateValue)},
		},
		map[string]any{"sku": newSKU, "category": map[string]any{"id": SeedItemCategoryID}},
	)
	defer cleanupProductIDs(created2)
	require.Len(t, created2, 1, "the new SKU creates")
	require.Len(t, updated2, 1, "the existing SKU updates")
	assert.Equal(t, productID, updated2[0])

	got = getPreview9Product(t, productID, "item")
	assert.Equal(t, "hidden", jsonField(got, "portal_visibility"))
	item = jsonObject(got, "item")
	assert.Equal(t, "updated description", jsonField(item, "description"))
	assertNilField(t, item, "notes")
	assertProductsPreview9Price(t, jsonObject(item, "unit_value"), "12.34", nonCurrencyUnitID, "item.unit_value")
	assertProductsPreview9Price(t, jsonObject(item, "unit_cost"), "6.66", nonCurrencyUnitID, "item.unit_cost")
	assert.Contains(t, catalogItemAttributeValues(t, jsonField(item, "id")), updateValue)
}

// category and product_line are fuzzy references: they resolve by name, case-insensitively, as well as by id.
func TestProductsPreview9_BulkUpsert_ResolvesRefsByName(t *testing.T) {
	t.Parallel()

	created, _ := bulkUpsertProductIDs(t, map[string]any{
		"sku":          uniqueName("e2e-bup-p9-prod-refname"),
		"category":     map[string]any{"name": "socks"},
		"product_line": map[string]any{"name": "socks"},
	})
	defer cleanupProductIDs(created)
	require.Len(t, created, 1)

	got := getPreview9Product(t, created[0], "product_line", "item.category")
	assert.Equal(t, SeedProductLineID, jsonField(jsonObject(got, "product_line"), "id"))
	assert.Equal(t, SeedItemCategoryID, jsonField(jsonObject(jsonObject(got, "item"), "category"), "id"))
}

// Rules the write applies fail the row, not the request: the job completes with the row among its errors.
func TestProductsPreview9_BulkUpsert_RowFailures(t *testing.T) {
	t.Parallel()

	t.Run("currency per unit", func(t *testing.T) {
		job := bulkUpsertProductsJob(t, map[string]any{
			"sku":        uniqueName("e2e-bup-p9-prod-badrate"),
			"category":   map[string]any{"id": SeedItemCategoryID},
			"unit_price": map[string]any{"amount": "1.00", "currency": "usd", "per_unit_id": currencyUnitID},
		})
		assert.Empty(t, jobWrittenResults(job), "a bad rate unit must not be written")
		require.Len(t, jobErrors(job), 1, "the rejected row is recorded in errors")
	})

	t.Run("sku of another item type", func(t *testing.T) {
		part := createForgeItem(t, itemsPath, forgeItemBody("part", uniqueName("e2e-bup-p9-prod-xtype")))
		job := bulkUpsertProductsJob(t, map[string]any{"sku": jsonField(part, "sku"), "type": "sale", "category": map[string]any{"id": SeedItemCategoryID}})
		assert.Empty(t, jobWrittenResults(job), "a conflicting SKU must not be written")
		require.Len(t, jobErrors(job), 1, "the conflicting row is recorded in errors")
		assert.Equal(t, "part", jsonField(getItem(t, jsonField(part, "id")), "type"))
	})
}

// Rows that can't resolve are refused at accept with a preview.9 400 naming the row, and nothing is written.
func TestProductsPreview9_BulkUpsert_RejectedAtAccept(t *testing.T) {
	t.Parallel()
	good := func() map[string]any {
		return map[string]any{"sku": uniqueName("e2e-bup-p9-prod-ok"), "category": map[string]any{"id": SeedItemCategoryID}}
	}
	dupSKU := uniqueName("e2e-bup-p9-prod-dup")

	cases := []struct {
		name  string
		rows  []map[string]any
		param string
	}{
		{"duplicate sku", []map[string]any{
			{"sku": dupSKU, "category": map[string]any{"id": SeedItemCategoryID}},
			{"sku": dupSKU, "category": map[string]any{"id": SeedItemCategoryID}},
		}, "products[1].sku"},
		{"unknown category", []map[string]any{good(), {"sku": uniqueName("e2e-bup-p9-prod-badcat"), "category": map[string]any{"id": "ic_does_not_exist_000000000"}}}, "products[1].category"},
		{"material category", []map[string]any{{"sku": uniqueName("e2e-bup-p9-prod-wrongcat"), "category": map[string]any{"id": SeedMaterialCategoryID}}}, "products[0].category"},
		{"unknown product line", []map[string]any{good(), {
			"sku": uniqueName("e2e-bup-p9-prod-plbad"), "category": map[string]any{"id": SeedItemCategoryID},
			"product_line": map[string]any{"id": "pdln_does_not_exist_00000"},
		}}, "products[1].product_line"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body := bulkUpsertProducts(t, tc.rows...)
			requireStatus(t, 400, status, body)
			assertErrorParam(t, requirePreview9Error(t, body, "invalid_format"), tc.param)
			assert.Empty(t, listItemIDs(t, map[string][]string{"skus": {jsonField(tc.rows[0], "sku")}}), "a refused batch writes nothing")
		})
	}

	t.Run("no rows", func(t *testing.T) {
		status, body := bulkUpsertProducts(t)
		requireStatus(t, 400, status, body)
		assertErrorParam(t, requirePreview9Error(t, body, "invalid_format"), "products")
	})
}
