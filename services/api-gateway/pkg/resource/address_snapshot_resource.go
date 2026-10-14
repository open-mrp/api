package apiresource

import apiexample "github.com/open-mrp/apikit/example"

// An address as a document printed it, copied when it was set on the document.
//
// `address` is the saved address it came from, if any. Later changes to that address don't affect this document.
type AddressSnapshot struct {
	// Name on the address, such as a company or a receiving dock.
	Name string `json:"name" validate:"required"`
	// First line of the street address.
	Line1 *string `json:"line_1"`
	// Second line of the street address.
	Line2 *string `json:"line_2"`
	// City or locality.
	City *string `json:"city"`
	// State or administrative area.
	State *string `json:"state"`
	// Postal or ZIP code.
	PostalCode *string `json:"postal_code"`
	// Two-letter ISO 3166-1 country code, such as `US`.
	Country string `json:"country" validate:"required"`
	// Phone number at the address.
	Phone *string `json:"phone"`
	// Email address at the address.
	Email *string `json:"email"`
	// The saved address this was copied from. `null` for an address given inline, and unless included.
	Address *Address `json:"address" expandable:"true"`
}

var SampleAddressSnapshot = &AddressSnapshot{
	Name:       "Headquarters",
	Line1:      new("4200 Industrial Pkwy"),
	City:       new("Columbus"),
	State:      new("OH"),
	PostalCode: new("43204"),
	Country:    "US",
	Phone:      new("+1-614-555-0142"),
}

func (*AddressSnapshot) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAddressSnapshot)
}
