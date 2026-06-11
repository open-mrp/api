package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleOrderDiscountID = "ords_01121c5e2f6937a6b896daad3a"

// A discount code that can be applied to a sales order.
//
// An order discount reduces the order total by either a percentage or a fixed amount, depending on `discount_type`.
type OrderDiscount struct {
	// Order discount ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=order_discount"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// The code entered to apply this discount to an order.
	//
	// Must be unique within the account.
	Code string `json:"code" validate:"required"`
	// Percent off as a decimal string (e.g. `10` for 10%).
	//
	// Applies when `discount_type` is `percentage`; otherwise `0`.
	Percentage string `json:"percentage" validate:"required" format:"decimal"`
	// Fixed amount off as a decimal string.
	//
	// Applies when `discount_type` is `amount`; otherwise `0`.
	Amount string `json:"amount" validate:"required" format:"decimal"`
	// How the discount is calculated, determining whether `percentage` or `amount` is used.
	//
	// - `percentage`: the discount is a percent off, taken from `percentage`.
	// - `amount`: the discount is a fixed amount off, taken from `amount`.
	DiscountType constants.OrderDiscountType `json:"discount_type" validate:"required"`
	// Number of orders currently using this discount.
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
