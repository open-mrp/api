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
