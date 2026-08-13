package messaging

import "github.com/augno/api/shared/contracts"

// names one async bulk operation, and stems its routing key, queue and inbox handler so
// those cannot drift apart. Persisted in the inbox: renaming one orphans in-flight records.
type BulkOperation string

// builds the AMQP command routing key
func (o BulkOperation) RoutingKey() contracts.AmqpRoutingKey {
	return contracts.AmqpRoutingKey("core.cmd." + o)
}

// builds the command queue name
func (o BulkOperation) Queue() string {
	return "core_cmd_" + o.String()
}

// builds the inbox handler key, which is part of the persisted dedup identity
func (o BulkOperation) Handler() string {
	return "core." + o.String()
}

func (o BulkOperation) String() string {
	return string(o)
}

// The canonical bulk operations
const (
	BulkCreateProductionRuns   BulkOperation = "bulk_create_production_runs"
	BulkUpsertProductionSteps  BulkOperation = "bulk_upsert_production_steps"
	BulkUpsertUnits            BulkOperation = "bulk_upsert_units"
	BulkUpsertUnitGroups       BulkOperation = "bulk_upsert_unit_groups"
	BulkUpsertLocations        BulkOperation = "bulk_upsert_locations"
	BulkUpsertDepartments      BulkOperation = "bulk_upsert_departments"
	BulkUpsertMachines         BulkOperation = "bulk_upsert_machines"
	BulkUpsertProductLines     BulkOperation = "bulk_upsert_product_lines"
	BulkUpsertScanningStations BulkOperation = "bulk_upsert_scanning_stations"
	BulkUpsertItemCategories   BulkOperation = "bulk_upsert_item_categories"
	BulkUpsertParts            BulkOperation = "bulk_upsert_parts"
	BulkUpsertProducts         BulkOperation = "bulk_upsert_products"
	BulkUpsertMaterials        BulkOperation = "bulk_upsert_materials"
	BulkUpsertProperties       BulkOperation = "bulk_upsert_properties"
	// BulkResolveHubspotCompanyReviews applies many company-match decisions at once, the path a reviewed spreadsheet comes back through.
	BulkResolveHubspotCompanyReviews BulkOperation = "bulk_resolve_hubspot_company_reviews"
)

// lists every registered bulk operation; the rabbitmq bindings declare a queue per entry
// and the core-service wiring pairs each with its executor
var BulkOperations = []BulkOperation{
	BulkCreateProductionRuns,
	BulkUpsertProductionSteps,
	BulkUpsertUnits,
	BulkUpsertUnitGroups,
	BulkUpsertLocations,
	BulkUpsertDepartments,
	BulkUpsertMachines,
	BulkUpsertProductLines,
	BulkUpsertScanningStations,
	BulkUpsertItemCategories,
	BulkUpsertParts,
	BulkUpsertProducts,
	BulkUpsertMaterials,
	BulkUpsertProperties,
	BulkResolveHubspotCompanyReviews,
}
