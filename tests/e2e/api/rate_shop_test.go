//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const rateShopPath = "/v1/operations/shipments/actions/rate-shop"

// seededExpressServiceLevelID is the second permanent service level on the
// seeded "delivery" carrier (see shared/db/seed/0003_accounts.sql).
// SeedServiceLevelID covers the "ground" option. Both are stable across runs —
// no test creates or deletes them — so they are safe to assert on even while
// cov_operations_carriers churns other service levels on the same carrier.
const seededExpressServiceLevelID = "crop_01seedexpress00000"

// rateShopRequestBody builds a minimal rate-shop request. The origin is omitted
// so core resolves the seller account's configured ship-from address.
func rateShopRequestBody() map[string]any {
	return map[string]any{
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

// TestRateShop_ReturnsFullyHydratedCarrierAndServiceLevel is the regression
// guard for the bug where the rate-shop response embedded thin carrier and
// service_level stubs (only id/object/name), leaving service_level_token,
// customer_portal_visibility, and the timestamps empty. Every option must now
// carry a fully-hydrated carrier and service level.
//
// Strict assertions are scoped to the two permanent seeded service levels on
// the non-Shippo "delivery" carrier (ground → fedex_ground, express →
// fedex_express), each rated at 0. Those rows are stable across parallel runs.
func TestRateShop_ReturnsFullyHydratedCarrierAndServiceLevel(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(rateShopPath, rateShopRequestBody(), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, "rate_shop_result", jsonField(got, "object"))

	options := jsonListData(got, "options")
	require.NotEmpty(t, options, "the seeded delivery carrier has service levels, so rate shopping must return options")

	expectedTokens := map[string]string{
		SeedServiceLevelID:          "fedex_ground",
		seededExpressServiceLevelID: "fedex_express",
	}
	seen := map[string]bool{}

	for i, raw := range options {
		opt, ok := raw.(map[string]any)
		require.True(t, ok, "options[%d] should be an object", i)
		assert.Equal(t, "rate_shop_option", jsonField(opt, "object"), "options[%d].object", i)

		carrier := jsonObject(opt, "carrier")
		require.NotNil(t, carrier, "options[%d].carrier must be present", i)
		serviceLevel := jsonObject(opt, "service_level")
		require.NotNil(t, serviceLevel, "options[%d].service_level must be present", i)

		slID := jsonField(serviceLevel, "id")
		wantToken, isSeeded := expectedTokens[slID]
		if !isSeeded {
			continue
		}
		seen[slID] = true

		// carrier — fully hydrated (previously only id/object/name were set).
		assert.Equal(t, SeedCarrierID, jsonField(carrier, "id"), "options[%d].carrier.id", i)
		assert.Equal(t, "carrier", jsonField(carrier, "object"), "options[%d].carrier.object", i)
		assert.Equal(t, "Delivery", jsonField(carrier, "name"), "options[%d].carrier.name", i)
		assert.NotEmpty(t, jsonField(carrier, "customer_portal_visibility"), "options[%d].carrier.customer_portal_visibility must not be empty", i)
		assertValidTimestamp(t, jsonField(carrier, "created_at"), "carrier.created_at")
		assertValidTimestamp(t, jsonField(carrier, "updated_at"), "carrier.updated_at")

		// service_level — fully hydrated; this is where the bug surfaced.
		assert.Equal(t, "service_level", jsonField(serviceLevel, "object"), "options[%d].service_level.object", i)
		assert.NotEmpty(t, jsonField(serviceLevel, "name"), "options[%d].service_level.name must not be empty", i)
		assert.Equal(t, wantToken, jsonField(serviceLevel, "service_level_token"), "options[%d].service_level.service_level_token", i)
		assert.NotEmpty(t, jsonField(serviceLevel, "customer_portal_visibility"), "options[%d].service_level.customer_portal_visibility must not be empty", i)
		assertValidTimestamp(t, jsonField(serviceLevel, "created_at"), "service_level.created_at")
		assertValidTimestamp(t, jsonField(serviceLevel, "updated_at"), "service_level.updated_at")
	}

	assert.True(t, seen[SeedServiceLevelID], "the seeded delivery/ground option must appear in rate-shop results")
	assert.True(t, seen[seededExpressServiceLevelID], "the seeded delivery/express option must appear in rate-shop results")
}

// TestRateShop_ValidationMissingToAddress asserts the endpoint rejects a
// request with no destination address.
func TestRateShop_ValidationMissingToAddress(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(rateShopPath, map[string]any{
		"parcels": []map[string]any{
			{"weight": 5.0, "length": 12.0, "width": 8.0, "height": 6.0},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"missing to_address should return 400 or 422, got %d: %s", status, string(body))
}
