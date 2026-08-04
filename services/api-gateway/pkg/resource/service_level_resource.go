package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleServiceLevelID = "crop_4ilk9p6gccrx"
const SampleServiceLevelName = "FedEx Ground"

// A shipping speed or method offered by a carrier, such as ground or overnight.
//
// Carriers connected through Shippo have their service levels synced from the carrier itself; any carrier can also have service levels you create by hand.
type ServiceLevel struct {
	// Service level ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=service_level"`
	// Human-readable name for the service level, shown to customers at checkout when the service level is visible.
	Name string `json:"name" validate:"required"`
	// Carrier-specific code identifying this service level (e.g. `fedex_ground`, `ups_next_day_air`).
	//
	// For service levels synced from a connected carrier this is the carrier's own token, which is what rate shopping and label purchase are keyed on; for service levels you create yourself it is the `code` you supplied.
	ServiceLevelToken constants.ServiceLevelCode `json:"service_level_token" validate:"required"`
	// Whether customers can see and select this service level at checkout in the customer portal.
	CustomerPortalVisibility constants.CustomerPortalVisibility `json:"customer_portal_visibility" validate:"required"`
	// Whether this is the carrier's default service level, pre-selected when the carrier is chosen.
	//
	// Each carrier has at most one default; setting a new default clears the previous one. A default service level cannot be deleted until another service level takes its place or the flag is cleared.
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
