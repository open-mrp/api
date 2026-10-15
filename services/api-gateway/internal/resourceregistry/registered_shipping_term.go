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
		ObjectType: constants.ObjectTypeShippingTerm,
		Load:       resourceloaders.LoadShippingTerms,
		Subs: []include.SubField{
			{Key: "owner", Populate: populateOwnerOnShippingTerm},
			{
				Key:         "owner.account",
				Target:      constants.ObjectTypeAccount,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractOwnerAccountIDFromShippingTerm,
				Populate:    populateOwnerAccountOnShippingTerm,
			},
			{
				Key:         "free_shipping.service_levels",
				Target:      constants.ObjectTypeServiceLevel,
				Cardinality: include.CardinalityList,
				ExtractIDs:  extractFreeShippingServiceLevelIDsFromShippingTerm,
				Populate:    populateFreeShippingServiceLevelsOnShippingTerm,
			},
		},
	})
}

func populateOwnerOnShippingTerm(ctx context.Context, parent any, _ map[string]any) {
	st := parent.(*apiresource.ShippingTerm)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeShippingTerm, st.ID, "owner_account_id")
	st.Owner = buildOwnerShell(id)
}

func extractOwnerAccountIDFromShippingTerm(ctx context.Context, parent any) []string {
	st := parent.(*apiresource.ShippingTerm)
	id, _ := include.GetLoadMeta(ctx).
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
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeShippingTerm, st.ID, "owner_account_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		st.Owner.Account = v.(*apiresource.Account)
	}
}

func extractFreeShippingServiceLevelIDsFromShippingTerm(ctx context.Context, parent any) []string {
	st := parent.(*apiresource.ShippingTerm)
	ids, _ := include.GetLoadMeta(ctx).
		GetStrings(constants.ObjectTypeShippingTerm, st.ID, "free_shipping_service_level_ids")
	return ids
}

func populateFreeShippingServiceLevelsOnShippingTerm(ctx context.Context, parent any, loaded map[string]any) {
	st := parent.(*apiresource.ShippingTerm)
	if st.FreeShipping == nil {
		return
	}
	ids, _ := include.GetLoadMeta(ctx).
		GetStrings(constants.ObjectTypeShippingTerm, st.ID, "free_shipping_service_level_ids")

	items := make([]apiresource.ServiceLevel, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			items = append(items, *(v.(*apiresource.ServiceLevel)))
		}
	}
	st.FreeShipping.ServiceLevels = apiresource.NewList(items, apiresource.PageInfo{})
}
