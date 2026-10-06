package service

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	apierror "github.com/open-mrp/api/shared/errors"
)

// A knit step makes twelve eaches of a component from twelve eaches of a $0.10 material; a sew step
// makes one each of the finished item from two dozen of the component. Each finished each therefore
// takes two knit runs: $2.40 of material.
const (
	msAccount   = "ac_multistep"
	msItem      = "it_finished"
	msComponent = "it_component"
	msMaterial  = "it_material"
	msSew       = "ps_sew"
	msKnit      = "ps_knit"
	msEach      = "un_ea"
	msDozen     = "un_dz"
	msGroup     = "ug_each"
)

type multistepFlow struct {
	sewQuantity         decimal.Decimal
	componentDrawn      decimal.Decimal
	componentDrawnRatio decimal.Decimal
}

func newMultistepCostSvc(t *testing.T, flow multistepFlow) *itemSvcImpl {
	t.Helper()
	ctrl := gomock.NewController(t)

	each := domain.LightUnit{ID: msEach, RatioNumerator: "1", RatioDenominator: "1"}
	sew := &domain.ProductionFlowStep{
		ID: msSew,
		Production: domain.StepProduction{
			ProducedItem: domain.LightItem{ID: msItem},
			Quantity:     domain.BatchQuantity{Measure: flow.sewQuantity, Unit: each},
		},
		LevelingFactor: "0", Allowances: "0",
	}
	knit := &domain.ProductionFlowStep{
		ID: msKnit,
		Production: domain.StepProduction{
			ProducedItem: domain.LightItem{ID: msComponent},
			Quantity:     domain.BatchQuantity{Measure: decimal.NewFromInt(12), Unit: each},
		},
		LevelingFactor: "0", Allowances: "0",
	}

	flowRepo := repositorymock.NewMockProductionFlowRepo(ctrl)
	flowRepo.EXPECT().FindStepsByProducedItem(gomock.Any(), msAccount, msItem).Return([]string{msSew}, nil).AnyTimes()
	flowRepo.EXPECT().GetAllStepEdgesForAccount(gomock.Any(), msAccount).
		Return([]domain.StepEdge{{ParentStepID: msKnit, ChildStepID: msSew}}, nil).AnyTimes()
	flowRepo.EXPECT().GetFlowStep(gomock.Any(), msAccount, msSew).Return(sew, nil).AnyTimes()
	flowRepo.EXPECT().GetFlowStep(gomock.Any(), msAccount, msKnit).Return(knit, nil).AnyTimes()

	itemRepo := repositorymock.NewMockItemRepo(ctrl)
	itemRepo.EXPECT().GetCostFlowConsumptions(gomock.Any(), msSew).Return([]domain.CostFlowConsumption{{
		ConsumedItemID:           msComponent,
		ConsumedItemType:         "part",
		ConsumptionQuantity:      flow.componentDrawn,
		ConsumptionUnitRatio:     flow.componentDrawnRatio,
		WasteQuantity:            decimal.Zero,
		WasteUnitRatio:           decimal.NewFromInt(1),
		UnitCostDenominatorRatio: decimal.NewFromInt(1),
	}}, nil).AnyTimes()
	itemRepo.EXPECT().GetCostFlowConsumptions(gomock.Any(), msKnit).Return([]domain.CostFlowConsumption{{
		ConsumedItemID:           msMaterial,
		ConsumedItemType:         "material",
		ConsumptionQuantity:      decimal.NewFromInt(12),
		ConsumptionUnitRatio:     decimal.NewFromInt(1),
		WasteQuantity:            decimal.Zero,
		WasteUnitRatio:           decimal.NewFromInt(1),
		UnitCost:                 decimal.RequireFromString("0.10"),
		UnitCostDenominatorRatio: decimal.NewFromInt(1),
	}}, nil).AnyTimes()
	itemRepo.EXPECT().GetStockingUnit(gomock.Any(), msAccount, msItem).
		Return(&domain.ItemStockingUnit{UnitGroupID: msGroup, BaseUnitID: msEach}, nil).AnyTimes()

	unitRepo := repositorymock.NewMockUnitRepo(ctrl)
	unitRepo.EXPECT().IsUnitInGroup(gomock.Any(), msGroup, msEach).Return(true, nil).AnyTimes()
	unitRepo.EXPECT().GetCurrencyBaseUnitID(gomock.Any()).Return("un_usd", nil).AnyTimes()

	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewProductionFlowRepo().Return(flowRepo).AnyTimes()
	repos.EXPECT().NewItemRepo().Return(itemRepo).AnyTimes()
	repos.EXPECT().NewUnitRepo().Return(unitRepo).AnyTimes()
	repos.EXPECT().NewUnitConversionRepo().Return(repositorymock.NewMockUnitConversionRepo(ctrl)).AnyTimes()

	return &itemSvcImpl{repos: repos}
}

// The component is drawn in dozens and made in eaches. Dividing the two as entered — 2 over 12 —
// costs a sixth of one knit run per finished each instead of two.
func TestComputeItemCosts_ScalesAnUpstreamStepThroughBothSidesUnits(t *testing.T) {
	t.Parallel()

	svc := newMultistepCostSvc(t, multistepFlow{
		sewQuantity:         decimal.NewFromInt(1),
		componentDrawn:      decimal.NewFromInt(2),
		componentDrawnRatio: decimal.NewFromInt(12),
	})

	costs, apiErr := svc.ComputeItemCosts(t.Context(), msAccount, msItem)
	require.Nil(t, apiErr)
	assert.True(t, decimal.RequireFromString(costs.DirectMaterialCost).Equal(decimal.RequireFromString("2.4")), "material = %s", costs.DirectMaterialCost)
	assert.True(t, decimal.RequireFromString(costs.TotalCost).Equal(decimal.RequireFromString("2.4")), "total = %s", costs.TotalCost)
	assert.Equal(t, msEach, costs.UnitID)
}

// The same flow written with the component drawn in eaches lands on the same cost.
func TestComputeItemCosts_UpstreamScalingDoesNotDependOnTheUnitWrittenDown(t *testing.T) {
	t.Parallel()

	svc := newMultistepCostSvc(t, multistepFlow{
		sewQuantity:         decimal.NewFromInt(1),
		componentDrawn:      decimal.NewFromInt(24),
		componentDrawnRatio: decimal.NewFromInt(1),
	})

	costs, apiErr := svc.ComputeItemCosts(t.Context(), msAccount, msItem)
	require.Nil(t, apiErr)
	assert.True(t, decimal.RequireFromString(costs.DirectMaterialCost).Equal(decimal.RequireFromString("2.4")), "material = %s", costs.DirectMaterialCost)
}

// A producing step that makes nothing cannot be costed per unit; that is the not-found a missing flow
// gets, not a server error.
func TestComputeItemCosts_ZeroProductionQuantityIsNotFound(t *testing.T) {
	t.Parallel()

	svc := newMultistepCostSvc(t, multistepFlow{
		sewQuantity:         decimal.Zero,
		componentDrawn:      decimal.NewFromInt(2),
		componentDrawnRatio: decimal.NewFromInt(12),
	})

	_, apiErr := svc.ComputeItemCosts(t.Context(), msAccount, msItem)
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeResourceNotFound, apiErr.Code)
}
