package apiresource

import (
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
)

// A candidate address returned by address autocomplete.
//
// A suggestion is a lookup result from the address provider, not a saved address in your account. Creating an address from one is a separate step.
type AddressSuggestion struct {
	// Identifier of the suggested place.
	//
	// Pass this value as the `id` path parameter of the address details endpoint to retrieve the full parsed address. It is issued by the underlying address provider rather than by OpenMRP, so it is not a durable OpenMRP resource ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=address_suggestion"`
	// Full description of the address.
	Description string `json:"description" validate:"required"`
	// Main text (typically the street address).
	MainText string `json:"main_text" validate:"required"`
	// Secondary text (typically city, state, country).
	SecondaryText string `json:"secondary_text" validate:"required"`
}

var SampleAddressSuggestion = &AddressSuggestion{
	ID:            "ChIJd8BlQ2BZwokRAFUEcm_qrcA",
	Object:        constants.ObjectTypeAddressSuggestion,
	Description:   "123 Main St, Springfield, IL 62701, USA",
	MainText:      "123 Main St",
	SecondaryText: "Springfield, IL 62701, USA",
}

func (*AddressSuggestion) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAddressSuggestion)
}

// Parsed address components.
type AddressComponents struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=address_components"`
	// First line of the street address.
	AddressLine1 string `json:"address_line_1" validate:"required"`
	// Second line of the street address.
	AddressLine2 *string `json:"address_line_2"`
	// City or locality.
	City string `json:"city" validate:"required"`
	// State or administrative area.
	State string `json:"state" validate:"required"`
	// Postal or ZIP code.
	PostalCode string `json:"postal_code" validate:"required"`
	// Country name or code.
	Country string `json:"country" validate:"required"`
	// Two-letter country code.
	CountryCode string `json:"country_code" validate:"required"`
}

// The full address behind an autocomplete suggestion.
type AddressDetailsResult struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=address_details_result"`
	// Parsed address components.
	Address *AddressComponents `json:"address" validate:"required"`
	// Full address formatted as a single line.
	FormattedAddress string `json:"formatted_address" validate:"required"`
}

// The outcome of checking a submitted address against an address validation service.
type ValidatedAddress struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=validated_address"`
	// Whether the address was confirmed as complete and specific enough to ship to.
	//
	// - `valid`: nothing required was missing and the address resolved to a specific building or block.
	// - `invalid`: required components were missing, or the address only resolved to a street or a wider area.
	//
	// When the status is `invalid`, read `validation_messages` and compare `components` against what you submitted to see what to correct.
	Status constants.AddressValidationStatus `json:"status" validate:"required"`
	// Formatted, single-line address as standardized by the validation service.
	//
	// The validation service may omit this regardless of `status`, so it can be absent even for a `valid` address.
	FormattedAddress *string `json:"formatted_address"`
	// Standardized, parsed address components returned by the validation service.
	//
	// The validation service may omit this regardless of `status`, so it can be absent even for a `valid` address.
	Components *AddressComponents `json:"components"`
	// Human-readable messages describing issues found during validation.
	//
	// May be non-empty even when `status` is `valid`, for example when components were inferred or replaced with standardized values. Empty when no issues were reported.
	ValidationMessages []string `json:"validation_messages"`
}

var sampleValidatedFormattedAddress = "123 Main St, Springfield, IL 62701, USA"

var SampleValidatedAddress = &ValidatedAddress{
	Object:           constants.ObjectTypeValidatedAddress,
	Status:           constants.AddressValidationStatusValid,
	FormattedAddress: &sampleValidatedFormattedAddress,
	Components: &AddressComponents{
		Object:       constants.ObjectTypeAddressComponents,
		AddressLine1: "123 Main St",
		City:         "Springfield",
		State:        "IL",
		PostalCode:   "62701",
		Country:      "United States",
		CountryCode:  "US",
	},
	ValidationMessages: []string{},
}

func (*ValidatedAddress) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleValidatedAddress)
}

var SampleAddressDetailsResult = &AddressDetailsResult{
	Object: constants.ObjectTypeAddressDetailsResult,
	Address: &AddressComponents{
		Object:       constants.ObjectTypeAddressComponents,
		AddressLine1: "123 Main St",
		City:         "Springfield",
		State:        "IL",
		PostalCode:   "62701",
		Country:      "United States",
		CountryCode:  "US",
	},
	FormattedAddress: "123 Main St, Springfield, IL 62701, USA",
}

func (*AddressDetailsResult) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAddressDetailsResult)
}
