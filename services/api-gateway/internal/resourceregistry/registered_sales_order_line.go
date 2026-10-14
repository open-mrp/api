package resourceregistry

import (
	"context"
	"github.com/open-mrp/apikit/include"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
)

func init() {
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeSalesOrderLine,
		Load:       stubLoadSalesOrderLines,
		Subs: []include.SubField{
			{
				// The product id is stashed in meta (not on a prebuilt stub) so the line's product field stays nil unless lines.product is requested. Loading via LoadProducts (rather than recursing into a stub) also stashes the product's item_id/product_line_id meta so nested includes (lines.product.item / lines.product.product_line) resolve.
				Key:         "product",
				Target:      constants.ObjectTypeProduct,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractProductIDFromSOLine,
				Populate:    populateProductOnSOLine,
			},
			// quantity_ordered / unit_price / unit_cost are populated inline from stashed
			// proto data (value + display_value). Target + ExtractRefs let the resolver
			// recurse into the populated Quantity/Rate so their own unit sub-fields
			// (quantity_ordered.unit, unit_price.numerator_unit, ...) resolve via LoadUnits —
			// the unit FKs are stashed on the Quantity/Rate in stashSalesOrderLineMeta. Mirrors
			// how customer.credit_limit (also a Quantity) enables credit_limit.unit.
			valueShapeSub(constants.ObjectTypeSalesOrderLine, "quantity_ordered", func(p any) string { return p.(*apiresource.SalesOrderLine).ID }, func(p, v any) {
				p.(*apiresource.SalesOrderLine).QuantityOrdered = v.(*apiresource.Quantity)
			}),
			valueShapeSub(constants.ObjectTypeSalesOrderLine, "unit_price", func(p any) string { return p.(*apiresource.SalesOrderLine).ID }, func(p, v any) {
				p.(*apiresource.SalesOrderLine).UnitPrice = v.(*apiresource.UnitPrice)
			}),
			valueShapeSub(constants.ObjectTypeSalesOrderLine, "unit_cost", func(p any) string { return p.(*apiresource.SalesOrderLine).ID }, func(p, v any) {
				p.(*apiresource.SalesOrderLine).UnitCost = v.(*apiresource.UnitPrice)
			}),
			{Key: "totals", Populate: populateTotalsOnSOLine},
		},
	})
}

func populateTotalsOnSOLine(ctx context.Context, parent any, _ map[string]any) {
	l := parent.(*apiresource.SalesOrderLine)
	v, ok := include.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrderLine, l.ID, "totals")
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
	v, ok := include.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrderLine, l.ID, "product_id")
	if !ok {
		return nil
	}
	return []string{v.(string)}
}

func populateProductOnSOLine(ctx context.Context, parent any, loaded map[string]any) {
	l := parent.(*apiresource.SalesOrderLine)
	v, ok := include.GetLoadMeta(ctx).Get(constants.ObjectTypeSalesOrderLine, l.ID, "product_id")
	if !ok {
		return
	}
	if p, ok := loaded[v.(string)]; ok {
		l.Product = p.(*apiresource.Product)
	}
}
