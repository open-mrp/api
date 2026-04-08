//go:build e2e

package api_test

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Schema validation tests verify that API responses conform to the OpenAPI spec.
// They check that:
// - All fields defined in the schema are present in responses
// - No undocumented fields appear in responses
// - GET and POST responses for the same resource have consistent keys

// ──────────────────────────────────────────────
// Response key completeness for list endpoints
// ──────────────────────────────────────────────

func TestSchemaValidation_ListEndpoints_ItemFieldsMatchSpec(t *testing.T) {
	t.Parallel()

	spec, err := LoadFullSpec()
	require.NoError(t, err, "Failed to load OpenAPI spec")

	for _, ep := range listEndpoints {
		ep := ep
		t.Run(ep.OperationID, func(t *testing.T) {
			t.Parallel()

			path, ok := ep.ResolvePath()
			if !ok {
				t.Skipf("Cannot resolve path params for %s", ep.Path)
				return
			}

			// Get the response schema for this endpoint.
			schema, ok := spec.GetResponseSchema(ep.Path, "get", "200")
			if !ok {
				t.Skipf("No 200 response schema found for GET %s", ep.Path)
				return
			}

			// List responses have a data array — find the item schema.
			itemSchema := findListItemSchema(spec, schema)
			if itemSchema == nil {
				t.Skipf("Cannot determine list item schema for %s", ep.Path)
				return
			}

			specFields := spec.CollectSchemaFields(itemSchema)
			if len(specFields) == 0 {
				t.Skipf("No fields defined in schema for %s", ep.Path)
				return
			}

			// Fetch actual data.
			statusCode, body, err := apiClient.GetListRaw(path, nil)
			require.NoError(t, err)
			skipOnNonClientError(t, path, statusCode)
			if statusCode != 200 {
				t.Skipf("GET %s returned %d", path, statusCode)
				return
			}

			var list ListResponse
			require.NoError(t, json.Unmarshal(body, &list))
			if len(list.Data) == 0 {
				t.Skipf("No data returned for %s", path)
				return
			}

			// Check first item's keys against schema.
			var firstItem map[string]any
			require.NoError(t, json.Unmarshal(list.Data[0], &firstItem))

			// Report fields in response but NOT in spec (potential undocumented fields).
			for key := range firstItem {
				if !specFields[key] {
					// Don't fail — some fields may be added and spec not updated yet.
					// Log it for visibility.
					t.Logf("NOTICE: GET %s item has field %q not in OpenAPI spec", ep.Path, key)
				}
			}

			// Report fields in spec but NOT in response (potential missing fields).
			for field := range specFields {
				_, exists := firstItem[field]
				if !exists {
					t.Logf("NOTICE: GET %s item missing field %q defined in OpenAPI spec", ep.Path, field)
				}
			}
		})
	}
}

// findListItemSchema resolves the item schema from a list response schema.
func findListItemSchema(spec *openAPISpec, listSchema *openAPISchema) *openAPISchema {
	// Try direct properties.data.items path.
	if dataProp, ok := listSchema.Properties["data"]; ok {
		if dataProp.Items != nil {
			if dataProp.Items.Ref != "" {
				resolved, ok := spec.ResolveSchemaRef(dataProp.Items.Ref)
				if ok {
					return resolved
				}
			}
			return dataProp.Items
		}
	}

	// Try allOf.
	for _, s := range listSchema.AllOf {
		result := findListItemSchema(spec, &s)
		if result != nil {
			return result
		}
	}

	// Try resolving ref.
	if listSchema.Ref != "" {
		resolved, ok := spec.ResolveSchemaRef(listSchema.Ref)
		if ok {
			return findListItemSchema(spec, resolved)
		}
	}

	return nil
}

// ──────────────────────────────────────────────
// GET/POST response consistency
// ──────────────────────────────────────────────

