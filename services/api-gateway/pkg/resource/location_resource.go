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

const SampleLocationTypeID = "lc_01gf7a8200er3ar3pkfrb6kk31"
const SampleLocationTypeCode = constants.LocationTypeCodeBuilding
const SampleLocationTypeName = "Building"

// LocationType represents a location type.
type LocationType struct {
	// The unique identifier for the location type.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=location_type"`
	// The unique code for this type.
	Code constants.LocationTypeCode `json:"code" validate:"required"`
	// The display name of the type.
	Name string `json:"name" validate:"required"`
	// The timestamp when the type was created.
	CreatedAt time.Time `json:"created_at"`
	// The timestamp when the type was last updated.
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

const SampleLocationID = "lc_01gf7a8200er3ar3pkfrb6kk30"
const SampleLocationName = "Warehouse A"

// Location represents a location.
type Location struct {
	// The unique identifier for the location.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=location"`
	// The display name of the location.
	Name string `json:"name" validate:"required"`
	// The code of the location type.
	TypeCode constants.LocationTypeCode `json:"type" validate:"required"`
	// The parent location. Null if this is a top-level location. Expandable.
	Parent *Location `json:"parent" expandable:"true"`
	// The child locations. Expandable.
	Children *List[Location] `json:"children" expandable:"true"`
	// The timestamp when the location was created.
	CreatedAt time.Time `json:"created_at"`
	// The timestamp when the location was last updated.
	UpdatedAt time.Time `json:"updated_at"`
}

var SampleLocation = &Location{
	ID:       SampleLocationID,
	Object:   constants.ObjectTypeLocation,
	Name:     SampleLocationName,
	TypeCode: SampleLocationTypeCode,
	Parent:   nil,
	Children: NewList([]Location{
		{ID: "lc_01gf7a8200er3ar3pkfrb6kk32", Object: constants.ObjectTypeLocation, Name: "Shelf A1", TypeCode: SampleLocationTypeCode},
	}, PageInfo{}),
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Location) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleLocation)
}
