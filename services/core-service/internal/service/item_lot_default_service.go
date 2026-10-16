package service

import (
	"context"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/scheduling"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/appctx"
	"github.com/open-mrp/apikit/tracing"
)

// GetItemLotDefault answers "how many, counted in what" for one item.
//
// This is the same precedence the solver applies, asked one item at a time so a person adding a batch by hand gets the lot the plan would have used.
func (s *itemSvcImpl) GetItemLotDefault(ctx context.Context, itemID string) (*domain.ItemLotDefault, *apierror.APIError) {
	ctx, span := itemSvcTracer.Start(ctx, "service.item.get_lot_default")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
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

	unitID, apiErr := s.itemBaseUnitID(ctx, accountID, item)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result, apiErr := resolveItemLotDefault(ctx, s.repos, accountID, itemID, unitID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return result, nil
}

// resolveItemLotDefault walks the lot precedence chain for one item: a per-item override, the item's category, and the
// account default.
//
// Shared rather than inlined so the lot a person is offered when adding a batch, and the lot a hand-added campaign is
// planned at, cannot disagree. unitID is the item's own counting unit, used by the rules that change how big a lot is
// without changing what it is counted in.
func resolveItemLotDefault(
	ctx context.Context,
	repos domain.RepoFactory,
	accountID, itemID, unitID string,
) (*domain.ItemLotDefault, *apierror.APIError) {
	override, apiErr := repos.NewProductLineRepo().GetItemLotOverride(ctx, accountID, itemID)
	if apiErr != nil {
		return nil, apiErr
	}
	categoryLots, apiErr := repos.NewProductionScheduleInputRepo().ListItemCategoryLots(ctx, accountID, []string{itemID})
	if apiErr != nil {
		return nil, apiErr
	}
	settings, apiErr := repos.NewProductionScheduleRepo().GetSettings(ctx, accountID)
	if apiErr != nil {
		return nil, apiErr
	}

	in := scheduling.LotResolutionInput{
		ItemOverrides:     map[string]float64{itemID: override},
		CategoryLotByItem: categoryLots,
		UnitByItem:        map[string]string{itemID: unitID},
	}
	if settings != nil {
		in.AccountLotUnits = settings.DefaultLotUnits
	}

	result := &domain.ItemLotDefault{ItemID: itemID, UnitID: unitID}
	// Nothing anywhere in the chain leaves a unit and no quantity, so a form defaults to empty instead of to a number nobody chose.
	if lot, ok := scheduling.ResolveLotDefault(itemID, in); ok {
		result.Quantity = lot.Quantity
		result.UnitID = lot.UnitID
		result.Source = lot.Source
		result.ItemCategoryID = lot.ItemCategoryID
	}
	return result, nil
}

// itemBaseUnitID is the unit an item is counted in: its stocking unit.
func (s *itemSvcImpl) itemBaseUnitID(ctx context.Context, accountID string, item *domain.Item) (string, *apierror.APIError) {
	if item.ItemCategoryID == "" {
		return "", nil
	}
	stocking, apiErr := s.repos.NewItemRepo().GetStockingUnit(ctx, accountID, item.ID)
	if apiErr != nil {
		return "", apiErr
	}
	return stocking.BaseUnitID, nil
}