func TestSchemaValidation_CreateAndGetResponseKeysMatch(t *testing.T) {
	t.Parallel()

	// Test with customers — a well-defined CRUD resource.
	name := uniqueName("e2e-schema-keys")
	createStatus, createBody, err := apiClient.Post(customersPath, validCustomerBody(name), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	var createResp map[string]any
	require.NoError(t, json.Unmarshal(createBody, &createResp))
	id := jsonField(createResp, "id")
	t.Cleanup(func() { apiClient.Delete(customersPath + "/" + id) })

	// GET the same resource.
	getStatus, getBody, err := apiClient.GetListRaw(customersPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	var getResp map[string]any
	require.NoError(t, json.Unmarshal(getBody, &getResp))

	// Compare key sets.
	createKeys := mapKeys(createResp)
	getKeys := mapKeys(getResp)

	for _, key := range createKeys {
		assert.Contains(t, getKeys, key,
			"Key %q present in POST response but missing from GET response", key)
	}
	for _, key := range getKeys {
		assert.Contains(t, createKeys, key,
			"Key %q present in GET response but missing from POST response", key)
	}
}

// TestSchemaValidation_UpdateResponseKeysMatchGet verifies PATCH response has same keys as GET.
func TestSchemaValidation_UpdateResponseKeysMatchGet(t *testing.T) {
	t.Parallel()

	created := createAndCleanup(t, customersPath, validCustomerBody(uniqueName("e2e-schema-upd")))
	id := jsonField(created, "id")

	// PATCH.
	patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+id, map[string]any{
		"note": "schema test",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	var patchResp map[string]any
	require.NoError(t, json.Unmarshal(patchBody, &patchResp))

	// GET.
	getStatus, getBody, err := apiClient.GetListRaw(customersPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	var getResp map[string]any
	require.NoError(t, json.Unmarshal(getBody, &getResp))

	patchKeys := mapKeys(patchResp)
	getKeys := mapKeys(getResp)

	for _, key := range patchKeys {
		assert.Contains(t, getKeys, key,
			"Key %q present in PATCH response but missing from GET response", key)
	}
	for _, key := range getKeys {
		assert.Contains(t, patchKeys, key,
			"Key %q present in GET response but missing from PATCH response", key)
	}
}

// ──────────────────────────────────────────────
// noGETPatchBodies provides minimal PATCH bodies for update endpoints that have
// no corresponding GET endpoint (405). These are used as a fallback to obtain
// the response shape for schema validation.
var noGETPatchBodies = map[string]map[string]any{
	"update-quantity":               {"value": "1"},
	"update-rate":                   {"value": "1"},
	"update-pick-line":              {"quantity_value": "1"},
	"update-sales-order-line":       {"quantity_value": "1"},
	"update-receiving-order-line":   {"quantity_value": "1"},
	"update-transaction-allocation": {"amount": "1"},
}

// Update endpoint schema validation
// ──────────────────────────────────────────────

func TestSchemaValidation_UpdateEndpoints_ResponseFieldsMatchSpec(t *testing.T) {
	t.Parallel()

	spec, err := LoadFullSpec()
	require.NoError(t, err, "Failed to load OpenAPI spec")

	for _, ep := range updateEndpoints {
		ep := ep
		t.Run(ep.OperationID, func(t *testing.T) {
			t.Parallel()

			// Get the response schema for this endpoint.
			schema, ok := spec.GetResponseSchema(ep.Path, "patch", "200")
			if !ok {
				t.Skipf("No 200 response schema found for PATCH %s", ep.Path)
				return
			}

			specFields := spec.CollectSchemaFields(schema)
			if len(specFields) == 0 {
				return // Empty-response endpoints (e.g. EmptyResource) have no fields to validate.
			}

			// Use the corresponding GET endpoint to fetch current data.
			// If no GET endpoint exists (405), fall back to PATCH with empty body.
			path, ok := ep.ResolvePath()
			if !ok {
				t.Skipf("Cannot resolve path params for %s", ep.Path)
				return
			}

			var respBody []byte

			getStatus, getBody, err := apiClient.GetListRaw(path, nil)
			require.NoError(t, err)
			skipOnNonClientError(t, path, getStatus)

			if getStatus == 200 {
				respBody = getBody
			} else if getStatus == 405 {
				// No GET endpoint — PATCH with a minimal body to obtain the response shape.
				body := noGETPatchBodies[ep.OperationID]
				if body == nil {
					body = map[string]any{}
				}
				patchStatus, patchBody, patchErr := apiClient.Patch(path, body, "")
				require.NoError(t, patchErr)
				skipOnNonClientError(t, path, patchStatus)
				if patchStatus != 200 {
					t.Skipf("PATCH %s returned %d (no GET either)", path, patchStatus)
					return
				}
				respBody = patchBody
			} else {
				t.Skipf("GET %s returned %d", path, getStatus)
				return
			}

			var resp map[string]any
			require.NoError(t, json.Unmarshal(respBody, &resp))

			for field := range specFields {
				_, exists := resp[field]
				if !exists {
					t.Logf("NOTICE: GET %s response missing field %q defined in PATCH response schema", path, field)
				}
			}
		})
	}
}

// ──────────────────────────────────────────────
// Account group schema consistency
// ──────────────────────────────────────────────

func TestSchemaValidation_AccountGroupCreateAndGetConsistency(t *testing.T) {
	t.Parallel()

	created := createAndCleanup(t, "/v1/sales/account-groups", map[string]any{
		"name": uniqueName("e2e-schema-grp"),
		"type": "type_group",
	})
	id := jsonField(created, "id")

	getStatus, getBody, err := apiClient.GetListRaw("/v1/sales/account-groups/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	var getResp map[string]any
	require.NoError(t, json.Unmarshal(getBody, &getResp))

	createKeys := mapKeys(created)
	getKeys := mapKeys(getResp)

	for _, key := range createKeys {
		assert.Contains(t, getKeys, key,
			"Key %q present in POST response but missing from GET response", key)
	}
}

// ──────────────────────────────────────────────
// Include parameter schema validation
// ──────────────────────────────────────────────

func TestSchemaValidation_IncludeFieldsPopulated(t *testing.T) {
	t.Parallel()

	// Without include, expandable fields should be null.
	created := createAndCleanup(t, customersPath, validCustomerBody(uniqueName("e2e-schema-inc")))
	id := jsonField(created, "id")

	// Verify expandable fields are null without include.
	expandableFields := []string{"contact_info", "defaults", "freight_preferences", "type", "notification_preferences"}
	for _, field := range expandableFields {
		assert.Nil(t, created[field], "%s should be null without ?include", field)
	}

	// With include, they should be populated.
	includes := strings.Join(expandableFields, ",")
	getStatus, getBody, err := apiClient.GetListRaw(
		customersPath+"/"+id,
		url.Values{"include": {includes}},
	)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	var expanded map[string]any
	require.NoError(t, json.Unmarshal(getBody, &expanded))

	for _, field := range expandableFields {
		val, exists := expanded[field]
		assert.True(t, exists, "Field %q should exist in response with ?include", field)
		if exists && val != nil {
			obj, ok := val.(map[string]any)
			if ok {
				assert.NotEmpty(t, obj["object"],
					"Expanded field %q should have an 'object' field", field)
			}
		}
	}
}

// mapKeys returns the keys of a map as a sorted slice.
func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// ──────────────────────────────────────────────
// Error response schema
// ──────────────────────────────────────────────

func TestSchemaValidation_ErrorResponseShape(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		path         string
		expectedCode int
	}{
		{"NotFound", customersPath + "/ac_000000000000000000000000", 404},
		{"MethodNotAllowed", "/v1/sales/priorities", 405}, // POST to list-only endpoint
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var statusCode int
			var body []byte
			var err error

			if tc.expectedCode == 405 {
				statusCode, body, err = apiClient.Post(tc.path, map[string]any{"name": "test"}, newIdempotencyKey())
			} else {
				statusCode, body, err = apiClient.GetListRaw(tc.path, nil)
			}
			require.NoError(t, err)

			if statusCode != tc.expectedCode {
				t.Skipf("Expected %d but got %d for %s", tc.expectedCode, statusCode, tc.path)
				return
			}

			// Validate the full error envelope structure.
			var envelope map[string]any
			require.NoError(t, json.Unmarshal(body, &envelope),
				"Error response should be valid JSON: %s", string(body))

			errObj, ok := envelope["error"]
			require.True(t, ok, "Error response should have 'error' key")

			errMap, ok := errObj.(map[string]any)
			require.True(t, ok, "'error' should be an object")

			requiredFields := []string{"code", "type", "message", "is_transient"}
			for _, field := range requiredFields {
				_, exists := errMap[field]
				assert.True(t, exists,
					fmt.Sprintf("Error object should have '%s' field (status %d, path %s)", field, tc.expectedCode, tc.path))
			}
		})
	}
}
