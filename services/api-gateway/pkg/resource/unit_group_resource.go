package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// UnitGroup is a unit group resource.
type UnitGroup struct {
	// Unit group ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=unit_group"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Notes.
	Notes *string `json:"notes"`
	// Dimension shared by every unit in this group.
	//
	// Only units of this dimension can belong to the group.
	//
	// - `currency`: monetary units such as dollars or euros.
	// - `quantity`: discrete countable units.
	// - `time`: time-based units such as hours or minutes.
	// - `mass`: weight-based units such as kilograms or pounds.
	// - `volume`: volumetric units such as liters or gallons.
	// - `length`: distance-based units such as meters or feet.
	// - `temperature`: temperature units such as Celsius or Fahrenheit.
	// - `area`: area-based units such as square meters or acres.
	Type constants.UnitType `json:"type" validate:"required"`
	// Base unit of the group.
	//
	// All other units' conversion ratios are expressed relative to this unit. Expandable.
	BaseUnit *Unit `json:"base_unit" expandable:"true"`
	// Associated units.
	AssociatedUnits *List[UnitGroupUnit] `json:"associated_units" expandable:"true"`
	// Owner.
	Owner *Owner `json:"owner" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// UnitGroupUnit is an associated unit within a unit group.
type UnitGroupUnit struct {
	// Unit group unit ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=unit_group_unit"`
	// Unit.
	Unit *Unit `json:"unit" expandable:"true"`
	// Percentage discount applied when ordering in this unit, as a number out of 100 (e.g. `1` means 1%).
	//
	// Defaults to `1`.
	DiscountPercentage float64 `json:"discount_percentage"`
	// Fixed per-unit discount amount applied when ordering in this unit, in the account's currency.
	//
	// Defaults to `0`.
	DiscountFixed float64 `json:"discount_fixed"`
	// Whether this unit is shown to customers in the customer portal.
	//
	// - `visible`: the unit is selectable in the customer portal.
	// - `hidden`: the unit is hidden from the customer portal.
	CustomerPortalVisibility constants.CustomerPortalVisibility `json:"customer_portal_visibility" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

const SampleUnitGroupUnitID = "ugu_01d75e0598ed09be56fd39fab5"

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

var SampleUnitGroup = &UnitGroup{
	ID:        SampleUnitGroupID,
	Object:    constants.ObjectTypeUnitGroup,
	Name:      SampleUnitGroupName,
	Type:      constants.UnitTypeMass,
	BaseUnit:  SampleUnit,
	Owner:     SampleOwnerSystem,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*UnitGroup) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleUnitGroup)
}
