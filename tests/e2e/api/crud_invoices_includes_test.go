//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// Invoice — Include Tests
// ──────────────────────────────────────────────
//
// Invoice GET endpoint whitelists: lines, allocations.
// (order, billing_address, shipment are always populated as summaries and
// are not expandable on this endpoint — the include definition registers them
// for use on other endpoints.)

func TestInvoices_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(invoicesPath+"/"+SeedInvoiceID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["lines"], "lines should be null without ?include=lines")
	assert.Nil(t, got["allocations"], "allocations should be null without ?include=allocations")
}

func TestInvoices_IncludeLines(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(invoicesPath+"/"+SeedInvoiceID, url.Values{"include": {"lines"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	lines := jsonObject(parseJSON(body), "lines")
	require.NotNil(t, lines, "lines should be present with ?include=lines")
	assert.Equal(t, "list", jsonField(lines, "object"))
}

func TestInvoices_IncludeAllocations(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(invoicesPath+"/"+SeedInvoiceID, url.Values{"include": {"allocations"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["allocations"]
	assert.True(t, ok, "allocations key should be present with ?include=allocations")
	if a := jsonObject(got, "allocations"); a != nil {
		assert.Equal(t, "list", jsonField(a, "object"))
	}
}

// The order include must carry the full sales order, not the reference the batch loader built without
// its lifecycle timestamps: issued_at/first_ship_at/completed_at rode the batch projection but were
// never mapped onto the reference, so a hydrated order reported them null even though the order stores
// them. See TestIncludes_HydratedToOneMatchesCanonical for the cross-endpoint guard.
func TestInvoices_IncludeOrderCarriesLifecycleTimestamps(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(invoicesPath+"/"+SeedInvoiceID, url.Values{"include": {"order"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	order := jsonObject(parseJSON(body), "order")
	require.NotNil(t, order, "order should be present with ?include=order")
	assert.Equal(t, "sales_order", jsonField(order, "object"))
	assert.NotEmpty(t, jsonField(order, "issued_at"),
		"the included order carries its issued_at, not a reference stripped of lifecycle timestamps")
	assert.NotEmpty(t, jsonField(order, "first_ship_at"),
		"the included order carries its first_ship_at, not a reference stripped of lifecycle timestamps")
	assert.NotEmpty(t, jsonField(order, "completed_at"),
		"the included order carries its completed_at, not a reference stripped of lifecycle timestamps")
}
