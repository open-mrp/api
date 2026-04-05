package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// UnitGroup represents a unit group resource with base unit and associated units.
type UnitGroup struct {
	// The unique identifier for the unit group.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=unit_group"`
	// The display name of the unit group.
	Name string `json:"name" validate:"required"`
	// Notes about the unit group.
	Notes *string `json:"notes"`
	// The unit type.
	Type constants.UnitType `json:"type" validate:"required"`
	// The base unit for this group.
	BaseUnit *Unit `json:"base_unit" expandable:"true"`
	// The associated units in this group.
	AssociatedUnits *List[UnitGroupUnit] `json:"associated_units" expandable:"true"`
	// The owner of this resource.
	Owner *Owner `json:"owner" expandable:"true"`
	// When this unit group was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this unit group was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// UnitGroupUnit represents an associated unit within a unit group.
type UnitGroupUnit struct {
	// The unique identifier for the unit group unit.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=unit_group_unit"`
	// The unit. Expandable.
	Unit *Unit `json:"unit" expandable:"true"`
	// The discount percentage for this associated unit.
	DiscountPercentage float64 `json:"discount_percentage"`
	// The fixed discount amount for this associated unit.
	DiscountFixed float64 `json:"discount_fixed"`
	// Whether this associated unit is visible in the customer portal.
	CustomerPortalVisibility constants.CustomerPortalVisibility `json:"customer_portal_visibility" validate:"required,enum"`
	// When this associated unit was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this associated unit was last updated.
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
