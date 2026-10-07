package service

import (
	"context"

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
