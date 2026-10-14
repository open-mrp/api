//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

// A measure is a value and the unit it is counted in. A bare number says nothing about what it counts,
// so every batch endpoint that returns a measure carries its whole unit inline. The batch endpoints
// take no include at all: there is nothing behind one, so the parameter is refused rather than ignored.

// assertMeasureCarriesItsUnit checks a measure arrives with its value and its whole unit record.
func assertMeasureCarriesItsUnit(t *testing.T, measure map[string]any, where string) {
	t.Helper()

	require.NotNil(t, measure, "%s must be present", where)
	require.NotEmpty(t, jsonField(measure, "value"), "%s.value", where)
	assertUnitHydrated(t, jsonObject(measure, "unit"), where+".unit")
	assertNoValueIdentity(t, measure, where)
}

// assertIncludeRefused checks a batch endpoint refuses ?include as an unknown parameter.
func assertIncludeRefused(t *testing.T, status int, body []byte) {
	t.Helper()

	requireStatus(t, 400, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "parameter_unknown", "invalid_request_error"), "include")
}

// --- Remaining quantity to split ---

// The response is a bare measure, so its unit is all that says what the number counts.
func TestBatches_RemainingQuantityUnitResolves(t *testing.T) {
	t.Parallel()

	// A batch waiting at a split step, so the endpoint answers with a measure.
	c := newScanChain(t)
	planned := planScanBatch(t, c.a.id, "10", unitEach)
	initializeScan(t, planned, c.initStation, nil)
	moveScan(t, c.moveStation, c.moveStep, planned)

	body := map[string]any{
		"batch_ids":          []string{planned},
		"production_step_id": c.splitStep,
	}

	status, plain, err := apiClient.Post(batchesPath+"/remaining-quantities", body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, plain)
	assertMeasureCarriesItsUnit(t, parseJSON(plain), "remaining quantity")

	withStatus, withUnit, err := apiClient.Post(
		batchesPath+"/remaining-quantities?include=unit", body, newIdempotencyKey())
	require.NoError(t, err)
	assertIncludeRefused(t, withStatus, withUnit)
}

// --- Batch flow ---

// Every node of a flow carries a whole batch, and the batch's measures carry their units.
func TestBatches_FlowMeasureUnitsResolveThroughTheBatch(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(batchesPath+"/"+SeedBatchID+"/flow", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "the flow must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	nodes := jsonArray(parseJSON(body), "data")
	require.NotEmpty(t, nodes, "the seeded batch has a flow: %s", string(body))

	var checked int
	for _, raw := range nodes {
		node, ok := raw.(map[string]any)
		require.True(t, ok)
		batch := jsonObject(node, "batch")
		require.NotNil(t, batch, "every flow node carries a batch: %v", node)

		quantity := jsonObject(batch, "quantity")
		if quantity == nil {
			continue
		}
		assertMeasureCarriesItsUnit(t, quantity, "flow node batch.quantity")
		checked++
	}
	require.Positive(t, checked, "at least one flow node reports a quantity: %s", string(body))

	withStatus, withBody, err := apiClient.GetListRaw(batchesPath+"/"+SeedBatchID+"/flow",
		url.Values{"include": {"batch.quantity"}})
	require.NoError(t, err)
	assertIncludeRefused(t, withStatus, withBody)
}

// --- Delete ---

// Delete returns a deleted stub, which carries no measures, so there is nothing for include to expand and
// the parameter is refused rather than silently ignored.
func TestBatches_DeleteRefusesInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Delete(batchesPath + "/bt_doesnotexist00000?include=" + url.QueryEscape("quantity"))
	require.NoError(t, err)
	assertIncludeRefused(t, status, body)
}
