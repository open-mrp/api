//go:build e2e

package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/open-mrp/api/shared/id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// e2eAsyncWaitTimeout and e2eAsyncPollInterval tune polling for side-effects that flow
// through outbox/RabbitMQ (audit events, request logs, etc.). Tests poll until the
// condition holds or the timeout elapses, then fail with the last error — there is no
// unconditional long sleep when the backend is already caught up.
const (
	e2eAsyncWaitTimeout  = 15 * time.Second
	e2eAsyncPollInterval = 50 * time.Millisecond
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

// commitmentOf extracts the commitment sub-resource an order, a pick, or a quote carries its ship-by date and derivation on.
func commitmentOf(m map[string]any) map[string]any {
	return jsonObject(m, "commitment")
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

// requirePageLen checks that a pagination page has the expected number of
// items. Only use it on lists that parallel tests cannot shrink (system
// enums, append-only logs, single-fetch checks) — for mutable shared lists
// use assertScopedCursorPagination or assertCursorPaginationAdvances instead.
func requirePageLen(t *testing.T, data []json.RawMessage, expected int) {
	t.Helper()
	if len(data) == 0 && expected > 0 {
		t.Fatal("Pagination page returned empty; likely parallel test interference")
	}
	require.Len(t, data, expected)
}

// assertListPageLen fetches a single page of the given list and asserts it
// holds exactly `expected` rows, retrying a few times first. List endpoints
// fetch a page of ids and then batch-hydrate them, silently dropping ids that
// a parallel test deleted in between, so a limit=N page can come back short
// (even empty) under churn while a real pagination bug fails every attempt.
// Use this for limit=1 pagination smoke checks on mutable shared lists that
// always have at least one undeletable row (e.g. a seeded global row).
func assertListPageLen(t *testing.T, path string, params url.Values, expected int) {
	t.Helper()
	const attempts = 3

	for attempt := 1; ; attempt++ {
		list, _, err := apiClient.GetList(path, params)
		require.NoError(t, err, "listing %s", path)
		if len(list.Data) != expected && attempt < attempts {
			t.Logf("page of %s held %d rows, want %d, on attempt %d (likely parallel deletes); retrying",
				path, len(list.Data), expected, attempt)
			continue
		}
		require.Len(t, list.Data, expected,
			"page of %s should hold %d row(s) (after %d attempts)", path, expected, attempt)
		return
	}
}

// maxListScanPages bounds how many pages listFindByField walks (pages of
// 1000, so 25k rows) before declaring a row absent.
const maxListScanPages = 25

// listFindByField pages through a cursor-paginated list endpoint (following
// next_page_url with limit=1000, up to maxListScanPages pages) and returns
// the first item whose field equals value, or nil when no page contains it.
//
// List tests must never assume a row lands on the first page: repeated e2e
// runs against the same database accumulate rows, and seed rows — the oldest —
// are the first to fall off the newest-first front page.
func listFindByField(t *testing.T, path string, params url.Values, field, value string) json.RawMessage {
	t.Helper()

	merged := url.Values{"limit": {"1000"}}
	for k, vs := range params {
		merged[k] = vs
	}

	list, _, err := apiClient.GetList(path, merged)
	require.NoError(t, err, "listing %s", path)
	for page := 0; page < maxListScanPages; page++ {
		for _, item := range list.Data {
			if DataItemField(item, field) == value {
				return item
			}
		}
		if !list.PageInfo.HasNextPage || list.PageInfo.NextPageURL == nil {
			return nil
		}
		list, _, err = apiClient.GetListFromPageURL(list.PageInfo.NextPageURL)
		require.NoError(t, err, "paging %s", path)
	}
	return nil
}

// assertListContainsID asserts an item with the given id appears somewhere in
// the paginated list.
func assertListContainsID(t *testing.T, path string, params url.Values, id string) {
	t.Helper()
	assert.NotNil(t, listFindByField(t, path, params, "id", id),
		"item %q should appear in the %s list (scanned up to %d pages)", id, path, maxListScanPages)
}

// assertCursorPaginationAdvances fetches two consecutive limit=1 pages of a
// shared (globally mutable) list and asserts the cursor advanced to a
// different row. Parallel tests can delete rows mid-flight and leave either
// page legitimately empty: list endpoints fetch a page of ids and then batch-
// hydrate them, silently dropping ids deleted in between, so even page 1 can
// come back empty under churn. The sequence is therefore retried a few times:
// transient interference passes on a later attempt while a real pagination
// bug fails every attempt. Prefer assertScopedCursorPagination (test-owned
// rows) where the resource supports search-scoped listing.
func assertCursorPaginationAdvances(t *testing.T, path string, params url.Values) {
	t.Helper()
	const attempts = 3

	merged := url.Values{"limit": {"1"}}
	for k, vs := range params {
		merged[k] = vs
	}

	for attempt := 1; ; attempt++ {
		page1, _, err := apiClient.GetList(path, merged)
		require.NoError(t, err, "listing %s", path)
		if (len(page1.Data) != 1 || !page1.PageInfo.HasNextPage || page1.PageInfo.NextPageURL == nil) && attempt < attempts {
			t.Logf("page 1 of %s incomplete on attempt %d (likely parallel deletes); retrying", path, attempt)
			continue
		}
		require.Len(t, page1.Data, 1, "first page of %s should hold one row (after %d attempts)", path, attempt)
		require.True(t, page1.PageInfo.HasNextPage, "%s should have a next page", path)
		require.NotNil(t, page1.PageInfo.NextPageURL)

		page2, _, err := apiClient.GetListFromPageURL(page1.PageInfo.NextPageURL)
		require.NoError(t, err, "paging %s", path)

		if len(page2.Data) == 0 && attempt < attempts {
			t.Logf("page 2 of %s empty on attempt %d (likely parallel deletes); retrying", path, attempt)
			continue
		}
		require.Len(t, page2.Data, 1, "second page of %s should hold one row (after %d attempts)", path, attempt)

		assert.NotEqual(t,
			DataItemField(page1.Data[0], "id"), DataItemField(page2.Data[0], "id"),
			"consecutive pages of %s should return different items", path)
		return
	}
}

// assertScopedCursorPagination walks a list one row per page and asserts that
// exactly the given ids are reached, each exactly once — proving the cursor
// advances without duplicating or skipping rows. Callers scope the list to
// rows they own (a search param matching a unique prefix), which makes the
// walk immune to rows that parallel tests create or delete.
func assertScopedCursorPagination(t *testing.T, path string, params url.Values, wantIDs []string) {
	t.Helper()
	require.GreaterOrEqual(t, len(wantIDs), 2, "scoped pagination needs at least two rows")

	merged := url.Values{"limit": {"1"}}
	for k, vs := range params {
		merged[k] = vs
	}

	list, _, err := apiClient.GetList(path, merged)
	require.NoError(t, err, "listing %s", path)

	var seen []string
	for page := 0; page <= len(wantIDs); page++ {
		require.LessOrEqual(t, len(list.Data), 1, "limit=1 pages should hold at most one row")
		for _, item := range list.Data {
			seen = append(seen, DataItemField(item, "id"))
		}
		if !list.PageInfo.HasNextPage || list.PageInfo.NextPageURL == nil {
			break
		}
		list, _, err = apiClient.GetListFromPageURL(list.PageInfo.NextPageURL)
		require.NoError(t, err, "paging %s", path)
	}

	assert.ElementsMatch(t, wantIDs, seen,
		"cursor walk over %s should visit each scoped row exactly once", path)
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

// mustGenID generates a well-formed ID with the given prefix via the real ID
// generator. Use it for nonexistent-reference probes instead of hand-written fake
// IDs, so tests never drift from the actual ID format.
func mustGenID(t *testing.T, prefix id.IDPrefix) string {
	t.Helper()
	generated, apiErr := id.GenID(prefix, nil)
	require.Nil(t, apiErr, "failed to generate id with prefix %q", prefix)
	return generated
}

// assertIDFormat asserts the ID starts with the given prefix followed by "_". The
// prefix should come from the shared/id prefix constants; untyped string literals
// also convert.
func assertIDFormat(t *testing.T, actualID string, expectedPrefix id.IDPrefix) {
	t.Helper()
	assert.True(t, strings.HasPrefix(actualID, string(expectedPrefix)+"_"),
		"id %q should start with %q", actualID, string(expectedPrefix)+"_")
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

// createE2EAddress creates an address (owned by the actor's target account) and
// registers cleanup, returning its ID for use as an order bill-to / ship-to.
func createE2EAddress(t *testing.T, name string) string {
	t.Helper()
	status, body, err := apiClient.Post("/v1/sales/addresses", map[string]any{
		"name":          name,
		"street_line_1": "123 Test St",
		"locality":      "Los Angeles",
		"state":         "CA",
		"postal_code":   "90001",
		"country":       "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete("/v1/sales/addresses/" + id) })
	return id
}

// minimalSalesOrderCreateBody returns a minimal payload for POST /v1/sales/sales-orders (internal e2e).
// Create only accepts existing address IDs, so this provisions a bill-to + ship-to address first.
// The buyer must have product line access for the product's line (grant via POST .../product-line-access/customers first).
func minimalSalesOrderCreateBody(t *testing.T, buyerAccountID string) map[string]any {
	return map[string]any{
		"buyer_account_id":   buyerAccountID,
		"carrier_id":         SeedCarrierID,
		"service_level_id":   SeedServiceLevelID,
		"priority_code":      "normal",
		"payment_term_id":    SeedPaymentTermID,
		"shipping_term_id":   SeedShippingTermID,
		"bill_to_address_id": createE2EAddress(t, "E2E Bill-To"),
		"ship_to_address_id": createE2EAddress(t, "E2E Ship-To"),
		// The item, SKU/description, unit cost, and unit price are resolved server-side
		// from the product. The quantity unit must belong to the product's unit group.
		"lines": []map[string]any{
			{
				"product_id": SeedProductID,
				"quantity": map[string]any{
					"value":   "1",
					"unit_id": SeedUnitID,
				},
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

// mustGet fetches a resource by path and requires a 200, returning the raw body. Used to
// re-read after a write so an assertion covers what was persisted rather than the echo.
func mustGet(t *testing.T, path string) []byte {
	t.Helper()
	status, body, err := apiClient.GetListRaw(path, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return body
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

// reads a completed job's row-indexed results into created and updated id slices, in
// results order. A rejected row names no resource, so it lands in neither.
func jobResultIDs(job map[string]any) (created, updated []string) {
	for _, m := range jobResults(job) {
		id := jobResultResourceID(m)
		switch m["status"] {
		case "created":
			created = append(created, id)
		case "updated":
			updated = append(updated, id)
		}
	}
	return created, updated
}

// reads a job's `results` list as raw entry maps, for asserting on an entry's
// index/status/resource/sub_resources directly. Every submitted row appears exactly
// once, written or rejected.
func jobResults(job map[string]any) []map[string]any {
	var out []map[string]any
	for _, raw := range jsonListData(job, "results") {
		if m, ok := raw.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

// picks the written rows out of a job's results — the entries that produced a resource.
func jobWrittenResults(job map[string]any) []map[string]any {
	var out []map[string]any
	for _, m := range jobResults(job) {
		if m["status"] != "failed" {
			out = append(out, m)
		}
	}
	return out
}

// picks the rejected rows out of a job's results — the entries carrying an error.
func jobErrors(job map[string]any) []map[string]any {
	var out []map[string]any
	for _, m := range jobResults(job) {
		if m["status"] == "failed" {
			out = append(out, m)
		}
	}
	return out
}

// reads the id of the resource one result row produced, "" when the row failed.
func jobResultResourceID(entry map[string]any) string {
	resource, _ := entry["resource"].(map[string]any)
	id, _ := resource["id"].(string)
	return id
}

// reads the ids of the resources produced alongside one result row's own.
func jobResultSubResourceIDs(entry map[string]any) []string {
	var out []string
	for _, raw := range jsonListData(entry, "sub_resources") {
		if m, ok := raw.(map[string]any); ok {
			if id, ok := m["id"].(string); ok {
				out = append(out, id)
			}
		}
	}
	return out
}

// extracts a JSON array of strings into a []string, skipping any non-string elements.
func jsonStringSlice(m map[string]any, key string) []string {
	arr := jsonArray(m, key)
	out := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// reads the canonical error object a rejected result row carries.
func jobRowError(entry map[string]any) map[string]any {
	obj, _ := entry["error"].(map[string]any)
	return obj
}

// reads the public message off a rejected result row.
func jobRowErrorMessage(entry map[string]any) string {
	msg, _ := jobRowError(entry)["message"].(string)
	return msg
}

// reads the public message of the failure that sank the job as a whole, "" when the job
// itself did not fail. A row rejected on its own merits does not set this.
func jobErrorMessage(job map[string]any) string {
	obj, _ := job["error"].(map[string]any)
	msg, _ := obj["message"].(string)
	return msg
}

// jsonListData extracts the "data" array from a List[T] field ({"object":"list","data":[...]}).
func jsonListData(m map[string]any, key string) []any {
	obj := jsonObject(m, key)
	if obj == nil {
		return nil
	}
	return jsonArray(obj, "data")
}
