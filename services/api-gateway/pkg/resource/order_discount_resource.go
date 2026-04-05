package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleOrderDiscountID = "ords_01jm4r6700f8nwq3v5hx2d9ktp"

// OrderDiscount represents an order discount.
type OrderDiscount struct {
	// The unique identifier for the order discount.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=order_discount"`
	// The human-readable name for the discount.
	Name string `json:"name" validate:"required"`
	// The unique code for this discount.
	Code string `json:"code" validate:"required"`
	// The percentage value of the discount as a decimal string.
	Percentage string `json:"percentage" validate:"required" format:"decimal"`
	// The fixed amount of the discount as a decimal string.
	Amount string `json:"amount" validate:"required" format:"decimal"`
	// The type of discount: "percentage" or "amount".
	DiscountType constants.OrderDiscountType `json:"discount_type" validate:"required"`
	// The number of orders using this discount.
	OrderCount int32 `json:"order_count" validate:"required"`
	// When this discount was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this discount was last updated.
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
