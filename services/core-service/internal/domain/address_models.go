package domain

import (
	"time"

	"github.com/augno/api/shared/patch"
	"github.com/augno/api/shared/pagination"
)

// Geolocation represents a geographic location.
type Geolocation struct {
	ID            string
	StreetLine1   *string  `audit:"street_line_1"`
	StreetLine2   *string  `audit:"street_line_2"`
	Locality      *string  `audit:"locality"`
	State         *string  `audit:"state"`
	PostalCode    *string  `audit:"postal_code"`
	Country       string   `audit:"country"`
	GooglePlaceID *string  `audit:"google_place_id"`
	Latitude      *float64 `audit:"latitude"`
	Longitude     *float64 `audit:"longitude"`
}

// Address represents an address with its associated geolocation.
type Address struct {
	ID          string
	Name        string       `audit:"name"`
	Phone       *string      `audit:"phone"`
	Email       *string      `audit:"email"`
	IsDropShip  bool         `audit:"is_drop_ship"`
	Geolocation *Geolocation `audit:"geolocation"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ListAddressesParams contains the parameters for listing addresses.
type ListAddressesParams struct {
	AccountID string
	Cursor    *string
	Limit     int32
	Query     *string
	DropShip  *bool
}

// ListAddressesResult contains the result of listing addresses.
type ListAddressesResult struct {
	Addresses []*Address
	PageInfo  pagination.PageInfo
}

// GetAddressParams contains the parameters for getting a single address.
type GetAddressParams struct {
	AccountID string
	AddressID string
}

// CreateAddressParams contains the parameters for creating an address.
type CreateAddressParams struct {
	AccountID   string
	Name        string
	Phone       *string
	Email       *string
	IsDropShip  bool
	StreetLine1 *string
	StreetLine2 *string
	Locality    *string
	State       *string
	PostalCode  *string
	Country     string
}

// UpdateAddressParams contains the parameters for updating an address.
type UpdateAddressParams struct {
	AccountID   string
	AddressID   string
	Name        *string
	Phone       patch.Field[string]
	Email       patch.Field[string]
	IsDropShip  *bool
	StreetLine1 *string
	StreetLine2 patch.Field[string]
	Locality    *string
	State       *string
	PostalCode  *string
	Country     *string
}

// DeleteAddressParams contains the parameters for deleting an address.
type DeleteAddressParams struct {
	AccountID string
	AddressID string
}

// AddressSuggestion represents an autocomplete suggestion.
type AddressSuggestion struct {
	ID            string
	Description   string
	MainText      string
	SecondaryText string
}

// AddressComponents represents parsed address components.
type AddressComponents struct {
	AddressLine1 string
	AddressLine2 *string
	City         string
	State        string
	PostalCode   string
	Country      string
	CountryCode  string
}

// AddressDetailsResult represents the result of a place details lookup.
type AddressDetailsResult struct {
	Address          *AddressComponents
	FormattedAddress string
}

// ValidatedAddress represents the result of address validation.
type ValidatedAddress struct {
	IsValid            bool
	FormattedAddress   *string
	Components         *AddressComponents
	ValidationMessages []string
}
