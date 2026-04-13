package apirequest

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
)

// AddressInput represents an address.
// Field names align with the Address resource shape.
type AddressInput struct {
	// The display name of the address.
	Name string `json:"name" validate:"required,max=255"`
	// The phone number associated with this address.
	Phone *string `json:"phone,omitempty" validate:"omitempty,max=255"`
	// The email address associated with this address.
	Email *string `json:"email,omitempty" validate:"omitnil,custom_email,max=255"`
	// Whether this is a drop ship address.
	IsDropShip bool `json:"is_drop_ship"`
	// The first line of the street address.
	StreetLine1 *string `json:"street_line_1,omitempty" validate:"omitempty,max=255"`
	// The second line of the street address.
	StreetLine2 *string `json:"street_line_2,omitempty" validate:"omitempty,max=255"`
	// The city or locality.
	Locality *string `json:"locality,omitempty" validate:"omitempty,max=255"`
	// The state or administrative area.
	State *string `json:"state,omitempty" validate:"omitempty,max=255"`
	// The postal or zip code.
	PostalCode *string `json:"postal_code,omitempty" validate:"omitempty,max=255"`
	// The two-letter country code.
	Country string `json:"country" validate:"required,max=2"`
}

var sampleAddressStreetLine1 = "123 Main St"
var sampleAddressLocality = "Springfield"
var sampleAddressState = "IL"
var sampleAddressPostalCode = "62701"

var sampleAddressInput = &AddressInput{
	Name:        "Headquarters",
	StreetLine1: &sampleAddressStreetLine1,
	Locality:    &sampleAddressLocality,
	State:       &sampleAddressState,
	PostalCode:  &sampleAddressPostalCode,
	Country:     "US",
}

func (*AddressInput) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleAddressInput)
}
