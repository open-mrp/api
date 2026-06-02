package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/augno/api/services/core-service/internal/domain"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var addressValidationSvcTracer = tracing.GetTracer("core-service.address_validation_service")

const (
	placesBaseURL        = "https://places.googleapis.com/v1"
	addressValidationURL = "https://addressvalidation.googleapis.com/v1:validateAddress"
)

type addressValidationSvcImpl struct {
	apiKey string
}

type AddressValidationSvcConfig struct {
	GoogleMapsAPIKey string
}

func NewAddressValidationSvc(config *AddressValidationSvcConfig) domain.AddressValidationSvc {
	return &addressValidationSvcImpl{
		apiKey: config.GoogleMapsAPIKey,
	}
}

// Autocomplete calls Google Places without tenant identity checks; used from public OpenAPI for checkout-style flows where callers may be unauthenticated.
func (s *addressValidationSvcImpl) Autocomplete(ctx context.Context, input string, sessionToken *string) ([]domain.AddressSuggestion, *apierror.APIError) {
	ctx, span := addressValidationSvcTracer.Start(ctx, "service.address_validation.autocomplete")
	defer span.End()

	if s.apiKey == "" {
		return nil, tracing.Trace(span, apierror.NewInternalError(fmt.Errorf("google maps api key not configured"), "Address autocomplete service not configured."))
	}

	body := map[string]any{
		"input":                input,
		"includedPrimaryTypes": []string{"street_address", "premise", "subpremise"},
	}
	if sessionToken != nil {
		body["sessionToken"] = *sessionToken
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to marshal autocomplete request."))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, placesBaseURL+"/places:autocomplete", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to create autocomplete request."))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", s.apiKey)

	resp, err := http.DefaultClient.Do(req) // #nosec G704 -- URL constructed from server-side config
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Address autocomplete service error."))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, tracing.Trace(span, apierror.NewInternalError(fmt.Errorf("google places autocomplete returned status %d", resp.StatusCode), "Address autocomplete service error."))
	}

	var data struct {
		Suggestions []struct {
			PlacePrediction struct {
				PlaceID string `json:"placeId"`
				Text    struct {
					Text string `json:"text"`
				} `json:"text"`
				StructuredFormat struct {
					MainText struct {
						Text string `json:"text"`
					} `json:"mainText"`
					SecondaryText struct {
						Text string `json:"text"`
					} `json:"secondaryText"`
				} `json:"structuredFormat"`
			} `json:"placePrediction"`
		} `json:"suggestions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to decode autocomplete response."))
	}

	suggestions := make([]domain.AddressSuggestion, len(data.Suggestions))
	for i, s := range data.Suggestions {
		suggestions[i] = domain.AddressSuggestion{
			ID:            s.PlacePrediction.PlaceID,
			Description:   s.PlacePrediction.Text.Text,
			MainText:      s.PlacePrediction.StructuredFormat.MainText.Text,
			SecondaryText: s.PlacePrediction.StructuredFormat.SecondaryText.Text,
		}
	}

	return suggestions, nil
}

// GetPlaceDetails calls Google Places without tenant identity checks; pairs with Autocomplete for public address entry flows.
func (s *addressValidationSvcImpl) GetPlaceDetails(ctx context.Context, placeID string, sessionToken *string) (*domain.AddressDetailsResult, *apierror.APIError) {
	ctx, span := addressValidationSvcTracer.Start(ctx, "service.address_validation.place_details")
	defer span.End()

	if s.apiKey == "" {
		return nil, tracing.Trace(span, apierror.NewInternalError(fmt.Errorf("google maps api key not configured"), "Address service not configured."))
	}

	placeResource := normalizePlaceResourceName(placeID)
	url := fmt.Sprintf("%s/%s?fields=addressComponents,formattedAddress,postalAddress", placesBaseURL, placeResource)
	if sessionToken != nil {
		url += "&sessionToken=" + *sessionToken
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to create place details request."))
	}
	req.Header.Set("X-Goog-Api-Key", s.apiKey)

	resp, err := http.DefaultClient.Do(req) // #nosec G704 -- URL constructed from server-side config
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Address details service error."))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, tracing.Trace(span, apierror.NewInternalError(fmt.Errorf("google places details returned status %d", resp.StatusCode), "Address details service error."))
	}

	var data struct {
		AddressComponents []addressComponent `json:"addressComponents"`
		FormattedAddress  string             `json:"formattedAddress"`
		PostalAddress     *postalAddress     `json:"postalAddress"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to decode place details response."))
	}

	components := parseAddressComponents(data.AddressComponents)
	applyPostalAddressFallback(components, data.PostalAddress)

	return &domain.AddressDetailsResult{
		Address:          components,
		FormattedAddress: data.FormattedAddress,
	}, nil
}

