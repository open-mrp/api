package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
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
	// Display name, shown to customers at checkout when the service level is visible.
	Name string `json:"name" validate:"required"`
	// Code identifying the service, such as `fedex_ground`.
	//
	// Unique among the carrier's service levels. For connected carriers it's the carrier's own code, used for rates and labels.
	Code constants.ServiceLevelCode `json:"code" validate:"required"`
	// Whether customers can see and pick this service level at checkout in the customer portal.
	PortalVisibility constants.CustomerPortalVisibility `json:"portal_visibility" validate:"required"`
	// Whether the service level can be newly assigned.
	//
	// `active` service levels can be set on orders and customer defaults. `archived` service levels stay on the records that use them but can't be newly set.
	Status constants.ArchiveStatus `json:"status" validate:"required"`
	// Business days this service usually takes, used when the carrier can't quote a lane.
	//
	// Works an order's ship-by date back from its promised date. `null` means unknown.
	EstimatedTransitDays *int32 `json:"estimated_transit_days"`
	// Provenance of this service level.
	//
	// System-owned service levels are platform-provided defaults that cannot be updated or deleted; account-owned service levels are custom to your account.
	Owner *Owner `json:"owner" expandable:"true"`
	// The carrier offering this service level.
	Carrier *Carrier `json:"carrier" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleServiceLevelTransitDays = int32(3)

var SampleServiceLevel = &ServiceLevel{
	ID:                   SampleServiceLevelID,
	Object:               constants.ObjectTypeServiceLevel,
	Name:                 SampleServiceLevelName,
	Code:                 "fedex_ground",
	PortalVisibility:     constants.CustomerPortalVisibilityVisible,
	Status:               constants.ArchiveStatusActive,
	EstimatedTransitDays: &sampleServiceLevelTransitDays,
	Owner:                SampleOwnerAccount,
	CreatedAt:            timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:            timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ServiceLevel) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleServiceLevel)
}
