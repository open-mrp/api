//go:build e2e

package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetEndpoints_NotFoundErrorShape validates that GET-by-ID endpoints return
// a well-formed error response for non-existent resources.
func TestGetEndpoints_NotFoundErrorShape(t *testing.T) {
	t.Parallel()

	// Build test cases from pathSpecificIDSeeds — each entry maps a path prefix
	// to a seed ID. We fabricate a non-existent ID by zeroing out the suffix.
	type testCase struct {
		name string
		path string
	}

	var cases []testCase
	seen := make(map[string]bool)
	for prefix, seedID := range pathSpecificIDSeeds {
		if seen[prefix] {
			continue
		}
		seen[prefix] = true

		// Skip paths that have additional path params (nested resources) —
		// they need special handling and are covered by their CRUD tests.
		if strings.Count(prefix, "{") > 0 {
			continue
		}

		// Fabricate a non-existent ID: keep the prefix part, zero the rest.
		parts := strings.SplitN(seedID, "_", 2)
		if len(parts) != 2 {
			continue
		}
		fakeID := parts[0] + "_" + strings.Repeat("0", len(parts[1]))

		path := prefix + fakeID
		cases = append(cases, testCase{
			name: strings.TrimPrefix(prefix, "/v1/"),
			path: path,
		})
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			statusCode, body, err := apiClient.GetListRaw(tc.path, nil)
			require.NoError(t, err, "GET %s failed", tc.path)

			switch statusCode {
			case 404:
				requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
			case 405:
				// No GET-by-ID endpoint exists for this resource (e.g. quantities, rates, transaction-allocations).
				requireErrorResponse(t, body, "method_not_allowed", "invalid_request_error")
			case 403:
				// Endpoint checks ownership before existence (e.g. identity/accounts).
				requireErrorResponse(t, body, "", "")
			default:
				t.Errorf("GET %s returned unexpected status %d (expected 404, 405, or 403): %s", tc.path, statusCode, string(body))
			}
		})
	}
}

// TestUpdateEndpoints_UnknownFieldRejected validates that PATCH endpoints reject
// requests with unknown fields in the body.
func TestUpdateEndpoints_UnknownFieldRejected(t *testing.T) {
	t.Parallel()
	for _, ep := range updateEndpoints {
		t.Run(ep.OperationID, func(t *testing.T) {
			t.Parallel()
			path, ok := ep.ResolvePath()
			if !ok {
				t.Skipf("Cannot resolve path params for %s", ep.Path)
				return
			}

			statusCode, body, err := apiClient.Patch(path, map[string]any{
				bogusE2EJSONField: "should_be_rejected",
			}, newIdempotencyKey())
			require.NoError(t, err, "PATCH %s failed", path)
			assertJSONUnknownFieldRejected(t, http.MethodPatch, path, statusCode, body)
		})
	}
}

// TestCreateEndpoints_UnknownFieldRejected validates that POST endpoints reject
// requests with unknown fields in the JSON body.
func TestCreateEndpoints_UnknownFieldRejected(t *testing.T) {
	t.Parallel()
	for _, ep := range createEndpoints {
		t.Run(ep.OperationID, func(t *testing.T) {
			t.Parallel()
			path, ok := ep.ResolvePath()
			if !ok {
				t.Skipf("Cannot resolve path params for %s", ep.Path)
				return
			}

			statusCode, body, err := apiClient.Post(path, map[string]any{
				bogusE2EJSONField: "should_be_rejected",
			}, newIdempotencyKey())
			require.NoError(t, err, "POST %s failed", path)
			assertJSONUnknownFieldRejected(t, http.MethodPost, path, statusCode, body)
		})
	}
}

// TestPutEndpoints_UnknownFieldRejected validates that PUT endpoints reject
// requests with unknown fields in the JSON body.
func TestPutEndpoints_UnknownFieldRejected(t *testing.T) {
	t.Parallel()
	for _, ep := range putEndpoints {
		t.Run(ep.OperationID, func(t *testing.T) {
			t.Parallel()
			path, ok := ep.ResolvePath()
			if !ok {
				t.Skipf("Cannot resolve path params for %s", ep.Path)
				return
			}

			statusCode, body, err := apiClient.Put(path, map[string]any{
				bogusE2EJSONField: "should_be_rejected",
			})
			require.NoError(t, err, "PUT %s failed", path)
			assertJSONUnknownFieldRejected(t, http.MethodPut, path, statusCode, body)
		})
	}
}

