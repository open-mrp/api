//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const shippingCasesPath = "/v1/operations/shipping-cases"

// ──────────────────────────────────────────────
// ShippingCase — Include Tests
// ──────────────────────────────────────────────

func TestShippingCases_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(shippingCasesPath+"/"+SeedShippingCaseID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["carrier"], "carrier should be null without ?include=carrier")
	assert.Nil(t, got["shipment"], "shipment should be null without ?include=shipment")
	assertNilField(t, got, "freight_amount")
	assertNilField(t, got, "freight_weight")
}

func TestShippingCases_IncludeCarrier(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(shippingCasesPath+"/"+SeedShippingCaseID, url.Values{"include": {"carrier"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	carrier := jsonObject(got, "carrier")
	require.NotNil(t, carrier, "carrier should be present with ?include=carrier")
	assert.Equal(t, "carrier", jsonField(carrier, "object"))
	assert.Equal(t, SeedCarrierID, jsonField(carrier, "id"))
	// The include must carry the full carrier, not a stub built from the shipping-case join: the join
	// projection dropped the carrier's own code, so a stub reads null here even though it's a carrier
	// with a code. See TestIncludes_HydratedToOneMatchesCanonical for the cross-endpoint guard.
	assert.Equal(t, "delivery", jsonField(carrier, "code"),
		"the included carrier carries its code, not a partial stub")
}

func TestShippingCases_IncludeShipment(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(shippingCasesPath+"/"+SeedShippingCaseID, url.Values{"include": {"shipment"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	shipment := jsonObject(got, "shipment")
	require.NotNil(t, shipment, "shipment should be present with ?include=shipment")
	assert.Equal(t, "shipment", jsonField(shipment, "object"))
	assert.Equal(t, SeedShipmentID, jsonField(shipment, "id"))
	// The include must carry the full shipment, not a stub built from the shipping-case join: the join
	// projection dropped the shipment's stored priority, so a stub reads empty here. See
	// TestIncludes_HydratedToOneMatchesCanonical for the cross-endpoint guard.
	assert.Equal(t, "normal", jsonField(shipment, "priority"),
		"the included shipment carries its priority, not a partial stub")
}

// The freight amount is money, so it names its currency rather than a unit.
func TestShippingCases_IncludeFreightAmount(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(shippingCasesPath+"/"+SeedShippingCaseID, url.Values{"include": {"freight_amount"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	if fa := jsonObject(parseJSON(body), "freight_amount"); fa != nil {
		assert.NotEmpty(t, jsonField(fa, "amount"))
		assert.Equal(t, "usd", jsonField(fa, "currency"))
		assert.NotContains(t, fa, "unit", "money carries a currency, not a unit")
	}
}

// The freight weight always carries its unit; there is no separate unit include.
func TestShippingCases_IncludeFreightWeightCarriesItsUnit(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(shippingCasesPath+"/"+SeedShippingCaseID, url.Values{"include": {"freight_weight"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	if fw := jsonObject(parseJSON(body), "freight_weight"); fw != nil {
		assert.NotEmpty(t, jsonField(fw, "value"))
		unit := jsonObject(fw, "unit")
		require.NotNil(t, unit, "the unit comes with the weight")
		assert.Equal(t, "unit", jsonField(unit, "object"))
		assert.NotEmpty(t, jsonField(unit, "id"))
	}
}

func TestShippingCases_UnitIncludesAreRejected(t *testing.T) {
	t.Parallel()
	for _, include := range []string{"freight_amount.unit", "freight_weight.unit"} {
		status, body, err := apiClient.GetListRaw(shippingCasesPath+"/"+SeedShippingCaseID, url.Values{"include": {include}})
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
		assertErrorParam(t, errObj, "include[]")
	}
}
