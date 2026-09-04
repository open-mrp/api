//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Bulk Reconcile Items: counting stock and correcting the books, by SKU, in one call.
//
// The endpoint's whole point is that one bad row does not fail the batch — an unknown SKU is
// skipped and an unknown unit is an error, while every other row still lands. That partition is
// what the dashboard's reconcile importer reports back to the operator, so each of the three
// outcome lists is exercised here rather than just the happy path.
//
// Every test reconciles items it created itself. Reconciling a seeded item would move a figure
// other tests assert against.

const bulkReconcilePath = "/v1/catalog/items/actions/bulk-reconcile"

// newReconcilableItem creates a material and returns the SKU and catalog item id it produces.
func newReconcilableItem(t *testing.T) (sku, itemID string) {
	t.Helper()

	sku = uniqueName("e2e-rec")
	createStatus, createBody, err := apiClient.Post(materialsPath, validMaterialBody(sku), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	materialID := jsonField(parseJSON(createBody), "id")
	require.NotEmpty(t, materialID)
	t.Cleanup(func() { _, _, _ = apiClient.Delete(materialsPath + "/" + materialID) })

	list, status, err := apiClient.GetList(itemsPath, url.Values{"q": {sku}})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.Len(t, list.Data, 1, "exactly one item should match the unique SKU %q", sku)

	return sku, DataItemField(list.Data[0], "id")
}

func bulkReconcile(t *testing.T, rows []map[string]any, reconcileType string) map[string]any {
	t.Helper()

	status, body, err := apiClient.Post(bulkReconcilePath, map[string]any{
		"data":           rows,
		"reconcile_type": reconcileType,
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "bulk reconcile must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	return parseJSON(body)
}

// onHandValue reads the item's on-hand figure, which is the basis a `force` reconcile measures
// against and therefore the figure these tests have to check.
func onHandValue(t *testing.T, itemID string) string {
	t.Helper()

	status, body, err := apiClient.GetListRaw(itemsPath+"/"+itemID+"/inventory", url.Values{"include": {"on_hand"}})
	require.NoError(t, err)
	require.Less(t, status, 500, "item inventory must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	onHand := jsonObject(parseJSON(body), "on_hand")
	require.NotNil(t, onHand, "on_hand must expand when asked for: %s", string(body))
	return jsonField(onHand, "value")
}

// --- Response shape ---

func TestBulkReconcileItems_ResponseShape(t *testing.T) {
	t.Parallel()

	sku, _ := newReconcilableItem(t)
	resp := bulkReconcile(t, []map[string]any{
		{"sku": sku, "unit": "ea", "quantity": "5"},
	}, "force")

	assertObjectField(t, resp, "bulk_reconcile_items_response")

	for _, key := range []string{"reconciled_items", "skipped_items", "errors"} {
		list := jsonObject(resp, key)
		require.NotNil(t, list, "%s is always present, even when empty: %v", key, resp)
		assert.NotNil(t, jsonArray(list, "data"), "%s carries a data array", key)
	}

	reconciled := jsonListData(resp, "reconciled_items")
	require.Len(t, reconciled, 1, "the one good row must be reported as reconciled: %v", resp)

	row, ok := reconciled[0].(map[string]any)
	require.True(t, ok)
	assertObjectField(t, row, "reconciled_item_result")

	// Every list in the response names its item the same way — as an entity carrying the SKU —
	// so the importer needs one code path to label a row whatever became of it.
	item := jsonObject(row, "item")
	require.NotNil(t, item, "a reconciled row names its item: %v", row)
	assertObjectField(t, item, "entity")
	assert.Equal(t, "item", jsonField(item, "type"))
	assert.NotEmpty(t, jsonField(item, "id"))
	assert.Equal(t, sku, jsonField(item, "name"), "the entity's name carries the SKU")

	for _, key := range []string{"previous_quantity", "new_quantity"} {
		quantity := jsonObject(row, key)
		require.NotNil(t, quantity, "%s is required on a reconciled row: %v", key, row)
		assert.NotEmpty(t, jsonField(quantity, "value"))
		assertObjectField(t, quantity, "computed_quantity")
	}
}

// A reconciled figure is meaningless without the unit it was recorded in, and the unit is not
// behind an include — a computed quantity carries it. Before this, the unit came back as an id
// with every other field blank, which the importer could not render.
func TestBulkReconcileItems_ReconciledQuantitiesCarryAFullyResolvedUnit(t *testing.T) {
	t.Parallel()

	sku, _ := newReconcilableItem(t)
	resp := bulkReconcile(t, []map[string]any{
		{"sku": sku, "unit": "ea", "quantity": "5"},
	}, "force")

	reconciled := jsonListData(resp, "reconciled_items")
	require.Len(t, reconciled, 1, "%v", resp)
	row, ok := reconciled[0].(map[string]any)
	require.True(t, ok)

	for _, key := range []string{"previous_quantity", "new_quantity"} {
		quantity := jsonObject(row, key)
		require.NotNil(t, quantity, "%s: %v", key, row)
		assertUnitHydrated(t, jsonObject(quantity, "unit"), key+".unit")
	}
}

// --- Reconcile types ---

func TestBulkReconcileItems_ForceSetsTheExactQuantity(t *testing.T) {
	t.Parallel()

	sku, itemID := newReconcilableItem(t)

	bulkReconcile(t, []map[string]any{{"sku": sku, "unit": "ea", "quantity": "25"}}, "force")
	assertDecimalEqual(t, "25", onHandValue(t, itemID), "force sets the quantity to exactly what was sent")

	// Forcing to the figure already on hand writes nothing, which is the property that makes a
	// re-uploaded spreadsheet safe.
	bulkReconcile(t, []map[string]any{{"sku": sku, "unit": "ea", "quantity": "25"}}, "force")
	assertDecimalEqual(t, "25", onHandValue(t, itemID), "forcing to the current figure is a no-op")
}

func TestBulkReconcileItems_AdditionAddsToTheCurrentQuantity(t *testing.T) {
	t.Parallel()

	sku, itemID := newReconcilableItem(t)

	bulkReconcile(t, []map[string]any{{"sku": sku, "unit": "ea", "quantity": "10"}}, "force")
	assertDecimalEqual(t, "10", onHandValue(t, itemID))

	bulkReconcile(t, []map[string]any{{"sku": sku, "unit": "ea", "quantity": "5"}}, "addition")
	assertDecimalEqual(t, "15", onHandValue(t, itemID), "addition adds rather than replaces")
}

func TestBulkReconcileItems_PreviousAndNewQuantityBracketTheChange(t *testing.T) {
	t.Parallel()

	sku, _ := newReconcilableItem(t)

	bulkReconcile(t, []map[string]any{{"sku": sku, "unit": "ea", "quantity": "7"}}, "force")
	resp := bulkReconcile(t, []map[string]any{{"sku": sku, "unit": "ea", "quantity": "12"}}, "force")

	reconciled := jsonListData(resp, "reconciled_items")
	require.Len(t, reconciled, 1)
	row, ok := reconciled[0].(map[string]any)
	require.True(t, ok)

	assertDecimalEqual(t, "7", jsonField(jsonObject(row, "previous_quantity"), "value"))
	assertDecimalEqual(t, "12", jsonField(jsonObject(row, "new_quantity"), "value"))
}

// --- Partial failure: the three outcome lists ---

func TestBulkReconcileItems_UnknownSKUIsSkippedAndDoesNotFailTheBatch(t *testing.T) {
	t.Parallel()

	sku, itemID := newReconcilableItem(t)
	missingSKU := uniqueName("e2e-rec-missing")

	resp := bulkReconcile(t, []map[string]any{
		{"sku": missingSKU, "unit": "ea", "quantity": "3"},
		{"sku": sku, "unit": "ea", "quantity": "9"},
	}, "force")

	skipped := jsonListData(resp, "skipped_items")
	require.Len(t, skipped, 1, "the unknown SKU is skipped: %v", resp)
	row, ok := skipped[0].(map[string]any)
	require.True(t, ok)
	assertObjectField(t, row, "skipped_item_result")
	assert.Equal(t, missingSKU, jsonField(row, "sku"))
	assert.NotEmpty(t, jsonField(row, "reason"), "a skipped row explains itself")

	assert.Len(t, jsonListData(resp, "reconciled_items"), 1, "the good row still lands")
	assertDecimalEqual(t, "9", onHandValue(t, itemID), "the good row was actually written")
}

// The importer reports failed rows back to the operator by SKU, and reads that SKU off the error's
// entity name — the error result carries an Entity, not a full item. If that ever stops being the
// submitted SKU the operator gets an unlabelled failure row.
func TestBulkReconcileItems_UnknownUnitIsAnErrorNamingTheSubmittedSKU(t *testing.T) {
	t.Parallel()

	sku, itemID := newReconcilableItem(t)

	resp := bulkReconcile(t, []map[string]any{
		{"sku": sku, "unit": "definitely-not-a-unit", "quantity": "4"},
	}, "force")

	errors := jsonListData(resp, "errors")
	require.Len(t, errors, 1, "an unknown unit is reported as an error, not a skip: %v", resp)

	row, ok := errors[0].(map[string]any)
	require.True(t, ok)
	assertObjectField(t, row, "reconcile_error_result")
	assert.NotEmpty(t, jsonField(row, "error"), "an errored row explains itself")

	item := jsonObject(row, "item")
	require.NotNil(t, item, "the error names the item the row referred to: %v", row)
	assert.Equal(t, sku, jsonField(item, "name"),
		"the entity's name carries the submitted SKU, which is what the importer displays")

	assertDecimalEqual(t, "0", onHandValue(t, itemID), "an errored row writes nothing")
}

func TestBulkReconcileItems_MixedBatchPartitionsEveryRow(t *testing.T) {
	t.Parallel()

	goodSKU, goodItemID := newReconcilableItem(t)
	badUnitSKU, _ := newReconcilableItem(t)
	missingSKU := uniqueName("e2e-rec-absent")

	resp := bulkReconcile(t, []map[string]any{
		{"sku": goodSKU, "unit": "ea", "quantity": "6"},
		{"sku": badUnitSKU, "unit": "not-a-unit", "quantity": "6"},
		{"sku": missingSKU, "unit": "ea", "quantity": "6"},
	}, "force")

	assert.Len(t, jsonListData(resp, "reconciled_items"), 1, "one row reconciled: %v", resp)
	assert.Len(t, jsonListData(resp, "errors"), 1, "one row errored: %v", resp)
	assert.Len(t, jsonListData(resp, "skipped_items"), 1, "one row skipped: %v", resp)

	assertDecimalEqual(t, "6", onHandValue(t, goodItemID), "the good row survived its neighbours")
}

// --- Units ---

// The unit is checked for existence but the quantity is always recorded in the item's own base
// unit, so a row naming a different real unit is not converted.
func TestBulkReconcileItems_QuantityIsRecordedInTheItemsBaseUnit(t *testing.T) {
	t.Parallel()

	sku, itemID := newReconcilableItem(t)

	resp := bulkReconcile(t, []map[string]any{{"sku": sku, "unit": "pr", "quantity": "8"}}, "force")
	require.Len(t, jsonListData(resp, "reconciled_items"), 1,
		"a real account unit is accepted: %v", resp)

	assertDecimalEqual(t, "8", onHandValue(t, itemID),
		"the figure is taken as already expressed in the item's base unit, not converted from pairs")
}

// --- Idempotency ---

func TestBulkReconcileItems_IsIdempotent(t *testing.T) {
	t.Parallel()

	sku, itemID := newReconcilableItem(t)
	key := newIdempotencyKey()
	body := map[string]any{
		"data":           []map[string]any{{"sku": sku, "unit": "ea", "quantity": "3"}},
		"reconcile_type": "addition",
	}

	firstStatus, firstBody, err := apiClient.Post(bulkReconcilePath, body, key)
	require.NoError(t, err)
	requireStatus(t, 200, firstStatus, firstBody)
	assertDecimalEqual(t, "3", onHandValue(t, itemID))

	secondStatus, secondBody, err := apiClient.Post(bulkReconcilePath, body, key)
	require.NoError(t, err)
	requireStatus(t, 200, secondStatus, secondBody)

	assertDecimalEqual(t, "3", onHandValue(t, itemID),
		"replaying the same key must not add the quantity a second time")
}

// --- Validation ---

func TestBulkReconcileItems_RejectsAnUnknownReconcileType(t *testing.T) {
	t.Parallel()

	sku, _ := newReconcilableItem(t)

	status, body, err := apiClient.Post(bulkReconcilePath, map[string]any{
		"data":           []map[string]any{{"sku": sku, "unit": "ea", "quantity": "1"}},
		"reconcile_type": "bogus_e2e_type",
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "an unknown reconcile type is a client error: %s", string(body))
	require.Equal(t, 400, status, "reconcile_type only accepts the documented values: %s", string(body))
}

func TestBulkReconcileItems_RejectsAMissingReconcileType(t *testing.T) {
	t.Parallel()

	sku, _ := newReconcilableItem(t)

	status, body, err := apiClient.Post(bulkReconcilePath, map[string]any{
		"data": []map[string]any{{"sku": sku, "unit": "ea", "quantity": "1"}},
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "a missing reconcile type is a client error: %s", string(body))
	assert.Equal(t, 400, status, "reconcile_type is required: %s", string(body))
}

func TestBulkReconcileItems_RejectsMissingData(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(bulkReconcilePath, map[string]any{
		"reconcile_type": "force",
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "a missing data array is a client error: %s", string(body))
	assert.Equal(t, 400, status, "data is required: %s", string(body))
}

// The three row fields are documented as required, but only the quantity is enforced up front. A
// row with no sku or no unit is reported per-row instead, which is consistent with the endpoint's
// promise that one bad row never fails the batch.
func TestBulkReconcileItems_RowMissingSKUOrUnitIsReportedPerRow(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		row  map[string]any
	}{
		{"no sku", map[string]any{"unit": "ea", "quantity": "1"}},
		{"no unit", map[string]any{"sku": "E2E-SKU-ABSENT", "quantity": "1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			status, body, err := apiClient.Post(bulkReconcilePath, map[string]any{
				"data":           []map[string]any{tc.row},
				"reconcile_type": "force",
			}, newIdempotencyKey())
			require.NoError(t, err)
			require.Less(t, status, 500, "a malformed row must not 5xx: %s", string(body))
			requireStatus(t, 200, status, body)

			resp := parseJSON(body)
			reported := len(jsonListData(resp, "skipped_items")) + len(jsonListData(resp, "errors"))
			assert.Equal(t, 1, reported, "the row is accounted for as skipped or errored: %v", resp)
			assert.Empty(t, jsonListData(resp, "reconciled_items"), "and nothing was written: %v", resp)
		})
	}
}

// A quantity that is not a decimal is the one row-level fault refused up front, failing the batch.
func TestBulkReconcileItems_RejectsAMalformedQuantity(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(bulkReconcilePath, map[string]any{
		"data":           []map[string]any{{"sku": "E2E-SKU", "unit": "ea", "quantity": "not-a-number"}},
		"reconcile_type": "force",
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "a malformed quantity is a client error: %s", string(body))
	assert.Equal(t, 400, status, "quantity must be a decimal: %s", string(body))
}

func TestBulkReconcileItems_RejectsAnUnknownBodyField(t *testing.T) {
	t.Parallel()

	sku, _ := newReconcilableItem(t)

	status, body, err := apiClient.Post(bulkReconcilePath, map[string]any{
		"data":            []map[string]any{{"sku": sku, "unit": "ea", "quantity": "1"}},
		"reconcile_type":  "force",
		bogusE2EJSONField: true,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assertJSONUnknownFieldRejected(t, "POST", bulkReconcilePath, status, body)
}