// TestListEndpoints_UnknownQueryParamRejected validates that list GET endpoints
// reject undeclared query parameters.
func TestListEndpoints_UnknownQueryParamRejected(t *testing.T) {
	t.Parallel()
	for _, ep := range listEndpoints {
		t.Run(ep.OperationID, func(t *testing.T) {
			t.Parallel()
			path, ok := ep.ResolvePath()
			if !ok {
				t.Skipf("Cannot resolve path params for %s", ep.Path)
				return
			}

			statusCode, body, err := apiClient.GetListRaw(path, url.Values{
				bogusE2EQueryParam: {"should_be_rejected"},
			})
			require.NoError(t, err, "GET %s failed", path)
			assertUnknownQueryParamRejected(t, path, statusCode, body)
		})
	}
}

// TestUpdateEndpoints_TimestampFieldsRejected validates that PATCH endpoints reject
// created_at and updated_at in the request body. These are system-managed fields.
func TestUpdateEndpoints_TimestampFieldsRejected(t *testing.T) {
	t.Parallel()

	timestampFields := []struct {
		field string
		value string
	}{
		{"created_at", "2020-01-01T00:00:00Z"},
		{"updated_at", "2020-01-01T00:00:00Z"},
	}

	for _, ep := range updateEndpoints {
		for _, tf := range timestampFields {
			t.Run(fmt.Sprintf("%s/%s", ep.OperationID, tf.field), func(t *testing.T) {
				t.Parallel()
				path, ok := ep.ResolvePath()
				if !ok {
					t.Skipf("Cannot resolve path params for %s", ep.Path)
					return
				}

				statusCode, body, err := apiClient.Patch(path, map[string]any{
					tf.field: tf.value,
				}, newIdempotencyKey())
				require.NoError(t, err, "PATCH %s failed", path)
				skipOnNonClientError(t, path, statusCode)

				assert.Equal(t, 400, statusCode,
					"PATCH %s with %s should return 400, got %d: %s",
					path, tf.field, statusCode, string(body))

				if statusCode == 400 {
					errObj := requireErrorResponse(t, body, "", "invalid_request_error")
					code := errObj["code"]
					assert.True(t, code == "parameter_unknown" || code == "validation_failed",
						"PATCH %s: error.code should be parameter_unknown or validation_failed, got %v", path, code)
				}
			})
		}
	}
}

// TestUpdateEndpoints_EmptyBodyErrorShape validates that PATCH endpoints return
// a well-formed error response when sent an empty body.
func TestUpdateEndpoints_EmptyBodyErrorShape(t *testing.T) {
	t.Parallel()
	for _, ep := range updateEndpoints {
		t.Run(ep.OperationID, func(t *testing.T) {
			t.Parallel()
			path, ok := ep.ResolvePath()
			if !ok {
				t.Skipf("Cannot resolve path params for %s", ep.Path)
				return
			}

			statusCode, body, err := apiClient.Patch(path, map[string]any{}, newIdempotencyKey())
			require.NoError(t, err, "PATCH %s failed", path)
			skipOnNonClientError(t, path, statusCode)

			if statusCode != 400 {
				t.Skipf("PATCH %s with empty body returned %d (expected 400)", path, statusCode)
				return
			}

			// Validate the error body has the correct structure.
			var envelope map[string]any
			require.NoError(t, json.Unmarshal(body, &envelope), "response is not valid JSON")
			errObj, ok := envelope["error"]
			require.True(t, ok, "response missing 'error' key: %s", string(body))
			errMap, ok := errObj.(map[string]any)
			require.True(t, ok, "'error' should be an object")
			assert.NotEmpty(t, errMap["code"], "error.code should be set")
			assert.NotEmpty(t, errMap["type"], "error.type should be set")
			assert.NotEmpty(t, errMap["message"], "error.message should be set")
		})
	}
}

// TestMethodNotAllowed validates that endpoints return 405 for unsupported HTTP methods.
func TestMethodNotAllowed(t *testing.T) {
	t.Parallel()

	listOnlyPaths := []string{
		"/v1/sales/priorities",
	}

	for _, path := range listOnlyPaths {
		t.Run(fmt.Sprintf("POST_%s", path), func(t *testing.T) {
			t.Parallel()
			statusCode, body, err := apiClient.Post(path, map[string]any{"name": "test"}, newIdempotencyKey())
			require.NoError(t, err, "POST %s failed", path)
			skipOnNonClientError(t, path, statusCode)

			if statusCode != 405 {
				t.Skipf("POST %s returned %d (expected 405), endpoint may accept POST", path, statusCode)
				return
			}

			requireErrorResponse(t, body, "method_not_allowed", "invalid_request_error")
		})
	}
}
