package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleProductLineID = "pl_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleProductLineName = "Industrial Fasteners"

// Product line resource.
type ProductLine struct {
	// Product line ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=product_line"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Description.
	Description *string `json:"description"`
	// Notes.
	Notes *string `json:"notes"`
	// Commission policy of products in this product line.
	CommissionPolicy constants.CommissionPolicy `json:"commission_policy" validate:"required"`
	// Freight policy for all items in this product line.
	FreightPolicy constants.FreightPolicy `json:"freight_policy" validate:"required"`
	// Owner of the product line.
	Owner *Owner `json:"owner" expandable:"true"`
	// Unit group associated with this product line. This unit group dictates the available units that products in this product line may embody in your production process.
	UnitGroup *UnitGroup `json:"unit_group" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last-updated timestamp.
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
