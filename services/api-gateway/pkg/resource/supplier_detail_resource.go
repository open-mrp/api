package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// SupplierDetail is the full supplier resource.
type SupplierDetail struct {
	// Supplier ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=supplier"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Human-facing supplier code, unique per account (e.g. `SUP-001`).
	Number string `json:"number" validate:"required"`
	// Free-form notes about the supplier.
	//
	// Null if none.
	Note *string `json:"note"`
	// Default billing address.
	BillToAddress *Address `json:"bill_to_address" expandable:"true"`
	// Default shipping address.
	ShipToAddress *Address `json:"ship_to_address" expandable:"true"`
	// Number of associated materials.
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

// SupplierSummary is the lightweight supplier resource for list results.
type SupplierSummary struct {
	// Supplier ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=supplier_summary"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Human-facing supplier code, unique per account (e.g. `SUP-001`).
	Number string `json:"number" validate:"required"`
	// Number of associated materials.
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
