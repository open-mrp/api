package resourceregistry

import (
	"context"

	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeProductLine,
		Load:       resourceloaders.LoadProductLines,
		Subs: []include.SubField{
			{Key: "owner", Populate: populateOwnerOnProductLine},
			{
				Key:         "owner.account",
				Target:      constants.ObjectTypeAccount,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractOwnerAccountIDFromProductLine,
				Populate:    populateOwnerAccountOnProductLine,
			},
			{
				Key:         "unit_group",
				Target:      constants.ObjectTypeUnitGroup,
				ExtractRefs: extractUnitGroupRefsFromProductLine,
				Populate:    populateUnitGroupOnProductLine,
			},
		},
	})
}

func populateOwnerOnProductLine(ctx context.Context, parent any, _ map[string]any) {
	pl := parent.(*apiresource.ProductLine)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeProductLine, pl.ID, "owner_account_id")
	pl.Owner = buildOwnerShell(id)
}

func extractOwnerAccountIDFromProductLine(ctx context.Context, parent any) []string {
	pl := parent.(*apiresource.ProductLine)
	id, _ := include.GetLoadMeta(ctx).
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
	id, _ := include.GetLoadMeta(ctx).
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
	// Unit groups are retired in 1.0.forge; only preview.9 callers still read them.
	if !resourceloaders.IsPreview9(ctx) {
		return
	}
	pl := parent.(*apiresource.ProductLine)
	v, ok := include.GetLoadMeta(ctx).
		Get(constants.ObjectTypeProductLine, pl.ID, "unit_group")
	if !ok || v == nil {
		return
	}
	pl.UnitGroup = v.(*apiresource.UnitGroup)
}
