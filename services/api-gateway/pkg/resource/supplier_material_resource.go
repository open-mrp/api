package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleSupplierMaterialID = "suml_gegrad0aqkhj"

// Links a material to a supplier that provides it, carrying the supplier's own part number and description for the material.
//
// Each material can be linked to a given supplier at most once.
type SupplierMaterial struct {
	// ID of the linked material, which also identifies this supplier material.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=supplier_material"`
	// Material provided by this supplier.
	Material *Material `json:"material" expandable:"true"`
	// The part number the supplier uses for this material in their own catalog.
	SupplierPartNumber string `json:"supplier_part_number" validate:"required"`
	// The supplier's own description of this material.
	SupplierDescription *string `json:"supplier_description"`
	// Whether this supplier is currently one you would source the material from.
	//
	// Inactive links are kept for reference and are still returned when listing or retrieving supplier materials; the status is a record-keeping flag and does not by itself prevent purchasing the material from this supplier.
	Status constants.SupplierMaterialStatus `json:"status" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleSupplierPartNumber = "SUP-PART-001"
var sampleSupplierDescription = "Cold-rolled steel sheet, 16 gauge, 48x96 in."

var SampleSupplierMaterial = &SupplierMaterial{
	ID:                  SampleSupplierMaterialID,
	Object:              constants.ObjectTypeSupplierMaterial,
	Material:            SampleMaterial,
	SupplierPartNumber:  sampleSupplierPartNumber,
	SupplierDescription: &sampleSupplierDescription,
	Status:              constants.SupplierMaterialStatusActive,
	CreatedAt:           timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:           timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*SupplierMaterial) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSupplierMaterial)
}
