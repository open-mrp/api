package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleOrderDiscountID = "ords_qnbrjvq5ih2q"

// A discount code that can be applied to a sales order.
//
// An order discount reduces the order total by either a percentage or a fixed amount, depending on `discount_type`. The reduction is capped at the order total and rounded to the nearest cent.
type OrderDiscount struct {
	// Order discount ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=order_discount"`
	// Display name of the discount.
	Name string `json:"name" validate:"required"`
	// The code a buyer enters to apply this discount to an order.
	//
	// Codes are unique within your account and are matched without regard to letter case.
	Code string `json:"code" validate:"required"`
	// The fraction of the order total taken off, as a decimal string.
	//
	// This is a multiplier, not a whole percent: `0.1` takes 10% off. Only read when `discount_type` is `percentage`.
	Percentage string `json:"percentage" validate:"required" format:"decimal"`
	// The flat amount taken off the order total, as a decimal string.
	//
	// Only read when `discount_type` is `amount`.
	Amount string `json:"amount" validate:"required" format:"decimal"`
	// How the discount is calculated.
	//
	// - `percentage`: the order total is reduced by the fraction in `percentage`.
	// - `amount`: the order total is reduced by the flat amount in `amount`.
	DiscountType constants.OrderDiscountType `json:"discount_type" validate:"required"`
	// How many sales orders this discount has been applied to, across all buyers.
	OrderCount int32 `json:"order_count" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleOrderDiscount = &OrderDiscount{
	ID:           SampleOrderDiscountID,
	Object:       constants.ObjectTypeOrderDiscount,
	Name:         "10% Off",
	Code:         "SAVE10",
	Percentage:   "10.000000000000000000000000000000",
	Amount:       "0.000000000000000000000000000000",
	DiscountType: constants.OrderDiscountTypePercentage,
	OrderCount:   5,
	CreatedAt:    timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:    timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*OrderDiscount) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleOrderDiscount)
}
