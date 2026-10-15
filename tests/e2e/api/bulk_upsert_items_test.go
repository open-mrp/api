//go:build e2e

package api_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/open-mrp/api/shared/id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const itemsBulkUpsertPath = "/v1/catalog/items/actions/bulk-upsert"

// postBulkUpsertItems posts the rows and returns the raw response.
func postBulkUpsertItems(t *testing.T, rows ...map[string]any) *Response {
	t.Helper()
	items := make([]any, len(rows))
	for i, r := range rows {
		items[i] = r
	}
	resp, err := apiClient.PostFull(itemsBulkUpsertPath, map[string]any{"items": items}, newIdempotencyKey())
	require.NoError(t, err)
	return resp
}

// bulkUpsertItemsJob requires the 202 job acknowledgment, follows the job to completion and returns it. Every item a
// row wrote is deleted after the test.
func bulkUpsertItemsJob(t *testing.T, rows ...map[string]any) map[string]any {
	t.Helper()
	resp := postBulkUpsertItems(t, rows...)
	requireStatus(t, 202, resp.StatusCode, resp.Body)
	ack := parseJSON(resp.Body)
	assert.Equal(t, "async_job", jsonField(ack, "object"))
	jobID := jsonField(ack, "id")
	require.NotEmpty(t, jobID, "202 must name the job to poll")
	assert.True(t, strings.HasSuffix(resp.Header.Get("Location"), "/v1/core/async-jobs/"+jobID),
		"Location %q points at the job", resp.Header.Get("Location"))

	job := pollJobUntilTerminal(t, jobID)
	require.Equal(t, "completed", jsonField(job, "status"), "job should complete: %v", job)
	created, _ := jobResultIDs(job)
	t.Cleanup(func() {
		for _, itemID := range created {
			apiClient.Delete(itemsPath + "/" + itemID)
		}
	})
	return job
}

// requireBulkRowRejected checks the whole request was refused for the named row field.
func requireBulkRowRejected(t *testing.T, resp *Response, param string) {
	t.Helper()
	requireClientRejection(t, resp.StatusCode, resp.Body)
	errObj := requireErrorResponse(t, resp.Body, "", "invalid_request_error")
	assert.Contains(t, fmt.Sprint(errObj["param"])+"\n"+errorMessages(errObj), param)
}

