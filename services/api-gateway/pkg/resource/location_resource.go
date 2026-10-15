package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
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

// A place stock is kept, from a whole site down to a single bin, arranged in a parent-child hierarchy.
//
// A site (a plant, warehouse or 3PL) is a top-level location with an address. Every location under a site belongs to it; a location under no site belongs to your default site.
type Location struct {
	// Location ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=location"`
	// Display name, unique (case-insensitive) among your account's locations.
	Name string `json:"name" validate:"required"`
	// What kind of place this is.
	//
	// `site` is a plant, warehouse or 3PL: always top-level, and the only type with an `address`. The rest run from largest to smallest: `building`, `section`, `aisle`, `rack`, `shelf`, `bin`. They are descriptive labels rather than a rule — a location's parent is not required to be the next level up.
	TypeCode constants.LocationTypeCode `json:"type" validate:"required"`
	// Whether the location can be newly assigned.
	//
	// `active` or `archived`. An archived location keeps its stock, history and children, but can't be chosen for new stock or as a new parent.
	Status constants.ArchiveStatus `json:"status" validate:"required"`
	// Where a site is.
	//
	// Only sites have an address; `null` for every other type.
	Address *Address `json:"address" expandable:"true"`
	// The location directly above this one in the hierarchy.
	//
	// `null` for top-level locations.
	Parent *Location `json:"parent" expandable:"true"`
	// The locations directly below this one, newest first.
	//
	// Holds the first 10; when there are more, `page_info.next_page_url` lists them all.
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
	TypeCode: constants.LocationTypeCodeSite,
	Status:   constants.ArchiveStatusActive,
	Address:  SampleAddress,
	Parent:   nil,
	Children: NewList([]Location{
		{
			ID:        SampleLocationChildID,
			Object:    constants.ObjectTypeLocation,
			Name:      "Shelf A1",
			TypeCode:  constants.LocationTypeCodeShelf,
			Status:    constants.ArchiveStatusActive,
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
