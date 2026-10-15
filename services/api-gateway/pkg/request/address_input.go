package apirequest

import (
	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/validate"
)

func init() {
	validate.RegisterWrappedTypes(field.Optional[AddressInput]{})
}

// Address details supplied when creating an address, either on its own or inline on another resource.
//
// A few requests, such as shipping rate estimates, take these same fields for a one-off address that is never saved to the account.
type AddressInput struct {
	// A label for the address, such as `Main warehouse`.
	Name string `json:"name" validate:"required,min=1,max=255"`
	// How the address is used.
	//
	// `standard`: a normal billing or shipping address. `drop_ship`: shipped to directly on the customer's behalf, such as an end customer.
	Type field.Optional[constants.AddressType] `json:"type,omitzero" default:"standard"`
	// Street address.
	Line1 field.Optional[string] `json:"line_1,omitzero" validate:"omitempty,max=255"`
	// Apartment, suite, unit or building.
	Line2 field.Optional[string] `json:"line_2,omitzero" validate:"omitempty,max=255"`
	// City or locality.
	City field.Optional[string] `json:"city,omitzero" validate:"omitempty,max=255"`
	// State, province or region; in the US and Canada, the two-letter code.
	State field.Optional[string] `json:"state,omitzero" validate:"omitempty,max=255"`
	// Postal or ZIP code.
	PostalCode field.Optional[string] `json:"postal_code,omitzero" validate:"omitempty,max=255"`
	// Two-letter country code (ISO 3166-1 alpha-2), such as `US`.
	Country string `json:"country" validate:"required,iso3166_1_alpha2"`
	// Phone number at the address.
	Phone field.Optional[string] `json:"phone,omitzero" validate:"omitempty,max=255"`
	// Email address at the address.
	Email field.Optional[string] `json:"email,omitzero" validate:"omitempty,custom_email,max=255"`
	// ID of the operating calendar naming the days this location accepts freight.
	ReceiveCalendarID field.Optional[string] `json:"receive_calendar_id,omitzero" validate:"omitempty" visibility:"internal"`
}

var sampleAddressInput = &AddressInput{
	Name:       "Headquarters",
	Type:       field.Some(constants.AddressTypeStandard),
	Line1:      field.Some("123 Main St"),
	Line2:      field.Some("Suite 400"),
	City:       field.Some("Springfield"),
	State:      field.Some("IL"),
	PostalCode: field.Some("62701"),
	Country:    "US",
	Phone:      field.Some("555-123-4567"),
	Email:      field.Some("warehouse@acme.com"),
}

func (*AddressInput) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleAddressInput)
}
