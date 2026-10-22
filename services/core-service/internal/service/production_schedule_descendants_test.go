package service

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/services/core-service/internal/scheduling"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// walkFromEdges drives walkDescendants against an in-memory routing graph, answering each level's
// frontier from the given (parent item -> child items) edges. The whole point of the structural walk
// is that it needs no batch history, so the fixture is just the graph.
func walkFromEdges(t *testing.T, edges map[string][]string, roots []string, maxDepth int) map[string][]string {
	t.Helper()
	ctrl := gomock.NewController(t)
	repo := repositorymock.NewMockProductionScheduleInputRepo(ctrl)

	repo.EXPECT().GetProductionFlowChildrenByItem(gomock.Any(), "acct", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, parents []string) ([]domain.ProductionFlowChildRow, *apierror.APIError) {
			var out []domain.ProductionFlowChildRow
			for _, p := range parents {
				for _, c := range edges[p] {
					out = append(out, domain.ProductionFlowChildRow{ParentItemID: p, ChildItemID: c})
				}
			}
			return out, nil
		}).AnyTimes()

	svc := &productionScheduleSvcImpl{}
	got, _, apiErr := svc.walkDescendants(context.Background(), repo, "acct", roots, maxDepth)
	assert.Nil(t, apiErr)
	return got
}

// Every stage a constraint item flows through is downstream stock that will become a finished good, so
// the walk has to reach the whole chain, not just the immediate next step.
func TestWalkDescendants_ReachesEveryDownstreamStage(t *testing.T) {
	t.Parallel()

	// greige -> sewn -> washed -> packed
	edges := map[string][]string{
		"greige": {"sewn"},
		"sewn":   {"washed"},
		"washed": {"packed"},
	}

	got := walkFromEdges(t, edges, []string{"greige"}, 10)

	assert.Equal(t, []string{"packed", "sewn", "washed"}, got["greige"],
		"the echelon must include every stage between the constraint item and the finished good")
}

// A routing loop must not spin forever or re-expand a stage: the visited set is the cycle guard.
func TestWalkDescendants_IsCycleSafe(t *testing.T) {
	t.Parallel()

	edges := map[string][]string{
		"greige": {"sewn"},
		"sewn":   {"washed"},
		"washed": {"greige"}, // loops back
	}

	got := walkFromEdges(t, edges, []string{"greige"}, 10)

	assert.Equal(t, []string{"sewn", "washed"}, got["greige"],
		"a cycle must terminate and never fold the constraint item back into its own descendants")
}

// A stage reachable from two constraint items belongs to exactly one echelon, or its stock is counted
// twice. Attribution is first-wins over the SKU-sorted roots, so it is stable.
func TestWalkDescendants_FirstWinsOnSharedStage(t *testing.T) {
	t.Parallel()

	// Both greige_a and greige_b feed the same shared wash stage.
	edges := map[string][]string{
		"greige_a": {"shared_wash"},
		"greige_b": {"shared_wash"},
	}

	got := walkFromEdges(t, edges, []string{"greige_a", "greige_b"}, 10)

	assert.Equal(t, []string{"shared_wash"}, got["greige_a"],
		"the first constraint item in sorted order claims the shared stage")
	assert.Empty(t, got["greige_b"],
		"the shared stage must not also count into the second constraint item's echelon")
}

// A constraint item that is itself downstream of another stays rooted to itself rather than being
// folded into the other's echelon, so its on-hand is never counted into two echelons at once.
func TestWalkDescendants_ConstraintDownstreamOfAnotherIsNotFolded(t *testing.T) {
	t.Parallel()

	// greige -> sub, and sub is itself a constraint item that flows to packed.
	edges := map[string][]string{
		"greige": {"sub"},
		"sub":    {"packed"},
	}

	got := walkFromEdges(t, edges, []string{"greige", "sub"}, 10)

	assert.Empty(t, got["greige"],
		"sub is a constraint item in its own right, so it must not be folded into greige's echelon")
	assert.Equal(t, []string{"packed"}, got["sub"])
}

// maxDepth bounds the number of round-trips regardless of chain length.
func TestWalkDescendants_StopsAtMaxDepth(t *testing.T) {
	t.Parallel()

	edges := map[string][]string{
		"a": {"b"},
		"b": {"c"},
		"c": {"d"},
	}

	got := walkFromEdges(t, edges, []string{"a"}, 2)

	assert.Equal(t, []string{"b", "c"}, got["a"],
		"a depth of 2 reaches two levels down and no further")
}

