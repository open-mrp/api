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
