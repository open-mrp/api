package service

import (
	"context"
	"sort"
	"testing"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// assembleFlowForItem must root the flow at a SINGLE producing step and walk only
// upstream. An item produced by two recipes (e.g. the White Sock made either Large or
// Small) that share an upstream step must not merge both recipes — nor pull in sibling
// branches reachable via downstream edges — or a run would build parts that aren't on the
// ordered item's recipe. This reproduces the sock-catalog bug (order or_85ix05sfvy3b).
func TestAssembleFlowForItem_SingleRecipeNoForwardExpansion(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	flowRepo := repositorymock.NewMockProductionFlowRepo(ctrl)
	stepQueryRepo := repositorymock.NewMockProductionStepQueryRepo(ctrl)

	const acct = "ac_test"
	// F is produced by two recipes: pack1 (fed by knit1) and pack2 (fed by knit2).
	// knit1 also feeds a sibling branch (dye1) for a different product — reachable only
	// via a downstream edge, so it must be excluded by the backward-only walk.
	flowRepo.EXPECT().FindStepsByProducedItem(gomock.Any(), acct, "F").
		Return([]string{"pack2", "pack1"}, nil).Times(1)
	flowRepo.EXPECT().GetAllStepEdgesForAccount(gomock.Any(), acct).Return([]domain.StepEdge{
		{ParentStepID: "knit1", ChildStepID: "pack1"},
		{ParentStepID: "knit2", ChildStepID: "pack2"},
		{ParentStepID: "knit1", ChildStepID: "dye1"},
	}, nil).Times(1)

	// Root is the lexicographically-smallest producing step (pack1); only pack1 and its
	// ancestor knit1 are relevant. Setting expectations for exactly these two steps makes
	// any call for pack2/knit2/dye1 an unexpected-call failure.
	for _, id := range []string{"pack1", "knit1"} {
		flowRepo.EXPECT().GetFlowStep(gomock.Any(), acct, id).
			Return(&domain.ProductionFlowStep{ID: id}, nil).Times(1)
		stepQueryRepo.EXPECT().Find(gomock.Any(), acct, id).
			Return(&domain.ProductionStepDetail{ID: id}, nil).Times(1)
	}

	steps, apiErr := assembleFlowForItem(context.Background(), flowRepo, stepQueryRepo, acct, "F")
	require.Nil(t, apiErr)

	got := make([]string, 0, len(steps))
	for _, s := range steps {
		got = append(got, s.ID)
	}
	sort.Strings(got)
	assert.Equal(t, []string{"knit1", "pack1"}, got, "flow must be the single chosen recipe, not both recipes or sibling branches")
}

// All quantities are in base units so ToBase is the identity, keeping the expected
// values hand-computable.
var baseFactors = map[string]domain.UnitFactors{"u": {IsBaseUnit: true}}

func batchQty(m int64) domain.BatchQuantity {
	return domain.BatchQuantity{Measure: decimal.NewFromInt(m), Unit: domain.LightUnit{ID: "u"}}
}

// One step making the finished good directly from a material is itself a material-only
// block: the batch is the finished good at the ordered quantity.
func TestComputeMaterialOnlyBlocks_SingleStep(t *testing.T) {
	t.Parallel()
	step := &domain.ProductionFlowStep{
		ID:           "S",
		Production:   domain.StepProduction{ProducedItem: domain.LightItem{ID: "F"}, Quantity: batchQty(2)},
		Consumptions: []domain.StepConsumption{{ConsumedItem: domain.LightItem{ID: "M", Type: "material"}, Quantity: batchQty(5)}},
	}
	out := map[string]*blockAccum{}
	computeMaterialOnlyBlocks([]*domain.ProductionFlowStep{step}, "F", decimal.NewFromInt(10), baseFactors, out)

	require.Len(t, out, 1)
	require.Contains(t, out, "F")
	// prodBase(2) × normFactor(1/2) × ordered(10) = 10.
	assert.True(t, out["F"].baseTotal.Equal(decimal.NewFromInt(10)), "got %s", out["F"].baseTotal)
}

// Two-step flow: B (source, material-only) produces intermediate I from material M;
// A (target) produces finished good F from I. Only B is a material-only block, so the
// single batch is for I — normalized and scaled by the ordered quantity of F.
func TestComputeMaterialOnlyBlocks_TwoStep(t *testing.T) {
	t.Parallel()
	stepB := &domain.ProductionFlowStep{
		ID:           "B",
		Production:   domain.StepProduction{ProducedItem: domain.LightItem{ID: "I"}, Quantity: batchQty(2)},
		Consumptions: []domain.StepConsumption{{ConsumedItem: domain.LightItem{ID: "M", Type: "material"}, Quantity: batchQty(4)}},
		OutStepIDs:   []string{"A"},
	}
	stepA := &domain.ProductionFlowStep{
		ID:           "A",
		Production:   domain.StepProduction{ProducedItem: domain.LightItem{ID: "F"}, Quantity: batchQty(3)},
		Consumptions: []domain.StepConsumption{{ConsumedItem: domain.LightItem{ID: "I", Type: "part"}, Quantity: batchQty(1)}},
		InStepIDs:    []string{"B"},
	}
	out := map[string]*blockAccum{}
	computeMaterialOnlyBlocks([]*domain.ProductionFlowStep{stepA, stepB}, "F", decimal.NewFromInt(6), baseFactors, out)

	require.Len(t, out, 1, "only the material-only block (I) should produce a batch, not F")
	require.Contains(t, out, "I")
	// normFactor(B) = ratio(1/2) × parent(1/3) = 1/6; prodBase(2) × 1/6 × ordered(6) = 2.
	// Rounded to the batch precision (the chained 1/6 division leaves float noise).
	assert.True(t, out["I"].baseTotal.Round(batchQuantityPrecision).Equal(decimal.NewFromInt(2)), "got %s", out["I"].baseTotal)
	// The finished good A is NOT a material-only block (it consumes a part, I).
	assert.NotContains(t, out, "F")
}

// A step that consumes a non-material (a part) and is not a source is not a material-only
// block, so it produces no batch.
func TestComputeMaterialOnlyBlocks_PartConsumptionNotBlock(t *testing.T) {
	t.Parallel()
	source := &domain.ProductionFlowStep{
		ID:           "src",
		Production:   domain.StepProduction{ProducedItem: domain.LightItem{ID: "P"}, Quantity: batchQty(1)},
		Consumptions: []domain.StepConsumption{{ConsumedItem: domain.LightItem{ID: "M", Type: "material"}, Quantity: batchQty(1)}},
		OutStepIDs:   []string{"mid"},
	}
	mid := &domain.ProductionFlowStep{
		ID:         "mid",
		Production: domain.StepProduction{ProducedItem: domain.LightItem{ID: "F"}, Quantity: batchQty(1)},
		Consumptions: []domain.StepConsumption{
			{ConsumedItem: domain.LightItem{ID: "P", Type: "part"}, Quantity: batchQty(1)},
			{ConsumedItem: domain.LightItem{ID: "M2", Type: "material"}, Quantity: batchQty(1)},
		},
		InStepIDs: []string{"src"},
	}
	out := map[string]*blockAccum{}
	computeMaterialOnlyBlocks([]*domain.ProductionFlowStep{mid, source}, "F", decimal.NewFromInt(1), baseFactors, out)

	// Only the source (P) is a material-only block; F's step mixes a part + a material.
	require.Contains(t, out, "P")
	assert.NotContains(t, out, "F")
}

// No target step (the item isn't produced by this flow) yields no batches.
func TestComputeMaterialOnlyBlocks_NoTarget(t *testing.T) {
	t.Parallel()
	step := &domain.ProductionFlowStep{
		ID:         "S",
		Production: domain.StepProduction{ProducedItem: domain.LightItem{ID: "OTHER"}, Quantity: batchQty(1)},
	}
	out := map[string]*blockAccum{}
	computeMaterialOnlyBlocks([]*domain.ProductionFlowStep{step}, "F", decimal.NewFromInt(1), baseFactors, out)
	assert.Empty(t, out)
}
