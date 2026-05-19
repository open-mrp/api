package apirequest

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/patch"
)

// Request to create an address.
type AddressInput struct {
	// Display name of the address.
	Name string `json:"name" validate:"required,max=255"`
	// Phone number associated with the address.
	Phone patch.Nullable[string] `json:"phone,omitzero" validate:"omitempty,max=255"`
	// Email address associated with the address.
	Email patch.Nullable[string] `json:"email,omitzero" validate:"omitempty,custom_email,max=255"`
	// Address type.
	Type *constants.AddressType `json:"type,omitempty" default:"standard"`
	// First line of the street address.
	StreetLine1 patch.Nullable[string] `json:"street_line_1,omitzero" validate:"omitempty,max=255"`
	// Second line of the street address.
	StreetLine2 patch.Nullable[string] `json:"street_line_2,omitzero" validate:"omitempty,max=255"`
	// City or locality.
	Locality patch.Nullable[string] `json:"locality,omitzero" validate:"omitempty,max=255"`
	// State or administrative area.
	State patch.Nullable[string] `json:"state,omitzero" validate:"omitempty,max=255"`
	// Postal or ZIP code.
	PostalCode patch.Nullable[string] `json:"postal_code,omitzero" validate:"omitempty,max=255"`
	// Two-letter country code.
	Country string `json:"country" validate:"required,max=2"`
}

var sampleAddressStreetLine1 = "123 Main St"
var sampleAddressLocality = "Springfield"
var sampleAddressState = "IL"
var sampleAddressPostalCode = "62701"

var sampleAddressInput = &AddressInput{
	Name:        "Headquarters",
	StreetLine1: patch.PtrNullable(&sampleAddressStreetLine1),
	Locality:    patch.PtrNullable(&sampleAddressLocality),
	State:       patch.PtrNullable(&sampleAddressState),
	PostalCode:  patch.PtrNullable(&sampleAddressPostalCode),
	Country:     "US",
}

func (*AddressInput) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleAddressInput)
}
