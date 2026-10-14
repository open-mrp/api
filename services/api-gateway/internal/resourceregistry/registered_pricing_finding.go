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
		ObjectType: constants.ObjectTypeCustomerPricingFinding,
		Load:       resourceloaders.LoadCustomerPricingFindings,
		Subs: []include.SubField{
			{
				Key:         "customer",
				Target:      constants.ObjectTypeCustomer,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractIDFromPricingFinding("customer_id"),
				Populate: func(ctx context.Context, parent any, loaded map[string]any) {
					f := parent.(*apiresource.CustomerPricingFinding)
					if v, ok := loadedOne(ctx, constants.ObjectTypeCustomerPricingFinding, f.ID, "customer_id", loaded); ok {
						f.Customer = v.(*apiresource.Customer)
					}
				},
			},
			{
				Key:         "product_line",
				Target:      constants.ObjectTypeProductLine,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractIDFromPricingFinding("product_line_id"),
				Populate: func(ctx context.Context, parent any, loaded map[string]any) {
					f := parent.(*apiresource.CustomerPricingFinding)
					if v, ok := loadedOne(ctx, constants.ObjectTypeCustomerPricingFinding, f.ID, "product_line_id", loaded); ok {
						f.ProductLine = v.(*apiresource.ProductLine)
					}
				},
			},
			{
				Key:         "attributes",
				Target:      constants.ObjectTypeAttribute,
				Cardinality: include.CardinalityList,
				ExtractIDs:  extractIDsFromPricingFinding("attribute_ids"),
				Populate: func(ctx context.Context, parent any, loaded map[string]any) {
					f := parent.(*apiresource.CustomerPricingFinding)
					ids, _ := include.GetLoadMeta(ctx).GetStrings(constants.ObjectTypeCustomerPricingFinding, f.ID, "attribute_ids")
					items := make([]apiresource.Attribute, 0, len(ids))
					for _, id := range ids {
						if v, ok := loaded[id]; ok {
							items = append(items, *(v.(*apiresource.Attribute)))
						}
					}
					f.Attributes = apiresource.NewList(items, apiresource.PageInfo{})
				},
			},
		},
	})

	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeRealizedMarginFinding,
		Load:       resourceloaders.LoadRealizedMarginFindings,
		Subs: []include.SubField{
			{
				Key:         "customer",
				Target:      constants.ObjectTypeCustomer,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractIDFromMarginFinding("customer_id"),
				Populate: func(ctx context.Context, parent any, loaded map[string]any) {
					f := parent.(*apiresource.RealizedMarginFinding)
					if v, ok := loadedOne(ctx, constants.ObjectTypeRealizedMarginFinding, f.ID, "customer_id", loaded); ok {
						f.Customer = v.(*apiresource.Customer)
					}
				},
			},
			{
				Key:         "customer_group",
				Target:      constants.ObjectTypeAccountGroup,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractIDFromMarginFinding("customer_group_id"),
				Populate: func(ctx context.Context, parent any, loaded map[string]any) {
					f := parent.(*apiresource.RealizedMarginFinding)
					if v, ok := loadedOne(ctx, constants.ObjectTypeRealizedMarginFinding, f.ID, "customer_group_id", loaded); ok {
						f.CustomerGroup = v.(*apiresource.AccountGroup)
					}
				},
			},
			{
				Key:         "item",
				Target:      constants.ObjectTypeItem,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractIDFromMarginFinding("item_id"),
				Populate: func(ctx context.Context, parent any, loaded map[string]any) {
					f := parent.(*apiresource.RealizedMarginFinding)
					if v, ok := loadedOne(ctx, constants.ObjectTypeRealizedMarginFinding, f.ID, "item_id", loaded); ok {
						f.Item = v.(*apiresource.Item)
					}
				},
			},
			{
				Key:         "product_line",
				Target:      constants.ObjectTypeProductLine,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractIDFromMarginFinding("product_line_id"),
				Populate: func(ctx context.Context, parent any, loaded map[string]any) {
					f := parent.(*apiresource.RealizedMarginFinding)
					if v, ok := loadedOne(ctx, constants.ObjectTypeRealizedMarginFinding, f.ID, "product_line_id", loaded); ok {
						f.ProductLine = v.(*apiresource.ProductLine)
					}
				},
			},
		},
	})
}

// loadedOne resolves the single FK stashed under key and returns the loaded resource for it.
func loadedOne(ctx context.Context, ot constants.ObjectType, id, key string, loaded map[string]any) (any, bool) {
	fk, _ := include.GetLoadMeta(ctx).GetString(ot, id, key)
	if fk == "" {
		return nil, false
	}
	v, ok := loaded[fk]
	return v, ok
}

func extractIDsFromPricingFinding(key string) func(context.Context, any) []string {
	return func(ctx context.Context, parent any) []string {
		f := parent.(*apiresource.CustomerPricingFinding)
		ids, _ := include.GetLoadMeta(ctx).GetStrings(constants.ObjectTypeCustomerPricingFinding, f.ID, key)
		return ids
	}
}

func extractIDFromMarginFinding(key string) func(context.Context, any) []string {
	return func(ctx context.Context, parent any) []string {
		f := parent.(*apiresource.RealizedMarginFinding)
		id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeRealizedMarginFinding, f.ID, key)
		if id == "" {
			return nil
		}
		return []string{id}
	}
}

func extractIDFromPricingFinding(key string) func(context.Context, any) []string {
	return func(ctx context.Context, parent any) []string {
		f := parent.(*apiresource.CustomerPricingFinding)
		id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeCustomerPricingFinding, f.ID, key)
		if id == "" {
			return nil
		}
		return []string{id}
	}
}
