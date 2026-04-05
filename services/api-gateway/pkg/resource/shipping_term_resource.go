package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleShippingTermID = "shtm_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleShippingTermName = "Prepaid"

// ShippingTerm represents a shipping term configuration.
type ShippingTerm struct {
	// The unique identifier for the shipping term.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=shipping_term"`
	// The display name of the shipping term.
	Name string `json:"name" validate:"required"`
	// The shipping term type.
	Type constants.ShippingTermType `json:"type" validate:"required"`
	// The owner of this resource.
	Owner *Owner `json:"owner" expandable:"true"`
	// The flat rate quantity for this shipping term, if any.
	FlatRate *Quantity `json:"flat_rate"`
	// The minimum order value quantity for this shipping term, if any.
	MinimumOrderValue *Quantity `json:"minimum_order_value"`
	// The service levels that qualify for free shipping under this term.
	FreeShippingServiceLevels *List[ServiceLevel] `json:"free_shipping_service_levels" expandable:"true"`
	// When this shipping term was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this shipping term was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleShippingTerm = &ShippingTerm{
	ID:                        SampleShippingTermID,
	Object:                    constants.ObjectTypeShippingTerm,
	Name:                      SampleShippingTermName,
	Type:                      constants.ShippingTermTypeCarrierRateFreight,
	Owner:                     SampleOwnerSystem,
	FlatRate:                  nil,
	MinimumOrderValue:         nil,
	FreeShippingServiceLevels: NewList([]ServiceLevel{}, PageInfo{}),
	CreatedAt:                 timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:                 timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ShippingTerm) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleShippingTerm)
}
