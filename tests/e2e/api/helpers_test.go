//go:build e2e

package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// uniqueName generates a unique name for test resources.
func uniqueName(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, uuid.New().String()[:8])
}

// newIdempotencyKey generates a UUID v4 for idempotency testing.
func newIdempotencyKey() string {
	return uuid.New().String()
}

// parseJSON unmarshals raw JSON bytes into a generic map.
func parseJSON(body []byte) map[string]any {
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return nil
	}
	return m
}

// jsonField extracts a string field from a parsed JSON map. Returns "" for
// missing keys and for JSON null values so callers can distinguish "no value"
// from a real value with a simple emptiness check.
func jsonField(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return fmt.Sprintf("%v", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// jsonObject extracts a nested object from a parsed JSON map.
func jsonObject(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	v, ok := m[key]
	if !ok {
		return nil
	}
	obj, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	return obj
}

// formatListDataForLog renders list response data as readable JSON (testify prints []json.RawMessage as raw byte decimals).
func formatListDataForLog(data []json.RawMessage) string {
	if len(data) == 0 {
		return "(empty)"
	}
	var b strings.Builder
	for i, raw := range data {
		if i > 0 {
			b.WriteString("\n---\n")
		}
		var buf bytes.Buffer
		if err := json.Indent(&buf, raw, "", "  "); err != nil {
			b.WriteString(string(raw))
			continue
		}
		b.Write(buf.Bytes())
	}
	return b.String()
}

// assertEmptyListData asserts list data is empty; on failure prints each item as indented JSON.
func assertEmptyListData(t *testing.T, data []json.RawMessage, msgAndArgs ...interface{}) {
	t.Helper()
	if len(data) == 0 {
		return
	}
	var prefix string
	switch len(msgAndArgs) {
	case 0:
		prefix = "expected empty list data"
	default:
		if format, ok := msgAndArgs[0].(string); ok {
			prefix = fmt.Sprintf(format, msgAndArgs[1:]...)
		} else {
			prefix = fmt.Sprint(msgAndArgs...)
		}
	}
	assert.Fail(t, fmt.Sprintf("%s\n\ngot %d items:\n%s", prefix, len(data), formatListDataForLog(data)))
}

// requireStatus asserts the HTTP status code matches and includes the body in the error message.
func requireStatus(t *testing.T, expected, actual int, body []byte) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected status %d, got %d: %s", expected, actual, string(body))
	}
}

// skipOnNonClientError skips the test if the endpoint returned 401/403,
// which means it requires a different auth method (e.g. cookie auth).
func skipOnNonClientError(t *testing.T, path string, statusCode int) {
	t.Helper()
	if statusCode == 401 || statusCode == 403 {
		t.Skipf("Endpoint %s requires different auth (status %d)", path, statusCode)
	}
}

// requireErrorResponse asserts the body is a valid API error response and returns the inner error object.
// Validates: top-level "error" key exists and is an object, "code" matches expectedCode,
// "type" matches expectedType, "message" is a non-empty string, "is_transient" is present.
func requireErrorResponse(t *testing.T, body []byte, expectedCode, expectedType string) map[string]any {
	t.Helper()
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(body, &envelope), "response body is not valid JSON: %s", string(body))

	errObj, ok := envelope["error"]
	require.True(t, ok, "response missing 'error' key: %s", string(body))

	errMap, ok := errObj.(map[string]any)
	require.True(t, ok, "'error' should be an object: %s", string(body))

	if expectedCode != "" {
		assert.Equal(t, expectedCode, errMap["code"], "error.code mismatch (body: %s)", string(body))
	}
	if expectedType != "" {
		assert.Equal(t, expectedType, errMap["type"], "error.type mismatch (body: %s)", string(body))
	}

	msg, ok := errMap["message"].(string)
	assert.True(t, ok && msg != "", "error.message should be a non-empty string (body: %s)", string(body))

	_, hasTransient := errMap["is_transient"]
	assert.True(t, hasTransient, "error should have is_transient field (body: %s)", string(body))

	return errMap
}

// assertErrorParam asserts the error object has the expected param value.
func assertErrorParam(t *testing.T, errObj map[string]any, expectedParam string) {
	t.Helper()
	assert.Equal(t, expectedParam, errObj["param"], "error.param mismatch")
}

// assertResponseHeader asserts a response header equals the expected value.
func assertResponseHeader(t *testing.T, header http.Header, name, expected string) {
	t.Helper()
	assert.Equal(t, expected, header.Get(name), "header %s mismatch", name)
}

// assertResponseHeaderPresent asserts a response header is non-empty.
func assertResponseHeaderPresent(t *testing.T, header http.Header, name string) {
	t.Helper()
	assert.NotEmpty(t, header.Get(name), "header %s should be present and non-empty", name)
}

// assertCreatedLocation asserts that a 201 response includes a Location header
// pointing at the newly created resource. The header must be present and must
// contain the new resource's id (typically as the last path segment).
func assertCreatedLocation(t *testing.T, header http.Header, id string) {
	t.Helper()
	location := header.Get("Location")
	require.NotEmpty(t, location, "201 Created response must include a Location header")
	assert.Contains(t, location, id,
		"Location header %q should contain the new resource id %q", location, id)
}

