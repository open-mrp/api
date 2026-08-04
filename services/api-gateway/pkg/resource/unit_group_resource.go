package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// A named collection of units that share one dimension, defining which units a product can be ordered in.
//
// Each associated unit carries its own discount and customer portal visibility, applied when an order line is priced in that unit. A product takes its unit group from its product line, falling back to its item category.
type UnitGroup struct {
	// Unit group ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=unit_group"`
	// Display name of the unit group.
	//
	// Unique within the account.
	Name string `json:"name" validate:"required"`
	// Free-form notes about the unit group.
	Notes *string `json:"notes"`
	// The dimension shared by every unit in this group, such as mass, volume, or currency.
	//
	// Only units of this dimension can belong to the group, and the dimension is fixed once the group is created.
	Type constants.UnitType `json:"type" validate:"required"`
	// The reference unit designated for this group.
	BaseUnit *Unit `json:"base_unit" expandable:"true"`
	// Units associated with this group, each with its own discount and customer portal visibility settings.
	AssociatedUnits *List[UnitGroupUnit] `json:"associated_units" expandable:"true"`
	// Owner of this resource.
	Owner *Owner `json:"owner" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// Membership of a unit in a unit group, carrying the discount and customer portal visibility settings applied when ordering in that unit.
type UnitGroupUnit struct {
	// Unit group unit ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=unit_group_unit"`
	// The unit this association refers to.
	Unit *Unit `json:"unit" expandable:"true"`
	// Share of the unit's price removed when an order is placed in this unit.
	//
	// Expressed as a decimal fraction rather than a whole number, so `0.1` is a 10% discount and `0` is no discount.
	DiscountPercentage float64 `json:"discount_percentage"`
	// Flat amount subtracted from the unit's price when an order is placed in this unit.
	//
	// Subtracted before `discount_percentage` is applied.
	DiscountFixed float64 `json:"discount_fixed"`
	// Whether this unit is shown to customers in the customer portal.
	CustomerPortalVisibility constants.CustomerPortalVisibility `json:"customer_portal_visibility" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

const SampleUnitGroupUnitID = "ugu_sfrqqziz49dw"

var SampleUnitGroupUnit = &UnitGroupUnit{
	ID:                       SampleUnitGroupUnitID,
	Object:                   constants.ObjectTypeUnitGroupUnit,
	Unit:                     SampleUnit,
	DiscountPercentage:       1,
	DiscountFixed:            0,
	CustomerPortalVisibility: constants.CustomerPortalVisibilityVisible,
	CreatedAt:                timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:                timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*UnitGroupUnit) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleUnitGroupUnit)
}

var sampleUnitGroupNotes = "Mass units used for ordering raw materials by weight."

var SampleUnitGroup = &UnitGroup{
	ID:              SampleUnitGroupID,
	Object:          constants.ObjectTypeUnitGroup,
	Name:            SampleUnitGroupName,
	Notes:           &sampleUnitGroupNotes,
	Type:            constants.UnitTypeMass,
	BaseUnit:        SampleUnit,
	AssociatedUnits: NewList([]UnitGroupUnit{*SampleUnitGroupUnit}, PageInfo{}),
	Owner:           SampleOwnerSystem,
	CreatedAt:       timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:       timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*UnitGroup) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleUnitGroup)
}
