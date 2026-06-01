package event

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/mediator"
	apierror "github.com/augno/api/shared/errors"
)

func (c *ExecuteProductionStepConsumer) updateInventoryWithAudit(
	ctx context.Context,
	accountID string,
	params domain.InventoryUpdateParams,
) *apierror.APIError {
	inventoryRepo := c.repos.NewInventoryMutationRepo()
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
		c.repos,
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
