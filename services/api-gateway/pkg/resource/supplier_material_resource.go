package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleSupplierMaterialID = "suml_01e05d8a2821bf791bd30d537f"

// Supplier material resource.
type SupplierMaterial struct {
	// Supplier material ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=supplier_material"`
	// Material provided by this supplier.
	Material *Material `json:"material" expandable:"true"`
	// Supplier part number for this material.
	SupplierPartNumber string `json:"supplier_part_number" validate:"required"`
	// Supplier description for this material.
	SupplierDescription *string `json:"supplier_description"`
	// Whether this supplier can currently be sourced for the material.
	//
	// - `active`: the supplier is available to source this material.
	// - `inactive`: the link is retained for history but the supplier is not   considered when sourcing this material.
	Status constants.SupplierMaterialStatus `json:"status" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleSupplierPartNumber = "SUP-PART-001"

var SampleSupplierMaterial = &SupplierMaterial{
	ID:                 SampleSupplierMaterialID,
	Object:             constants.ObjectTypeSupplierMaterial,
	Material:           SampleMaterial,
	SupplierPartNumber: sampleSupplierPartNumber,
	Status:             constants.SupplierMaterialStatusActive,
	CreatedAt:          timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:          timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*SupplierMaterial) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSupplierMaterial)
}
