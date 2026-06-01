package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleAdjustmentTypeID = "adjt_01200338b135dc51aba62d4bf8"
const SampleAdjustmentTypeName = "Discount"
const SampleAdjustmentTypeCode = string(constants.AdjustmentTypeDiscount)

// Adjustment type resource.
type AdjustmentType struct {
	// Adjustment ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=adjustment_type"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Machine-readable code.
	Code constants.AdjustmentType `json:"code" validate:"required"`
	// Resource owner.
	Owner *Owner `json:"owner" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
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
