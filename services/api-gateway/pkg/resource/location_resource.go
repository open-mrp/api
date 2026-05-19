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

// LocationType resource.
type LocationType struct {
	// Location ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=location_type"`
	// Location type code.
	Code constants.LocationTypeCode `json:"code" validate:"required"`
	// Display name.
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

const SampleLocationID = "lc_01gf7a8200er3ar3pkfrb6kk30"
const SampleLocationChildID = "lc_01gf7a8200er3ar3pkfrb6kk32"
const SampleLocationName = "Warehouse A"

// Location resource.
type Location struct {
	// Location ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=location"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Location type code.
	TypeCode constants.LocationTypeCode `json:"type" validate:"required"`
	// Parent location. Null for top-level locations. Expandable.
	Parent *Location `json:"parent" expandable:"true"`
	// Child locations. Expandable.
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
		{ID: SampleLocationChildID, Object: constants.ObjectTypeLocation, Name: "Shelf A1", TypeCode: SampleLocationTypeCode},
	}, PageInfo{}),
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Location) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleLocation)
}
