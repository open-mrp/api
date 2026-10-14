package productionstepep

// productionStepIncludes is every include a production step endpoint exposes: enough to cost a step client-side
// (rate units, the produced quantity's unit, and each item's unit group and unit cost).
var productionStepIncludes = []string{
	"production", "production.produced_item", "production.produced_item.category", "production.produced_item.category.unit_group", "production.produced_item.category.unit_group.base_unit", "production.produced_item.category.unit_group.associated_units", "production.produced_item.category.unit_group.associated_units.unit", "production.produced_item.unit_cost", "consumptions", "consumptions.consumed_item", "consumptions.consumed_item.category", "consumptions.consumed_item.category.unit_group", "consumptions.consumed_item.category.unit_group.base_unit", "consumptions.consumed_item.category.unit_group.associated_units", "consumptions.consumed_item.category.unit_group.associated_units.unit", "consumptions.consumed_item.unit_cost", "machines", "machines.department", "scanning_station", "department", "in_steps", "out_steps",
}

// productionIncludes is every include a production endpoint exposes.
var productionIncludes = []string{
	"produced_item", "produced_item.category", "produced_item.category.unit_group", "produced_item.category.unit_group.base_unit", "produced_item.category.unit_group.associated_units", "produced_item.category.unit_group.associated_units.unit", "produced_item.unit_cost",
}
