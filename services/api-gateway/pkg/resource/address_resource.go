package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleCRUDAddressID = "ad_012100950cfaa34aa0e0ad7258"
const SampleGeolocationID = "gl_013e4c26412103c6757ba71806"

// Geolocation sub-resource.
type Geolocation struct {
	// Geolocation ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=geolocation"`
	// First line of the street address.
	StreetLine1 *string `json:"street_line_1"`
	// Second line of the street address.
	StreetLine2 *string `json:"street_line_2"`
	// City or locality.
	Locality *string `json:"locality"`
	// State or administrative area.
	State *string `json:"state"`
	// Postal or ZIP code.
	PostalCode *string `json:"postal_code"`
	// Two-letter country code.
	Country string `json:"country" validate:"required"`
}

// Address with associated geolocation.
type Address struct {
	// Address ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=address"`
	// Display name of the address.
	Name string `json:"name" validate:"required"`
	// Phone number associated with the address.
	Phone *string `json:"phone"`
	// Email address associated with the address.
	Email *string `json:"email"`
	// Address type.
	Type constants.AddressType `json:"type" validate:"required"`
	// Geolocation details for the address.
	Geolocation *Geolocation `json:"geolocation" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
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
	Type:   constants.AddressTypeStandard,
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

func ExpandableAddressStub(id, name, country string, ts time.Time) *Address {
	if id == "" {
		id = SampleCRUDAddressID
	}
	if name == "" {
		name = "Address"
	}
	if country == "" {
		country = "US"
	}
	if ts.IsZero() {
		ts = time.Unix(0, 0).UTC()
	}
	return &Address{
		ID:     id,
		Object: constants.ObjectTypeAddress,
		Name:   name,
		Type:   constants.AddressTypeStandard,
		Geolocation: &Geolocation{
			ID:      SampleGeolocationID,
			Object:  constants.ObjectTypeGeolocation,
			Country: country,
		},
		CreatedAt: ts,
		UpdatedAt: ts,
	}
}
