package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleSalesTargetID = "ta_01jm4r6700f8nwq3v5hx2d9ktp"

// SalesTarget represents a sales target for an account user.
type SalesTarget struct {
	// The unique identifier for the sales target.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_target"`
	// The start date for this sales target.
	StartAt time.Time `json:"start_at" validate:"required"`
	// The end date for this sales target.
	EndAt time.Time `json:"end_at" validate:"required"`
	// The sales representative this target belongs to.
	SalesRep *User `json:"sales_rep" expandable:"true"`
	// The target amount. Contains the value and unit.
	Amount *Quantity `json:"amount" validate:"required"`
	// When the sales target was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When the sales target was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleSalesTarget = &SalesTarget{
	ID:        SampleSalesTargetID,
	Object:    constants.ObjectTypeSalesTarget,
	StartAt:   timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	EndAt:     timeutil.TimestampToTime(sampleExpiresAtTimestamp),
	SalesRep:  SampleUser,
	Amount:    SampleQuantity,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*SalesTarget) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSalesTarget)
}
