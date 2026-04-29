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
	// Unit type.
	Type constants.UnitType `json:"type" validate:"required"`
	// Base unit.
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
	// Discount percentage.
	DiscountPercentage float64 `json:"discount_percentage"`
	// Fixed discount amount.
	DiscountFixed float64 `json:"discount_fixed"`
	// Customer portal visibility.
	CustomerPortalVisibility constants.CustomerPortalVisibility `json:"customer_portal_visibility" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleUnitGroup = &UnitGroup{
	ID:        SampleUnitGroupID,
	Object:    constants.ObjectTypeUnitGroup,
	Name:      SampleUnitGroupName,
	Type:      constants.UnitTypeMass,
	Owner:     SampleOwnerSystem,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*UnitGroup) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleUnitGroup)
}
