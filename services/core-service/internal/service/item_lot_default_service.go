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

// maxLotInheritanceDepth bounds the walk from an intermediate item to something sellable. It matches the solver's own max_flow_depth default: past that the item being made and the line it is inherited from have little to do with each other.
const maxLotInheritanceDepth = 10

// GetItemLotDefault answers "how many, counted in what" for one item.
//
// This is the same precedence the solver applies, asked one item at a time so a person adding a batch by hand gets the lot the plan would have used. The greige case prefers the greige-to-finished decomposition the last plan recorded — reading what the plan decided keeps the two answers from disagreeing, and it is demand-weighted — and falls back to the configured production flow for anything that plan never reached.
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

	// The item has to exist and be ours before anything is resolved against it.
	item, apiErr := s.repos.NewItemRepo().Get(ctx, domain.GetItemParams{
		AccountID: accountID,
		ItemID:    itemID,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	unitID, apiErr := s.itemBaseUnitID(ctx, item)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result, apiErr := resolveItemLotDefault(ctx, s.repos, accountID, itemID, unitID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return result, nil
}

// resolveItemLotDefault walks the lot precedence chain for one item.
//
// The chain, most specific first: a per-item override, the item's own product line, what the item becomes, and the account default. Shared rather than inlined so the lot a person is offered when adding a batch, and the lot a hand-added campaign is planned at, cannot disagree — a campaign released in sixties and a batch typed in at the account default would put two different-sized doffs of the same thing on the floor.
//
// unitID is the item's own counting unit, used by the rules that change how big a lot is without changing what it is counted in.
func resolveItemLotDefault(
	ctx context.Context,
	repos domain.RepoFactory,
	accountID, itemID, unitID string,
) (*domain.ItemLotDefault, *apierror.APIError) {
	repo := repos.NewProductLineRepo()
	result := &domain.ItemLotDefault{ItemID: itemID, UnitID: unitID}

	// 1. A per-item override changes the size but keeps the item's own unit.
	override, apiErr := repo.GetItemLotOverride(ctx, accountID, itemID)
	if apiErr != nil {
		return nil, apiErr
	}
	if override > 0 {
		result.Quantity = override
		result.Source = scheduling.LotSourceItemOverride
		return result, nil
	}

	// 2. The item's own product line, for anything that is itself sellable.
	ownLine, apiErr := repo.GetProductLineLotForItem(ctx, accountID, itemID)
	if apiErr != nil {
		return nil, apiErr
	}
	if ownLine != nil {
		result.Quantity = ownLine.Quantity
		result.UnitID = ownLine.UnitID
		result.ProductLineID = ownLine.ProductLineID
		result.Source = scheduling.LotSourceProductLine
		return result, nil
	}

	// 3. What the item becomes, as the last plan's greige decomposition recorded it. Greige is not sold and has no line of its own.
	inherited, apiErr := repo.GetDownstreamProductLineLot(ctx, accountID, itemID)
	if apiErr != nil {
		return nil, apiErr
	}
	if inherited != nil {
		result.Quantity = inherited.Quantity
		result.UnitID = inherited.UnitID
		result.ProductLineID = inherited.ProductLineID
		result.Source = scheduling.LotSourceDownstreamProductLine
		return result, nil
	}

	// 3b. What the flow says the item becomes, for anything the last plan did not reach.
	//
	// The demand-weighted lookup above only covers items a generated schedule decomposed. An account that has never run the solver, or an intermediate item that has never been produced, would otherwise fall straight past every line convention it has been given and land on the account default.
	flowInherited, apiErr := repo.GetFlowProductLineLot(ctx, accountID, itemID, maxLotInheritanceDepth)
	if apiErr != nil {
		return nil, apiErr
	}
	if flowInherited != nil {
		result.Quantity = flowInherited.Quantity
		result.UnitID = flowInherited.UnitID
		result.ProductLineID = flowInherited.ProductLineID
		result.Source = scheduling.LotSourceDownstreamProductLine
		return result, nil
	}

	// 4. The account default, counted in the item's own unit.
	settings, apiErr := repos.NewProductionScheduleRepo().GetSettings(ctx, accountID)
	if apiErr != nil {
		return nil, apiErr
	}
	if settings != nil && settings.DefaultLotUnits > 0 {
		result.Quantity = settings.DefaultLotUnits
		result.Source = scheduling.LotSourceAccountDefault
		return result, nil
	}

	// Nothing anywhere in the chain. The caller gets a unit and no quantity rather than a guessed one, so a form defaults to empty instead of to a number nobody chose.
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
