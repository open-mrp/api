package service

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/event"
	apierror "github.com/open-mrp/api/shared/errors"
)

// publishStepCostBasisChanged announces that a step's inputs changed, naming the item the step
// produces. A step with no production has no cost to restate, so nothing is published for it.
func publishStepCostBasisChanged(ctx context.Context, repos domain.RepoFactory, accountID, productionStepID, reason string) *apierror.APIError {
	itemID, apiErr := repos.NewProductionStepQueryRepo().FindProducedItemID(ctx, accountID, productionStepID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return nil
		}
		return apiErr
	}
	return event.PublishItemCostBasisChanged(ctx, repos, accountID, itemID, reason)
}

func applyItemRatesPublishingCostInTx(txCtx context.Context, repos domain.RepoFactory, accountID, itemID, unitValueRateID, unitCostRateID string, unitPrice, unitCost *domain.CreateRateParams) *apierror.APIError {
	var before *domain.Rate
	if unitCost != nil {
		current, apiErr := repos.NewRateRepo().Get(txCtx, unitCostRateID)
		if apiErr != nil {
			return apiErr
		}
		before = current
	}
	if apiErr := applyItemRatesInTx(txCtx, repos, unitValueRateID, unitCostRateID, unitPrice, unitCost); apiErr != nil {
		return apiErr
	}
	if unitCost == nil || !rateDiffers(before, unitCost) {
		return nil
	}
	return event.PublishItemCostBasisChanged(txCtx, repos, accountID, itemID, event.CostBasisUnitCostUpdated)
}

func rateDiffers(current *domain.Rate, next *domain.CreateRateParams) bool {
	if current.NumeratorUnitID != next.NumeratorUnitID || current.DenominatorUnitID != next.DenominatorUnitID {
		return true
	}
	a, errA := decimal.NewFromString(current.Value)
	b, errB := decimal.NewFromString(next.Value)
	if errA != nil || errB != nil {
		return current.Value != next.Value
	}
	return !a.Equal(b)
}
