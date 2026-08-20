//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Coverage for the estimate-rate action, which had none. The seeded "delivery" carrier is not
// linked to a Shippo account, so every path here lands on the "no live rating" branch and quotes
// 0 — the same branch rate_shop_test relies on.
//
// NOT covered, and deliberately so: the branches that need state no fixture has — a freight-exempt
// product line or customer, a flat-rate or free-freight shipping term (the seeded flat-rate term
// shtm_01seedcustflat000 is not any customer's default), a met free-shipping minimum, and a live
// Shippo rate. Those need seed rows before they can be asserted rather than guessed at.

const estimateRatePath = "/v1/operations/shipments/actions/estimate-rate"

func estimateRateRequestBody() map[string]any {
	return map[string]any{
		"carrier_id":       SeedCarrierID,
		"service_level_id": SeedServiceLevelID,
		"from_address": map[string]any{
			"name":          "Origin",
			"street_line_1": "123 Main Street",
			"locality":      "Austin",
			"state":         "TX",
			"postal_code":   "78701",
			"country":       "US",
		},
		"to_address": map[string]any{
			"name":          "Destination",
			"street_line_1": "456 Oak Avenue",
			"locality":      "Los Angeles",
			"state":         "CA",
			"postal_code":   "90001",
			"country":       "US",
		},
		"parcels": []map[string]any{
			{"weight": 5.0, "length": 12.0, "width": 8.0, "height": 6.0},
		},
	}
}

func TestEstimateRate_ReturnsWrappedResultForANonShippoCarrier(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(estimateRatePath, estimateRateRequestBody(), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	// The rate comes back in an object envelope the FE caller unwraps, not as a bare number.
	assert.Equal(t, "estimate_rate_result", jsonField(got, "object"))
	require.Contains(t, got, "rate")
	assert.Equal(t, 0.0, got["rate"], "a carrier with no Shippo account quotes 0 rather than failing")
}

func TestEstimateRate_RejectsMissingRequiredFields(t *testing.T) {
	t.Parallel()

	for _, field := range []string{"carrier_id", "service_level_id", "to_address", "from_address", "parcels"} {
		t.Run(field, func(t *testing.T) {
			t.Parallel()
			body := estimateRateRequestBody()
			delete(body, field)

			status, resp, err := apiClient.Post(estimateRatePath, body, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 400, status, resp)
		})
	}
}

func TestEstimateRate_RejectsEmptyParcelList(t *testing.T) {
	t.Parallel()

	body := estimateRateRequestBody()
	body["parcels"] = []map[string]any{}

	status, resp, err := apiClient.Post(estimateRatePath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, resp)
}

func TestEstimateRate_CustomerActorCannotRateForAnotherCustomer(t *testing.T) {
	t.Parallel()

	customer := apiClient.WithBearerToken(SeedCustomerAPIKey, SeedAccountID)

	body := estimateRateRequestBody()
	// The seller's own account id is not the calling customer, so the rate must be refused
	// rather than quoted against someone else's freight policy.
	body["customer_id"] = SeedAccountID

	status, resp, err := customer.Post(estimateRatePath, body, newIdempotencyKey())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, status, 400, "a customer actor must not rate for another customer; got %d: %s", status, resp)
}
