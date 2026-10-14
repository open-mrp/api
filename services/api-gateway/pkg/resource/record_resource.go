package apiresource

import (
	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
)

// A lightweight reference to another record, such as a sales order's shipments or the record a note is on.
//
// It carries enough to identify and label the record without embedding it. Fetch the full record by `type` and `id` when you need more.
type Record struct {
	// Unique identifier for the record.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=record"`
	// The kind of record referenced: the `object` of the full resource, such as `sales_order`, `invoice` or `job`.
	//
	// More types may be added; treat an unknown one as a record you can't open.
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

// Names one record by type and ID, for a request field that can point at records of several types.
type RecordInput struct {
	// The kind of record: the `object` of the full resource, such as `sales_order`.
	Type constants.RecordType `json:"type" validate:"required"`
	// ID of the record.
	ID string `json:"id" validate:"required"`
}

var SampleRecordInput = &RecordInput{Type: constants.RecordTypeSalesOrder, ID: SampleSalesOrderID}

func (*RecordInput) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleRecordInput)
}
