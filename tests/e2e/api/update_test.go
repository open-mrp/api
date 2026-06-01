//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpdateEndpoints_NullableClearFields verifies that every PATCH endpoint
// with x-nullable-clear fields supports three states: set a value (200),
// clear by sending null (200), and restore.
//
// Fields are discovered from the OpenAPI spec. Only fields with a known seed
// value are tested; the rest are covered by per-resource CRUD tests.
func TestUpdateEndpoints_NullableClearFields(t *testing.T) {
	for _, ep := range updateEndpoints {
		if len(ep.NullableClearFields) == 0 {
			continue
		}
		t.Run(ep.OperationID, func(t *testing.T) {
			path, ok := ep.ResolvePath()
			if !ok {
				t.Skipf("Cannot resolve path params for %s", ep.Path)
				return
			}

			for _, field := range ep.NullableClearFields {
				seedVal, hasSeed := nullableFieldSeeds[field]
				if !hasSeed {
					continue
				}

				t.Run("clear_"+field, func(t *testing.T) {
					getStatus, getBody, err := apiClient.GetListRaw(path, nil)
					require.NoError(t, err)
					if getStatus == 404 || getStatus == 401 || getStatus == 403 {
						t.Skipf("Seed resource not accessible for %s: %d", path, getStatus)
						return
					}
					requireStatus(t, 200, getStatus, getBody)
					originalValue := parseJSON(getBody)[field]

					// Phase 1: Set the field to a known value.
					setStatus, setBody, err := apiClient.Patch(path, map[string]any{
						field: seedVal,
					}, newIdempotencyKey())
					require.NoError(t, err)
					if setStatus == 404 || setStatus == 401 || setStatus == 403 {
						t.Skipf("Seed resource not accessible for %s: %d", path, setStatus)
						return
					}
					requireStatus(t, 200, setStatus, setBody)

					// Phase 2: Clear the field by sending null.
					clearStatus, clearBody, err := apiClient.Patch(path, map[string]any{
						field: nil,
					}, newIdempotencyKey())
					require.NoError(t, err)
					requireStatus(t, 200, clearStatus, clearBody)

					// Phase 3: Restore the original value so other tests aren't affected.
					restoreStatus, restoreBody, err := apiClient.Patch(path, map[string]any{
						field: originalValue,
					}, newIdempotencyKey())
					require.NoError(t, err)
					requireStatus(t, 200, restoreStatus, restoreBody)
				})
			}
		})
	}
}

func TestUpdateEndpoints_EmptyBodyRejected(t *testing.T) {
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

			assert.Equal(t, 400, statusCode,
				"PATCH %s with empty body should return 400, got %d: %s",
				path, statusCode, string(body))
		})
	}
}
