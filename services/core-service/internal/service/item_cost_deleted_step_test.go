package service

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	apierror "github.com/open-mrp/api/shared/errors"
)

// The edge graph is read before the steps behind it are, so a step deleted in between is skipped. The
// edge that pointed at it must be dropped with it: the normalization walk follows in-edges and reads
// the step data behind each one, so an edge left pointing at a step that never loaded walked into a
// step with no data at all.
func TestComputeItemCosts_DeletedParentStepDoesNotFailTheRollup(t *testing.T) {
	t.Parallel()

	const (
		accountID = "ac_deleted_parent"
		itemID    = "itm_sewn"
		targetID  = "ps_sew"
		parentID  = "ps_knit"
		unitID    = "un_ea"
		currency  = "un_usd"
		groupID   = "ug_each"
	)

	ctrl := gomock.NewController(t)

	target := &domain.ProductionFlowStep{
		ID: targetID,
		Production: domain.StepProduction{
			ID:           "prod_target",
			ProducedItem: domain.LightItem{ID: itemID, SKU: "SEWN"},
			Quantity: domain.BatchQuantity{
				ID:      "qty_target",
				Measure: decimal.NewFromInt(1),
				Unit:    domain.LightUnit{ID: unitID},
			},
		},
		LevelingFactor: "0",
		Allowances:     "0",
	}

	flowRepo := repositorymock.NewMockProductionFlowRepo(ctrl)
	flowRepo.EXPECT().FindStepsByProducedItem(gomock.Any(), accountID, itemID).
		Return([]string{targetID}, nil).AnyTimes()
	// Knit feeds sew, so the walk back from sew picks the knit step up...
	flowRepo.EXPECT().GetAllStepEdgesForAccount(gomock.Any(), accountID).
		Return([]domain.StepEdge{{ParentStepID: parentID, ChildStepID: targetID}}, nil).AnyTimes()
	flowRepo.EXPECT().GetFlowStep(gomock.Any(), accountID, targetID).Return(target, nil).AnyTimes()
	// ...and by the time it is read, it has been deleted.
	flowRepo.EXPECT().GetFlowStep(gomock.Any(), accountID, parentID).
		Return(nil, apierror.NewResourceNotFoundError("Production step not found.")).AnyTimes()

	itemRepo := repositorymock.NewMockItemRepo(ctrl)
	itemRepo.EXPECT().GetCostFlowConsumptions(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	itemRepo.EXPECT().GetStockingUnit(gomock.Any(), accountID, itemID).
		Return(&domain.ItemStockingUnit{UnitGroupID: groupID, BaseUnitID: unitID}, nil).AnyTimes()

	stepQueryRepo := repositorymock.NewMockProductionStepQueryRepo(ctrl)
	stepQueryRepo.EXPECT().Find(gomock.Any(), accountID, gomock.Any()).
		Return(&domain.ProductionStepDetail{ID: targetID}, nil).AnyTimes()

	unitRepo := repositorymock.NewMockUnitRepo(ctrl)
	unitRepo.EXPECT().IsUnitInGroup(gomock.Any(), groupID, unitID).Return(true, nil).AnyTimes()
	unitRepo.EXPECT().GetCurrencyBaseUnitID(gomock.Any()).Return(currency, nil).AnyTimes()

	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewProductionFlowRepo().Return(flowRepo).AnyTimes()
	repos.EXPECT().NewItemRepo().Return(itemRepo).AnyTimes()
	repos.EXPECT().NewProductionStepQueryRepo().Return(stepQueryRepo).AnyTimes()
	repos.EXPECT().NewUnitRepo().Return(unitRepo).AnyTimes()
	repos.EXPECT().NewUnitConversionRepo().Return(repositorymock.NewMockUnitConversionRepo(ctrl)).AnyTimes()

	svc := &itemSvcImpl{repos: repos}

	costs, apiErr := svc.ComputeItemCosts(t.Context(), accountID, itemID)
	require.Nil(t, apiErr, "a step deleted mid-read is not part of the flow, and must not fail the rollup")
	require.NotNil(t, costs)
	require.Equal(t, unitID, costs.UnitID)
}
