package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

// AddressSuggestion represents an autocomplete suggestion.
type AddressSuggestion struct {
	// The Google Places ID for the suggestion.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=address_suggestion"`
	// The full text description of the suggestion.
	Description string `json:"description" validate:"required"`
	// The main text of the suggestion (typically the street address).
	MainText string `json:"main_text" validate:"required"`
	// The secondary text of the suggestion (typically city, state, country).
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

// AddressComponents represents parsed address components.
type AddressComponents struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=address_components"`
	// The first line of the street address.
	AddressLine1 string `json:"address_line_1" validate:"required"`
	// The second line of the street address.
	AddressLine2 *string `json:"address_line_2"`
	// The city.
	City string `json:"city" validate:"required"`
	// The state or administrative area.
	State string `json:"state" validate:"required"`
	// The postal or zip code.
	PostalCode string `json:"postal_code" validate:"required"`
	// The country name or code.
	Country string `json:"country" validate:"required"`
	// The two-letter country code.
	CountryCode string `json:"country_code" validate:"required"`
}

// AddressDetailsResult represents the result of a place details lookup.
type AddressDetailsResult struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=address_details_result"`
	// The parsed address components.
	Address *AddressComponents `json:"address" validate:"required"`
	// The formatted full address string.
	FormattedAddress string `json:"formatted_address" validate:"required"`
}

// ValidatedAddress represents the result of address validation.
type ValidatedAddress struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=validated_address"`
	// Whether the address is considered valid.
	IsValid bool `json:"is_valid"`
	// The formatted address as returned by the validation service.
	FormattedAddress *string `json:"formatted_address"`
	// The standardized address components.
	Components *AddressComponents `json:"components"`
	// Validation messages describing any issues found.
	ValidationMessages []string `json:"validation_messages"`
}

var sampleValidatedFormattedAddress = "123 Main St, Springfield, IL 62701, USA"

var SampleValidatedAddress = &ValidatedAddress{
	Object:           constants.ObjectTypeValidatedAddress,
	IsValid:          true,
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
