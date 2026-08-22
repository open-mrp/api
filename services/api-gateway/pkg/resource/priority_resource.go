package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SamplePriorityID = "pi_dubkbqpnz45f"
const SamplePriorityCode = constants.PriorityCodeNormal
const SamplePriorityName = "Normal"

// Priority level used to order work on sales orders, purchase orders, and picks.
//
// The levels are platform-provided and the same for every account, so they cannot be created, renamed, or removed. A customer can carry a default priority that pre-fills new orders for them.
type Priority struct {
	// Priority ID.
	ID string `json:"id"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=priority"`
	// Machine-readable code identifying the priority level.
	//
	// Other resources refer to a priority by this code rather than by its ID, such as a sales order's `priority`, and it can be used in place of the ID when retrieving a priority.
	Code constants.PriorityCode `json:"code" validate:"required"`
	// Display name of the priority level.
	Name string `json:"name" validate:"required"`
	// Owner of this resource.
	//
	// Priorities are platform-provided and shared across all accounts, so the owner is always the OpenMRP system owner.
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
