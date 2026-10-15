package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

const SampleShippingTermID = "shtm_c5gxy05whw6r"
const SampleShippingTermName = "Free over $500"

// How freight is charged on orders, assigned to customers and sales orders.
type ShippingTerm struct {
	// Shipping term ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=shipping_term"`
	// Display name, such as `Free over $500`.
	//
	// Unique (case-insensitive) among your account's terms and the system defaults.
	Name string `json:"name" validate:"required"`
	// How freight is charged.
	//
	// `free`: never charged. `flat_rate`: charged `flat_rate` on every order. `carrier_rate`: charged the carrier's quoted rate for the order's service level.
	Type constants.ShippingTermType `json:"type" validate:"required"`
	// Whether the term can be assigned to customers and sales orders.
	//
	// `archived` terms stay attached to the records already using them.
	Status constants.ArchiveStatus `json:"status" validate:"required"`
	// Freight charged on every order.
	//
	// Present only when `type` is `flat_rate`.
	FlatRate *Money `json:"flat_rate"`
	// Waives freight on orders over a minimum value.
	//
	// `null` when the term has no threshold. Not allowed when `type` is `free`.
	FreeShipping *ShippingTermFreeShipping `json:"free_shipping"`
	// Who owns this shipping term.
	//
	// `system` terms are shared defaults and are read-only; `account` terms belong to your account.
	Owner *Owner `json:"owner" expandable:"true"`
	// Time the shipping term was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Time the shipping term was last changed.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// When a shipping term waives freight.
type ShippingTermFreeShipping struct {
	// Order total that must be exceeded for freight to be waived.
	MinimumOrderValue Money `json:"minimum_order_value" validate:"required"`
	// Which service levels qualify for free freight.
	//
	// `all`: every service level. `selected`: only the levels in `service_levels`.
	ServiceLevelScope constants.ServiceLevelScope `json:"service_level_scope" validate:"required"`
	// Service levels that qualify for free freight when `service_level_scope` is `selected`. Empty when it is `all`.
	ServiceLevels *List[ServiceLevel] `json:"service_levels" expandable:"true"`
}

var SampleShippingTerm = &ShippingTerm{
	ID:       SampleShippingTermID,
	Object:   constants.ObjectTypeShippingTerm,
	Name:     SampleShippingTermName,
	Type:     constants.ShippingTermTypeCarrierRate,
	Status:   constants.ArchiveStatusActive,
	Owner:    SampleOwnerSystem,
	FlatRate: nil,
	FreeShipping: &ShippingTermFreeShipping{
		MinimumOrderValue: Money{Amount: "500.00", Currency: "usd"},
		ServiceLevelScope: constants.ServiceLevelScopeAll,
		ServiceLevels:     NewList([]ServiceLevel{}, PageInfo{}),
	},
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ShippingTerm) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleShippingTerm)
}
