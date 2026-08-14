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
		ObjectType: constants.ObjectTypeCustomerPricingFinding,
		Load:       resourceloaders.LoadCustomerPricingFindings,
		Subs: []resourcekit.SubField{
			{
				Key:         "customer",
				Target:      constants.ObjectTypeCustomer,
				Cardinality: resourcekit.CardinalityOnePtr,
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
				Cardinality: resourcekit.CardinalityOnePtr,
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
				Cardinality: resourcekit.CardinalityList,
				ExtractIDs:  extractIDsFromPricingFinding("attribute_ids"),
				Populate: func(ctx context.Context, parent any, loaded map[string]any) {
					f := parent.(*apiresource.CustomerPricingFinding)
					ids, _ := resourcekit.GetLoadMeta(ctx).GetStrings(constants.ObjectTypeCustomerPricingFinding, f.ID, "attribute_ids")
					items := make([]apiresource.Attribute, 0, len(ids))
					for _, id := range ids {
						if v, ok := loaded[id]; ok {
							items = append(items, *(v.(*apiresource.Attribute)))
						}
					}
					f.Attributes = apiresource.NewList(items, apiresource.PageInfo{})
				},
			},
			{
				Key:         "unit_price.numerator_unit",
				Target:      constants.ObjectTypeUnit,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractIDFromPricingFinding("numerator_unit_id"),
				Populate:    populateUnitOnPricingFinding("numerator_unit_id", true),
			},
			{
				Key:         "unit_price.denominator_unit",
				Target:      constants.ObjectTypeUnit,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractIDFromPricingFinding("denominator_unit_id"),
				Populate:    populateUnitOnPricingFinding("denominator_unit_id", false),
			},
		},
	})

	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeRealizedMarginFinding,
		Load:       resourceloaders.LoadRealizedMarginFindings,
		Subs: []resourcekit.SubField{
			{
				Key:         "customer",
				Target:      constants.ObjectTypeCustomer,
				Cardinality: resourcekit.CardinalityOnePtr,
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
				Cardinality: resourcekit.CardinalityOnePtr,
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
				Cardinality: resourcekit.CardinalityOnePtr,
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
				Cardinality: resourcekit.CardinalityOnePtr,
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
	fk, _ := resourcekit.GetLoadMeta(ctx).GetString(ot, id, key)
	if fk == "" {
		return nil, false
	}
	v, ok := loaded[fk]
	return v, ok
}

func extractIDFromPricingFinding(key string) func(context.Context, any) []string {
	return func(ctx context.Context, parent any) []string {
		f := parent.(*apiresource.CustomerPricingFinding)
		id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeCustomerPricingFinding, f.ID, key)
		if id == "" {
			return nil
		}
		return []string{id}
	}
}

func extractIDsFromPricingFinding(key string) func(context.Context, any) []string {
	return func(ctx context.Context, parent any) []string {
		f := parent.(*apiresource.CustomerPricingFinding)
		ids, _ := resourcekit.GetLoadMeta(ctx).GetStrings(constants.ObjectTypeCustomerPricingFinding, f.ID, key)
		return ids
	}
}

func extractIDFromMarginFinding(key string) func(context.Context, any) []string {
	return func(ctx context.Context, parent any) []string {
		f := parent.(*apiresource.RealizedMarginFinding)
		id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeRealizedMarginFinding, f.ID, key)
		if id == "" {
			return nil
		}
		return []string{id}
	}
}

// populateUnitOnPricingFinding writes a loaded unit onto the finding's price rate. The rate is always built by the presenter, so there is nothing to create here — only the unit to attach.
func populateUnitOnPricingFinding(key string, numerator bool) func(context.Context, any, map[string]any) {
	return func(ctx context.Context, parent any, loaded map[string]any) {
		f := parent.(*apiresource.CustomerPricingFinding)
		v, ok := loadedOne(ctx, constants.ObjectTypeCustomerPricingFinding, f.ID, key, loaded)
		if !ok {
			return
		}
		unit := v.(*apiresource.Unit)
		for _, rate := range []*apiresource.ComputedRate{f.UnitPrice, f.PeerMedianPrice} {
			if rate == nil {
				continue
			}
			if numerator {
				rate.NumeratorUnit = unit
			} else {
				rate.DenominatorUnit = unit
			}
		}
	}
}
