package apirequest

// AddressInput represents an inline address to create with a parent resource.
// Field names align with the Address resource shape.
type AddressInput struct {
	// The display name of the address.
	Name string `json:"name" validate:"required,max=255"`
	// The phone number associated with this address.
	Phone *string `json:"phone" validate:"omitempty,max=255"`
	// The email address associated with this address.
	Email *string `json:"email" validate:"omitempty,max=255"`
	// Whether this is a drop ship address.
	IsDropShip bool `json:"is_drop_ship"`
	// The first line of the street address.
	StreetLine1 *string `json:"street_line_1" validate:"omitempty,max=255"`
	// The second line of the street address.
	StreetLine2 *string `json:"street_line_2" validate:"omitempty,max=255"`
	// The city or locality.
	Locality *string `json:"locality" validate:"omitempty,max=255"`
	// The state or region.
	State *string `json:"state" validate:"omitempty,max=255"`
	// The postal or ZIP code.
	PostalCode *string `json:"postal_code" validate:"omitempty,max=255"`
	// The ISO country code.
	Country string `json:"country" validate:"required,max=2"`
}

// ShippingAddressInput represents an address for shipping operations.
// Uses the same field naming convention as AddressInput but with
// different required/optional semantics for shipping use cases.
type ShippingAddressInput struct {
	// The name of the recipient.
	Name *string `json:"name" validate:"omitempty,max=255"`
	// The company name.
	Company *string `json:"company" validate:"omitempty,max=255"`
	// The first line of the street address.
	StreetLine1 string `json:"street_line_1" validate:"required,max=255"`
	// The second line of the street address.
	StreetLine2 *string `json:"street_line_2" validate:"omitempty,max=255"`
	// The city or locality.
	Locality string `json:"locality" validate:"required,max=255"`
	// The state or province.
	State string `json:"state" validate:"required,max=255"`
	// The postal or ZIP code.
	PostalCode string `json:"postal_code" validate:"required,max=255"`
	// The two-letter country code.
	Country string `json:"country" validate:"required,max=2"`
	// The phone number.
	Phone *string `json:"phone" validate:"omitempty,max=255"`
	// The email address.
	Email *string `json:"email" validate:"omitempty,max=255"`
}
