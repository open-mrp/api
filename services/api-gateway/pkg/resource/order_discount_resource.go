package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

const SampleOrderDiscountID = "ords_qnbrjvq5ih2q"

// A code a buyer enters to take money off an order.
//
// Codes are unique within your account and matched case-insensitively.
type OrderDiscount struct {
	// Order discount ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=order_discount"`
	// Whether the code can be applied to new orders.
	//
	// `archived` codes can't be applied to new orders; orders that already used one keep their discount.
	Status constants.ArchiveStatus `json:"status" validate:"required"`
	// Display name of the discount, such as `Spring promotion`.
	Name string `json:"name" validate:"required"`
	// The code a buyer enters. Unique within your account, matched case-insensitively.
	Code string `json:"code" validate:"required"`
	// How much comes off.
	//
	// `percent_off`: a whole-number percent of the order total. `amount_off`: a fixed amount.
	Discount *Discount `json:"discount" validate:"required"`
	// How many orders this code has been applied to.
	TimesRedeemed int32 `json:"times_redeemed" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleOrderDiscount = &OrderDiscount{
	ID:            SampleOrderDiscountID,
	Object:        constants.ObjectTypeOrderDiscount,
	Status:        constants.ArchiveStatusActive,
	Name:          "10% Off",
	Code:          "SAVE10",
	Discount:      &Discount{Type: constants.DiscountTypePercentOff, PercentOff: new("10")},
	TimesRedeemed: 5,
	CreatedAt:     timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:     timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*OrderDiscount) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleOrderDiscount)
}
