package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleProductLineID = "pl_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleProductLineName = "Industrial Fasteners"

// ProductLine represents a full product line resource.
type ProductLine struct {
	// The unique identifier for the product line.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=product_line"`
	// The display name of the product line.
	Name string `json:"name" validate:"required"`
	// Optional description of the product line.
	Description *string `json:"description"`
	// Optional notes about the product line.
	Notes *string `json:"notes"`
	// The commission policy for this product line.
	CommissionPolicy constants.CommissionPolicy `json:"commission_policy" validate:"required"`
	// The freight policy for this product line.
	FreightPolicy constants.FreightPolicy `json:"freight_policy" validate:"required"`
	// The owner of this resource.
	Owner *Owner `json:"owner" expandable:"true"`
	// The unit group associated with this product line.
	UnitGroup *UnitGroup `json:"unit_group" expandable:"true"`
	// The timestamp when the product line was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the product line was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleProductLine = &ProductLine{
	ID:               SampleProductLineID,
	Object:           constants.ObjectTypeProductLine,
	Name:             SampleProductLineName,
	CommissionPolicy: constants.CommissionPolicyExempt,
	FreightPolicy:    constants.FreightPolicyBilled,
	Owner:            SampleOwnerSystem,
	CreatedAt:        timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:        timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ProductLine) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductLine)
}
