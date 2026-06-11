package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleVolumeDiscountID = "quds_01b64658b647f3c5266b8f6ae1"
const SampleVolumeDiscountTierID = "qudstr_01576d26526ad625c3dd0725a9"

// A quantity-based discount with tiered percentage rates.
//
// A volume discount reduces the price once the ordered quantity reaches a tier's threshold. The customer group, product line, category, attribute, and acceptable unit associations scope which orders qualify; an empty association list means no restriction on that dimension.
type VolumeDiscount struct {
	// Volume discount ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=volume_discount"`
	// Display name.
	//
	// Must be unique within the account.
	Name string `json:"name" validate:"required"`
	// Quantity tiers that define the discount.
	//
	// Each tier sets a discount percentage that applies once the ordered quantity reaches its threshold.
	Tiers *List[VolumeDiscountTier] `json:"tiers" validate:"required"`
	// Customer groups this discount is scoped to.
	//
	// When set, only customers belonging to at least one of these groups qualify; when empty, all customers qualify.
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
	// Quantities ordered in other units are converted to an acceptable unit before being compared against tier thresholds.
	AcceptableUnits *List[Unit] `json:"acceptable_units" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// Tier within a volume discount.
type VolumeDiscountTier struct {
	// Volume discount tier ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=volume_discount_tier"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Percentage taken off the price once the threshold is met, as a decimal string.
	//
	// For example, `5` means a 5% discount.
	DiscountPercentage string `json:"discount_percentage" validate:"required" format:"decimal"`
	// Minimum ordered quantity at which this tier's discount begins to apply, as a decimal string.
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
