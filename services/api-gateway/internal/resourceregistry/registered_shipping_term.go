package resourceregistry

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeShippingTerm,
		Load:       resourceloaders.LoadShippingTerms,
		Subs: []resourcekit.SubField{
			{Key: "owner", Populate: populateOwnerOnShippingTerm},
			{
				Key:         "owner.account",
				Target:      constants.ObjectTypeAccount,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractOwnerAccountIDFromShippingTerm,
				Populate:    populateOwnerAccountOnShippingTerm,
			},
			{
				Key:         "flat_rate.unit",
				Target:      constants.ObjectTypeUnit,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractFlatRateUnitIDFromShippingTerm,
				Populate:    populateFlatRateUnitOnShippingTerm,
			},
			{
				Key:         "minimum_order_value.unit",
				Target:      constants.ObjectTypeUnit,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractMinimumOrderValueUnitIDFromShippingTerm,
				Populate:    populateMinimumOrderValueUnitOnShippingTerm,
			},
			{
				Key:         "free_shipping_service_levels",
				Target:      constants.ObjectTypeServiceLevel,
				Cardinality: resourcekit.CardinalityList,
				ExtractIDs:  extractFreeShippingServiceLevelIDsFromShippingTerm,
				Populate:    populateFreeShippingServiceLevelsOnShippingTerm,
			},
		},
	})
}

func populateOwnerOnShippingTerm(ctx context.Context, parent any, _ map[string]any) {
	st := parent.(*apiresource.ShippingTerm)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeShippingTerm, st.ID, "owner_account_id")
	st.Owner = buildOwnerShell(id)
}

func extractOwnerAccountIDFromShippingTerm(ctx context.Context, parent any) []string {
	st := parent.(*apiresource.ShippingTerm)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeShippingTerm, st.ID, "owner_account_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateOwnerAccountOnShippingTerm(ctx context.Context, parent any, loaded map[string]any) {
	st := parent.(*apiresource.ShippingTerm)
	if st.Owner == nil {
		return
	}
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeShippingTerm, st.ID, "owner_account_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		st.Owner.Account = v.(*apiresource.Account)
	}
}

func extractFlatRateUnitIDFromShippingTerm(ctx context.Context, parent any) []string {
	st := parent.(*apiresource.ShippingTerm)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeShippingTerm, st.ID, "flat_rate_unit_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateFlatRateUnitOnShippingTerm(ctx context.Context, parent any, loaded map[string]any) {
	st := parent.(*apiresource.ShippingTerm)
	if st.FlatRate == nil {
		return
	}
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeShippingTerm, st.ID, "flat_rate_unit_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		st.FlatRate.Unit = v.(*apiresource.Unit)
	}
}

func extractMinimumOrderValueUnitIDFromShippingTerm(ctx context.Context, parent any) []string {
	st := parent.(*apiresource.ShippingTerm)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeShippingTerm, st.ID, "minimum_order_value_unit_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateMinimumOrderValueUnitOnShippingTerm(ctx context.Context, parent any, loaded map[string]any) {
	st := parent.(*apiresource.ShippingTerm)
	if st.MinimumOrderValue == nil {
		return
	}
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeShippingTerm, st.ID, "minimum_order_value_unit_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		st.MinimumOrderValue.Unit = v.(*apiresource.Unit)
	}
}

func extractFreeShippingServiceLevelIDsFromShippingTerm(ctx context.Context, parent any) []string {
	st := parent.(*apiresource.ShippingTerm)
	ids, _ := resourcekit.GetLoadMeta(ctx).
		GetStrings(constants.ObjectTypeShippingTerm, st.ID, "free_shipping_service_level_ids")
	return ids
}

func populateFreeShippingServiceLevelsOnShippingTerm(ctx context.Context, parent any, loaded map[string]any) {
	st := parent.(*apiresource.ShippingTerm)
	ids, _ := resourcekit.GetLoadMeta(ctx).
		GetStrings(constants.ObjectTypeShippingTerm, st.ID, "free_shipping_service_level_ids")

	items := make([]apiresource.ServiceLevel, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			items = append(items, *(v.(*apiresource.ServiceLevel)))
		}
	}
	st.FreeShippingServiceLevels = apiresource.NewList(items, apiresource.PageInfo{})
}
