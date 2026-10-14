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
		ObjectType: constants.ObjectTypeShipmentLine,
		Load:       resourceloaders.LoadShipmentLines,
		Subs: []include.SubField{
			// ExtractRefs (not a loader Target) because the line is already stashed by the shipment
			// presenter; it exists so the resolver descends into it for lines.sales_order_line.product.
			{
				Key:         "sales_order_line",
				Target:      constants.ObjectTypeSalesOrderLine,
				ExtractRefs: extractSalesOrderLineRefFromShipmentLine,
				Populate:    populateSalesOrderLineOnShipmentLine,
			},
			{
				Key:         "item",
				Target:      constants.ObjectTypeItem,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractItemIDFromShipmentLine,
				Populate:    populateItemOnShipmentLine,
			},
		},
	})
}

func extractItemIDFromShipmentLine(ctx context.Context, parent any) []string {
	l := parent.(*apiresource.ShipmentLine)
	id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeShipmentLine, l.ID, "item_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateItemOnShipmentLine(ctx context.Context, parent any, loaded map[string]any) {
	l := parent.(*apiresource.ShipmentLine)
	id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeShipmentLine, l.ID, "item_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		l.Item = v.(*apiresource.Item)
	}
}

func populateSalesOrderLineOnShipmentLine(ctx context.Context, parent any, _ map[string]any) {
	l := parent.(*apiresource.ShipmentLine)
	v, ok := include.GetLoadMeta(ctx).Get(constants.ObjectTypeShipmentLine, l.ID, "sales_order_line")
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
