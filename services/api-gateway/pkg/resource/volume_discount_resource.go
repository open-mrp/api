package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleVolumeDiscountID = "quds_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleVolumeDiscountTierID = "qudstr_01jm4r6700f8nwq3v5hx2d9ktp"

// VolumeDiscount represents a volume discount with tiered pricing.
type VolumeDiscount struct {
	// The unique identifier for the volume discount.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=volume_discount"`
	// The human-readable name for the volume discount.
	Name string `json:"name" validate:"required"`
	// The tiers for this volume discount.
	Tiers *List[VolumeDiscountTier] `json:"tiers" validate:"required"`
	// The customer groups associated with this volume discount.
	CustomerGroups *List[AccountGroup] `json:"customer_groups" expandable:"true"`
	// The product lines associated with this volume discount.
	ProductLines *List[ProductLine] `json:"product_lines" expandable:"true"`
	// The item categories associated with this volume discount.
	Categories *List[ItemCategory] `json:"categories" expandable:"true"`
	// The attributes associated with this volume discount.
	Attributes *List[Attribute] `json:"attributes" expandable:"true"`
	// The acceptable units for this volume discount.
	AcceptableUnits *List[Unit] `json:"acceptable_units" expandable:"true"`
	// When this volume discount was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this volume discount was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// VolumeDiscountTier represents a tier within a volume discount.
type VolumeDiscountTier struct {
	// The unique identifier for the volume discount tier.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=volume_discount_tier"`
	// The human-readable name for the tier.
	Name string `json:"name" validate:"required"`
	// The discount percentage as a decimal string.
	DiscountPercentage string `json:"discount_percentage" validate:"required" format:"decimal"`
	// The quantity threshold for this tier as a decimal string.
	Threshold string `json:"threshold" validate:"required" format:"decimal"`
	// When this tier was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this tier was last updated.
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
