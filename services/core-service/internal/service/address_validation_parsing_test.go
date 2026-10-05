package service

import (
	"net/http"
	"testing"

	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifyAddressValidationError_UnsupportedRegion(t *testing.T) {
	t.Parallel()

	body := []byte(`{"error":{"code":400,"message":"Unsupported region code: \"BT\".","status":"INVALID_ARGUMENT"}}`)
	apiErr := classifyAddressValidationError(http.StatusBadRequest, body, "BT")

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusBadRequest, apierror.GetHTTPStatusCode(apiErr.Code))
	assert.Equal(t, "country", apiErr.Param)
	assert.Contains(t, apiErr.PublicMessage, "BT")
}

func TestClassifyAddressValidationError_OtherBadRequest(t *testing.T) {
	t.Parallel()

	body := []byte(`{"error":{"code":400,"message":"Invalid value at 'address.postal_code'.","status":"INVALID_ARGUMENT"}}`)
	apiErr := classifyAddressValidationError(http.StatusBadRequest, body, "US")

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusBadRequest, apierror.GetHTTPStatusCode(apiErr.Code))
	assert.Empty(t, apiErr.Param)
}

func TestClassifyAddressValidationError_UpstreamFailure(t *testing.T) {
	t.Parallel()

	body := []byte(`{"error":{"code":503,"message":"The service is currently unavailable.","status":"UNAVAILABLE"}}`)
	apiErr := classifyAddressValidationError(http.StatusServiceUnavailable, body, "US")

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusInternalServerError, apierror.GetHTTPStatusCode(apiErr.Code))
}

func TestNormalizePlaceResourceName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "places/ChIJabc", normalizePlaceResourceName("ChIJabc"))
	assert.Equal(t, "places/ChIJabc", normalizePlaceResourceName("places/ChIJabc"))
}

func TestParseAddressComponents_USStreetAddress(t *testing.T) {
	t.Parallel()

	got := parseAddressComponents([]addressComponent{
		{ShortText: "1600", LongText: "1600", Types: []string{"street_number"}},
		{ShortText: "Amphitheatre Pkwy", LongText: "Amphitheatre Parkway", Types: []string{"route"}},
		{ShortText: "Mountain View", LongText: "Mountain View", Types: []string{"locality"}},
		{ShortText: "CA", LongText: "California", Types: []string{"administrative_area_level_1"}},
		{ShortText: "94043", LongText: "94043", Types: []string{"postal_code"}},
		{ShortText: "US", LongText: "United States", Types: []string{"country"}},
	})

	assert.Equal(t, "1600 Amphitheatre Pkwy", got.AddressLine1)
	assert.Equal(t, "Mountain View", got.City)
	assert.Equal(t, "CA", got.State)
	assert.Equal(t, "94043", got.PostalCode)
	assert.Equal(t, "US", got.CountryCode)
}

func TestParseAddressComponents_UKPostalTown(t *testing.T) {
	t.Parallel()

	got := parseAddressComponents([]addressComponent{
		{LongText: "The Shrubberies", ShortText: "", Types: []string{"premise"}},
		{LongText: "Selby", ShortText: "Selby", Types: []string{"postal_town"}},
		{LongText: "North Yorkshire", ShortText: "North Yorkshire", Types: []string{"administrative_area_level_1"}},
		{LongText: "YO8", ShortText: "YO8", Types: []string{"postal_code"}},
		{LongText: "United Kingdom", ShortText: "GB", Types: []string{"country"}},
	})

	assert.Equal(t, "The Shrubberies", got.AddressLine1)
	assert.Equal(t, "Selby", got.City)
	assert.NotEmpty(t, got.PostalCode)
}

func TestParseAddressComponents_PrefersLongTextWhenShortEmpty(t *testing.T) {
	t.Parallel()

	got := parseAddressComponents([]addressComponent{
		{LongText: "350 5th Ave", ShortText: "", Types: []string{"route"}},
		{LongText: "New York", ShortText: "", Types: []string{"locality"}},
		{LongText: "New York", ShortText: "NY", Types: []string{"administrative_area_level_1"}},
		{LongText: "10118", ShortText: "", Types: []string{"postal_code"}},
		{LongText: "United States", ShortText: "US", Types: []string{"country"}},
	})

	assert.Equal(t, "350 5th Ave", got.AddressLine1)
	assert.Equal(t, "New York", got.City)
	assert.Equal(t, "NY", got.State)
	assert.Equal(t, "10118", got.PostalCode)
}

func TestApplyPostalAddressFallback(t *testing.T) {
	t.Parallel()

	got := parseAddressComponents([]addressComponent{
		{ShortText: "US", Types: []string{"country"}},
	})
	applyPostalAddressFallback(got, &postalAddress{
		RegionCode:         "US",
		AdministrativeArea: "CA",
		Locality:           "Mountain View",
		PostalCode:         "94043",
		AddressLines:       []string{"1600 Amphitheatre Parkway"},
	})

	require.NotNil(t, got)
	assert.Equal(t, "1600 Amphitheatre Parkway", got.AddressLine1)
	assert.Equal(t, "Mountain View", got.City)
	assert.Equal(t, "CA", got.State)
	assert.Equal(t, "94043", got.PostalCode)
}
