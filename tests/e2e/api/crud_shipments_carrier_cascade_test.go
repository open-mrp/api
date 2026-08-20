//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The shipping cases of the seeded SHP-001 shipment, which hangs off this order.
const cascadeOrderID = "or_01k0a8bs2ye3f9p8sj0m4dfmwe"

func caseCarrierIDs(t *testing.T, shipmentID string) map[string]string {
	t.Helper()

	status, body, err := apiClient.GetListRaw(shipmentsPath+"/"+shipmentID, url.Values{"include": {"shipping_cases"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	// shipping_cases is an expanded list resource, so the cases hang off its data array.
	cases := jsonObject(parseJSON(body), "shipping_cases")
	require.NotNil(t, cases, "shipping_cases must be expanded")

	carriers := map[string]string{}
	for _, raw := range jsonArray(cases, "data") {
		c, ok := raw.(map[string]any)
		require.True(t, ok, "shipping_cases[] entries must be objects")
		carrier := jsonObject(c, "carrier")
		require.NotNil(t, carrier, "shipping_cases[].carrier must be present")
		carriers[jsonField(c, "id")] = jsonField(carrier, "id")
	}
	return carriers
}

// Re-pointing an order's carrier must carry its shipments' cases along: a case's tracking link is
// built from its own carrier, so one left behind deep-links to the wrong carrier.
func TestShipments_OrderCarrierChangeCascadesToShippingCases(t *testing.T) {
	// Not parallel: this mutates a shared order's carrier and restores it.
	before := caseCarrierIDs(t, SeedShipmentID)
	require.NotEmpty(t, before, "the seeded shipment must have shipping cases for this to mean anything")

	var original string
	for _, carrierID := range before {
		original = carrierID
		break
	}
	require.NotEmpty(t, original)

	// Pick any other carrier in the account to move to.
	list, _, err := apiClient.GetList("/v1/operations/carriers", nil)
	require.NoError(t, err)
	var target string
	for _, raw := range list.Data {
		if id := jsonField(parseJSON(raw), "id"); id != "" && id != original {
			target = id
			break
		}
	}
	require.NotEmpty(t, target, "the account needs a second carrier to move to")

	restore := func(carrierID string) {
		status, body, err := apiClient.Patch(salesOrdersPath+"/"+cascadeOrderID,
			map[string]any{"carrier_id": carrierID}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
	}
	t.Cleanup(func() { restore(original) })

	restore(target)

	// The sync runs out of band, so the cases move shortly after the order write commits.
	eventually(t, 20*time.Second, 500*time.Millisecond, func() error {
		for caseID, carrierID := range caseCarrierIDs(t, SeedShipmentID) {
			if carrierID != target {
				return fmt.Errorf("case %s still on carrier %s, want %s", caseID, carrierID, target)
			}
		}
		return nil
	})

	assert.NotEqual(t, original, target, "the test must actually change the carrier to prove anything")
}
