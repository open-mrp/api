package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleSupplierMaterialID = "suml_01jm4r6700f8nwq3v5hx2d9ktp"

// SupplierMaterial represents a link between a supplier and a material.
type SupplierMaterial struct {
	// The unique identifier for the supplier material.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=supplier_material"`
	// The material this supplier provides.
	Material *Material `json:"material" expandable:"true"`
	// The supplier's part number for this material.
	SupplierPartNumber string `json:"supplier_part_number" validate:"required"`
	// The supplier's description for this material.
	SupplierDescription *string `json:"supplier_description"`
	// Whether this supplier material is active.
	IsActive bool `json:"is_active"`
	// The timestamp when the supplier material was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the supplier material was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleSupplierPartNumber = "SUP-PART-001"

var SampleSupplierMaterial = &SupplierMaterial{
	ID:                 SampleSupplierMaterialID,
	Object:             constants.ObjectTypeSupplierMaterial,
	Material:           SampleMaterial,
	SupplierPartNumber: sampleSupplierPartNumber,
	IsActive:           true,
	CreatedAt:          timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:          timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*SupplierMaterial) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSupplierMaterial)
}