// The walk also returns the edges inside each echelon, which is what locates the branch point. An edge
// into another constraint item's echelon is not part of this family's routing.
func TestWalkDescendants_ReturnsEdgesWithinEachEchelon(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repo := repositorymock.NewMockProductionScheduleInputRepo(ctrl)
	edges := map[string][]string{
		"greige":     {"dyed_blue", "dyed_black"},
		"dyed_blue":  {"blue"},
		"dyed_black": {"black", "sub"},
		"sub":        {"sub_packed"},
	}
	repo.EXPECT().GetProductionFlowChildrenByItem(gomock.Any(), "acct", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, parents []string) ([]domain.ProductionFlowChildRow, *apierror.APIError) {
			var out []domain.ProductionFlowChildRow
			for _, p := range parents {
				for _, c := range edges[p] {
					out = append(out, domain.ProductionFlowChildRow{ParentItemID: p, ChildItemID: c})
				}
			}
			return out, nil
		}).AnyTimes()

	svc := &productionScheduleSvcImpl{}
	_, children, apiErr := svc.walkDescendants(context.Background(), repo, "acct", []string{"greige", "sub"}, 10)
	assert.Nil(t, apiErr)

	assert.Equal(t, map[string][]string{
		"greige":     {"dyed_black", "dyed_blue"},
		"dyed_blue":  {"blue"},
		"dyed_black": {"black"},
		"sub":        {"sub_packed"},
	}, children, "the edge into sub belongs to sub's own echelon, not greige's")
}

// stepEdge is one routing edge with the step that makes it.
type stepEdge struct{ child, step string }

// walkFromStepEdges is walkFromEdges for a graph whose edges name their step, which is what lets the
// walk tell an assembly from an alternative.
func walkFromStepEdges(t *testing.T, edges map[string][]stepEdge, roots []string) map[string][]string {
	t.Helper()
	ctrl := gomock.NewController(t)
	repo := repositorymock.NewMockProductionScheduleInputRepo(ctrl)

	repo.EXPECT().GetProductionFlowChildrenByItem(gomock.Any(), "acct", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, parents []string) ([]domain.ProductionFlowChildRow, *apierror.APIError) {
			var out []domain.ProductionFlowChildRow
			for _, p := range parents {
				for _, e := range edges[p] {
					out = append(out, domain.ProductionFlowChildRow{ParentItemID: p, ChildItemID: e.child, ProductionStepID: e.step})
				}
			}
			return out, nil
		}).AnyTimes()

	svc := &productionScheduleSvcImpl{}
	got, _, apiErr := svc.walkDescendants(context.Background(), repo, "acct", roots, 10)
	assert.Nil(t, apiErr)
	return got
}

// Tights are sewn from a left leg and a right leg knitted as two items. Every pair sewn holds one of
// each, so the sewn stock and everything after it is cover and demand for both legs.
func TestWalkDescendants_StageAssembledFromTwoConstraintItemsBelongsToBoth(t *testing.T) {
	t.Parallel()

	edges := map[string][]stepEdge{
		"leg_left":  {{child: "sewn", step: "step_sew"}},
		"leg_right": {{child: "sewn", step: "step_sew"}},
		"sewn":      {{child: "dyed", step: "step_dye"}},
		"dyed":      {{child: "boxed", step: "step_box"}},
	}

	got := walkFromStepEdges(t, edges, []string{"leg_left", "leg_right"})

	assert.Equal(t, []string{"boxed", "dyed", "sewn"}, got["leg_left"])
	assert.Equal(t, []string{"boxed", "dyed", "sewn"}, got["leg_right"],
		"the right leg is half of every pair sewn, so the whole family is its echelon too")
}

// Two different steps that each make the same stage from a different constraint item are alternatives,
// not an assembly: the stage's stock can only have come from one of them, so it stays first-wins.
func TestWalkDescendants_AlternativeStepsStayFirstWins(t *testing.T) {
	t.Parallel()

	edges := map[string][]stepEdge{
		"greige_a": {{child: "dyed", step: "step_dye_a"}},
		"greige_b": {{child: "dyed", step: "step_dye_b"}},
	}

	got := walkFromStepEdges(t, edges, []string{"greige_a", "greige_b"})

	assert.Equal(t, []string{"dyed"}, got["greige_a"])
	assert.Empty(t, got["greige_b"], "an alternative route must not count the same stock twice")
}

