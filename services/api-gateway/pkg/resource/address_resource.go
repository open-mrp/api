package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleCRUDAddressID = "ad_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleGeolocationID = "gl_01jm4r6700f8nwq3v5hx2d9ktp"

// Geolocation represents a geolocation sub-resource within an address.
type Geolocation struct {
	// The unique identifier for the geolocation.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=geolocation"`
	// The first line of the street address.
	StreetLine1 *string `json:"street_line_1"`
	// The second line of the street address.
	StreetLine2 *string `json:"street_line_2"`
	// The city or locality.
	Locality *string `json:"locality"`
	// The state or administrative area.
	State *string `json:"state"`
	// The postal or zip code.
	PostalCode *string `json:"postal_code"`
	// The two-letter country code.
	Country string `json:"country" validate:"required"`
}

// Address represents an address with its associated geolocation.
type Address struct {
	// The unique identifier for the address.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=address"`
	// The display name of the address.
	Name string `json:"name" validate:"required"`
	// The phone number associated with this address.
	Phone *string `json:"phone"`
	// The email address associated with this address.
	Email *string `json:"email"`
	// Whether this is a drop ship address.
	IsDropShip bool `json:"is_drop_ship"`
	// The geolocation details for this address.
	Geolocation *Geolocation `json:"geolocation" validate:"required"`
	// When this address was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this address was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleStreetLine1 = "4200 Industrial Pkwy"
var sampleLocality = "Columbus"
var sampleState = "OH"
var samplePostalCode = "43204"

var SampleAddress = &Address{
	ID:     SampleCRUDAddressID,
	Object: constants.ObjectTypeAddress,
	Name:   "Headquarters",
	Geolocation: &Geolocation{
		ID:          SampleGeolocationID,
		Object:      constants.ObjectTypeGeolocation,
		StreetLine1: &sampleStreetLine1,
		Locality:    &sampleLocality,
		State:       &sampleState,
		PostalCode:  &samplePostalCode,
		Country:     "US",
	},
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Address) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAddress)
}
