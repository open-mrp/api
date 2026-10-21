package repository

import (
	"strings"
	"testing"
)

// The echelon's downstream discovery is structural — the routing graph — not batch genealogy, and that
// is the whole point of the query. Genealogy (`_batch_flow`) is written only when the floor links
// batches through move/split/merge, so a walk that depended on it saw no downstream stages whenever a
// plant scanned each stage as its own batch, and the echelon collapsed to the constraint item alone.
// Reintroducing a batch table here would quietly bring that failure back: the query would still return
// rows, just far fewer of them, and only in accounts that never linked their batches.
func TestGetProductionFlowChildrenByItem_WalksTheRoutingGraphNotGenealogy(t *testing.T) {
	t.Parallel()

	body := queryBody(t, "production_schedule_input.sql", "GetProductionFlowChildrenByItem")

	if strings.Contains(body, "_batch_flow") || strings.Contains(body, "FROM batch") || strings.Contains(body, "JOIN batch") {
		t.Error("GetProductionFlowChildrenByItem reads batch history: downstream stages must come from the " +
			"routing graph (consumption -> production), or the echelon collapses when batches were never linked")
	}
	if !strings.Contains(body, "consumption") || !strings.Contains(body, "production") {
		t.Error("GetProductionFlowChildrenByItem must join consumption to production to derive the item each " +
			"consuming step produces — the structural downstream edge")
	}
	// Consumption and production carry no account_id of their own; the scope comes from the step.
	if !strings.Contains(body, "ps.account_id") {
		t.Error("GetProductionFlowChildrenByItem must scope by production_step.account_id: consumption and " +
			"production have no account column, so without it the walk crosses tenant boundaries")
	}
}

// The routing edge carries the step that makes it, because the step is the only thing that tells an
// assembly (one step joining a left and a right leg) from two alternative ways to make the same stage.
// Without it the walk can only guess, and its guess handed a whole pair-of-legs family to one leg.
func TestGetProductionFlowChildrenByItem_ReturnsTheStep(t *testing.T) {
	t.Parallel()

	body := queryBody(t, "production_schedule_input.sql", "GetProductionFlowChildrenByItem")

	if !strings.Contains(body, "c.production_step_id") {
		t.Error("GetProductionFlowChildrenByItem must return the consuming step, or the walk cannot see " +
			"that one step assembles several constraint items into the same stage")
	}
}

// An allocation is counted in its own unit, not the receipt's. A receipt of pairs drawn on in eaches
// otherwise loses a whole pair for every each, and a stage holding thousands of pairs reads as almost
// empty: the plan then knits greige to replace stock already on the shelf.
func TestGetEchelonOnHand_NormalizesEachAllocationThroughItsOwnUnit(t *testing.T) {
	t.Parallel()

	body := queryBody(t, "production_schedule_input.sql", "GetEchelonOnHand")

	if !strings.Contains(body, "JOIN unit au ON au.id = aq.unit_id") {
		t.Error("GetEchelonOnHand must join each allocation's own unit")
	}
	if !strings.Contains(body, "aq.value * (au.ratio_numerator / au.ratio_denominator)") {
		t.Error("GetEchelonOnHand must scale each allocation by its own unit's ratio, not the receipt's")
	}
}

// Carry-forward looks for any open ticket on a machine for the item, not only those an earlier schedule
// release printed: work already on a machine is outstanding either way, and a second ticket for it is
// the failure carry-forward exists to prevent. Runs released for this week or later stay untouched.
func TestListCarryForwardBatchesForItem_TakesAnyOpenBatchOnAMachine(t *testing.T) {
	t.Parallel()

	body := queryBody(t, "production_schedule_lifecycle.sql", "ListCarryForwardBatchesForItem")

	if strings.Contains(body, "production_run_id IN (") {
		t.Error("ListCarryForwardBatchesForItem must not limit candidates to runs an earlier schedule week released")
	}
	if !strings.Contains(body, "_batches_machines") {
		t.Error("ListCarryForwardBatchesForItem must only take batches already on a machine")
	}
	if !strings.Contains(body, "production_run_id NOT IN (") || !strings.Contains(body, "l.week_start_date >= sqlc.arg('week_start_date')") {
		t.Error("ListCarryForwardBatchesForItem must leave runs released for this week or a later one alone")
	}
	for _, guard := range []string{"b.scanned_at IS NULL", "b.closed_at IS NULL"} {
		if !strings.Contains(body, guard) {
			t.Errorf("ListCarryForwardBatchesForItem must keep %q: only unworked tickets can be moved", guard)
		}
	}
}
