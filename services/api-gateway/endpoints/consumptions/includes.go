package consumptionep

// consumptionEndpointIncludes is every include a consumption endpoint exposes.
var consumptionEndpointIncludes = []string{
	"consumed_item",
	"consumed_item.category",
	"consumed_item.category.unit_group",
	"consumed_item.category.unit_group.base_unit",
	"consumed_item.category.unit_group.associated_units",
	"consumed_item.category.unit_group.associated_units.unit",
	"consumed_item.unit_cost",
	"quantity.unit",
	"waste_quantity.unit",
}
