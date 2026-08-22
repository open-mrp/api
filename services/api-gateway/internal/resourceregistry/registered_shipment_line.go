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
		ObjectType: constants.ObjectTypeShipmentLine,
		Load:       resourceloaders.LoadShipmentLines,
		Subs: []resourcekit.SubField{
			// ExtractRefs (not a loader Target) because the line is already stashed by the shipment
			// presenter; it exists so the resolver descends into it for lines.sales_order_line.product.
			{
				Key:         "sales_order_line",
				Target:      constants.ObjectTypeSalesOrderLine,
				ExtractRefs: extractSalesOrderLineRefFromShipmentLine,
				Populate:    populateSalesOrderLineOnShipmentLine,
			},
		},
	})
}

func populateSalesOrderLineOnShipmentLine(ctx context.Context, parent any, _ map[string]any) {
	l := parent.(*apiresource.ShipmentLine)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeShipmentLine, l.ID, "sales_order_line")
	if !ok {
		return
	}
	l.SalesOrderLine = v.(*apiresource.SalesOrderLine)
}

func extractSalesOrderLineRefFromShipmentLine(_ context.Context, parent any) []any {
	l := parent.(*apiresource.ShipmentLine)
	if l.SalesOrderLine == nil {
		return nil
	}
	return []any{l.SalesOrderLine}
}
