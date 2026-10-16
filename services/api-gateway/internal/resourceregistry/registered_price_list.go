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
		ObjectType: constants.ObjectTypePriceList,
		Load:       resourceloaders.LoadPriceLists,
		Subs: []include.SubField{
			{
				Key:         "rules",
				Target:      constants.ObjectTypePriceListRule,
				Cardinality: include.CardinalityList,
				ExtractIDs:  metaIDs(constants.ObjectTypePriceList, func(p any) string { return p.(*apiresource.PriceList).ID }, resourceloaders.PriceListMetaRuleIDs),
				Populate: func(ctx context.Context, parent any, loaded map[string]any) {
					l := parent.(*apiresource.PriceList)
					ids, _ := include.GetLoadMeta(ctx).GetStrings(constants.ObjectTypePriceList, l.ID, resourceloaders.PriceListMetaRuleIDs)
					l.Rules = apiresource.NewEmbeddedList(loadedInOrder[apiresource.PriceListRule](ids, loaded), false,
						resourceloaders.PriceListRulesPath(l.ID),
						metaCursor[apiresource.PriceListRule](ctx, constants.ObjectTypePriceList, l.ID, resourceloaders.PriceListMetaRulesNextCursor))
				},
			},
		},
	})

	ruleID := func(p any) string { return p.(*apiresource.PriceListRule).ID }
	scopeList := func(key, metaKey string, target constants.ObjectType, set func(*apiresource.PriceListRule, []string, map[string]any)) include.SubField {
		return include.SubField{
			Key:         key,
			Target:      target,
			Cardinality: include.CardinalityList,
			ExtractIDs:  metaIDs(constants.ObjectTypePriceListRule, ruleID, metaKey),
			Populate: func(ctx context.Context, parent any, loaded map[string]any) {
				r := parent.(*apiresource.PriceListRule)
				ids, _ := include.GetLoadMeta(ctx).GetStrings(constants.ObjectTypePriceListRule, r.ID, metaKey)
				set(r, ids, loaded)
			},
		}
	}
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypePriceListRule,
		Load:       resourceloaders.LoadPriceListRules,
		Subs: []include.SubField{
			scopeList("scope.items", resourceloaders.PriceListRuleMetaItemIDs, constants.ObjectTypeItem, func(r *apiresource.PriceListRule, ids []string, loaded map[string]any) {
				r.Scope.Items = apiresource.NewList(loadedInOrder[apiresource.Item](ids, loaded), apiresource.PageInfo{})
			}),
			scopeList("scope.product_lines", resourceloaders.PriceListRuleMetaLineIDs, constants.ObjectTypeProductLine, func(r *apiresource.PriceListRule, ids []string, loaded map[string]any) {
				r.Scope.ProductLines = apiresource.NewList(loadedInOrder[apiresource.ProductLine](ids, loaded), apiresource.PageInfo{})
			}),
			scopeList("scope.categories", resourceloaders.PriceListRuleMetaCategoryIDs, constants.ObjectTypeItemCategory, func(r *apiresource.PriceListRule, ids []string, loaded map[string]any) {
				r.Scope.Categories = apiresource.NewList(loadedInOrder[apiresource.ItemCategory](ids, loaded), apiresource.PageInfo{})
			}),
			scopeList("scope.attributes", resourceloaders.PriceListRuleMetaAttrIDs, constants.ObjectTypeAttribute, func(r *apiresource.PriceListRule, ids []string, loaded map[string]any) {
				r.Scope.Attributes = apiresource.NewList(loadedInOrder[apiresource.Attribute](ids, loaded), apiresource.PageInfo{})
			}),
		},
	})
}

// metaIDs extracts the IDs a parent's load meta holds under key.
func metaIDs(parentType constants.ObjectType, idOf func(any) string, key string) func(ctx context.Context, parent any) []string {
	return func(ctx context.Context, parent any) []string {
		ids, _ := include.GetLoadMeta(ctx).GetStrings(parentType, idOf(parent), key)
		return ids
	}
}

// loadedInOrder is the loaded records of ids, in ids' order, skipping any that didn't load.
func loadedInOrder[T any](ids []string, loaded map[string]any) []T {
	out := make([]T, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			out = append(out, *(v.(*T)))
		}
	}
	return out
}
