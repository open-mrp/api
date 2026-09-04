package sqlc

import (
	"strings"
	"testing"
)

// TestListConsumptionChangeLogsForBurnRate_ActionTypeFilter guards the burn-rate consumption filter
// against silent regressions in the generated query. 'system_action' (product order fulfillment) must
// be counted or products compute a burn rate of 0; 'user_correction' must not be counted or a single
// large manual re-baseline of on-hand counts skews the rate. Keep in sync with the action-type gate in
// mediator.MaybeRecalculateAfterConsumption.
func TestListConsumptionChangeLogsForBurnRate_ActionTypeFilter(t *testing.T) {
	q := listConsumptionChangeLogsForBurnRate

	for _, want := range []string{"'scan'", "'system_action'"} {
		if !strings.Contains(q, want) {
			t.Errorf("burn-rate consumption query must count %s as consumption; query was:\n%s", want, q)
		}
	}
	if strings.Contains(q, "user_correction") {
		t.Errorf("burn-rate consumption query must not count user_correction (manual re-baseline, not demand); query was:\n%s", q)
	}
}
