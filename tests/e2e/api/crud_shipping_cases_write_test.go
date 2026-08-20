//go:build e2e

package api_test

import (
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The case SHP-SB-005 exists purely for these edits; the shared seeded case must keep its blank
// tracking number, which crud_pack_list_test.go asserts.
const sb5CaseID = "shcs_01seedsb5_c1_000"

// Covers the shipping-case writes the shipment detail page performs (ShipmentDetails.tsx): editing a
// case's tracking and freight weight, admin-retracking a case, and fetching its label. The existing
// shipping-case tests only read includes, so none of these had coverage.

// Reads one shipping case, with its freight_weight expanded.
func readShippingCase(t *testing.T, caseID string) map[string]any {
	t.Helper()
	status, body, err := apiClient.GetListRaw(shippingCasesPath+"/"+caseID, url.Values{"include": {"freight_weight.unit"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}

// Returns a case's freight weight as (measure, unit id). The API returns the raw DECIMAL, so the
// value is parsed rather than string-compared.
func caseFreightWeight(t *testing.T, caseID string) (float64, string) {
	t.Helper()
	weight := jsonObject(readShippingCase(t, caseID), "freight_weight")
	require.NotNil(t, weight, "freight_weight must expand")
	unitID := ""
	if unit := jsonObject(weight, "unit"); unit != nil {
		unitID = jsonField(unit, "id")
	}
	measure, err := strconv.ParseFloat(jsonField(weight, "value"), 64)
	require.NoError(t, err, "freight weight parses as a number")
	return measure, unitID
}

// The detail page edits a case row by sending the weight as a bare string and deliberately omitting
// the unit, so the case must keep the unit it was stored with rather than falling back to a default.
func TestShippingCases_UpdateTrackingAndWeightKeepsTheStoredUnit(t *testing.T) {
	// Not parallel: mutates the shared seeded case and restores it.
	beforeValue, beforeUnit := caseFreightWeight(t, sb5CaseID)
	require.NotEmpty(t, beforeUnit, "the seeded case must carry a weight unit for this to mean anything")
	beforeTracking := jsonField(readShippingCase(t, sb5CaseID), "tracking_number")

	patch := func(body map[string]any) {
		t.Helper()
		status, resp, err := apiClient.Patch(shippingCasesPath+"/"+sb5CaseID, body, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, resp)
	}
	// A blank tracking number is rejected, so it is only restored when the case had one.
	t.Cleanup(func() {
		restore := map[string]any{"freight_weight_value": strconv.FormatFloat(beforeValue, 'f', -1, 64)}
		if beforeTracking != "" {
			restore["tracking_number"] = beforeTracking
		}
		patch(restore)
	})

	patch(map[string]any{"tracking_number": "1Z-EDITED-BY-TEST", "freight_weight_value": "17.5"})

	got := readShippingCase(t, sb5CaseID)
	assert.Equal(t, "1Z-EDITED-BY-TEST", jsonField(got, "tracking_number"))

	afterValue, afterUnit := caseFreightWeight(t, sb5CaseID)
	assert.InDelta(t, 17.5, afterValue, 0.001, "the new weight must persist")
	assert.Equal(t, beforeUnit, afterUnit,
		"omitting freight_weight_unit_id must keep the case's stored unit, not reset it")
}

// Editing one field must not blank the other — the page sends only what the user touched.
func TestShippingCases_UpdateLeavesUntouchedFieldsAlone(t *testing.T) {
	// Not parallel: shares the seeded case with the test above.
	beforeValue, _ := caseFreightWeight(t, sb5CaseID)
	beforeTracking := jsonField(readShippingCase(t, sb5CaseID), "tracking_number")

	status, resp, err := apiClient.Patch(shippingCasesPath+"/"+sb5CaseID,
		map[string]any{"tracking_number": "1Z-ONLY-TRACKING"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, resp)
	t.Cleanup(func() {
		if beforeTracking == "" {
			return
		}
		_, _, _ = apiClient.Patch(shippingCasesPath+"/"+sb5CaseID,
			map[string]any{"tracking_number": beforeTracking}, newIdempotencyKey())
	})

	afterValue, _ := caseFreightWeight(t, sb5CaseID)
	assert.InDelta(t, beforeValue, afterValue, 0.001, "a tracking-only edit must leave the freight weight alone")
}

// The detail page opens a case's label in a new tab; a case that never bought one must say so
// cleanly rather than 500.
func TestShippingCases_LabelURLForACaseWithoutALabel(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(shippingCasesPath+"/"+sb5CaseID+"/label", nil)
	require.NoError(t, err)
	assert.Less(t, status, 500, "fetching a label must never 5xx: %s", string(body))

	if status == 200 {
		assert.Equal(t, "shipping_case_label_url", jsonField(parseJSON(body), "object"))
	}
}
