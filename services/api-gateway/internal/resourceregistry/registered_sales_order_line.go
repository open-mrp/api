package resourceregistry

import (
	"context"

	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeSalesOrderLine,
		Load:       stubLoadSalesOrderLines,
		Subs: []resourcekit.SubField{
			{
				// Load the product via LoadProducts (rather than recursing into the prebuilt stub) so the product's item_id/product_line_id meta is stashed and nested includes (lines.product.item / lines.product.product_line) resolve.
				Key:         "product",
				Target:      constants.ObjectTypeProduct,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractProductIDFromSOLine,
				Populate:    populateProductOnSOLine,
			},
			{Key: "quantity_ordered", Populate: populateQuantityOrderedOnSOLine},
			{Key: "unit_price", Populate: populateUnitPriceOnSOLine},
			{Key: "unit_cost", Populate: populateUnitCostOnSOLine},
			{Key: "totals", Populate: populateTotalsOnSOLine},
		},
	})
}

func populateQuantityOrderedOnSOLine(ctx context.Context, parent any, _ map[string]any) {
	l := parent.(*apiresource.SalesOrderLine)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrderLine, l.ID, "quantity_ordered")
	if !ok {
		return
	}
	l.QuantityOrdered = v.(*apiresource.Quantity)
}

func populateUnitPriceOnSOLine(ctx context.Context, parent any, _ map[string]any) {
	l := parent.(*apiresource.SalesOrderLine)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrderLine, l.ID, "unit_price")
	if !ok {
		return
	}
	l.UnitPrice = v.(*apiresource.Rate)
}

func populateUnitCostOnSOLine(ctx context.Context, parent any, _ map[string]any) {
	l := parent.(*apiresource.SalesOrderLine)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrderLine, l.ID, "unit_cost")
	if !ok {
		return
	}
	l.UnitCost = v.(*apiresource.Rate)
}

func populateTotalsOnSOLine(ctx context.Context, parent any, _ map[string]any) {
	l := parent.(*apiresource.SalesOrderLine)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrderLine, l.ID, "totals")
	if !ok {
		return
	}
	l.Totals = v.(*apiresource.SalesOrderTotals)
}

func stubLoadSalesOrderLines(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, nil
}

func extractProductIDFromSOLine(_ context.Context, parent any) []string {
	l := parent.(*apiresource.SalesOrderLine)
	if l.Product == nil || l.Product.ID == "" {
		return nil
	}
	return []string{l.Product.ID}
}

func populateProductOnSOLine(_ context.Context, parent any, loaded map[string]any) {
	l := parent.(*apiresource.SalesOrderLine)
	if l.Product == nil {
		return
	}
	if v, ok := loaded[l.Product.ID]; ok {
		l.Product = v.(*apiresource.Product)
	}
}
