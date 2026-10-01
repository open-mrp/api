package receivingorderep

import (
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	"github.com/open-mrp/api/shared/constants"
)

// receivingOrderLineIncludeFields are the sub-objects a receiving order line reveals on request.
//
// The item's unit group is offered because the receiving screens measure against it: a line is checked off in the unit it was ordered in, and the stocking dialog offers that group's units to put it away in.
var receivingOrderLineIncludeFields = []string{
	"item", "item.category", "item.category.unit_group", "item.category.unit_group.base_unit", "item.category.unit_group.associated_units", "item.category.unit_group.associated_units.unit",
	"order_line", "order_line.item", "order_line.quantity_ordered", "order_line.quantity_ordered.unit", "order_line.unit_price", "order_line.unit_price.numerator_unit", "order_line.unit_price.denominator_unit",
	"quantity", "quantity.unit", "quantity_ordered", "quantity_ordered.unit",
}

// receivingOrderIncludes is the include set of every endpoint that returns a receiving order. The actions return the same order the retrieve endpoint does, so a client can refresh its screen from the action's response rather than fetching the order again.
func receivingOrderIncludes() *apiendpoint.IncludeConfig {
	fields := []string{"supplier", "totals", "related", "related.purchase_order", "related.deliveries", "lines"}
	for _, f := range receivingOrderLineIncludeFields {
		fields = append(fields, "lines."+f)
	}
	return apiendpoint.IncludesFor(apiendpoint.IncludesParams{
		ObjectType: constants.ObjectTypeReceivingOrder,
		Fields:     fields,
	})
}

// receivingOrderLineIncludes is the include set of every endpoint that returns a single receiving order line.
func receivingOrderLineIncludes() *apiendpoint.IncludeConfig {
	return apiendpoint.IncludesFor(apiendpoint.IncludesParams{
		ObjectType: constants.ObjectTypeReceivingOrderLine,
		Fields:     receivingOrderLineIncludeFields,
	})
}
