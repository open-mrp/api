//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const idempotencyTestCustomerPath = "/v1/sales/customers"

// TestIdempotency_CreateConflict validates that sending the same idempotency key
// with a different request body returns an error.
func TestIdempotency_CreateConflict(t *testing.T) {
	t.Parallel()
	idemKey := newIdempotencyKey()

	// First request succeeds.
	name1 := uniqueName("e2e-idem-a")
	status1, body1, err := apiClient.Post(idempotencyTestCustomerPath, map[string]any{
		"name": name1,
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id := jsonField(parseJSON(body1), "id")

	// Second request with same key but different body.
	name2 := uniqueName("e2e-idem-b")
	status2, body2, err := apiClient.Post(idempotencyTestCustomerPath, map[string]any{
		"name": name2,
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 400, status2, body2)

	requireErrorResponse(t, body2, "validation_failed", "idempotency_error")

	apiClient.Delete(idempotencyTestCustomerPath + "/" + id)
}

// TestIdempotency_UpdateConflict validates that sending the same idempotency key
// with a different PATCH body returns an error.
func TestIdempotency_UpdateConflict(t *testing.T) {
	t.Parallel()

	// Create a resource to update.
	name := uniqueName("e2e-idem-upd")
	createStatus, createBody, err := apiClient.Post(idempotencyTestCustomerPath, map[string]any{
		"name": name,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	path := idempotencyTestCustomerPath + "/" + id

	idemKey := newIdempotencyKey()

	// First PATCH succeeds.
	status1, body1, err := apiClient.Patch(path, map[string]any{
		"note": "first note",
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)

	// Second PATCH with same key but different body.
	status2, body2, err := apiClient.Patch(path, map[string]any{
		"note": "different note",
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 400, status2, body2)

	requireErrorResponse(t, body2, "validation_failed", "idempotency_error")

	apiClient.Delete(path)
}
