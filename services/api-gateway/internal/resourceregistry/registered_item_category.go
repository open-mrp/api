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
		ObjectType: constants.ObjectTypeItemCategory,
		Load:       resourceloaders.LoadItemCategories,
		Subs: []resourcekit.SubField{
			{Key: "owner", Populate: populateOwnerOnItemCategory},
			{
				Key:         "owner.account",
				Target:      constants.ObjectTypeAccount,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractOwnerAccountIDFromItemCategory,
				Populate:    populateOwnerAccountOnItemCategory,
			},
			{Key: "properties", Populate: populatePropertiesOnItemCategory},
			{
				Key:         "unit_group",
				Target:      constants.ObjectTypeUnitGroup,
				ExtractRefs: extractUnitGroupRefsFromItemCategory,
				Populate:    populateUnitGroupOnItemCategory,
			},
		},
	})
}

func populateOwnerOnItemCategory(ctx context.Context, parent any, _ map[string]any) {
	ic := parent.(*apiresource.ItemCategory)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeItemCategory, ic.ID, "owner_account_id")
	ic.Owner = buildOwnerShell(id)
}

func extractOwnerAccountIDFromItemCategory(ctx context.Context, parent any) []string {
	ic := parent.(*apiresource.ItemCategory)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeItemCategory, ic.ID, "owner_account_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateOwnerAccountOnItemCategory(ctx context.Context, parent any, loaded map[string]any) {
	ic := parent.(*apiresource.ItemCategory)
	if ic.Owner == nil {
		return
	}
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeItemCategory, ic.ID, "owner_account_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		ic.Owner.Account = v.(*apiresource.Account)
	}
}

func populatePropertiesOnItemCategory(ctx context.Context, parent any, _ map[string]any) {
	ic := parent.(*apiresource.ItemCategory)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeItemCategory, ic.ID, "properties_list")
	if !ok || v == nil {
		return
	}
	ic.Properties = v.(*apiresource.List[apiresource.Property])
}

func extractUnitGroupRefsFromItemCategory(_ context.Context, parent any) []any {
	ic := parent.(*apiresource.ItemCategory)
	if ic.UnitGroup == nil {
		return nil
	}
	return []any{ic.UnitGroup}
}

func populateUnitGroupOnItemCategory(ctx context.Context, parent any, _ map[string]any) {
	ic := parent.(*apiresource.ItemCategory)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeItemCategory, ic.ID, "unit_group")
	if !ok || v == nil {
		return
	}
	ic.UnitGroup = v.(*apiresource.UnitGroup)
}
