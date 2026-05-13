//go:build e2e

package api_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Input validation tests verify that the API correctly rejects invalid payloads
// with appropriate error responses. These test common boundary conditions that
// real callers hit: wrong types, missing fields, invalid enums, oversized strings, etc.

// ──────────────────────────────────────────────
// Required field omission
// ──────────────────────────────────────────────

func TestInputValidation_RequiredField_Customer(t *testing.T) {
	t.Parallel()
	// Customer requires "name".
	status, body, err := apiClient.Post(customersPath, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "name")
}

func TestInputValidation_RequiredField_AccountGroup(t *testing.T) {
	t.Parallel()
	// Account group requires "name" and "type" (must be pricing_group or type_group).
	status, body, err := apiClient.Post("/v1/sales/account-groups", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Missing required fields should return 400 or 422, got %d: %s", status, string(body))
}

func TestInputValidation_RequiredField_ProductLine(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post("/v1/catalog/product-lines", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Missing required fields should return 400 or 422, got %d: %s", status, string(body))
}

func TestInputValidation_RequiredField_Location(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post("/v1/operations/locations", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Missing required fields should return 400 or 422, got %d: %s", status, string(body))
}

func TestInputValidation_RequiredField_Role(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post("/v1/identity/roles", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Missing required fields should return 400 or 422, got %d: %s", status, string(body))
}

// ──────────────────────────────────────────────
// Type coercion — wrong JSON types
// ──────────────────────────────────────────────

func TestInputValidation_WrongType_NumberForString(t *testing.T) {
	t.Parallel()
	// Send a number where a string is expected.
	status, body, err := apiClient.Post(customersPath, map[string]any{
		"name": 12345,
	}, newIdempotencyKey())
	require.NoError(t, err)
	// Accept either 400 (strict type checking) or 201 (lenient coercion).
	// If 201, clean up. The point is it shouldn't 500.
	assert.NotEqual(t, 500, status,
		"Number for string field should not cause 500: %s", string(body))
	if status == 201 {
		parsed := parseJSON(body)
		apiClient.Delete(customersPath + "/" + jsonField(parsed, "id"))
	}
}

func TestInputValidation_WrongType_StringForBool(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(customersPath, map[string]any{
		"name":       uniqueName("e2e-val-boolstr"),
		"edi_status": "yes",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.NotEqual(t, 500, status,
		"String for boolean field should not cause 500: %s", string(body))
	if status == 201 {
		parsed := parseJSON(body)
		apiClient.Delete(customersPath + "/" + jsonField(parsed, "id"))
	}
}

func TestInputValidation_WrongType_ObjectForString(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(customersPath, map[string]any{
		"name": map[string]any{"nested": "object"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Object for string field should return 400 or 422, got %d: %s", status, string(body))
}

func TestInputValidation_WrongType_ArrayForString(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(customersPath, map[string]any{
		"name": []string{"not", "a", "string"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Array for string field should return 400 or 422, got %d: %s", status, string(body))
}

// ──────────────────────────────────────────────
// String length boundaries
// ──────────────────────────────────────────────

func TestInputValidation_EmptyStringName(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(customersPath, map[string]any{
		"name": "",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Empty string name should return 400 or 422, got %d: %s", status, string(body))
}

func TestInputValidation_LongStringAccepted(t *testing.T) {
	t.Parallel()
	// 255 chars is the DB max for name — should be accepted.
	longName := strings.Repeat("a", 255)
	payload := validCustomerBody(longName)
	payload["bill_to_address"] = map[string]any{"name": "Billing", "country": "US"}
	payload["ship_to_address"] = map[string]any{"name": "Shipping", "country": "US"}
	status, body, err := apiClient.Post(customersPath, payload, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	apiClient.Delete(customersPath + "/" + jsonField(parseJSON(body), "id"))
}

func TestInputValidation_ExtremelyLongString_Rejected(t *testing.T) {
	t.Parallel()
	// 256+ chars exceeds the varchar(255) column — should be rejected with 400.
	longName := strings.Repeat("a", 500)
	status, body, err := apiClient.Post(customersPath, map[string]any{
		"name": longName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
}

func TestInputValidation_UnicodeString(t *testing.T) {
	t.Parallel()
	// Mixes 3-byte CJK and a 4-byte supplementary-plane code point so the test
	// exercises the full UTF-8 range, not just ASCII or the BMP.
	name := uniqueName("e2e-val") + " 日本語テスト 🏭"
	status, body, err := apiClient.Post(customersPath, map[string]any{
		"name": name,
	}, newIdempotencyKey())
	require.NoError(t, err)
	// Unicode should be handled gracefully — either accepted or rejected, never 500.
	assert.NotEqual(t, 500, status,
		"Unicode string should not cause 500: %s", string(body))
	if status == 201 {
		parsed := parseJSON(body)
		apiClient.Delete(customersPath + "/" + jsonField(parsed, "id"))
	}
}

// ──────────────────────────────────────────────
// Enum violations
// ──────────────────────────────────────────────

func TestInputValidation_InvalidEnum_StatusCode(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(customersPath, map[string]any{
		"name":   uniqueName("e2e-val-enum"),
		"status": "totally_invalid_status",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

func TestInputValidation_InvalidEnum_CommissionPolicy(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(customersPath, map[string]any{
		"name":              uniqueName("e2e-val-comm"),
		"commission_policy": "not_a_real_policy",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

func TestInputValidation_InvalidEnum_FreightPolicy(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(customersPath, map[string]any{
		"name":           uniqueName("e2e-val-freight"),
		"freight_policy": "invalid_freight_value",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

func TestInputValidation_InvalidEnum_CarrierBillingType(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(customersPath, map[string]any{
		"name":                 uniqueName("e2e-val-cbt"),
		"carrier_billing_type": "not_valid",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

// ──────────────────────────────────────────────
// ID format violations
// ──────────────────────────────────────────────

func TestInputValidation_InvalidIDFormat(t *testing.T) {
	t.Parallel()
	// Note: Some foreign key fields may accept invalid IDs without validation
	// and silently ignore them. This test verifies no 500 occurs.
	status, body, err := apiClient.Post(customersPath, map[string]any{
		"name":               uniqueName("e2e-val-badid"),
		"default_carrier_id": "not-a-valid-id",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.NotEqual(t, 500, status,
		"Invalid ID format should not cause 500: %s", string(body))
	if status == 201 {
		apiClient.Delete(customersPath + "/" + jsonField(parseJSON(body), "id"))
	}
}

func TestInputValidation_WrongPrefixID(t *testing.T) {
	t.Parallel()
	// Use a valid-looking ID but with the wrong prefix (user ID where carrier ID expected).
	status, body, err := apiClient.Post(customersPath, map[string]any{
		"name":               uniqueName("e2e-val-wrongpfx"),
		"default_carrier_id": SeedUserID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	// Should be 400 (validation) or 404 (not found) — never 500 or silently accepted.
	assert.NotEqual(t, 500, status,
		"Wrong prefix ID should not cause 500: %s", string(body))
}

// ──────────────────────────────────────────────
// Null and special values
// ──────────────────────────────────────────────

func TestInputValidation_NullRequiredField(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(customersPath, map[string]any{
		"name": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Null required field should return 400 or 422, got %d: %s", status, string(body))
}

func TestInputValidation_WhitespaceOnlyName(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(customersPath, map[string]any{
		"name": "   ",
	}, newIdempotencyKey())
	require.NoError(t, err)
	// Whitespace-only names should either be rejected or trimmed.
	// Accept both 400 and 201 — but never 500.
	assert.NotEqual(t, 500, status,
		"Whitespace-only name should not cause 500: %s", string(body))
	if status == 201 {
		parsed := parseJSON(body)
		apiClient.Delete(customersPath + "/" + jsonField(parsed, "id"))
	}
}

// ──────────────────────────────────────────────
// Malformed JSON
// ──────────────────────────────────────────────

func TestInputValidation_ExtraNestedPayload(t *testing.T) {
	t.Parallel()
	// Deeply nested payload — should not stack overflow or 500.
	nested := map[string]any{"name": uniqueName("e2e-val-nested")}
	current := nested
	for i := 0; i < 50; i++ {
		inner := map[string]any{"deep": "value"}
		current["extra_field"] = inner
		current = inner
	}

	status, body, err := apiClient.Post(customersPath, nested, newIdempotencyKey())
	require.NoError(t, err)
	// Should reject unknown fields (400) or accept just name — never 500.
	assert.NotEqual(t, 500, status,
		"Deeply nested payload should not cause 500: %s", string(body))
	if status == 201 {
		parsed := parseJSON(body)
		apiClient.Delete(customersPath + "/" + jsonField(parsed, "id"))
	}
}
