//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const preview8APIVersion = "1.0.forge-preview.8"

// preview.9 matches a pick search against the start of the pick and PO numbers. A client still pinned
// to preview.8 keeps matching anywhere in them.
func TestVersionCompat_Preview8PickSearchMatchesAnywhere(t *testing.T) {
	t.Parallel()
	pinned := apiClient.WithAPIVersion(preview8APIVersion)
	customerID := leadTimeCustomer(t, "e2e-compat-pick", nil, "")
	po := uniqueName("PO-COMPAT")
	_, pickID := issuedOrderAndPick(t, customerID, map[string]any{"customer_purchase_order_number": po})
	number := jsonField(retrievePick(t, pickID), "number")
	require.GreaterOrEqual(t, len(number), 4, "the pick number must leave a searchable tail once its first character is dropped")

	for what, q := range map[string]string{
		"middle of the PO number":   po[3:],
		"middle of the pick number": number[1:],
	} {
		params := url.Values{"q": {q}, "customer_ids": {customerID}}
		assert.Contains(t, dashPicksAllIDs(t, pinned, picksPath, params), pickID, "preview.8 %s %q", what, q)
		assert.NotContains(t, dashPicksAllIDs(t, apiClient, picksPath, params), pickID, "latest %s %q", what, q)
	}

	// A preview.8 client that already asks for prefix matching gets it.
	params := url.Values{"q": {po[3:]}, "q_match": {"prefix"}, "customer_ids": {customerID}}
	assert.NotContains(t, dashPicksAllIDs(t, pinned, picksPath, params), pickID)
}
