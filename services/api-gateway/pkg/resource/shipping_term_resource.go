package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleShippingTermID = "shtm_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleShippingTermName = "Prepaid"

// ShippingTerm resource.
type ShippingTerm struct {
	// Shipping term ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=shipping_term"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Shipping term type.
	Type constants.ShippingTermType `json:"type" validate:"required"`
	// Owner.
	Owner *Owner `json:"owner" expandable:"true"`
	// Flat rate quantity, if any.
	FlatRate *Quantity `json:"flat_rate"`
	// Minimum order value quantity, if any.
	MinimumOrderValue *Quantity `json:"minimum_order_value"`
	// Service levels that qualify for free shipping.
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
