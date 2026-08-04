package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

// Record is a lightweight reference to a business record — a sales order, purchase order, pick, shipment, production run, invoice, etc.
//
// Like the `actor` and `entity` references, it carries just enough to identify and label the referenced record without embedding its full resource. The `status` and `metadata` fields hold type-specific detail that varies by the kind of record referenced.
type Record struct {
	// Unique identifier for the record.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=record"`
	// The kind of business record referenced.
	//
	// Determines how to resolve the record and which `status` and `metadata` keys may appear.
	//
	// - `sales_order`: a customer order.
	// - `purchase_order`: an order placed with a supplier.
	// - `receiving_order`: an inbound order being received into inventory.
	// - `pick`: a warehouse pick task.
	// - `shipment`: an outbound shipment.
	// - `delivery`: a delivery of one or more shipments to a destination.
	// - `production_run`: a manufacturing production run.
	// - `invoice`: a customer invoice.
	// - `transaction`: a payment or financial transaction.
	// - `settlement`: a settlement reconciling transactions against invoices.
	Type constants.RecordType `json:"type" validate:"required"`
	// Human-readable record number, when the record has one.
	Number *string `json:"number"`
	// Type-specific status code, when applicable.
	Status *string `json:"status"`
	// Type-specific metadata.
	//
	// The set of keys varies by record type.
	Metadata map[string]string `json:"metadata"`
}

// NewRecord constructs a Record reference for the given record type.
func NewRecord(id string, recordType constants.RecordType) *Record {
	return &Record{
		ID:     id,
		Object: constants.ObjectTypeRecord,
		Type:   recordType,
	}
}

var sampleRecordNumber = "SHIP-001"

var SampleRecord = &Record{
	ID:     SampleShipmentID,
	Object: constants.ObjectTypeRecord,
	Type:   constants.RecordTypeShipment,
	Number: &sampleRecordNumber,
	Status: new("fulfilled"),
	Metadata: map[string]string{
		"carrier":         "UPS",
		"tracking_number": "1Z999AA10123456784",
	},
}

func (*Record) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleRecord)
}
