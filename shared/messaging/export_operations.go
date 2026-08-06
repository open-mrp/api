package messaging

import "github.com/augno/api/shared/contracts"

// names one async export, and stems its routing key, queue and inbox handler so those
// cannot drift apart. Persisted in the inbox: renaming one orphans in-flight records.
type ExportOperation string

// builds the AMQP command routing key
func (o ExportOperation) RoutingKey() contracts.AmqpRoutingKey {
	return contracts.AmqpRoutingKey("core.cmd." + o)
}

// builds the command queue name
func (o ExportOperation) Queue() string {
	return "core_cmd_" + o.String()
}

// builds the inbox handler key, which is part of the persisted dedup identity
func (o ExportOperation) Handler() string {
	return "core." + o.String()
}

func (o ExportOperation) String() string {
	return string(o)
}

// The canonical export operations
const (
	ExportUnits            ExportOperation = "export_units"
	ExportUnitGroups       ExportOperation = "export_unit_groups"
	ExportProductLines     ExportOperation = "export_product_lines"
	ExportItemCategories   ExportOperation = "export_item_categories"
	ExportDepartments      ExportOperation = "export_departments"
	ExportLocations        ExportOperation = "export_locations"
	ExportMachines         ExportOperation = "export_machines"
	ExportScanningStations ExportOperation = "export_scanning_stations"
	ExportProductionRuns   ExportOperation = "export_production_runs"
	ExportProductionSteps  ExportOperation = "export_production_steps"
	ExportParts            ExportOperation = "export_parts"
	ExportProducts         ExportOperation = "export_products"
	ExportMaterials        ExportOperation = "export_materials"
	ExportProperties       ExportOperation = "export_properties"
)

// lists every registered export; the rabbitmq bindings declare a queue per entry and the
// core-service wiring pairs each with the resource it renders
var ExportOperations = []ExportOperation{
	ExportUnits,
	ExportUnitGroups,
	ExportProductLines,
	ExportItemCategories,
	ExportDepartments,
	ExportLocations,
	ExportMachines,
	ExportScanningStations,
	ExportProductionRuns,
	ExportProductionSteps,
	ExportParts,
	ExportProducts,
	ExportMaterials,
	ExportProperties,
}

// finds the export command for a resource slug, an export being named for its resource.
// The second return is false for an unregistered resource, which is a wiring mistake.
func ExportOperationFor(resourceSlug string) (ExportOperation, bool) {
	candidate := ExportOperation("export_" + resourceSlug)
	for _, op := range ExportOperations {
		if op == candidate {
			return op, true
		}
	}
	return "", false
}
