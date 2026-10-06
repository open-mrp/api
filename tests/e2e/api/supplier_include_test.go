//go:build e2e

package api_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// supplierListedFields are the supplier's own fields, which a document's supplier carries exactly as the supplier list does.
var supplierListedFields = []string{"id", "object", "name", "number", "note", "material_count", "created_at", "updated_at"}

// A role that may read purchase or receiving orders sees the supplier they name in full, though it cannot read suppliers itself.
func TestSupplierInclude_DocumentReadersSeeTheSupplierAsListed(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		permission string
		path       string
	}{
		{permission: "purchase_orders:read", path: purchaseOrdersPath + "/" + SeedPurchaseOrderID},
		{permission: "receiving_orders:read", path: receivingOrdersPath + "/" + SeedReceivingOrderID},
	} {
		t.Run(tc.permission, func(t *testing.T) {
			t.Parallel()
			reader := customRoleClient(t, tc.permission)
			supplier := jsonObject(parseJSON(mustGetAs(t, reader, tc.path, url.Values{"include": {"supplier"}})), "supplier")
			require.NotNil(t, supplier, "the document's supplier expands")
			supplierID := jsonField(supplier, "id")

			status, body, err := reader.GetListRaw(suppliersPath+"/"+supplierID, nil)
			require.NoError(t, err)
			require.Equal(t, http.StatusForbidden, status, "the role cannot read the supplier on its own: %s", string(body))

			canonical := parseJSON(mustGetAs(t, apiClient, suppliersPath+"/"+supplierID, nil))
			require.NotNil(t, canonical["material_count"], "the supplier reports its material count")
			for _, field := range supplierListedFields {
				assert.Equal(t, canonical[field], supplier[field], "supplier.%s", field)
			}
			assertNilField(t, supplier, "bill_to_address")
			assertNilField(t, supplier, "ship_to_address")
		})
	}
}

// A list page's suppliers are read for the page at once and carry the same fields a single order's do.
func TestSupplierInclude_ListedReceivingOrdersCarryTheirSupplier(t *testing.T) {
	t.Parallel()

	list := parseJSON(mustGetAs(t, apiClient, receivingOrdersPath, url.Values{"include": {"supplier"}, "limit": {"25"}}))
	rows := jsonArray(list, "data")
	require.NotEmpty(t, rows)
	carried := 0
	for _, raw := range rows {
		row := raw.(map[string]any)
		supplier := jsonObject(row, "supplier")
		if supplier == nil {
			continue
		}
		carried++
		assert.Equal(t, "supplier", jsonField(supplier, "object"))
		assert.NotNil(t, supplier["material_count"], "receiving order %s: supplier.material_count", jsonField(row, "id"))
		assertValidTimestamp(t, jsonField(supplier, "created_at"), "supplier.created_at")
	}
	assert.Positive(t, carried, "some receiving order on the page names a supplier the account still has")
}

// The supplier stays null until asked for.
func TestSupplierInclude_NullWithoutInclude(t *testing.T) {
	t.Parallel()

	for _, path := range []string{purchaseOrdersPath + "/" + SeedPurchaseOrderID, receivingOrdersPath + "/" + SeedReceivingOrderID} {
		assertNilField(t, parseJSON(mustGetAs(t, apiClient, path, nil)), "supplier")
	}
}
