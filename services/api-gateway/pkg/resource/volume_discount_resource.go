package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleVolumeDiscountID = "quds_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleVolumeDiscountTierID = "qudstr_01jm4r6700f8nwq3v5hx2d9ktp"

// Volume discount with tiered pricing.
type VolumeDiscount struct {
	// Volume discount ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=volume_discount"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Tiers.
	Tiers *List[VolumeDiscountTier] `json:"tiers" validate:"required"`
	// Customer groups associated with this volume discount.
	CustomerGroups *List[AccountGroup] `json:"customer_groups" expandable:"true"`
	// Product lines associated with this volume discount.
	ProductLines *List[ProductLine] `json:"product_lines" expandable:"true"`
	// Item categories associated with this volume discount.
	Categories *List[ItemCategory] `json:"categories" expandable:"true"`
	// Attributes associated with this volume discount.
	Attributes *List[Attribute] `json:"attributes" expandable:"true"`
	// Acceptable units for this volume discount.
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
	// Discount percentage as a decimal string.
	DiscountPercentage string `json:"discount_percentage" validate:"required" format:"decimal"`
	// Quantity threshold as a decimal string.
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
