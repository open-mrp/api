package event

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/mediator"
	apierror "github.com/open-mrp/api/shared/errors"
)

// updateInventoryWithAudit moves inventory and records the movement on the audit trail. Takes the
// repo factory rather than hanging off a consumer so the batch-scanned and execute-production-step
// consumers share one implementation while the latter is being retired.
func updateInventoryWithAudit(
	ctx context.Context,
	repos domain.RepoFactory,
	accountID string,
	params domain.InventoryUpdateParams,
) *apierror.APIError {
	inventoryRepo := repos.NewInventoryMutationRepo()
	if apiErr := inventoryRepo.UpdateInventory(ctx, params); apiErr != nil {
		return apiErr
	}

	var scanningStationID *string
	if params.ScanningStationID != "" {
		id := params.ScanningStationID
		scanningStationID = &id
	}

	mediator.RecordInventoryAuditTrailOrLog(
		ctx,
		repos,
		accountID,
		params.ItemID,
		params.Measure,
		params.UnitID,
		params.ActionType,
		scanningStationID,
		params.ResponsibleUserID,
	)
	return nil
}

func (c *ExecuteProductionStepConsumer) updateInventoryWithAudit(
	ctx context.Context,
	accountID string,
	params domain.InventoryUpdateParams,
) *apierror.APIError {
	return updateInventoryWithAudit(ctx, c.repos, accountID, params)
}
