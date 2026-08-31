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
	// freight_amount/weight are quantity wrappers; unit subfield should be null by default
	if fa := jsonObject(got, "freight_amount"); fa != nil {
		assert.Nil(t, fa["unit"], "freight_amount.unit should be null without ?include=freight_amount.unit")
	}
	if fw := jsonObject(got, "freight_weight"); fw != nil {
		assert.Nil(t, fw["unit"], "freight_weight.unit should be null without ?include=freight_weight.unit")
	}
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

func TestShippingCases_IncludeFreightAmountUnit(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(shippingCasesPath+"/"+SeedShippingCaseID, url.Values{"include": {"freight_amount.unit"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	if fa := jsonObject(got, "freight_amount"); fa != nil {
		if unit := jsonObject(fa, "unit"); unit != nil {
			assert.Equal(t, "unit", jsonField(unit, "object"))
			assert.NotEmpty(t, jsonField(unit, "id"))
		}
	}
}

func TestShippingCases_IncludeFreightWeightUnit(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(shippingCasesPath+"/"+SeedShippingCaseID, url.Values{"include": {"freight_weight.unit"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	if fw := jsonObject(got, "freight_weight"); fw != nil {
		if unit := jsonObject(fw, "unit"); unit != nil {
			assert.Equal(t, "unit", jsonField(unit, "object"))
			assert.NotEmpty(t, jsonField(unit, "id"))
		}
	}
}
