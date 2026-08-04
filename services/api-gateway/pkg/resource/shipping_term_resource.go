package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleShippingTermID = "shtm_c5gxy05whw6r"
const SampleShippingTermName = "Prepaid"

// A named freight pricing rule that decides what a buyer pays for shipping.
//
// A customer's default shipping term is evaluated whenever freight is quoted for one of their orders. Freight exemptions on the customer, its type group, or any of its price groups are checked first and zero the freight charge before the shipping term is considered.
type ShippingTerm struct {
	// Shipping term ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=shipping_term"`
	// Human-readable name for the shipping term, used to identify it when assigning shipping terms to customers and orders.
	Name string `json:"name" validate:"required"`
	// Freight pricing model applied by this shipping term.
	//
	// - `free_freight`: the buyer is never charged for shipping.
	// - `flat_rate_freight`: the buyer is charged the fixed amount in `flat_rate`, regardless of what the carrier would have charged.
	// - `carrier_rate_freight`: the buyer is charged the rate the carrier quotes for the order's carrier and service level.
	Type constants.ShippingTermType `json:"type" validate:"required"`
	// Provenance of this shipping term.
	//
	// System-owned shipping terms are platform-provided defaults shared across all accounts and cannot be updated or deleted; account-owned shipping terms are custom to your account.
	Owner *Owner `json:"owner" expandable:"true"`
	// Fixed shipping charge applied to the order.
	//
	// Used only when `type` is `flat_rate_freight`; ignored for other freight pricing models. A `flat_rate_freight` term with no flat rate falls through to the carrier's quoted rate.
	FlatRate *Quantity `json:"flat_rate"`
	// Order total a buyer must exceed for this term's free-shipping rules to apply.
	//
	// Above this total, freight is free for the service levels in `free_shipping_service_levels`.
	MinimumOrderValue *Quantity `json:"minimum_order_value"`
	// Service levels that ship for free once an order exceeds `minimum_order_value`.
	//
	// When this list is empty, every service level ships free above the threshold. When it is not empty, an order that picks a service level outside the list is not shipped free even above the threshold.
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
