package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleShippingTermID = "shtm_014341ab4bb5bf94d5b6936f86"
const SampleShippingTermName = "Prepaid"

// ShippingTerm resource.
type ShippingTerm struct {
	// Shipping term ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=shipping_term"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Freight pricing model applied by this shipping term.
	//
	// - `free_freight`: no shipping cost to the buyer.
	// - `flat_rate_freight`: a fixed shipping cost regardless of order details (see `flat_rate`).
	// - `carrier_rate_freight`: shipping cost is determined by the carrier's quoted rate.
	Type constants.ShippingTermType `json:"type" validate:"required"`
	// Owner.
	Owner *Owner `json:"owner" expandable:"true"`
	// Fixed shipping charge applied to the order.
	//
	// Set when `type` is `flat_rate_freight`; null otherwise.
	FlatRate *Quantity `json:"flat_rate"`
	// Order subtotal a buyer must reach for this term's free-shipping rules to apply.
	//
	// Null if no threshold is configured.
	MinimumOrderValue *Quantity `json:"minimum_order_value"`
	// Service levels that ship for free under this term (typically once `minimum_order_value` is met).
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
