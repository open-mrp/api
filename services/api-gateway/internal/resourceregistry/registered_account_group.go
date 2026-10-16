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
		ObjectType: constants.ObjectTypeAccountGroup,
		Load:       resourceloaders.LoadAccountGroups,
		Subs: []include.SubField{
			{
				Key:         "price_lists",
				Target:      constants.ObjectTypePriceList,
				Cardinality: include.CardinalityList,
				ExtractIDs:  metaIDs(constants.ObjectTypeAccountGroup, func(p any) string { return p.(*apiresource.AccountGroup).ID }, resourceloaders.AccountGroupMetaPriceListIDs),
				Populate: func(ctx context.Context, parent any, loaded map[string]any) {
					g := parent.(*apiresource.AccountGroup)
					ids, _ := include.GetLoadMeta(ctx).GetStrings(constants.ObjectTypeAccountGroup, g.ID, resourceloaders.AccountGroupMetaPriceListIDs)
					g.PriceLists = apiresource.NewList(loadedInOrder[apiresource.PriceList](ids, loaded), apiresource.PageInfo{})
				},
			},
		},
	})
}
