package service

import (
	"context"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/scheduling"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

// GetItemLotDefault answers "how many, counted in what" for one item.
//
// This is the same precedence the solver applies, asked one item at a time so a person adding a batch by hand gets the lot the plan would have used. The greige case reads the greige-to-finished decomposition the last plan recorded rather than re-walking batch genealogy: the walk is expensive, and reading what the plan decided keeps the two answers from disagreeing.
func (s *itemSvcImpl) GetItemLotDefault(ctx context.Context, itemID string) (*domain.ItemLotDefault, *apierror.APIError) {
	ctx, span := itemSvcTracer.Start(ctx, "service.item.get_lot_default")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainItems, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	repo := s.repos.NewProductLineRepo()

	// The item has to exist and be ours before anything is resolved against it.
	item, apiErr := s.repos.NewItemRepo().Get(ctx, domain.GetItemParams{
		AccountID: accountID,
		ItemID:    itemID,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result := &domain.ItemLotDefault{ItemID: itemID}

	// 1. A per-item override changes the size but keeps the item's own unit.
	override, apiErr := repo.GetItemLotOverride(ctx, accountID, itemID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	unitID, apiErr := s.itemBaseUnitID(ctx, item)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if override > 0 {
		result.Quantity = override
		result.UnitID = unitID
		result.Source = scheduling.LotSourceItemOverride
		return result, nil
	}

	// 2. The item's own product line, for anything that is itself sellable.
	ownLine, apiErr := repo.GetProductLineLotForItem(ctx, accountID, itemID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if ownLine != nil {
		result.Quantity = ownLine.Quantity
		result.UnitID = ownLine.UnitID
		result.ProductLineID = ownLine.ProductLineID
		result.Source = scheduling.LotSourceProductLine
		return result, nil
	}

	// 3. What the item becomes. Greige is not sold and has no line of its own.
	inherited, apiErr := repo.GetDownstreamProductLineLot(ctx, accountID, itemID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if inherited != nil {
		result.Quantity = inherited.Quantity
		result.UnitID = inherited.UnitID
		result.ProductLineID = inherited.ProductLineID
		result.Source = scheduling.LotSourceDownstreamProductLine
		return result, nil
	}

	// 4. The account default, counted in the item's own unit.
	settings, apiErr := s.repos.NewProductionScheduleRepo().GetSettings(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if settings != nil && settings.DefaultLotUnits > 0 {
		result.Quantity = settings.DefaultLotUnits
		result.UnitID = unitID
		result.Source = scheduling.LotSourceAccountDefault
		return result, nil
	}

	// Nothing anywhere in the chain. The caller gets a unit and no quantity rather than a guessed one, so a form defaults to empty instead of to a number nobody chose.
	result.UnitID = unitID
	return result, nil
}

// itemBaseUnitID is the unit an item is counted in, taken from its category's unit group — the same source the rest of the system uses.
func (s *itemSvcImpl) itemBaseUnitID(ctx context.Context, item *domain.Item) (string, *apierror.APIError) {
	if item.ItemCategoryID == "" {
		return "", nil
	}
	unitID, _, apiErr := s.repos.NewItemRepo().GetCategoryBaseUnitID(ctx, item.ItemCategoryID)
	return unitID, apiErr
}
