package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleOrderDiscountID = "ords_01121c5e2f6937a6b896daad3a"

// Order discount resource.
type OrderDiscount struct {
	// Order discount ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=order_discount"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Discount code.
	Code string `json:"code" validate:"required"`
	// Percentage value as a decimal string.
	Percentage string `json:"percentage" validate:"required" format:"decimal"`
	// Fixed amount as a decimal string.
	Amount string `json:"amount" validate:"required" format:"decimal"`
	// Discount type: "percentage" or "amount".
	DiscountType constants.OrderDiscountType `json:"discount_type" validate:"required"`
	// Number of orders using this discount.
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