// ValidateAddress calls Google Address Validation without tenant identity checks; exposed publicly alongside autocomplete/details.
func (s *addressValidationSvcImpl) ValidateAddress(ctx context.Context, addressLine1 string, addressLine2 *string, city, state, postalCode, country string) (*domain.ValidatedAddress, *apierror.APIError) {
	ctx, span := addressValidationSvcTracer.Start(ctx, "service.address_validation.validate")
	defer span.End()

	if s.apiKey == "" {
		return nil, tracing.Trace(span, apierror.NewInternalError(fmt.Errorf("google maps api key not configured"), "Address validation service not configured."))
	}

	addressLines := []string{addressLine1}
	if addressLine2 != nil && *addressLine2 != "" {
		addressLines = append(addressLines, *addressLine2)
	}

	regionCode := getRegionCode(country)

	body := map[string]any{
		"address": map[string]any{
			"addressLines":       addressLines,
			"locality":           city,
			"administrativeArea": state,
			"postalCode":         postalCode,
			"regionCode":         regionCode,
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to marshal validation request."))
	}

	url := fmt.Sprintf("%s?key=%s", addressValidationURL, s.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to create validation request."))
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req) // #nosec G704 -- URL constructed from server-side config
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Address validation service error."))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, tracing.Trace(span, apierror.NewInternalError(fmt.Errorf("google address validation returned status %d: %s", resp.StatusCode, string(respBody)), "Address validation service error."))
	}

	var data struct {
		Result struct {
			Verdict struct {
				AddressComplete          bool   `json:"addressComplete"`
				ValidationGranularity    string `json:"validationGranularity"`
				HasUnconfirmedComponents bool   `json:"hasUnconfirmedComponents"`
				HasInferredComponents    bool   `json:"hasInferredComponents"`
				HasReplacedComponents    bool   `json:"hasReplacedComponents"`
			} `json:"verdict"`
			Address struct {
				FormattedAddress string `json:"formattedAddress"`
				PostalAddress    struct {
					RegionCode         string   `json:"regionCode"`
					AddressLines       []string `json:"addressLines"`
					Locality           string   `json:"locality"`
					AdministrativeArea string   `json:"administrativeArea"`
					PostalCode         string   `json:"postalCode"`
				} `json:"postalAddress"`
			} `json:"address"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to decode validation response."))
	}

	verdict := data.Result.Verdict

	var validationMessages []string
	if verdict.HasUnconfirmedComponents {
		validationMessages = append(validationMessages, "Some address components could not be confirmed")
	}
	if verdict.HasInferredComponents {
		validationMessages = append(validationMessages, "Some address components were inferred")
	}
	if verdict.HasReplacedComponents {
		validationMessages = append(validationMessages, "Some address components were replaced with standardized values")
	}

	isValid := verdict.AddressComplete &&
		verdict.ValidationGranularity != "OTHER" &&
		verdict.ValidationGranularity != "ROUTE"

	result := &domain.ValidatedAddress{
		IsValid: isValid,
	}

	if data.Result.Address.FormattedAddress != "" {
		fa := data.Result.Address.FormattedAddress
		result.FormattedAddress = &fa
	}

	if len(validationMessages) > 0 {
		result.ValidationMessages = validationMessages
	}

	postal := data.Result.Address.PostalAddress
	if postal.RegionCode != "" || len(postal.AddressLines) > 0 {
		rc := postal.RegionCode
		if rc == "" {
			rc = regionCode
		}

		line1 := addressLine1
		if len(postal.AddressLines) > 0 {
			line1 = postal.AddressLines[0]
		}
		var line2 string
		if len(postal.AddressLines) > 1 {
			line2 = postal.AddressLines[1]
		}

		// Line2 preservation logic
		if line2 == "" && addressLine2 != nil && *addressLine2 != "" {
			origLine2 := strings.TrimSpace(*addressLine2)
			if origLine2 != "" {
				lower1 := strings.ToLower(line1)
				lowerOrig := strings.ToLower(origLine2)
				if strings.HasSuffix(lower1, lowerOrig) && len(line1) > len(origLine2) {
					line2 = line1[len(line1)-len(origLine2):]
					line1 = strings.TrimRight(line1[:len(line1)-len(origLine2)], ", ")
				} else {
					match := unitDesignatorRegex.FindStringSubmatch(line1)
					if match != nil {
						line1 = match[1]
						line2 = match[2]
					}
				}
			}
		}

		resolvedCity := postal.Locality
		if resolvedCity == "" {
			resolvedCity = city
		}
		resolvedState := postal.AdministrativeArea
		if resolvedState == "" {
			resolvedState = state
		}
		resolvedPostal := postal.PostalCode
		if resolvedPostal == "" {
			resolvedPostal = postalCode
		}

		var line2Ptr *string
		if line2 != "" {
			line2Ptr = &line2
		} else if addressLine2 != nil {
			line2Ptr = addressLine2
		}

		result.Components = &domain.AddressComponents{
			AddressLine1: line1,
			AddressLine2: line2Ptr,
			City:         resolvedCity,
			State:        resolvedState,
			PostalCode:   resolvedPostal,
			Country:      rc,
			CountryCode:  rc,
		}
	}

	return result, nil
}

var unitDesignatorRegex = regexp.MustCompile(`(?i)^(.+),?\s+((?:Suite|Ste|Apt|Apartment|Unit|Bldg|Building|Fl|Floor|Rm|Room|Dept|Department|#)\s*\S+.*)$`)

func getRegionCode(country string) string {
	countryMap := map[string]string{
		"united states":  "US",
		"usa":            "US",
		"us":             "US",
		"canada":         "CA",
		"ca":             "CA",
		"united kingdom": "GB",
		"uk":             "GB",
		"gb":             "GB",
		"australia":      "AU",
		"au":             "AU",
		"germany":        "DE",
		"de":             "DE",
		"france":         "FR",
		"fr":             "FR",
		"mexico":         "MX",
		"mx":             "MX",
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
