//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Verifies the sales-order PATCH clearable-field contract (customer PO / note):
//   - omit  → leave unchanged
//   - null  → clear (SQL NULL)
//   - value → set; an empty string is a literal empty value, NEVER a clear.
func TestUpdateSalesOrder_ClearableFieldSemantics(t *testing.T) {
	t.Parallel()

	body := minimalSalesOrderCreateBody(t, SeedCustomerAccountID)
	body["note"] = "original note"
	body["customer_purchase_order_number"] = uniqueName("PO-ORIG")
	created := createAndCleanup(t, salesOrdersPath, body)
	id := jsonField(created, "id")
	path := salesOrdersPath + "/" + id
	origPO := body["customer_purchase_order_number"].(string)

	retrieve := func() map[string]any {
		status, b, err := apiClient.GetListRaw(path, url.Values{})
		require.NoError(t, err)
		requireStatus(t, 200, status, b)
		return parseJSON(b)
	}

	patch := func(patchBody map[string]any) {
		t.Helper()
		status, b, err := apiClient.Patch(path, patchBody, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, b)
	}

	// Baseline: both fields set from create.
	got := retrieve()
	assert.Equal(t, "original note", got["note"])
	assert.Equal(t, origPO, got["customer_purchase_order_number"])

	// null → clear the note; the omitted customer PO is left untouched.
	patch(map[string]any{"note": nil})
	got = retrieve()
	assert.Nil(t, got["note"], "sending null must clear the note")
	assert.Equal(t, origPO, got["customer_purchase_order_number"], "an omitted field must be left unchanged")

	// empty string → a literal empty value, NOT a clear.
	patch(map[string]any{"customer_purchase_order_number": ""})
	got = retrieve()
	assert.Equal(t, "", got["customer_purchase_order_number"], "an empty string sets a literal empty value, never clears")

	// value → set.
	patch(map[string]any{"note": "new note"})
	got = retrieve()
	assert.Equal(t, "new note", got["note"])
}

// The line PATCH applies the same contract to the line's own snapshot of the product.
// product_description is the field the order UI edits inline, so both halves matter: an
// unrelated line edit must not wipe it, and an empty string must actually stick — a
// repository that folded "" back into "leave unchanged" would silently restore the old
// text the next time the order is read.
func TestUpdateSalesOrderLine_ProductDescriptionSemantics(t *testing.T) {
	t.Parallel()

	orderID := createLifecycleOrder(t)
	lineID := orderSaleLineID(t, orderID)
	linePath := salesOrdersPath + "/" + orderID + "/lines/" + lineID

	description := func() any {
		t.Helper()
		got := getSalesOrder(t, orderID, url.Values{"include": {"lines"}})
		lines := jsonObject(got, "lines")
		require.NotNil(t, lines, "lines present with ?include=lines")
		for _, raw := range jsonArray(lines, "data") {
			line, ok := raw.(map[string]any)
			require.True(t, ok)
			if jsonField(line, "id") == lineID {
				return line["product_description"]
			}
		}
		require.FailNow(t, "updated line missing from the order")
		return nil
	}

	patchLine := func(body map[string]any) {
		t.Helper()
		status, b, err := apiClient.Patch(linePath, body, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, b)
	}

	// value → set.
	patchLine(map[string]any{"product_description": "Hand-written line note"})
	assert.Equal(t, "Hand-written line note", description())

	// omitted → left unchanged by an unrelated field's update.
	patchLine(map[string]any{"quantity": map[string]any{"value": "4", "unit_id": SeedUnitID}})
	assert.Equal(t, "Hand-written line note", description(), "an omitted description must be left unchanged")

	// empty string → a literal empty value, not a no-op.
	patchLine(map[string]any{"product_description": ""})
	assert.Equal(t, "", description(), "an empty string must clear the line's description")
}
