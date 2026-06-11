package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleServiceLevelID = "crop_01cfaf03f104e90ef9680e2a30"
const SampleServiceLevelName = "FedEx Ground"

// Shipping service level for a carrier.
type ServiceLevel struct {
	// Service level ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=service_level"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Carrier-specific code identifying this service level (e.g. `fedex_ground`, `ups_next_day_air`).
	//
	// Values are carrier-defined, so any non-empty string is accepted.
	ServiceLevelToken constants.ServiceLevelCode `json:"service_level_token" validate:"required"`
	// Whether this service level is shown to customers in the customer portal.
	//
	// - `visible`: customers can see and select this service level.
	// - `hidden`: the service level is concealed from the customer portal.
	CustomerPortalVisibility constants.CustomerPortalVisibility `json:"customer_portal_visibility" validate:"required"`
	// Whether this is the carrier's default service level, pre-selected when the carrier is chosen.
	IsDefault bool `json:"is_default"`
	// Owner.
	Owner *Owner `json:"owner" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleServiceLevel = &ServiceLevel{
	ID:                       SampleServiceLevelID,
	Object:                   constants.ObjectTypeServiceLevel,
	Name:                     SampleServiceLevelName,
	ServiceLevelToken:        "fedex_ground",
	CustomerPortalVisibility: constants.CustomerPortalVisibilityVisible,
	Owner:                    SampleOwnerAccount,
	CreatedAt:                timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:                timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ServiceLevel) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleServiceLevel)
}