// An order for a finished good assembled from two constraint items is owed by both of them: tights
// ordered are a left leg and a right leg owed.
func TestLoadOpenOrderBook_AssembledFinishedGoodIsOwedByEveryConstraintItem(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repo := repositorymock.NewMockProductionScheduleInputRepo(ctrl)
	repo.EXPECT().GetOpenOrderRequirements(gomock.Any(), "acct", []string{"prd_tights"}).
		Return([]domain.OpenOrderRequirementRow{
			{SalesOrderID: "so_1", SalesOrderLineID: "sol_1", ProductID: "prd_tights", OutstandingQty: 24},
		}, nil)

	in := &scheduling.SolverInput{}
	apiErr := loadOpenOrderBook(context.Background(), repo, domain.LoadSolverInputParams{AccountID: "acct"}, in,
		[]string{"leg_left", "leg_right"},
		map[string][]string{"leg_left": {"sewn", "tights"}, "leg_right": {"sewn", "tights"}},
		map[string]string{"prd_tights": "tights"},
		func(string) float64 { return 2 })
	assert.Nil(t, apiErr)

	assert.Equal(t, []scheduling.OpenOrderLine{
		{SalesOrderID: "so_1", SalesOrderLineID: "sol_1", FinishedItemID: "tights", ConstraintItemID: "leg_left", Units: 12},
		{SalesOrderID: "so_1", SalesOrderLineID: "sol_1", FinishedItemID: "tights", ConstraintItemID: "leg_right", Units: 12},
	}, in.OpenOrders)
}

func identityRatio(string) float64 { return 1 }

// Undyed stock can still become any colorway and stays fungible; dyed and packed stock has passed the
// branch and is committed to the colorway it can still become.
func TestCommittedStockByItem_SplitsStockPastTheBranchPoint(t *testing.T) {
	t.Parallel()

	// greige -> boarded -> dyed_black -> black
	//                   -> dyed_blue  -> blue
	descendants := map[string][]string{"greige": {"black", "blue", "boarded", "dyed_black", "dyed_blue"}}
	children := map[string][]string{
		"greige":     {"boarded"},
		"boarded":    {"dyed_black", "dyed_blue"},
		"dyed_black": {"black"},
		"dyed_blue":  {"blue"},
	}
	sellable := map[string]bool{"black": true, "blue": true}
	onHand := map[string]float64{
		"greige":     50,
		"boarded":    80,
		"dyed_black": 300,
		"black":      300,
		"dyed_blue":  0,  // nothing there: nothing to split out
		"blue":       -5, // drawn negative: keeps netting against the family as before
	}

	got := committedStockByItem([]string{"greige"}, descendants, children, sellable, onHand, identityRatio)

	assert.Equal(t, map[string][]scheduling.CommittedStock{"greige": {
		{ItemID: "black", OnHand: 300, FinishedItemIDs: []string{"black"}},
		{ItemID: "dyed_black", OnHand: 300, FinishedItemIDs: []string{"black"}},
	}}, got, "boarded can still become either colorway, so only the dyed and packed black is committed")
}

// Committed stock is expressed in the constraint item's unit, the same as the echelon it is part of.
func TestCommittedStockByItem_UsesTheConstraintItemsUnit(t *testing.T) {
	t.Parallel()

	descendants := map[string][]string{"greige": {"black", "blue"}}
	children := map[string][]string{"greige": {"black", "blue"}}
	sellable := map[string]bool{"black": true, "blue": true}
	onHand := map[string]float64{"black": 600} // eaches

	got := committedStockByItem([]string{"greige"}, descendants, children, sellable, onHand,
		func(string) float64 { return 2 }) // greige counted in pairs

	assert.Equal(t, []scheduling.CommittedStock{
		{ItemID: "black", OnHand: 300, FinishedItemIDs: []string{"black"}},
	}, got["greige"])
}

// A family with a single finished good has no branch, so nothing in it is committed.
func TestCommittedStockByItem_SingleFinishedGoodHasNoBranch(t *testing.T) {
	t.Parallel()

	descendants := map[string][]string{"greige": {"dyed", "packed"}}
	children := map[string][]string{"greige": {"dyed"}, "dyed": {"packed"}}
	sellable := map[string]bool{"packed": true}
	onHand := map[string]float64{"dyed": 100, "packed": 100}

	got := committedStockByItem([]string{"greige"}, descendants, children, sellable, onHand, identityRatio)
	assert.Empty(t, got)
}

// A routing loop must not recurse forever while working out what a stage can become.
func TestCommittedStockByItem_IsCycleSafe(t *testing.T) {
	t.Parallel()

	descendants := map[string][]string{"greige": {"black", "blue", "dyed_black", "rework"}}
	children := map[string][]string{
		"greige":     {"dyed_black", "blue"},
		"dyed_black": {"black", "rework"},
		"rework":     {"dyed_black"},
	}
	sellable := map[string]bool{"black": true, "blue": true}
	onHand := map[string]float64{"rework": 10}

	got := committedStockByItem([]string{"greige"}, descendants, children, sellable, onHand, identityRatio)
	assert.Equal(t, []scheduling.CommittedStock{
		{ItemID: "rework", OnHand: 10, FinishedItemIDs: []string{"black"}},
	}, got["greige"])
}
