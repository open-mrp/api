package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleSalesTargetID = "ta_0139fc283170d6e226c81719af"

// Sales target for an account user.
type SalesTarget struct {
	// Sales target ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_target"`
	// Start of the period this target applies to (inclusive).
	StartAt time.Time `json:"start_at" validate:"required"`
	// End of the period this target applies to (e.g. the close of a quarter).
	EndAt time.Time `json:"end_at" validate:"required"`
	// Sales representative the target is assigned to.
	//
	// Expandable: by default only the user reference is returned; request the full user object via `include[]=sales_rep`.
	SalesRep *User `json:"sales_rep" expandable:"true"`
	// Goal amount the representative is expected to reach over the period, expressed as a monetary quantity.
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
