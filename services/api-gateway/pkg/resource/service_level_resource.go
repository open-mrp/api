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
	// Human-readable name for the service level, shown to customers at checkout when the service level is visible.
	Name string `json:"name" validate:"required"`
	// Carrier-specific code identifying this service level (e.g. `fedex_ground`, `ups_next_day_air`).
	//
	// Values are carrier-defined, so any non-empty string is accepted.
	ServiceLevelToken constants.ServiceLevelCode `json:"service_level_token" validate:"required"`
	// Whether customers can see and select this service level at checkout in the customer portal.
	CustomerPortalVisibility constants.CustomerPortalVisibility `json:"customer_portal_visibility" validate:"required"`
	// Whether this is the carrier's default service level, pre-selected when the carrier is chosen.
	//
	// Each carrier has at most one default; setting a new default clears the previous one.
	IsDefault bool `json:"is_default"`
	// Provenance of this service level.
	//
	// System-owned service levels are platform-provided defaults that cannot be updated or deleted; account-owned service levels are custom to your account.
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
	IsDefault:                true,
	Owner:                    SampleOwnerAccount,
	CreatedAt:                timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:                timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ServiceLevel) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleServiceLevel)
}
