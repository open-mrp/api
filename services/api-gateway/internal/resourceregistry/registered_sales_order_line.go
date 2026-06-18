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
				// The product ID is stashed in meta (not on a prebuilt stub) so
				// the line's `product` field stays nil unless lines.product is
				// requested. Loading via LoadProducts (rather than recursing into
				// a stub) also stashes the product's item_id/product_line_id meta
				// so nested includes (lines.product.item / lines.product.product_line) resolve.
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

func extractProductIDFromSOLine(ctx context.Context, parent any) []string {
	l := parent.(*apiresource.SalesOrderLine)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrderLine, l.ID, "product_id")
	if !ok {
		return nil
	}
	return []string{v.(string)}
}

func populateProductOnSOLine(ctx context.Context, parent any, loaded map[string]any) {
	l := parent.(*apiresource.SalesOrderLine)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrderLine, l.ID, "product_id")
	if !ok {
		return
	}
	if p, ok := loaded[v.(string)]; ok {
		l.Product = p.(*apiresource.Product)
	}
}
