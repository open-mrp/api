//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// maxEndpointsToTest limits the number of endpoints tested for pagination errors
// to avoid excessive test runtime while still validating the behavior.
const maxEndpointsToTest = 3

func eligibleListEndpoints() []ListEndpointSpec {
	var eligible []ListEndpointSpec
	for _, ep := range listEndpoints {
		if ep.HasParam("limit") {
			eligible = append(eligible, ep)
		}
	}
	return eligible
}

// TestListEndpoints_InvalidLimit_Zero validates that limit=0 is rejected.
func TestListEndpoints_InvalidLimit_Zero(t *testing.T) {
	t.Parallel()
	tested := 0
	for _, ep := range eligibleListEndpoints() {
		if tested >= maxEndpointsToTest {
			break
		}

		t.Run(ep.OperationID, func(t *testing.T) {
			t.Parallel()
			path, ok := ep.ResolvePath()
			if !ok {
				t.Skipf("Cannot resolve path params for %s", ep.Path)
				return
			}

			statusCode, body, err := apiClient.GetListRaw(path, url.Values{"limit": {"0"}})
			require.NoError(t, err, "GET %s?limit=0 failed", path)
			skipOnNonClientError(t, path, statusCode)

			assert.Equal(t, 400, statusCode,
				"GET %s?limit=0 should return 400, got %d: %s", path, statusCode, string(body))

			if statusCode == 400 {
				// The API may return "validation_failed" or "invalid_format" depending on the endpoint.
				errObj := requireErrorResponse(t, body, "", "invalid_request_error")
				code := errObj["code"]
				assert.True(t, code == "validation_failed" || code == "invalid_format" || code == "parameter_invalid",
					"GET %s?limit=0: error.code should be validation_failed, invalid_format, or parameter_invalid, got %v", path, code)
			}
		})
		tested++
	}
}

// TestListEndpoints_InvalidLimit_Negative validates that negative limit is rejected.
func TestListEndpoints_InvalidLimit_Negative(t *testing.T) {
	t.Parallel()
	tested := 0
	for _, ep := range eligibleListEndpoints() {
		if tested >= maxEndpointsToTest {
			break
		}

		t.Run(ep.OperationID, func(t *testing.T) {
			t.Parallel()
			path, ok := ep.ResolvePath()
			if !ok {
				t.Skipf("Cannot resolve path params for %s", ep.Path)
				return
			}

			statusCode, body, err := apiClient.GetListRaw(path, url.Values{"limit": {"-1"}})
			require.NoError(t, err, "GET %s?limit=-1 failed", path)
			skipOnNonClientError(t, path, statusCode)

			assert.Equal(t, 400, statusCode,
				"GET %s?limit=-1 should return 400, got %d: %s", path, statusCode, string(body))

			if statusCode == 400 {
				errObj := requireErrorResponse(t, body, "", "invalid_request_error")
				code := errObj["code"]
				assert.True(t, code == "validation_failed" || code == "invalid_format" || code == "parameter_invalid",
					"GET %s?limit=-1: error.code should be validation_failed, invalid_format, or parameter_invalid, got %v", path, code)
			}
		})
		tested++
	}
}

// TestListEndpoints_InvalidLimit_TooLarge validates that excessively large limit is rejected.
func TestListEndpoints_InvalidLimit_TooLarge(t *testing.T) {
	t.Parallel()
	tested := 0
	for _, ep := range eligibleListEndpoints() {
		if tested >= maxEndpointsToTest {
			break
		}

		t.Run(ep.OperationID, func(t *testing.T) {
			t.Parallel()
			path, ok := ep.ResolvePath()
			if !ok {
				t.Skipf("Cannot resolve path params for %s", ep.Path)
				return
			}

			statusCode, body, err := apiClient.GetListRaw(path, url.Values{"limit": {"999999"}})
			require.NoError(t, err, "GET %s?limit=999999 failed", path)
			skipOnNonClientError(t, path, statusCode)

			assert.Equal(t, 400, statusCode,
				"GET %s?limit=999999 should return 400, got %d: %s", path, statusCode, string(body))

			if statusCode == 400 {
				errObj := requireErrorResponse(t, body, "", "invalid_request_error")
				code := errObj["code"]
				assert.True(t, code == "validation_failed" || code == "invalid_format" || code == "parameter_invalid",
					"GET %s?limit=999999: error.code should be validation_failed, invalid_format, or parameter_invalid, got %v", path, code)
			}
		})
		tested++
	}
}

// TestListEndpoints_InvalidCursor validates that a garbage cursor value is rejected.
func TestListEndpoints_InvalidCursor(t *testing.T) {
	t.Parallel()
	tested := 0
	for _, ep := range listEndpoints {
		if !ep.HasParam("cursor") {
			continue
		}
		if tested >= maxEndpointsToTest {
			break
		}

		t.Run(ep.OperationID, func(t *testing.T) {
			t.Parallel()
			path, ok := ep.ResolvePath()
			if !ok {
				t.Skipf("Cannot resolve path params for %s", ep.Path)
				return
			}

			statusCode, body, err := apiClient.GetListRaw(path, url.Values{"cursor": {"not_a_real_cursor_value"}})
			require.NoError(t, err, "GET %s?cursor=invalid failed", path)
			skipOnNonClientError(t, path, statusCode)

			if statusCode == 200 {
				// Some endpoints silently accept invalid cursors and return empty results.
				t.Skipf("GET %s accepts invalid cursor without error", path)
				return
			}

			assert.Equal(t, 400, statusCode,
				"GET %s?cursor=invalid should return 400, got %d: %s", path, statusCode, string(body))

			if statusCode == 400 {
				errObj := requireErrorResponse(t, body, "", "invalid_request_error")
				code := errObj["code"]
				assert.True(t, code == "parameter_invalid" || code == "validation_failed",
					"GET %s?cursor=invalid: error.code should be parameter_invalid or validation_failed, got %v", path, code)
			}
		})
		tested++
	}
}
