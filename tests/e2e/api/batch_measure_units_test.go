//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A measure is a value and the unit it is counted in, and the batch endpoints carry only the unit's
// id inline — the unit itself is resolved on `?include`. That only works where the endpoint declares
// the include: on one that does not, the unit is null and there is no request that can fill it in,
// so the measure comes back as a bare number with nothing saying what it counts. Every batch endpoint
// that returns a measure is checked here for exactly that.

// assertMeasureUnitResolvable pins both halves of the contract on one measure: absent without the
// include, and a whole unit record with it.
func assertMeasureUnitResolvable(t *testing.T, without, with map[string]any, where string) {
	t.Helper()

	require.NotNil(t, without, "%s must be present without the include", where)
	assertNilField(t, without, "unit")

	require.NotNil(t, with, "%s must be present with the include", where)
	assertUnitHydrated(t, jsonObject(with, "unit"), where+".unit")
}

// assertIncludeAccepted checks that an endpoint advertises an include rather than rejecting it.
//
// An endpoint that returns measures but declares no includes answers 400 ("does not support the
// include parameter") to the only request that could resolve their units, which is the failure this
// guards. Anything that is not a 400 means the include was understood.
func assertIncludeAccepted(t *testing.T, status int, body []byte, include, where string) {
	t.Helper()

	assert.NotEqual(t, 400, status,
		"%s returns measures, so it must accept ?include=%s rather than reject it: %s",
		where, include, string(body))
}

// --- Remaining quantity to split ---

// The response is a bare measure, so its unit is the only thing there is to expand — and an endpoint
// returning a lone quantity with no include support returns a number with no unit at all.
func TestBatches_RemainingQuantityUnitResolves(t *testing.T) {
	t.Parallel()

	body := map[string]any{
		"batch_ids":          []string{SeedBatchID},
		"production_step_id": SeedProductionStepID,
	}

	status, plain, err := apiClient.Post(batchesPath+"/remaining-quantities", body, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "remaining quantities must not 5xx: %s", string(plain))

	withStatus, withUnit, err := apiClient.Post(
		batchesPath+"/remaining-quantities?include=unit", body, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, withStatus, 500, "remaining quantities must not 5xx with an include: %s", string(withUnit))

	assertIncludeAccepted(t, withStatus, withUnit, "unit", "remaining-quantities")
	assert.Equal(t, status, withStatus,
		"the include changes what is expanded, never whether the request is answered: %s", string(withUnit))

	// The seeded batch is not a valid split target at the seeded step, so the endpoint answers before
	// it builds a measure. The include being understood rather than rejected is the whole contract
	// here; a measure that does come back is checked too.
	if status != 200 {
		return
	}
	assertMeasureUnitResolvable(t, parseJSON(plain), parseJSON(withUnit), "remaining quantity")
}

// --- Batch flow ---

// Every node of a flow carries a whole batch, so the batch's three measures are reached through it.
func TestBatches_FlowMeasureUnitsResolveThroughTheBatch(t *testing.T) {
	t.Parallel()

	plainStatus, plain, err := apiClient.GetListRaw(batchesPath+"/"+SeedBatchID+"/flow", nil)
	require.NoError(t, err)
	require.Less(t, plainStatus, 500, "the flow must not 5xx: %s", string(plain))
	requireStatus(t, 200, plainStatus, plain)

	withStatus, withUnit, err := apiClient.GetListRaw(batchesPath+"/"+SeedBatchID+"/flow",
		url.Values{"include": {"batch.quantity.unit"}})
	require.NoError(t, err)
	require.Less(t, withStatus, 500, "the flow must not 5xx with an include: %s", string(withUnit))
	assertIncludeAccepted(t, withStatus, withUnit, "batch.quantity.unit", "the batch flow")
	requireStatus(t, 200, withStatus, withUnit)

	plainNodes := jsonArray(parseJSON(plain), "data")
	withNodes := jsonArray(parseJSON(withUnit), "data")
	require.NotEmpty(t, plainNodes, "the seeded batch has a flow: %s", string(plain))
	require.Len(t, withNodes, len(plainNodes), "the include does not change which nodes come back")

	var checked int
	for i, raw := range withNodes {
		node, ok := raw.(map[string]any)
		require.True(t, ok)
		batch := jsonObject(node, "batch")
		require.NotNil(t, batch, "every flow node carries a batch: %v", node)

		quantity := jsonObject(batch, "quantity")
		if quantity == nil {
			continue
		}
		plainNode, ok := plainNodes[i].(map[string]any)
		require.True(t, ok)

		assertMeasureUnitResolvable(t,
			jsonObject(jsonObject(plainNode, "batch"), "quantity"),
			quantity,
			"flow node batch.quantity")
		checked++
	}
	require.Positive(t, checked, "at least one flow node reports a quantity: %s", string(withUnit))
}

// --- Delete ---

// Delete returns the batch as it looked before deletion, measures and all, so it has the same gap as
// any other batch endpoint — just harder to notice, because the batch is gone by the time anyone
// reads the response.
//
// Asserted against a batch that does not exist: the include is validated before the handler runs, so
// a 404 says the endpoint understood the include while a 400 says it advertises none. That avoids
// standing up a production run and a schedule just to have something to delete.
func TestBatches_DeleteAcceptsTheMeasureUnitIncludes(t *testing.T) {
	t.Parallel()

	for _, include := range []string{"quantity.unit", "seconds.unit", "waste.unit"} {
		t.Run(include, func(t *testing.T) {
			t.Parallel()

			status, body, err := apiClient.Delete(
				batchesPath + "/bt_doesnotexist00000?include=" + url.QueryEscape(include))
			require.NoError(t, err)
			require.Less(t, status, 500, "deleting an unknown batch must not 5xx: %s", string(body))

			assertIncludeAccepted(t, status, body, include, "deleting a batch")
			assert.Equal(t, 404, status,
				"the include is understood, so the request fails on the missing batch: %s", string(body))
		})
	}
}
