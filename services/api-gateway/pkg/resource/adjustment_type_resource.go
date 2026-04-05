package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleAdjustmentTypeID = "adjt_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleAdjustmentTypeName = "Discount"
const SampleAdjustmentTypeCode = string(constants.AdjustmentTypeDiscount)

// AdjustmentType represents a type of inventory adjustment.
type AdjustmentType struct {
	// The unique identifier for the adjustment type.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=adjustment_type"`
	// The display name of the adjustment type.
	Name string `json:"name" validate:"required"`
	// The machine-readable code for the adjustment type.
	Code constants.AdjustmentType `json:"code" validate:"required"`
	// The owner of this resource.
	Owner *Owner `json:"owner" expandable:"true"`
	// When this adjustment type was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this adjustment type was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleAdjustmentType = &AdjustmentType{
	ID:        SampleAdjustmentTypeID,
	Object:    constants.ObjectTypeAdjustmentType,
	Name:      SampleAdjustmentTypeName,
	Code:      constants.AdjustmentTypeDiscount,
	Owner:     SampleOwnerSystem,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*AdjustmentType) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAdjustmentType)
}
