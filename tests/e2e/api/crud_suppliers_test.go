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

	got := parseJSON(body)
	assert.Nil(t, got["bill_to_address"], "bill_to_address should be null without ?include=bill_to_address")
	assert.Nil(t, got["ship_to_address"], "ship_to_address should be null without ?include=ship_to_address")
}

func TestSuppliers_IncludeBillToAddress(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(suppliersPath+"/"+SeedSupplierAccountID, url.Values{"include": {"bill_to_address"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	// Address may legitimately be null if supplier has no billing address set
	if addr := jsonObject(got, "bill_to_address"); addr != nil {
		assert.Equal(t, "address", jsonField(addr, "object"))
		assert.NotEmpty(t, jsonField(addr, "id"))
	}
}

func TestSuppliers_IncludeShipToAddress(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(suppliersPath+"/"+SeedSupplierAccountID, url.Values{"include": {"ship_to_address"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	if addr := jsonObject(got, "ship_to_address"); addr != nil {
		assert.Equal(t, "address", jsonField(addr, "object"))
		assert.NotEmpty(t, jsonField(addr, "id"))
	}
}
