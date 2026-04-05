package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleServiceLevelID = "crop_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleServiceLevelName = "FedEx Ground"

// ServiceLevel represents a shipping service level for a carrier.
type ServiceLevel struct {
	// The unique identifier for the service level.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=service_level"`
	// The display name of the service level.
	Name string `json:"name" validate:"required"`
	// The service level token identifying this shipping service level.
	ServiceLevelToken constants.ServiceLevelCode `json:"service_level_token" validate:"required"`
	// Whether this service level is visible in the customer portal.
	CustomerPortalVisibility constants.CustomerPortalVisibility `json:"customer_portal_visibility" validate:"required,enum"`
	// Whether this is the default service level for the carrier.
	IsDefault bool `json:"is_default"`
	// When the service level was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When the service level was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleServiceLevel = &ServiceLevel{
	ID:                       SampleServiceLevelID,
	Object:                   constants.ObjectTypeServiceLevel,
	Name:                     SampleServiceLevelName,
	ServiceLevelToken:        "fedex_ground",
	CustomerPortalVisibility: constants.CustomerPortalVisibilityVisible,
	CreatedAt:                timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:                timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ServiceLevel) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleServiceLevel)
}
