//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Behavioral coverage for POST /v1/sales/sales-orders/{id}/actions/quote-freight, pinning
// that the quote's unit_price presents its numerator/denominator units in full — the same
// Unit shape (name, abbreviation, type, conversion fields) every other endpoint returns —
// rather than a bare {id, object} reference, which is what the endpoint used to emit.

// assertFullyPresentedUnit fails unless the given object is a fully presented Unit resource,
// not a bare {id, object} reference.
func assertFullyPresentedUnit(t *testing.T, unit map[string]any, label string) {
	t.Helper()
	require.NotNil(t, unit, "%s is present", label)
	assert.Equal(t, "unit", jsonField(unit, "object"), "%s.object", label)
	assert.NotEmpty(t, jsonField(unit, "id"), "%s.id", label)
	// The bug: only {id, object} came back. A fully presented unit carries its display and
	// conversion fields — these are exactly what was missing.
	assert.NotEmpty(t, jsonField(unit, "name"), "%s.name is present (fully presented unit)", label)
	assert.NotEmpty(t, jsonField(unit, "abbreviation"), "%s.abbreviation is present", label)
	assert.NotEmpty(t, jsonField(unit, "type"), "%s.type is present", label)
	assert.Contains(t, unit, "ratio_numerator", "%s carries conversion fields", label)
	assert.Contains(t, unit, "ratio_denominator", "%s carries conversion fields", label)
}

func TestQuoteSalesOrderFreight_FullyPresentsUnits(t *testing.T) {
	t.Parallel()
	orderID := createLifecycleOrder(t)

	status, body, err := apiClient.Post(salesOrdersPath+"/"+orderID+"/actions/quote-freight", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	resp := parseJSON(body)
	assert.Equal(t, "sales_order_freight_quote", jsonField(resp, "object"))

	unitPrice := jsonObject(resp, "unit_price")
	require.NotNil(t, unitPrice, "the freight quote carries a unit_price")
	assert.Equal(t, "sales_order_quote_rate", jsonField(unitPrice, "object"))
	assert.NotEmpty(t, jsonField(unitPrice, "value"), "the rate carries a value")

	// Both units of the rate must be fully presented, not bare {id, object} refs.
	assertFullyPresentedUnit(t, jsonObject(unitPrice, "numerator_unit"), "numerator_unit")
	assertFullyPresentedUnit(t, jsonObject(unitPrice, "denominator_unit"), "denominator_unit")
}
