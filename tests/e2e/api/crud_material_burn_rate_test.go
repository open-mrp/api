//go:build e2e

package api_test

import (
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A manual inventory correction must not move the burn rate.
//
// Burn rate is a demand signal, so it is computed only from consumption booked as 'scan' (production
// draw-down) or 'system_action' (order fulfillment). The inventory PATCH endpoint books
// 'user_correction' whatever the operation, because a manual edit is a re-baseline of the on-hand
// count rather than something the plant used — and a single large correction would dwarf real usage
// and skew the rate for thirty days. Both the mediator gate and the SQL that computes the rate
// exclude it; see ListConsumptionChangeLogsForBurnRate.
//
// This test used to drive consumption through this endpoint and assert the rate moved, which is the
// behaviour that was removed. The rate rising here again would mean corrections had leaked back into
// the demand signal.
//
// The path that does move the rate — scan consumption — is covered in units rather than here:
// BurnRateMedTestSuite.TestRecalculate_ComputesAndWritesRate for the arithmetic,
// TestMaybeRecalculateAfterConsumption_ActionTypeGate for which action types qualify, and
// TestListConsumptionChangeLogsForBurnRate_ActionTypeFilter for keeping the SQL in step. Driving a
// real scan end to end needs a production run and a published schedule, which is the machine-status
// test's territory.
func TestMaterials_BurnRate_IgnoresManualInventoryCorrections(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-mat-burn")
	body := validMaterialBody(sku)
	createStatus, createBody, err := apiClient.Post(materialsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	created := parseJSON(createBody)
	materialID := jsonField(created, "id")
	require.NotEmpty(t, materialID)
	t.Cleanup(func() { apiClient.Delete(materialsPath + "/" + materialID) })

	getStatus, getBody, err := apiClient.GetListRaw(materialsPath+"/"+materialID, url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	item := jsonObject(parseJSON(getBody), "item")
	require.NotNil(t, item, "item should be present with ?include=item")
	itemID := jsonField(item, "id")
	require.NotEmpty(t, itemID)

	adjust := func(delta string) {
		t.Helper()
		patchBody := map[string]any{
			"quantity":  map[string]any{"value": delta, "unit_id": nonCurrencyUnitID},
			"operation": "adjust",
		}
		status, resp, patchErr := apiClient.Patch(itemsPath+"/"+itemID+"/inventory", patchBody, newIdempotencyKey())
		require.NoError(t, patchErr)
		requireStatus(t, 200, status, resp)
	}

	// Enough corrections to compute a rate from, were they eligible: the mediator needs two logs and
	// a non-zero total, and both are satisfied here.
	adjust("-10")
	adjust("-5")

	// Held for longer than a recompute needs to land, rather than read once: a recompute runs off the
	// consumption transaction via the outbox, so reading immediately would report zero because nothing
	// had run yet, and the test would pass whether or not corrections were excluded.
	burnRate := requireBurnRateStaysZero(t, materialID, burnRateSettleWindow)

	denUnit := jsonObject(burnRate, "denominator_unit")
	require.NotNil(t, denUnit, "burn rate denominator_unit should be present")
	assert.Equal(t, "day", jsonField(denUnit, "id"))
}

// burnRateSettleWindow is how long the rate is watched for a recompute that must never arrive. It is
// comfortably longer than the outbox round trip the old test waited on, so a rate that was going to
// move has had every chance to.
const burnRateSettleWindow = 5 * time.Second

// requireBurnRateStaysZero polls the material's burn rate for the whole window and fails the moment it
// moves off zero, returning the last rate read.
func requireBurnRateStaysZero(t *testing.T, materialID string, window time.Duration) map[string]any {
	t.Helper()

	var burnRate map[string]any
	deadline := time.Now().Add(window)
	for {
		status, body, getErr := apiClient.GetListRaw(materialsPath+"/"+materialID, url.Values{
			"include": {"item.burn_rate"},
		})
		require.NoError(t, getErr)
		requireStatus(t, 200, status, body)

		gotItem := jsonObject(parseJSON(body), "item")
		require.NotNil(t, gotItem, "item should be present with include=item.burn_rate: %s", body)
		burnRate = jsonObject(gotItem, "burn_rate")
		require.NotNil(t, burnRate, "item.burn_rate should be present: %s", body)

		measure, parseErr := strconv.ParseFloat(jsonField(burnRate, "value"), 64)
		require.NoError(t, parseErr, "burn rate value must be a decimal")
		require.Zero(t, measure,
			"a manual correction is not demand: it must leave the burn rate where it was")

		if !time.Now().Before(deadline) {
			return burnRate
		}
		time.Sleep(e2eAsyncPollInterval)
	}
}
