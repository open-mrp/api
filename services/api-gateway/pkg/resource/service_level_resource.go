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
	// Service level token.
	ServiceLevelToken constants.ServiceLevelCode `json:"service_level_token" validate:"required"`
	// Customer portal visibility.
	CustomerPortalVisibility constants.CustomerPortalVisibility `json:"customer_portal_visibility" validate:"required"`
	// Default service level for the carrier.
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
