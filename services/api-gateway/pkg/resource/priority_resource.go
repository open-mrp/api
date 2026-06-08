package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SamplePriorityID = "pi_01fc435701244bb3978bfb77ff"
const SamplePriorityCode = constants.PriorityCodeNormal
const SamplePriorityName = "Normal"

// Priority level used by sales orders and picks.
type Priority struct {
	// Priority ID.
	ID string `json:"id"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=priority"`
	// Machine-readable code.
	Code constants.PriorityCode `json:"code" validate:"required"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Owner of this resource.
	Owner *Owner `json:"owner" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at"`
}

var SamplePriority = &Priority{
	ID:        SamplePriorityID,
	Object:    constants.ObjectTypePriority,
	Code:      SamplePriorityCode,
	Name:      SamplePriorityName,
	Owner:     SampleOwnerSystem,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Priority) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePriority)
}
