//go:build e2e

package api_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const headersTestCustomerPath = "/v1/sales/customers"

// TestResponseHeaders_RequestID validates that every response includes a unique Request-Id header.
func TestResponseHeaders_RequestID(t *testing.T) {
	t.Parallel()
	path := headersTestCustomerPath + "/" + SeedCustomerAccountID

	resp1, err := apiClient.GetFull(path, nil)
	require.NoError(t, err)
	requireStatus(t, 200, resp1.StatusCode, resp1.Body)
	assertResponseHeaderPresent(t, resp1.Header, "Request-Id")

	resp2, err := apiClient.GetFull(path, nil)
	require.NoError(t, err)
	requireStatus(t, 200, resp2.StatusCode, resp2.Body)
	assertResponseHeaderPresent(t, resp2.Header, "Request-Id")

	assert.NotEqual(t, resp1.Header.Get("Request-Id"), resp2.Header.Get("Request-Id"),
		"two requests should have different Request-Id values")
}

// TestResponseHeaders_ContentType validates that JSON responses have the correct Content-Type.
func TestResponseHeaders_ContentType(t *testing.T) {
	t.Parallel()
	path := headersTestCustomerPath + "/" + SeedCustomerAccountID

	resp, err := apiClient.GetFull(path, nil)
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)

	ct := resp.Header.Get("Content-Type")
	assert.True(t, strings.HasPrefix(ct, "application/json"),
		"Content-Type should start with application/json, got %q", ct)
}

// TestResponseHeaders_AugnoVersion validates that the Augno-Version header is echoed back.
func TestResponseHeaders_AugnoVersion(t *testing.T) {
	t.Parallel()
	path := headersTestCustomerPath + "/" + SeedCustomerAccountID

	resp, err := apiClient.GetFull(path, nil)
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	assertResponseHeaderPresent(t, resp.Header, "Augno-Version")
}

// TestResponseHeaders_LocationOnCreate validates that 201 Created includes a
// Location header containing the new resource's id.
func TestResponseHeaders_LocationOnCreate(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-hdr-loc")

	resp, err := apiClient.PostFull(headersTestCustomerPath, validCustomerBody(name), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	id := jsonField(parseJSON(resp.Body), "id")
	assertCreatedLocation(t, resp.Header, id)

	apiClient.Delete(headersTestCustomerPath + "/" + id)
}

// TestResponseHeaders_IdempotentReplayed validates the Idempotent-Replayed header
// is set on the second request with the same idempotency key.
func TestResponseHeaders_IdempotentReplayed(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-hdr-idem")
	idemKey := newIdempotencyKey()
	payload := validCustomerBody(name)

	resp1, err := apiClient.PostFull(headersTestCustomerPath, payload, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, resp1.StatusCode, resp1.Body)
	assert.NotEqual(t, "true", resp1.Header.Get("Idempotent-Replayed"),
		"first request should not have Idempotent-Replayed header")

	resp2, err := apiClient.PostFull(headersTestCustomerPath, payload, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, resp2.StatusCode, resp2.Body)
	assert.Equal(t, "true", resp2.Header.Get("Idempotent-Replayed"),
		"second request with same key should have Idempotent-Replayed: true")

	id := jsonField(parseJSON(resp1.Body), "id")
	apiClient.Delete(headersTestCustomerPath + "/" + id)
}

// TestResponseHeaders_OnErrors validates that error responses also include standard headers.
func TestResponseHeaders_OnErrors(t *testing.T) {
	t.Parallel()
	path := headersTestCustomerPath + "/ac_000000000000000000000000"

	resp, err := apiClient.GetFull(path, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)

	assertResponseHeaderPresent(t, resp.Header, "Request-Id")

	ct := resp.Header.Get("Content-Type")
	assert.True(t, strings.HasPrefix(ct, "application/json"),
		"error responses should have Content-Type application/json, got %q", ct)
}
