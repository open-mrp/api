package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SamplePriorityID = "pi_01jm4r6700f8nwq3v5hx2d9ktp"
const SamplePriorityCode = constants.PriorityCodeNormal
const SamplePriorityName = "Normal"

// Priority represents a priority level used by sales orders and picks.
type Priority struct {
	// The unique identifier for the priority.
	ID string `json:"id"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=priority"`
	// The machine-readable code.
	Code constants.PriorityCode `json:"code" validate:"required"`
	// The display name of the priority.
	Name string `json:"name" validate:"required"`
	// The owner of this resource.
	Owner *Owner `json:"owner" expandable:"true"`
	// When this priority was created.
	CreatedAt time.Time `json:"created_at"`
	// When this priority was last updated.
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
