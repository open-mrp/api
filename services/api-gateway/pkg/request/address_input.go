package apirequest

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
)

// Request to create an address.
type AddressInput struct {
	// Display name of the address.
	Name string `json:"name" validate:"required,max=255"`
	// Phone number associated with the address.
	Phone *string `json:"phone,omitempty" validate:"omitempty,max=255"`
	// Email address associated with the address.
	Email *string `json:"email,omitempty" validate:"omitnil,custom_email,max=255"`
	// Whether the address is a drop ship location.
	IsDropShip bool `json:"is_drop_ship"`
	// First line of the street address.
	StreetLine1 *string `json:"street_line_1,omitempty" validate:"omitempty,max=255"`
	// Second line of the street address.
	StreetLine2 *string `json:"street_line_2,omitempty" validate:"omitempty,max=255"`
	// City or locality.
	Locality *string `json:"locality,omitempty" validate:"omitempty,max=255"`
	// State or administrative area.
	State *string `json:"state,omitempty" validate:"omitempty,max=255"`
	// Postal or ZIP code.
	PostalCode *string `json:"postal_code,omitempty" validate:"omitempty,max=255"`
	// Two-letter country code.
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