func TestItems_BulkUpsert_CreatesAndUpdates(t *testing.T) {
	t.Parallel()
	existing := createForgeItem(t, itemsPath, forgeItemBody("part", uniqueName("e2e-bup-item-old")))
	existingID := jsonField(existing, "id")

	materialSKU := uniqueName("e2e-bup-item-mat")
	productSKU := uniqueName("e2e-bup-item-prod")
	job := bulkUpsertItemsJob(t,
		map[string]any{
			"type": "material", "sku": materialSKU, "category_id": SeedMaterialCategoryID,
			"description": "bulk yarn",
			"supply":      "buy",
			"unit_cost":   priceInputPer("2.00", SeedMaterialUnitID),
			"material": map[string]any{
				"order_point": map[string]any{"value": "25", "unit_id": SeedMaterialUnitID},
				"lead_time":   map[string]any{"value": "7", "unit_id": dayUnitID},
			},
		},
		map[string]any{
			"type": "product", "sku": productSKU, "category_id": SeedItemCategoryID,
			"status": "archived",
			"sales": map[string]any{
				"unit_price":        priceInputPer("12.00", SeedUnitID),
				"product_line_id":   SeedProductLineID,
				"portal_visibility": "visible",
			},
		},
		// Matched by SKU regardless of case.
		map[string]any{
			"type": "part", "sku": strings.ToUpper(jsonField(existing, "sku")), "category_id": SeedItemCategoryID,
			"description": "updated by bulk",
		},
	)

	results := jobResults(job)
	require.Len(t, results, 3, "one result per row: %v", job)
	for i, want := range []string{"created", "created", "updated"} {
		assert.Equal(t, want, results[i]["status"], "row %d", i)
		assertIDFormat(t, jobResultResourceID(results[i]), id.ItemIDPrefix)
	}
	assert.Equal(t, existingID, jobResultResourceID(results[2]), "the update names the matched item")
	assert.Empty(t, jobErrors(job))

	material := getItem(t, jobResultResourceID(results[0]))
	assertItemEnvelope(t, material, "material", materialSKU)
	assert.Equal(t, "bulk yarn", jsonField(material, "description"))
	assert.Equal(t, "buy", jsonField(material, "supply"))
	assert.Equal(t, "active", jsonField(material, "status"), "new items default to active")
	assertItemPrice(t, jsonObject(material, "unit_cost"), "2.00", SeedMaterialUnitID, "unit_cost")
	assertQuantityIn(t, jsonObject(jsonObject(material, "material"), "order_point"), "25", SeedMaterialUnitID, "material.order_point")
	assertQuantityIn(t, jsonObject(jsonObject(material, "material"), "lead_time"), "7", dayUnitID, "material.lead_time")
	assertNilField(t, material, "sales")

	product := getItem(t, jsonResultResourceIDOrFail(t, results[1]), "sales.product_line")
	assertItemEnvelope(t, product, "product", productSKU)
	assert.Equal(t, "archived", jsonField(product, "status"))
	sales := jsonObject(product, "sales")
	assertItemPrice(t, jsonObject(sales, "unit_price"), "12.00", SeedUnitID, "sales.unit_price")
	assert.Equal(t, "visible", jsonField(sales, "portal_visibility"))
	assert.Equal(t, SeedProductLineID, jsonField(jsonObject(sales, "product_line"), "id"))

	updated := getItem(t, existingID)
	assert.Equal(t, "updated by bulk", jsonField(updated, "description"))
	assert.Equal(t, jsonField(existing, "sku"), jsonField(updated, "sku"), "a case-insensitive match keeps the stored SKU")
}

func jsonResultResourceIDOrFail(t *testing.T, entry map[string]any) string {
	t.Helper()
	resourceID := jobResultResourceID(entry)
	require.NotEmpty(t, resourceID, "result row names its item: %v", entry)
	return resourceID
}

// Attributes are given by property and value, matched case-insensitively; a value that doesn't exist is created.
func TestItems_BulkUpsert_AttributesByPropertyAndValue(t *testing.T) {
	t.Parallel()
	value := uniqueName("e2e-bup-item-color")
	sku := uniqueName("e2e-bup-item-attr")
	job := bulkUpsertItemsJob(t, map[string]any{
		"type": "part", "sku": sku, "category_id": SeedItemCategoryID,
		"attributes": []any{map[string]any{"property": strings.ToLower(SeedPropertyName), "value": value}},
	})
	created, _ := jobResultIDs(job)
	require.Len(t, created, 1)
	itemID := created[0]

	item := getItem(t, itemID, "attributes", "attributes.property")
	attrs := jsonListData(item, "attributes")
	require.Len(t, attrs, 1)
	attr, _ := attrs[0].(map[string]any)
	assert.Equal(t, value, jsonField(attr, "value"))
	assert.Equal(t, SeedPropertyID, jsonField(jsonObject(attr, "property"), "id"))
	attrID := jsonField(attr, "id")
	// The test ends with the attribute detached, so it can go before the item does.
	t.Cleanup(func() { apiClient.Delete(attributePath(SeedPropertyID, attrID)) })

	// Re-sending the same value reuses the attribute; an empty list removes it.
	job = bulkUpsertItemsJob(t, map[string]any{
		"type": "part", "sku": sku, "category_id": SeedItemCategoryID,
		"attributes": []any{map[string]any{"property": SeedPropertyName, "value": strings.ToUpper(value)}},
	})
	_, updated := jobResultIDs(job)
	assert.Equal(t, []string{itemID}, updated)
	assert.Equal(t, []string{attrID}, attributeIDsOf(getItem(t, itemID, "attributes")))

	bulkUpsertItemsJob(t, map[string]any{
		"type": "part", "sku": sku, "category_id": SeedItemCategoryID, "attributes": []any{},
	})
	assert.Empty(t, attributeIDsOf(getItem(t, itemID, "attributes")))
	// Without attributes the row leaves them alone.
	patchItem(t, itemID, map[string]any{"attribute_ids": []string{attrID}})
	bulkUpsertItemsJob(t, map[string]any{"type": "part", "sku": sku, "category_id": SeedItemCategoryID, "description": "x"})
	assert.Equal(t, []string{attrID}, attributeIDsOf(getItem(t, itemID, "attributes")))
	patchItem(t, itemID, map[string]any{"attribute_ids": []string{}})
}