// assertIDFormat asserts the ID starts with the given prefix followed by "_".
func assertIDFormat(t *testing.T, id, expectedPrefix string) {
	t.Helper()
	assert.True(t, strings.HasPrefix(id, expectedPrefix+"_"),
		"id %q should start with %q", id, expectedPrefix+"_")
}

// assertValidTimestamp asserts the value is a valid RFC3339 timestamp.
func assertValidTimestamp(t *testing.T, value, fieldName string) {
	t.Helper()
	require.NotEmpty(t, value, "%s should not be empty", fieldName)
	_, err := time.Parse(time.RFC3339, value)
	if err != nil {
		_, err = time.Parse(time.RFC3339Nano, value)
	}
	assert.NoError(t, err, "%s %q is not a valid RFC3339 timestamp", fieldName, value)
}

// assertObjectField asserts the parsed JSON map has the expected object field value.
func assertObjectField(t *testing.T, m map[string]any, expected string) {
	t.Helper()
	assert.Equal(t, expected, jsonField(m, "object"), "object field mismatch")
}

// assertNilField asserts a field in the parsed JSON map is nil (null in JSON or absent).
func assertNilField(t *testing.T, m map[string]any, field string) {
	t.Helper()
	assert.Nil(t, m[field], "%s should be null", field)
}

// validCustomerBody returns a map with all required fields for creating a customer.
// Tests can override individual fields by writing to the returned map before posting.
func validCustomerBody(name string) map[string]any {
	return map[string]any{
		"name":                     name,
		"status":                   "normal",
		"default_carrier_id":       SeedCarrierID,
		"default_payment_term_id":  SeedPaymentTermID,
		"default_shipping_term_id": SeedShippingTermID,
		"customer_type_group_id":   SeedCustomerGroupID,
		"bill_to_address": map[string]any{
			"name":    name + " Billing",
			"country": "US",
		},
		"ship_to_address": map[string]any{
			"name":    name + " Shipping",
			"country": "US",
		},
	}
}

// createAndCleanup creates a resource via POST and registers t.Cleanup to delete it.
// Returns the parsed response body. Fails the test if creation fails.
func createAndCleanup(t *testing.T, path string, body map[string]any) map[string]any {
	t.Helper()
	status, respBody, err := apiClient.Post(path, body, newIdempotencyKey())
	require.NoError(t, err, "POST %s failed", path)
	requireStatus(t, 201, status, respBody)
	parsed := parseJSON(respBody)
	id := jsonField(parsed, "id")
	require.NotEmpty(t, id, "created resource should have an id")
	t.Cleanup(func() { apiClient.Delete(path + "/" + id) })
	return parsed
}

// createAndCleanupRaw creates a resource via POST, registers t.Cleanup, and returns
// both the parsed body and the raw response bytes. Useful when you need the raw body
// for schema validation.
func createAndCleanupRaw(t *testing.T, path string, body map[string]any) (map[string]any, []byte) {
	t.Helper()
	status, respBody, err := apiClient.Post(path, body, newIdempotencyKey())
	require.NoError(t, err, "POST %s failed", path)
	requireStatus(t, 201, status, respBody)
	parsed := parseJSON(respBody)
	id := jsonField(parsed, "id")
	require.NotEmpty(t, id, "created resource should have an id")
	t.Cleanup(func() { apiClient.Delete(path + "/" + id) })
	return parsed, respBody
}

// createAPIKeyAndCleanup creates an API key and registers cleanup. Returns the
// parsed response (which has api_key_secret + api_key_info at the top level).
func createAPIKeyAndCleanup(t *testing.T, name string) map[string]any {
	t.Helper()
	status, body, err := apiClient.Post(apiKeysPath, map[string]any{
		"name":    name,
		"role_id": SeedAdminRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	parsed := parseJSON(body)
	info := jsonObject(parsed, "api_key_info")
	require.NotNil(t, info)
	id := jsonField(info, "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { apiClient.Delete(apiKeysPath + "/" + id) })
	return parsed
}

// eventually retries fn until it returns nil or the timeout expires.
// fn should perform an assertion and return an error describing the failure.
// Typical usage:
//
//	eventually(t, 10*time.Second, 500*time.Millisecond, func() error {
//	    status, body, err := apiClient.Get(auditEventsPath, url.Values{"resource_id": {id}})
//	    if err != nil { return err }
//	    list := parseJSON(body)
//	    data := jsonArray(list, "data")
//	    if len(data) == 0 { return fmt.Errorf("no audit events yet for %s", id) }
//	    return nil
//	})
func eventually(t *testing.T, timeout, interval time.Duration, fn func() error) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		if lastErr = fn(); lastErr == nil {
			return
		}
		time.Sleep(interval)
	}
	t.Fatalf("condition not met after %s: %v", timeout, lastErr)
}

// jsonArray extracts a JSON array from a parsed JSON map.
func jsonArray(m map[string]any, key string) []any {
	if m == nil {
		return nil
	}
	v, ok := m[key]
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	return arr
}

// jsonListData extracts the "data" array from a List[T] field ({"object":"list","data":[...]}).
func jsonListData(m map[string]any, key string) []any {
	obj := jsonObject(m, key)
	if obj == nil {
		return nil
	}
	return jsonArray(obj, "data")
}
