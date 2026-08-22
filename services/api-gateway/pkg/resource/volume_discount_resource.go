package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleVolumeDiscountID = "quds_bn7hto9s10pp"
const SampleVolumeDiscountTierID = "qudstr_iylnkrlr3uhm"

// A quantity-based discount with tiered percentage rates.
//
// A volume discount reduces the price once the ordered quantity reaches a tier's threshold. The customer group associations scope which customers qualify, and the product line, category, and attribute associations scope which order lines qualify; an empty list on any of them means no restriction on that dimension. Acceptable units are not a scope: they are the units the ordered quantity is measured in, and a discount with none of them never reaches a threshold above zero.
//
// At most one volume discount is applied to a given order line: among the discounts whose scope the line matches and whose thresholds are met, those scoped to a customer group the buyer belongs to take precedence. An account price for the same line overrides the discounted price entirely.
type VolumeDiscount struct {
	// Volume discount ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=volume_discount"`
	// Display name of the volume discount.
	//
	// Must be unique within the account.
	Name string `json:"name" validate:"required"`
	// Quantity tiers that define the discount.
	//
	// Every tier whose threshold the ordered quantity reaches is applied, and their reductions compound. A discount with no tiers never changes a price.
	Tiers *List[VolumeDiscountTier] `json:"tiers" validate:"required"`
	// Customer groups this discount is scoped to.
	//
	// When set, only customers belonging to at least one of these groups qualify; when empty, all customers qualify. A customer belongs to a group either by being assigned to it directly or through the price groups on their customer relationship.
	CustomerGroups *List[AccountGroup] `json:"customer_groups" expandable:"true"`
	// Product lines this discount is scoped to.
	//
	// When set, only items in one of these product lines qualify; when empty, all product lines qualify.
	ProductLines *List[ProductLine] `json:"product_lines" expandable:"true"`
	// Item categories this discount is scoped to.
	//
	// When set, only items in one of these categories qualify; when empty, all categories qualify.
	Categories *List[ItemCategory] `json:"categories" expandable:"true"`
	// Attributes this discount is scoped to.
	//
	// When set, an item qualifies only if it has every listed attribute; when empty, attributes are not considered.
	Attributes *List[Attribute] `json:"attributes" expandable:"true"`
	// Units that ordered quantities are measured in when evaluating tier thresholds.
	//
	// Quantities ordered in other units are converted to an acceptable unit before being compared against tier thresholds; a quantity that cannot be converted contributes nothing. A discount with no acceptable units always evaluates to a quantity of zero, so it never reaches a threshold above zero.
	AcceptableUnits *List[Unit] `json:"acceptable_units" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// A quantity threshold within a volume discount, and the reduction that applies at or above it.
type VolumeDiscountTier struct {
	// Volume discount tier ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=volume_discount_tier"`
	// Display name of the tier.
	Name string `json:"name" validate:"required"`
	// Fraction of the price taken off once the threshold is met, as a decimal string.
	//
	// This is a multiplier, not a whole percent: `0.05` takes 5% off. When an order meets several tiers of the same discount, their reductions compound: meeting a `0.1` tier and a `0.2` tier multiplies the price by `0.9 × 0.8`, a 28% reduction overall.
	DiscountPercentage string `json:"discount_percentage" validate:"required" format:"decimal"`
	// Minimum ordered quantity at which this tier's discount begins to apply, as a decimal string.
	//
	// The quantity compared against the threshold is the total across every line on the order that falls within the discount's scope, converted into one of the discount's acceptable units — not the quantity of a single line.
	Threshold string `json:"threshold" validate:"required" format:"decimal"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleVolumeDiscountTier = VolumeDiscountTier{
	ID:                 SampleVolumeDiscountTierID,
	Object:             constants.ObjectTypeVolumeDiscountTier,
	Name:               "100+ Units",
	DiscountPercentage: "5.000000000000000000000000000000",
	Threshold:          "100.000000000000000000000000000000",
	CreatedAt:          timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:          timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

var SampleVolumeDiscount = &VolumeDiscount{
	ID:              SampleVolumeDiscountID,
	Object:          constants.ObjectTypeVolumeDiscount,
	Name:            "Bulk Order Discount",
	Tiers:           NewList([]VolumeDiscountTier{SampleVolumeDiscountTier}, PageInfo{}),
	CustomerGroups:  nil,
	ProductLines:    nil,
	Categories:      nil,
	Attributes:      nil,
	AcceptableUnits: nil,
	CreatedAt:       timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:       timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*VolumeDiscount) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleVolumeDiscount)
}

func (*VolumeDiscountTier) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&SampleVolumeDiscountTier)
}
