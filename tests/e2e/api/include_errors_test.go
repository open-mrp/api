//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInclude_InvalidFieldRejected validates that requesting an unknown include
// field on a single resource returns a parameter_invalid error.
func TestInclude_InvalidFieldRejected(t *testing.T) {
	t.Parallel()
	path := "/v1/sales/customers/" + SeedCustomerAccountID

	statusCode, body, err := apiClient.GetListRaw(path, url.Values{
		"include": {"nonexistent_bogus_field"},
	})
	require.NoError(t, err)
	assert.Equal(t, 400, statusCode,
		"GET %s?include=nonexistent_bogus_field should return 400, got %d: %s",
		path, statusCode, string(body))

	if statusCode == 400 {
		requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	}
}

// TestInclude_InvalidFieldOnList validates that requesting an unknown include
// field on a list endpoint returns a parameter_invalid error.
func TestInclude_InvalidFieldOnList(t *testing.T) {
	t.Parallel()
	path := "/v1/sales/customers"

	statusCode, body, err := apiClient.GetListRaw(path, url.Values{
		"include": {"nonexistent_bogus_field"},
	})
	require.NoError(t, err)
	assert.Equal(t, 400, statusCode,
		"GET %s?include=nonexistent_bogus_field should return 400, got %d: %s",
		path, statusCode, string(body))

	if statusCode == 400 {
		requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	}
}
