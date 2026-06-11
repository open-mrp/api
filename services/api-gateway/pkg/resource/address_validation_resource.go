package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

// Autocomplete address suggestion.
type AddressSuggestion struct {
	// Address suggestion ID.
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

// Result of a place details lookup.
type AddressDetailsResult struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=address_details_result"`
	// Parsed address components.
	Address *AddressComponents `json:"address" validate:"required"`
	// Formatted full address string.
	FormattedAddress string `json:"formatted_address" validate:"required"`
}

// Result of address validation.
type ValidatedAddress struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=validated_address"`
	// Address validation status.
	//
	// - `valid`: the address was validated successfully.
	// - `invalid`: the address could not be validated; see `validation_messages` for the issues found.
	Status constants.AddressValidationStatus `json:"status" validate:"required"`
	// Formatted, single-line address as standardized by the validation service.
	//
	// `null` when the service did not return a formatted address (this can occur regardless of `status`).
	FormattedAddress *string `json:"formatted_address"`
	// Standardized, parsed address components returned by the validation service.
	//
	// `null` when the service did not return parsed components (independent of `status`).
	Components *AddressComponents `json:"components"`
	// Human-readable messages describing issues found during validation.
	//
	// Empty when no issues were reported.
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
