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
		ObjectType: constants.ObjectTypeUnitGroup,
		Load:       resourceloaders.LoadUnitGroups,
		Subs: []resourcekit.SubField{
			{Key: "owner", Populate: populateOwnerOnUnitGroup},
			{
				Key:         "owner.account",
				Target:      constants.ObjectTypeAccount,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractOwnerAccountIDFromUnitGroup,
				Populate:    populateOwnerAccountOnUnitGroup,
			},
			{
				Key:         "base_unit",
				Target:      constants.ObjectTypeUnit,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractBaseUnitIDFromUnitGroup,
				Populate:    populateBaseUnitOnUnitGroup,
			},
			{
				Key:         "associated_units",
				Target:      constants.ObjectTypeUnitGroupUnit,
				ExtractRefs: extractAssociatedUnitRefsFromUnitGroup,
				Populate:    populateAssociatedUnitsOnUnitGroup,
			},
		},
	})
}

func populateOwnerOnUnitGroup(ctx context.Context, parent any, _ map[string]any) {
	ug := parent.(*apiresource.UnitGroup)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeUnitGroup, ug.ID, "owner_account_id")
	ug.Owner = buildOwnerShell(id)
}

func extractOwnerAccountIDFromUnitGroup(ctx context.Context, parent any) []string {
	ug := parent.(*apiresource.UnitGroup)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeUnitGroup, ug.ID, "owner_account_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateOwnerAccountOnUnitGroup(ctx context.Context, parent any, loaded map[string]any) {
	ug := parent.(*apiresource.UnitGroup)
	if ug.Owner == nil {
		return
	}
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeUnitGroup, ug.ID, "owner_account_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		ug.Owner.Account = v.(*apiresource.Account)
	}
}

func extractBaseUnitIDFromUnitGroup(ctx context.Context, parent any) []string {
	ug := parent.(*apiresource.UnitGroup)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeUnitGroup, ug.ID, "base_unit_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateBaseUnitOnUnitGroup(ctx context.Context, parent any, loaded map[string]any) {
	ug := parent.(*apiresource.UnitGroup)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeUnitGroup, ug.ID, "base_unit_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		ug.BaseUnit = v.(*apiresource.Unit)
	}
}

func extractAssociatedUnitRefsFromUnitGroup(_ context.Context, parent any) []any {
	ug := parent.(*apiresource.UnitGroup)
	if ug.AssociatedUnits == nil {
		return nil
	}
	refs := make([]any, len(ug.AssociatedUnits.Data))
	for i := range ug.AssociatedUnits.Data {
		refs[i] = &ug.AssociatedUnits.Data[i]
	}
	return refs
}

func populateAssociatedUnitsOnUnitGroup(ctx context.Context, parent any, _ map[string]any) {
	ug := parent.(*apiresource.UnitGroup)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeUnitGroup, ug.ID, "associated_units_data")
	if !ok || v == nil {
		ug.AssociatedUnits = apiresource.NewList([]apiresource.UnitGroupUnit{}, apiresource.PageInfo{})
		return
	}
	items := v.([]apiresource.UnitGroupUnit)
	ug.AssociatedUnits = apiresource.NewList(items, apiresource.PageInfo{})
}
