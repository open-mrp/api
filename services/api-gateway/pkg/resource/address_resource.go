package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

const SampleCRUDAddressID = "ad_j8cz0b79pwdb"

// A saved address, used as a billing or shipping address on orders, invoices and shipments.
type Address struct {
	// Address ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=address"`
	// A label for the address, such as `Main warehouse`.
	Name string `json:"name" validate:"required"`
	// How the address is used.
	//
	// `standard`: a normal billing or shipping address. `drop_ship`: shipped to directly on the customer's behalf, such as an end customer.
	Type constants.AddressType `json:"type" validate:"required"`
	// Street address.
	Line1 *string `json:"line_1"`
	// Apartment, suite, unit or building.
	Line2 *string `json:"line_2"`
	// City or locality.
	City *string `json:"city"`
	// State, province or region; in the US and Canada, the two-letter code.
	State *string `json:"state"`
	// Postal or ZIP code.
	PostalCode *string `json:"postal_code"`
	// Two-letter country code (ISO 3166-1 alpha-2), such as `US`.
	Country string `json:"country" validate:"required"`
	// Phone number at the address.
	Phone *string `json:"phone"`
	// Email address at the address.
	Email *string `json:"email"`
	// Days this location accepts freight.
	//
	// Set it when one site keeps different days from the rest. `null` uses the customer's calendar, then your account default.
	ReceiveCalendar *OperatingCalendar `json:"receive_calendar" expandable:"true" visibility:"internal"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleAddress = &Address{
	ID:         SampleCRUDAddressID,
	Object:     constants.ObjectTypeAddress,
	Name:       "Headquarters",
	Type:       constants.AddressTypeStandard,
	Line1:      new("4200 Industrial Pkwy"),
	City:       new("Columbus"),
	State:      new("OH"),
	PostalCode: new("43204"),
	Country:    "US",
	Phone:      new("+1-614-555-0142"),
	Email:      new("receiving@acme.example.com"),
	CreatedAt:  timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:  timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Address) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAddress)
}
