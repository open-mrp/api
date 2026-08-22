package resourceregistry

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeProductLine,
		Load:       resourceloaders.LoadProductLines,
		Subs: []resourcekit.SubField{
			{Key: "owner", Populate: populateOwnerOnProductLine},
			{
				Key:         "owner.account",
				Target:      constants.ObjectTypeAccount,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractOwnerAccountIDFromProductLine,
				Populate:    populateOwnerAccountOnProductLine,
			},
			{
				Key:         "unit_group",
				Target:      constants.ObjectTypeUnitGroup,
				ExtractRefs: extractUnitGroupRefsFromProductLine,
				Populate:    populateUnitGroupOnProductLine,
			},
			// default_lot is a Quantity built inline by the loader; declaring its Target lets the resolver recurse into default_lot.unit, whose id is stashed under ObjectTypeQuantity. Same shape as a customer's credit_limit.
			{
				Key:         "default_lot",
				Target:      constants.ObjectTypeQuantity,
				ExtractRefs: extractDefaultLotRefFromProductLine,
				Populate:    populateDefaultLotOnProductLine,
			},
		},
	})
}

func populateOwnerOnProductLine(ctx context.Context, parent any, _ map[string]any) {
	pl := parent.(*apiresource.ProductLine)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeProductLine, pl.ID, "owner_account_id")
	pl.Owner = buildOwnerShell(id)
}

func extractOwnerAccountIDFromProductLine(ctx context.Context, parent any) []string {
	pl := parent.(*apiresource.ProductLine)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeProductLine, pl.ID, "owner_account_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateOwnerAccountOnProductLine(ctx context.Context, parent any, loaded map[string]any) {
	pl := parent.(*apiresource.ProductLine)
	if pl.Owner == nil {
		return
	}
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeProductLine, pl.ID, "owner_account_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		pl.Owner.Account = v.(*apiresource.Account)
	}
}

func extractUnitGroupRefsFromProductLine(_ context.Context, parent any) []any {
	pl := parent.(*apiresource.ProductLine)
	if pl.UnitGroup == nil {
		return nil
	}
	return []any{pl.UnitGroup}
}

func populateUnitGroupOnProductLine(ctx context.Context, parent any, _ map[string]any) {
	pl := parent.(*apiresource.ProductLine)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeProductLine, pl.ID, "unit_group")
	if !ok || v == nil {
		return
	}
	pl.UnitGroup = v.(*apiresource.UnitGroup)
}

func populateDefaultLotOnProductLine(ctx context.Context, parent any, _ map[string]any) {
	pl := parent.(*apiresource.ProductLine)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeProductLine, pl.ID, "default_lot")
	if !ok || v == nil {
		return
	}
	pl.DefaultLot = v.(*apiresource.Quantity)
}

// extractDefaultLotRefFromProductLine returns the inline lot Quantity so the resolver can recurse into default_lot.unit.
func extractDefaultLotRefFromProductLine(_ context.Context, parent any) []any {
	pl := parent.(*apiresource.ProductLine)
	if pl.DefaultLot == nil {
		return nil
	}
	return []any{pl.DefaultLot}
}
