package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// A business you purchase materials from, with its default billing and shipping addresses.
type SupplierDetail struct {
	// Supplier ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=supplier"`
	// The supplier's name, as shown in the dashboard and on documents.
	Name string `json:"name" validate:"required"`
	// Human-facing supplier code, unique per account (e.g. `SUP-001`).
	Number string `json:"number" validate:"required"`
	// Free-form notes about the supplier.
	Note *string `json:"note"`
	// The supplier's default billing address.
	//
	// A new address can be created inline when the supplier is created; afterwards this default is changed by passing `bill_to_address_id` to the update endpoint.
	BillToAddress *Address `json:"bill_to_address" expandable:"true"`
	// The supplier's default shipping address.
	//
	// When a supplier is created with only a bill-to address, that same address also becomes the default shipping address.
	ShipToAddress *Address `json:"ship_to_address" expandable:"true"`
	// Number of materials sourced from this supplier.
	//
	// Counts every material linked to the supplier, including links whose status is `inactive`.
	MaterialCount int64 `json:"material_count"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleSupplierDetail = &SupplierDetail{
	ID:            SampleSupplierID,
	Object:        constants.ObjectTypeSupplier,
	Name:          SampleSupplierName,
	Number:        SampleSupplierNumber,
	Note:          nil,
	BillToAddress: SampleAddress,
	ShipToAddress: SampleAddress,
	MaterialCount: 5,
	CreatedAt:     timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:     timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*SupplierDetail) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSupplierDetail)
}

// A condensed supplier returned by the supplier list endpoint.
//
// The supplier's note and its default bill-to and ship-to addresses are only available when a single supplier is retrieved.
type SupplierSummary struct {
	// Supplier ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=supplier_summary"`
	// The supplier's name, as shown in the dashboard and on documents.
	Name string `json:"name" validate:"required"`
	// Human-facing supplier code, unique per account (e.g. `SUP-001`).
	Number string `json:"number" validate:"required"`
	// Number of materials sourced from this supplier.
	//
	// Counts every material linked to the supplier, including links whose status is `inactive`.
	MaterialCount int64 `json:"material_count"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
}

var SampleSupplierSummary = &SupplierSummary{
	ID:            SampleSupplierID,
	Object:        constants.ObjectTypeSupplierSummary,
	Name:          SampleSupplierName,
	Number:        SampleSupplierNumber,
	MaterialCount: 5,
	CreatedAt:     timeutil.TimestampToTime(sampleCreatedAtTimestamp),
}

func (*SupplierSummary) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSupplierSummary)
}
