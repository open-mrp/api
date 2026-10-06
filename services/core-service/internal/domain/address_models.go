package domain

import (
	"fmt"
	"time"

	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
	"github.com/open-mrp/api/shared/pagination"
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
	// Timezone is the IANA zone resolved from country and state on write. Nil means it has not been resolved yet, and readers fall back to deriving it.
	Timezone *string `audit:"timezone"`
}

// Address represents an address with its associated geolocation.
type Address struct {
	ID         string
	Name       string  `audit:"name"`
	Phone      *string `audit:"phone"`
	Email      *string `audit:"email"`
	IsDropShip bool    `audit:"is_drop_ship"`
	// ReceiveCalendarID names the days this dock accepts freight, overriding the customer's own calendar. Set when one of a customer's sites keeps different days from the rest.
	ReceiveCalendarID *string      `audit:"receive_calendar_id"`
	Geolocation       *Geolocation `audit:"geolocation"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
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
	AccountID         string
	Name              string
	Phone             *string
	Email             *string
	IsDropShip        bool
	ReceiveCalendarID *string
	StreetLine1       *string
	StreetLine2       *string
	Locality          *string
	State             *string
	PostalCode        *string
	Country           string
}

// UpdateAddressParams contains the parameters for updating an address.
type UpdateAddressParams struct {
	AccountID         string
	AddressID         string
	Name              *string
	Phone             field.Clearable[string]
	Email             field.Clearable[string]
	IsDropShip        *bool
	ReceiveCalendarID field.Clearable[string]
	StreetLine1       *string
	StreetLine2       field.Clearable[string]
	Locality          *string
	State             *string
	PostalCode        *string
	Country           *string
}

// InlineAddressParams is an address saved by the write of the record that uses it. With ID set, that stored address is updated and unset fields keep their values; without it a new address is created, so Name and Country are required.
type InlineAddressParams struct {
	ID                *string
	Name              *string
	Phone             field.Clearable[string]
	Email             field.Clearable[string]
	IsDropShip        *bool
	ReceiveCalendarID field.Clearable[string]
	StreetLine1       *string
	StreetLine2       field.Clearable[string]
	Locality          *string
	State             *string
	PostalCode        *string
	Country           *string
}

// CreateParams is the inline address as a new address in accountID.
func (p InlineAddressParams) CreateParams(accountID string) CreateAddressParams {
	params := CreateAddressParams{
		AccountID:         accountID,
		Phone:             p.Phone.ValuePtr(),
		Email:             p.Email.ValuePtr(),
		IsDropShip:        p.IsDropShip != nil && *p.IsDropShip,
		ReceiveCalendarID: p.ReceiveCalendarID.ValuePtr(),
		StreetLine1:       p.StreetLine1,
		StreetLine2:       p.StreetLine2.ValuePtr(),
		Locality:          p.Locality,
		State:             p.State,
		PostalCode:        p.PostalCode,
	}
	if p.Name != nil {
		params.Name = *p.Name
	}
	if p.Country != nil {
		params.Country = *p.Country
	}
	return params
}

// UpdateParams is the inline address as an update to the stored address its ID names, in accountID.
func (p InlineAddressParams) UpdateParams(accountID string) UpdateAddressParams {
	params := UpdateAddressParams{
		AccountID:         accountID,
		Name:              p.Name,
		Phone:             p.Phone,
		Email:             p.Email,
		IsDropShip:        p.IsDropShip,
		ReceiveCalendarID: p.ReceiveCalendarID,
		StreetLine1:       p.StreetLine1,
		StreetLine2:       p.StreetLine2,
		Locality:          p.Locality,
		State:             p.State,
		PostalCode:        p.PostalCode,
		Country:           p.Country,
	}
	if p.ID != nil {
		params.AddressID = *p.ID
	}
	return params
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

// UnsupportedAddressRegionError is the error for an address in a country Google Address Validation does
// not cover: the caller's input, so a 400 against the country rather than a 500 clients retry.
func UnsupportedAddressRegionError(country string) *apierror.APIError {
	return apierror.NewValidationErrorWithParam(fmt.Sprintf("Address validation is not supported for country %q.", country), "country")
}
