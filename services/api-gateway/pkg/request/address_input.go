package apirequest

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/field"
)

// Address details supplied when creating an address, either on its own or inline on another resource.
//
// A few requests, such as shipping rate estimates, take these same fields for a one-off address that is never saved to the account.
type AddressInput struct {
	// Display name of the address.
	Name string `json:"name" validate:"required,min=1,max=255"`
	// Phone number associated with the address.
	Phone field.Optional[string] `json:"phone,omitzero" validate:"omitempty,max=255"`
	// Email address associated with the address.
	Email field.Optional[string] `json:"email,omitzero" validate:"omitempty,custom_email,max=255"`
	// How the address is used.
	//
	// - `standard`: a normal shipping or billing address.
	// - `drop_ship`: an address an order is shipped to directly, typically a third party or end customer rather than the account itself.
	Type field.Optional[constants.AddressType] `json:"type,omitzero" default:"standard"`
	// First line of the street address.
	StreetLine1 field.Optional[string] `json:"street_line_1,omitzero" validate:"omitempty,max=255"`
	// Second line of the street address.
	StreetLine2 field.Optional[string] `json:"street_line_2,omitzero" validate:"omitempty,max=255"`
	// City or locality.
	Locality field.Optional[string] `json:"locality,omitzero" validate:"omitempty,max=255"`
	// State or administrative area.
	State field.Optional[string] `json:"state,omitzero" validate:"omitempty,max=255"`
	// Postal or ZIP code.
	PostalCode field.Optional[string] `json:"postal_code,omitzero" validate:"omitempty,max=255"`
	// Two-letter ISO 3166-1 country code, such as `US`.
	Country string `json:"country" validate:"required,max=2"`
}

var sampleAddressPhone = "555-123-4567"
var sampleAddressEmail = "warehouse@acme.com"
var sampleAddressStreetLine1 = "123 Main St"
var sampleAddressStreetLine2 = "Suite 400"
var sampleAddressLocality = "Springfield"
var sampleAddressState = "IL"
var sampleAddressPostalCode = "62701"

var sampleAddressInput = &AddressInput{
	Name:        "Headquarters",
	Phone:       field.SomePtr(&sampleAddressPhone),
	Email:       field.SomePtr(&sampleAddressEmail),
	Type:        field.Some(constants.AddressTypeStandard),
	StreetLine1: field.SomePtr(&sampleAddressStreetLine1),
	StreetLine2: field.SomePtr(&sampleAddressStreetLine2),
	Locality:    field.SomePtr(&sampleAddressLocality),
	State:       field.SomePtr(&sampleAddressState),
	PostalCode:  field.SomePtr(&sampleAddressPostalCode),
	Country:     "US",
}

func (*AddressInput) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleAddressInput)
}
