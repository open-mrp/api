package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleSalesTargetID = "ta_01jm4r6700f8nwq3v5hx2d9ktp"

// Sales target for an account user.
type SalesTarget struct {
	// Sales target ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_target"`
	// Start date.
	StartAt time.Time `json:"start_at" validate:"required"`
	// End date.
	EndAt time.Time `json:"end_at" validate:"required"`
	// Sales representative.
	SalesRep *User `json:"sales_rep" expandable:"true"`
	// Target amount.
	Amount *Quantity `json:"amount" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
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