// A row whose SKU names an item of another type fails on its own; the rest of the batch is written.
func TestItems_BulkUpsert_TypeChangeFailsTheRow(t *testing.T) {
	t.Parallel()
	existing := createForgeItem(t, itemsPath, forgeItemBody("part", uniqueName("e2e-bup-item-xtype")))
	okSKU := uniqueName("e2e-bup-item-xtype-ok")

	job := bulkUpsertItemsJob(t,
		map[string]any{"type": "product", "sku": jsonField(existing, "sku"), "category_id": SeedItemCategoryID},
		map[string]any{"type": "part", "sku": okSKU, "category_id": SeedItemCategoryID},
	)
	results := jobResults(job)
	require.Len(t, results, 2)
	assert.Equal(t, "failed", results[0]["status"])
	assert.Contains(t, jobRowErrorMessage(results[0]), "type")
	assert.Equal(t, "created", results[1]["status"])
	assert.Equal(t, "part", jsonField(getItem(t, jsonField(existing, "id")), "type"), "the existing item keeps its type")
}

func TestItems_BulkUpsert_RejectedAtAccept(t *testing.T) {
	t.Parallel()
	good := func() map[string]any { return forgeItemBody("part", uniqueName("e2e-bup-item-ok")) }
	dupSKU := uniqueName("e2e-bup-item-dup")

	cases := []struct {
		name  string
		rows  []map[string]any
		param string
	}{
		{"unknown type", []map[string]any{good(), {"type": "widget", "sku": uniqueName("e2e-bup-item-bad"), "category_id": SeedItemCategoryID}}, "items[1].type"},
		{"unknown category", []map[string]any{good(), {"type": "part", "sku": uniqueName("e2e-bup-item-bad"), "category_id": "itcg_00000000000000000000000000"}}, "items[1].category_id"},
		{"category of the wrong kind", []map[string]any{{"type": "material", "sku": uniqueName("e2e-bup-item-bad"), "category_id": SeedItemCategoryID}}, "items[0].category_id"},
		{"material section on a product", []map[string]any{{
			"type": "product", "sku": uniqueName("e2e-bup-item-bad"), "category_id": SeedItemCategoryID,
			"material": map[string]any{"order_point": map[string]any{"value": "1", "unit_id": SeedUnitID}},
		}}, "items[0].material"},
		{"duplicate sku", []map[string]any{
			{"type": "part", "sku": dupSKU, "category_id": SeedItemCategoryID},
			{"type": "part", "sku": dupSKU, "category_id": SeedItemCategoryID},
		}, "items[1].sku"},
		{"missing category", []map[string]any{{"type": "part", "sku": uniqueName("e2e-bup-item-bad")}}, "items[0].category_id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requireBulkRowRejected(t, postBulkUpsertItems(t, tc.rows...), tc.param)
		})
	}

	t.Run("no rows", func(t *testing.T) {
		resp := postBulkUpsertItems(t)
		requireClientRejection(t, resp.StatusCode, resp.Body)
	})

	// Nothing from a refused batch is written, including its good rows.
	t.Run("refused batch writes nothing", func(t *testing.T) {
		row := good()
		resp := postBulkUpsertItems(t, row, map[string]any{"type": "widget", "sku": uniqueName("e2e-bup-item-bad"), "category_id": SeedItemCategoryID})
		requireClientRejection(t, resp.StatusCode, resp.Body)
		assert.Empty(t, listItemIDs(t, map[string][]string{"skus": {jsonField(row, "sku")}}))
	})
}
