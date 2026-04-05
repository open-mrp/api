package stub

import (
	"context"
	"strings"

	"github.com/augno/api/services/core-service/internal/domain"
	apierror "github.com/augno/api/shared/errors"
)

// AddressValidationSvc is a stub AddressValidationSvc implementation for use in
// test mode. It returns realistic canned responses so that e2e tests can exercise
// the full address validation flow without an external Google Maps API key.
type AddressValidationSvc struct{}

// stubPlaces maps fake place IDs to their address details.
var stubPlaces = map[string]domain.AddressDetailsResult{
	"stub_place_1600_amphitheatre": {
		FormattedAddress: "1600 Amphitheatre Parkway, Mountain View, CA 94043, USA",
		Address: &domain.AddressComponents{
			AddressLine1: "1600 Amphitheatre Parkway",
			City:         "Mountain View",
			State:        "CA",
			PostalCode:   "94043",
			Country:      "United States",
			CountryCode:  "US",
		},
	},
	"stub_place_350_fifth_ave": {
		FormattedAddress: "350 5th Ave, New York, NY 10118, USA",
		Address: &domain.AddressComponents{
			AddressLine1: "350 5th Ave",
			City:         "New York",
			State:        "NY",
			PostalCode:   "10118",
			Country:      "United States",
			CountryCode:  "US",
		},
	},
}

// stubSuggestions maps input substrings to autocomplete suggestions.
var stubSuggestions = []struct {
	match      string
	suggestion domain.AddressSuggestion
}{
	{
		match: "1600 amphitheatre",
		suggestion: domain.AddressSuggestion{
			ID:            "stub_place_1600_amphitheatre",
			Description:   "1600 Amphitheatre Parkway, Mountain View, CA, USA",
			MainText:      "1600 Amphitheatre Parkway",
			SecondaryText: "Mountain View, CA, USA",
		},
	},
	{
		match: "350 fifth",
		suggestion: domain.AddressSuggestion{
			ID:            "stub_place_350_fifth_ave",
			Description:   "350 5th Ave, New York, NY 10118, USA",
			MainText:      "350 5th Ave",
			SecondaryText: "New York, NY, USA",
		},
	},
}

func (s *AddressValidationSvc) Autocomplete(_ context.Context, input string, _ *string) ([]domain.AddressSuggestion, *apierror.APIError) {
	lower := strings.ToLower(input)
	var results []domain.AddressSuggestion
	for _, entry := range stubSuggestions {
		if strings.Contains(lower, entry.match) {
			results = append(results, entry.suggestion)
		}
	}
	return results, nil
}

func (s *AddressValidationSvc) GetPlaceDetails(_ context.Context, placeID string, _ *string) (*domain.AddressDetailsResult, *apierror.APIError) {
	if details, ok := stubPlaces[placeID]; ok {
		return &details, nil
	}
	return nil, apierror.NewResourceNotFoundError("Place not found.")
}

func (s *AddressValidationSvc) ValidateAddress(_ context.Context, addressLine1 string, addressLine2 *string, city, state, postalCode, country string) (*domain.ValidatedAddress, *apierror.APIError) {
	regionCode := stubRegionCode(country)

	// Detect clearly fake addresses by checking for nonsensical state codes or
	// obviously invalid street numbers.
	isFake := state == "ZZ" || strings.Contains(strings.ToLower(city), "faketown") ||
		strings.HasPrefix(addressLine1, "99999")

	if isFake {
		formatted := addressLine1 + ", " + city + ", " + state + " " + postalCode
		return &domain.ValidatedAddress{
			IsValid:          false,
			FormattedAddress: &formatted,
			Components: &domain.AddressComponents{
				AddressLine1: addressLine1,
				City:         city,
				State:        state,
				PostalCode:   postalCode,
				Country:      country,
				CountryCode:  regionCode,
			},
			ValidationMessages: []string{
				"Some address components could not be confirmed",
			},
		}, nil
	}

	formatted := addressLine1
	if addressLine2 != nil && *addressLine2 != "" {
		formatted += ", " + *addressLine2
	}
	formatted += ", " + city + ", " + state + " " + postalCode + ", " + regionCode

	var line2Ptr *string
	if addressLine2 != nil && *addressLine2 != "" {
		line2Ptr = addressLine2
	}

	return &domain.ValidatedAddress{
		IsValid:          true,
		FormattedAddress: &formatted,
		Components: &domain.AddressComponents{
			AddressLine1: addressLine1,
			AddressLine2: line2Ptr,
			City:         city,
			State:        state,
			PostalCode:   postalCode,
			Country:      country,
			CountryCode:  regionCode,
		},
	}, nil
}

func stubRegionCode(country string) string {
	countryMap := map[string]string{
		"united states": "US",
		"usa":           "US",
		"us":            "US",
		"canada":        "CA",
		"ca":            "CA",
	}
	normalized := strings.ToLower(strings.TrimSpace(country))
	if code, ok := countryMap[normalized]; ok {
		return code
	}
	upper := strings.ToUpper(country)
	if len(upper) >= 2 {
		return upper[:2]
	}
	return upper
}
