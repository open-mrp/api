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
