package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// SupplierDetail represents a full supplier record returned by the API.
type SupplierDetail struct {
	// The unique identifier for the supplier.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=supplier"`
	// The display name of the supplier.
	Name string `json:"name" validate:"required"`
	// The supplier number.
	Number string `json:"number" validate:"required"`
	// Notes about the supplier.
	Note *string `json:"note"`
	// The default billing address.
	BillToAddress *Address `json:"bill_to_address" expandable:"true"`
	// The default shipping address.
	ShipToAddress *Address `json:"ship_to_address" expandable:"true"`
	// The number of materials associated with this supplier.
	MaterialCount int64 `json:"material_count"`
	// When this supplier was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this supplier was last updated.
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

// SupplierSummary represents a lightweight supplier record for list results.
type SupplierSummary struct {
	// The unique identifier for the supplier.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=supplier_summary"`
	// The display name of the supplier.
	Name string `json:"name" validate:"required"`
	// The supplier number.
	Number string `json:"number" validate:"required"`
	// The number of materials associated with this supplier.
	MaterialCount int64 `json:"material_count"`
	// When this supplier was created.
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
