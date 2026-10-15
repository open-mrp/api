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

// The parts bulk upsert answers 1.0.forge-preview.9 callers only; forge upserts parts through the items bulk upsert.

const partsBulkUpsertPath = partsPath + "/actions/bulk-upsert"

func bulkUpsertParts(t *testing.T, parts ...map[string]any) (int, []byte) {
	t.Helper()
	rows := make([]any, len(parts))
	for i, p := range parts {
		rows[i] = p
	}
	status, body, err := apiClient.WithAPIVersion(preview9Version).Post(partsBulkUpsertPath, map[string]any{"parts": rows}, newIdempotencyKey())
	require.NoError(t, err)
	return status, body
}

func cleanupPartIDs(ids []string) {
	for _, partID := range ids {
		if partID != "" {
			apiClient.WithAPIVersion(preview9Version).Delete(partsPath + "/" + partID)
		}
	}
}

// posts a bulk upsert, requires the 202 job acknowledgment, and returns the completed job
func bulkUpsertPartsJob(t *testing.T, parts ...map[string]any) map[string]any {
	t.Helper()
	status, body := bulkUpsertParts(t, parts...)
	requireStatus(t, 202, status, body)

	m := parseJSON(body)
	assert.Equal(t, "job", jsonField(m, "object"), "preview.9 acknowledges with its job object")
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

// partItemPreview9 reads a part's item on preview.9.
func partItemPreview9(t *testing.T, partID string) map[string]any {
	t.Helper()
	status, body := preview9PartGet(t, partsPath+"/"+partID, map[string][]string{"include": {"item,item.category"}})
	requireStatus(t, 200, status, body)
	item := jsonObject(parseJSON(body), "item")
	require.NotNil(t, item, "item is included")
	assertPreview9ItemShape(t, item)
	return item
}

// One upsert creates a part with every field, and a second, matching it by SKU, rewrites them; notes are ignored and
// the category resolves by name.
func TestParts_BulkUpsert_CreatesThenUpdatesEveryField(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-bup-part-full")
	created, updated := bulkUpsertPartIDs(t,
		map[string]any{
			"sku":         sku,
			"category":    map[string]any{"name": "socks"}, // by name, case-insensitive
			"description": "original description",
			"notes":       "ignored",
			"unit_price":  partPreview9Price("5.00", nonCurrencyUnitID),
			"unit_cost":   partPreview9Price("2.50", nonCurrencyUnitID),
			"properties":  []any{bulkProperty(uniqueName("e2e-bup-prop"), uniqueName("red"))},
		},
		map[string]any{"sku": uniqueName("e2e-bup-part-b"), "category": map[string]any{"id": SeedItemCategoryID}},
	)
	defer cleanupPartIDs(created)
	require.Len(t, created, 2)
	assert.Empty(t, updated)
	for _, createdID := range created {
		assertIDFormat(t, createdID, id.PartIDPrefix)
	}
	partID := created[0]

	item := partItemPreview9(t, partID)
	assert.Equal(t, sku, jsonField(item, "sku"))
	assert.Equal(t, "original description", jsonField(item, "description"))
	assertNilField(t, item, "notes")
	assert.Equal(t, SeedItemCategoryID, jsonField(jsonObject(item, "category"), "id"))
	assertPartPreview9Price(t, jsonObject(item, "unit_value"), "5.00", nonCurrencyUnitID, "item.unit_value")
	assertPartPreview9Price(t, jsonObject(item, "unit_cost"), "2.50", nonCurrencyUnitID, "item.unit_cost")

	propValue := uniqueName("blue")
	created2, updated2 := bulkUpsertPartIDs(t, map[string]any{
		"sku":         sku,
		"category":    map[string]any{"id": SeedItemCategoryID},
		"description": "updated description",
		"unit_price":  partPreview9Price("7.77", nonCurrencyUnitID),
		"unit_cost":   partPreview9Price("3.33", nonCurrencyUnitID),
		"properties":  []any{bulkProperty(uniqueName("e2e-bup-upd-prop"), propValue)},
	})
	assert.Empty(t, created2, "an existing SKU updates")
	require.Equal(t, []string{partID}, updated2)

	item = partItemPreview9(t, partID)
	assert.Equal(t, "updated description", jsonField(item, "description"))
	assertPartPreview9Price(t, jsonObject(item, "unit_value"), "7.77", nonCurrencyUnitID, "item.unit_value")
	assertPartPreview9Price(t, jsonObject(item, "unit_cost"), "3.33", nonCurrencyUnitID, "item.unit_cost")
	assert.Contains(t, catalogItemAttributeValues(t, jsonField(item, "id")), propValue)
}

// Problems only the write can see fail their row on the job; the job still completes.
func TestParts_BulkUpsert_RowErrorsLandOnTheJob(t *testing.T) {
	t.Parallel()

	t.Run("price per a currency", func(t *testing.T) {
		job := bulkUpsertPartsJob(t, map[string]any{
			"sku":       uniqueName("e2e-bup-part-badrate"),
			"category":  map[string]any{"id": SeedItemCategoryID},
			"unit_cost": partPreview9Price("1.00", e2eCurrencyUnitID),
		})
		assert.Empty(t, jobWrittenResults(job), "a bad rate unit must not be written")
		require.Len(t, jobErrors(job), 1, "the rejected row is recorded in errors")
	})

	t.Run("sku taken by another item type", func(t *testing.T) {
		sku := uniqueName("e2e-bup-part-xtype")
		createForgeItem(t, itemsPath, forgeItemBody("product", sku))
		job := bulkUpsertPartsJob(t, map[string]any{"sku": sku, "category": map[string]any{"id": SeedItemCategoryID}})
		assert.Empty(t, jobWrittenResults(job), "a conflicting SKU must not be written")
		require.Len(t, jobErrors(job), 1, "the conflicting row is recorded in errors")
	})
}

// Problems visible in the request refuse it whole with a preview.9 400 naming the row.
func TestParts_BulkUpsert_RejectedAtAccept(t *testing.T) {
	t.Parallel()
	dupSKU := uniqueName("e2e-bup-part-dup")

	cases := []struct {
		name  string
		rows  []map[string]any
		param string
	}{
		{"empty", nil, "parts"},
		{"duplicate sku", []map[string]any{
			{"sku": dupSKU, "category": map[string]any{"id": SeedItemCategoryID}},
			{"sku": dupSKU, "category": map[string]any{"id": SeedItemCategoryID}},
		}, "parts[1].sku"},
		{"unknown category", []map[string]any{
			{"sku": uniqueName("e2e-bup-part-ok"), "category": map[string]any{"id": SeedItemCategoryID}},
			{"sku": uniqueName("e2e-bup-part-badcat"), "category": map[string]any{"id": "ic_does_not_exist_000000000"}},
		}, "parts[1].category"},
		{"material category", []map[string]any{
			{"sku": uniqueName("e2e-bup-part-wrongcat"), "category": map[string]any{"id": SeedMaterialCategoryID}},
		}, "parts[0].category"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body := bulkUpsertParts(t, tc.rows...)
			requireStatus(t, 400, status, body)
			assert.Equal(t, tc.param, requirePreview9Error(t, body, "invalid_format")["param"])
		})
	}
}

// bulkCatalogPriceAmount fetches a part/material/product with one price included and returns its
// amount, checking it is in dollars per the each the bulk rows price in.
func bulkCatalogPriceAmount(t *testing.T, basePath, id, priceInclude string) string {
	t.Helper()
	status, body, err := apiClient.GetListRaw(basePath+"/"+id, url.Values{"include": {priceInclude}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	key := strings.TrimPrefix(priceInclude, "item.")
	price := jsonObject(jsonObject(parseJSON(body), "item"), key)
	require.NotNil(t, price, "%s should be populated with ?include=%s", key, priceInclude)
	assert.Equal(t, "usd", jsonField(price, "currency"), "%s.currency", key)
	assert.Equal(t, nonCurrencyUnitID, jsonField(jsonObject(price, "per_unit"), "id"), "%s.per_unit", key)
	return jsonField(price, "amount")
}
