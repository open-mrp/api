//go:build e2e

package api_test

import (
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
// The rate rising here again would mean corrections had leaked back into the demand signal.
//
// The path that does move the rate — scan consumption — is covered in units rather than here:
// BurnRateMedTestSuite.TestRecalculate_ComputesAndWritesRate for the arithmetic,
// TestMaybeRecalculateAfterConsumption_ActionTypeGate for which action types qualify, and
// TestListConsumptionChangeLogsForBurnRate_ActionTypeFilter for keeping the SQL in step. Driving a
// real scan end to end needs a production run and a published schedule, which is the machine-status
// test's territory.
func TestMaterials_BurnRate_IgnoresManualInventoryCorrections(t *testing.T) {
	t.Parallel()

	itemID := jsonField(createForgeItem(t, itemsPath, forgeItemBody("material", uniqueName("e2e-mat-burn"))), "id")

	adjust := func(delta string) {
		t.Helper()
		patchBody := map[string]any{
			"quantity":  map[string]any{"value": delta, "unit_id": poundUnitID},
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
	burnRate := requireBurnRateStaysZero(t, itemID, burnRateSettleWindow)

	assert.Equal(t, dayUnitID, jsonField(jsonObject(burnRate, "per_unit"), "id"), "burn rate is counted per day")
	assert.Equal(t, SeedMaterialUnitID, jsonField(jsonObject(burnRate, "unit"), "id"), "burn rate is in the category's base unit")
}

// burnRateSettleWindow is how long the rate is watched for a recompute that must never arrive. It is
// comfortably longer than the outbox round trip the old test waited on, so a rate that was going to
// move has had every chance to.
const burnRateSettleWindow = 5 * time.Second

// requireBurnRateStaysZero polls the item's burn rate, which every item carries, for the whole window and fails the
// moment it moves off zero, returning the last rate read.
func requireBurnRateStaysZero(t *testing.T, itemID string, window time.Duration) map[string]any {
	t.Helper()

	var burnRate map[string]any
	deadline := time.Now().Add(window)
	for {
		burnRate = jsonObject(getItem(t, itemID), "burn_rate")
		require.NotNil(t, burnRate, "burn_rate is always present")

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
