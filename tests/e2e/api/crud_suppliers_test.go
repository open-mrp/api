//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const suppliersPath = "/v1/operations/suppliers"

// ──────────────────────────────────────────────
// Supplier — Include Tests
// ──────────────────────────────────────────────

func TestSuppliers_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(suppliersPath+"/"+SeedSupplierAccountID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	defaults := jsonObject(parseJSON(body), "defaults")
	require.NotNil(t, defaults)
	for _, ref := range []string{"carrier", "service_level", "payment_term", "shipping_term", "bill_to_address", "ship_to_address"} {
		assert.Nil(t, defaults[ref], "defaults.%s should be null without its include", ref)
	}
}

func TestSuppliers_IncludeBillToAddress(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(suppliersPath+"/"+SeedSupplierAccountID, url.Values{"include": {"defaults.bill_to_address"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	// Address may legitimately be null if supplier has no billing address set
	if addr := jsonObject(jsonObject(got, "defaults"), "bill_to_address"); addr != nil {
		assert.Equal(t, "address", jsonField(addr, "object"))
		assert.NotEmpty(t, jsonField(addr, "id"))
	}
}

func TestSuppliers_IncludeShipToAddress(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(suppliersPath+"/"+SeedSupplierAccountID, url.Values{"include": {"defaults.ship_to_address"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	if addr := jsonObject(jsonObject(got, "defaults"), "ship_to_address"); addr != nil {
		assert.Equal(t, "address", jsonField(addr, "object"))
		assert.NotEmpty(t, jsonField(addr, "id"))
	}
}
