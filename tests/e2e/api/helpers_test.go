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

// e2eAsyncWaitTimeout and e2eAsyncPollInterval tune polling for side-effects that flow
// through outbox/RabbitMQ (audit events, request logs, etc.). Tests poll until the
// condition holds or the timeout elapses, then fail with the last error — there is no
// unconditional long sleep when the backend is already caught up.
const (
	e2eAsyncWaitTimeout  = 15 * time.Second
	e2eAsyncPollInterval = 200 * time.Millisecond
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

// assertSearchRankOrder asserts each expected SKU appears in list order (by first occurrence
// index in list.Data). skuFromRow should return the comparable SKU for each row (top-level
// or nested under include=item).
func assertSearchRankOrder(t *testing.T, list []json.RawMessage, expectedSKUs []string, skuFromRow func(map[string]any) string) {
	t.Helper()
	indexBySKU := make(map[string]int)
	for i, raw := range list {
		var row map[string]any
		require.NoError(t, json.Unmarshal(raw, &row))
		sku := skuFromRow(row)
		if sku == "" {
			continue
		}
		if _, ok := indexBySKU[sku]; !ok {
			indexBySKU[sku] = i
		}
	}
	var prevIdx int
	var havePrev bool
	var prevSKU string
	for _, want := range expectedSKUs {
		idx, ok := indexBySKU[want]
		require.True(t, ok, "expected SKU %q in list (checked %d rows):\n%s", want, len(list), formatListDataForLog(list))
		if havePrev {
			assert.Less(t, prevIdx, idx, "SKU %q should sort before %q", prevSKU, want)
		}
		prevIdx = idx
		prevSKU = want
		havePrev = true
	}
}

// requirePageLen checks that a pagination page has the expected number of items.
// If the page is unexpectedly empty it skips the test rather than failing, since
// parallel CRUD tests can interfere with cursor pagination results.
func requirePageLen(t *testing.T, data []json.RawMessage, expected int) {
	t.Helper()
	if len(data) == 0 && expected > 0 {
		t.Fatal("Pagination page returned empty; likely parallel test interference")
	}
	require.Len(t, data, expected)
}

// requireStatus asserts the HTTP status code matches and includes the body in the error message.
func requireStatus(t *testing.T, expected, actual int, body []byte) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected status %d, got %d: %s", expected, actual, string(body))
	}
}

const bogusE2EJSONField = "__bogus_e2e_field__"
const bogusE2EQueryParam = "__bogus_e2e_query__"

// assertJSONUnknownFieldRejected asserts a JSON body with only bogusE2EJSONField was rejected with 400.
func assertJSONUnknownFieldRejected(t *testing.T, method, path string, statusCode int, body []byte) {
	t.Helper()
	skipOnNonClientError(t, path, statusCode)

	assert.Equal(t, 400, statusCode,
		"%s %s with unknown field should return 400, got %d: %s",
		method, path, statusCode, string(body))

	if statusCode != 400 {
		return
	}

	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	code := errObj["code"]
	assert.True(t, code == "parameter_unknown" || code == "validation_failed",
		"%s %s: error.code should be parameter_unknown or validation_failed, got %v", method, path, code)
	if code == "parameter_unknown" {
		assert.Equal(t, bogusE2EJSONField, errObj["param"],
			"%s %s: error.param should name the unknown field", method, path)
	}
}

// assertUnknownQueryParamRejected asserts an undeclared query parameter was rejected with 400.
func assertUnknownQueryParamRejected(t *testing.T, path string, statusCode int, body []byte) {
	t.Helper()
	skipOnNonClientError(t, path, statusCode)

	assert.Equal(t, 400, statusCode,
		"GET %s with unknown query param should return 400, got %d: %s",
		path, statusCode, string(body))

	if statusCode != 400 {
		return
	}

	errObj := requireErrorResponse(t, body, "parameter_unknown", "invalid_request_error")
	assert.Equal(t, bogusE2EQueryParam, errObj["param"],
		"GET %s: error.param should name the unknown query parameter", path)
}

// skipOnNonClientError skips the test if the endpoint returned 401/403,
// which means it requires a different auth method (e.g. cookie auth).
func skipOnNonClientError(t *testing.T, path string, statusCode int) {
	t.Helper()
	if statusCode == 401 || statusCode == 403 {
		t.Fatalf("Endpoint %s requires different auth (status %d)", path, statusCode)
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

// e2eCurrencyUnitID is the global currency base unit from shared/db/seed/0005_measures.sql.
const e2eCurrencyUnitID = "dollar"

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

// minimalSalesOrderCreateBody returns a minimal payload for POST /v1/sales/sales-orders (internal e2e).
// The buyer must have product line access for the product's line (grant via POST .../product-line-access/customers first).
func minimalSalesOrderCreateBody(buyerAccountID string) map[string]any {
	return map[string]any{
		"buyer_account_id":      buyerAccountID,
		"carrier_id":            SeedCarrierID,
		"service_level_id":      SeedServiceLevelID,
		"priority_code":         "normal",
		"sales_order_type_code": "sales_order",
		"payment_term_id":       SeedPaymentTermID,
		"shipping_term_id":      SeedShippingTermID,
		"bill_to_name":          "E2E Bill-To",
		"bill_to_street_line_1": "456 Test Ave",
		"bill_to_locality":      "Denver",
		"bill_to_state":         "CO",
		"bill_to_postal_code":   "80202",
		"bill_to_country":       "US",
		"ship_to_name":          "E2E Ship-To",
		"ship_to_street_line_1": "123 Test St",
		"ship_to_locality":      "Los Angeles",
		"ship_to_state":         "CA",
		"ship_to_postal_code":   "90001",
		"ship_to_country":       "US",
		"lines": []map[string]any{
			{
				"product_id":                     SeedProductID,
				"product_sku":                    SeedItemSKU,
				"quantity_value":                 "1",
				"quantity_unit_id":               SeedUnitID,
				"unit_price_value":               "10.00",
				"unit_price_numerator_unit_id":   e2eCurrencyUnitID,
				"unit_price_denominator_unit_id": SeedUnitID,
			},
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

// eventually polls fn until it returns nil or timeout elapses. Each failure sleeps
// interval before the next attempt. On deadline expiry it fails the test with the
// last error from fn (use e2eAsyncWaitTimeout / e2eAsyncPollInterval for async pipelines).
// Typical usage:
//
//	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
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
