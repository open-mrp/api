package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// ---------------------------------------------------------------------------
// LocationType — location type resource
// ---------------------------------------------------------------------------

const SampleLocationTypeID = "lc_ra08sngd0fxr"
const SampleLocationTypeCode = constants.LocationTypeCodeBuilding
const SampleLocationTypeName = "Building"

// A level in the storage location hierarchy, such as a building or a bin.
//
// Location types are platform-defined and identical for every account: you choose one when creating a location, but you cannot add or modify the types themselves.
type LocationType struct {
	// Location type ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=location_type"`
	// The level of the storage hierarchy this type represents.
	//
	// The levels run from largest to smallest: `building`, `section`, `aisle`, `rack`, `shelf`, `bin`.
	Code constants.LocationTypeCode `json:"code" validate:"required"`
	// Display name of the location type.
	Name string `json:"name" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at"`
	// Last-updated timestamp.
	UpdatedAt time.Time `json:"updated_at"`
}

var SampleLocationType = &LocationType{
	ID:        SampleLocationTypeID,
	Object:    constants.ObjectTypeLocationType,
	Code:      SampleLocationTypeCode,
	Name:      SampleLocationTypeName,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*LocationType) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleLocationType)
}

// ---------------------------------------------------------------------------
// Location — location resource
// ---------------------------------------------------------------------------

const SampleLocationID = "lc_yonnys0hx3ju"
const SampleLocationChildID = "lc_huci6mict6ys"
const SampleLocationName = "Warehouse A"

// A physical storage location, such as a warehouse, aisle, or bin, arranged in a parent-child hierarchy.
type Location struct {
	// Location ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=location"`
	// Display name of the location.
	Name string `json:"name" validate:"required"`
	// This location's level in the storage hierarchy.
	//
	// The levels run from largest to smallest: `building`, `section`, `aisle`, `rack`, `shelf`, `bin`. They are descriptive labels rather than a rule — a location's parent is not required to be the next level up.
	TypeCode constants.LocationTypeCode `json:"type" validate:"required"`
	// The location directly above this one in the storage hierarchy.
	//
	// Absent for top-level locations.
	Parent *Location `json:"parent" expandable:"true"`
	// The locations directly below this one in the storage hierarchy.
	Children *List[Location] `json:"children" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at"`
	// Last-updated timestamp.
	UpdatedAt time.Time `json:"updated_at"`
}

var SampleLocation = &Location{
	ID:       SampleLocationID,
	Object:   constants.ObjectTypeLocation,
	Name:     SampleLocationName,
	TypeCode: SampleLocationTypeCode,
	Parent:   nil,
	Children: NewList([]Location{
		{
			ID:        SampleLocationChildID,
			Object:    constants.ObjectTypeLocation,
			Name:      "Shelf A1",
			TypeCode:  SampleLocationTypeCode,
			CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
			UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
		},
	}, PageInfo{}),
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Location) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleLocation)
}
