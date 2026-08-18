//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Idempotency contract for the discount resources (docs/patterns/important-patterns.md):
// POST and PATCH must respect a caller-supplied Idempotency-Key; DELETE must be idempotent
// by default without one.
//
// Two behaviours make up "respects the key", and each test covers both. Replaying a key with
// the SAME body returns the first response without re-applying the write. Replaying it with a
// DIFFERENT body is refused outright by the gateway's idempotency middleware, which
// fingerprints the request — so a key can never be silently reused for a different mutation.

func TestAccountPrices_UpdateIsIdempotent(t *testing.T) {
	t.Parallel()
	created := createAccountPrice(t, SeedCustomerAccountID, "20.00")
	priceID := jsonField(created, "id")

	key := newIdempotencyKey()
	rate := func(value string) map[string]any {
		return map[string]any{
			"value":               value,
			"numerator_unit_id":   e2eCurrencyUnitID,
			"denominator_unit_id": SeedUnitID,
		}
	}

	status1, body1, err := apiClient.Patch(accountPricesPath+"/"+priceID,
		map[string]any{"rate": rate("21.00")}, key)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)
	assertDecimalEquals(t, 21.00, jsonField(jsonObject(parseJSON(body1), "rate"), "value"), "first PATCH rate")

	// Same key, same body: the cached response comes back and nothing is re-applied.
	status2, body2, err := apiClient.Patch(accountPricesPath+"/"+priceID,
		map[string]any{"rate": rate("21.00")}, key)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)
	assertDecimalEquals(t, 21.00, jsonField(jsonObject(parseJSON(body2), "rate"), "value"),
		"replaying the key returns the first response")

	// Same key, different body: refused rather than silently applied or silently ignored.
	status3, body3, err := apiClient.Patch(accountPricesPath+"/"+priceID,
		map[string]any{"rate": rate("99.00")}, key)
	require.NoError(t, err)
	requireStatus(t, 400, status3, body3)
	requireErrorResponse(t, body3, "validation_failed", "idempotency_error")

	persisted := parseJSON(mustGet(t, accountPricesPath+"/"+priceID))
	assertDecimalEquals(t, 21.00, jsonField(jsonObject(persisted, "rate"), "value"),
		"the rejected replay must not have changed the price")
}

func TestVolumeDiscounts_UpdateIsIdempotent(t *testing.T) {
	t.Parallel()
	created := createVolumeDiscount(t, map[string]any{})
	discountID := jsonField(created, "id")

	key := newIdempotencyKey()
	firstName := uniqueName("e2e-quds-idem-1")
	secondName := uniqueName("e2e-quds-idem-2")

	status1, body1, err := apiClient.Patch(volumeDiscountsPath+"/"+discountID,
		map[string]any{"name": firstName}, key)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)
	assert.Equal(t, firstName, jsonField(parseJSON(body1), "name"))

	status2, body2, err := apiClient.Patch(volumeDiscountsPath+"/"+discountID,
		map[string]any{"name": firstName}, key)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)
	assert.Equal(t, firstName, jsonField(parseJSON(body2), "name"),
		"replaying the key returns the first response")

	status3, body3, err := apiClient.Patch(volumeDiscountsPath+"/"+discountID,
		map[string]any{"name": secondName}, key)
	require.NoError(t, err)
	requireStatus(t, 400, status3, body3)
	requireErrorResponse(t, body3, "validation_failed", "idempotency_error")

	persisted := parseJSON(mustGet(t, volumeDiscountsPath+"/"+discountID))
	assert.Equal(t, firstName, jsonField(persisted, "name"),
		"the rejected replay must not have renamed the discount")
}

func TestOrderDiscounts_UpdateIsIdempotent(t *testing.T) {
	t.Parallel()
	created := createOrderDiscount(t, map[string]any{})
	discountID := jsonField(created, "id")

	key := newIdempotencyKey()
	firstName := uniqueName("e2e-ords-idem-1")
	secondName := uniqueName("e2e-ords-idem-2")

	status1, body1, err := apiClient.Patch(orderDiscountsPath+"/"+discountID,
		map[string]any{"name": firstName}, key)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)
	assert.Equal(t, firstName, jsonField(parseJSON(body1), "name"))

	status2, body2, err := apiClient.Patch(orderDiscountsPath+"/"+discountID,
		map[string]any{"name": firstName}, key)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)
	assert.Equal(t, firstName, jsonField(parseJSON(body2), "name"),
		"replaying the key returns the first response")

	status3, body3, err := apiClient.Patch(orderDiscountsPath+"/"+discountID,
		map[string]any{"name": secondName}, key)
	require.NoError(t, err)
	requireStatus(t, 400, status3, body3)
	requireErrorResponse(t, body3, "validation_failed", "idempotency_error")

	persisted := parseJSON(mustGet(t, orderDiscountsPath+"/"+discountID))
	assert.Equal(t, firstName, jsonField(persisted, "name"),
		"the rejected replay must not have renamed the discount")
}

// DELETE carries no key, so repeating it must be safe on its own: the resource stays gone
// and the repeat answers with a stable 410 from the deleted-record tombstone rather than a
// 5xx or a second round of side effects.
func TestDiscounts_DeleteIsIdempotentWithoutKey(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		path   string
		create func(t *testing.T) string
	}{
		{
			name: "account price",
			path: accountPricesPath,
			create: func(t *testing.T) string {
				return jsonField(createAccountPrice(t, SeedCustomerAccountID, "31.00"), "id")
			},
		},
		{
			name: "volume discount",
			path: volumeDiscountsPath,
			create: func(t *testing.T) string {
				return jsonField(createVolumeDiscount(t, map[string]any{}), "id")
			},
		},
		{
			name: "order discount",
			path: orderDiscountsPath,
			create: func(t *testing.T) string {
				return jsonField(createOrderDiscount(t, map[string]any{}), "id")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resourceID := tc.create(t)

			status, body, err := apiClient.Delete(tc.path + "/" + resourceID)
			require.NoError(t, err)
			requireStatus(t, 200, status, body)

			repeatStatus, repeatBody, err := apiClient.Delete(tc.path + "/" + resourceID)
			require.NoError(t, err)
			requireClientError(t, repeatStatus, repeatBody)
			assert.Equal(t, 410, repeatStatus,
				"a repeat delete should report the tombstone, not a generic 404 or a 5xx")
			requireErrorResponse(t, repeatBody, "resource_gone", "invalid_request_error")

			getStatus, getBody, err := apiClient.GetListRaw(tc.path+"/"+resourceID, nil)
			require.NoError(t, err)
			requireClientError(t, getStatus, getBody)
		})
	}
}

// The export enqueues a background job, so a retried request must not spawn a second one.
func TestAccountPrices_ExportPriceListIsIdempotent(t *testing.T) {
	t.Parallel()

	key := newIdempotencyKey()
	body := map[string]any{"customer_id": SeedCustomerAccountID}

	status1, respBody1, err := apiClient.Post(accountPricesPath+"/actions/export-price-list", body, key)
	require.NoError(t, err)
	requireStatus(t, 202, status1, respBody1)
	firstJobID := jsonField(parseJSON(respBody1), "id")
	require.NotEmpty(t, firstJobID)

	status2, respBody2, err := apiClient.Post(accountPricesPath+"/actions/export-price-list", body, key)
	require.NoError(t, err)
	requireStatus(t, 202, status2, respBody2)
	assert.Equal(t, firstJobID, jsonField(parseJSON(respBody2), "id"),
		"replaying the key must return the same job rather than starting a second export")
}
