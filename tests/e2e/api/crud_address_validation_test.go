//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	addressAutocompletePath = "/v1/core/addresses/autocomplete"
	addressDetailsPath      = "/v1/core/addresses/details"
	addressValidatePath     = "/v1/core/addresses/validate"
)

// ---------------------------------------------------------------------------
// Validate Address (POST /v1/core/addresses/validate)
// ---------------------------------------------------------------------------

func TestAddressValidation_ValidAddress(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(addressValidatePath, map[string]any{
		"address_line_1": "1600 Amphitheatre Parkway",
		"city":           "Mountain View",
		"state":          "CA",
		"postal_code":    "94043",
		"country":        "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	result := parseJSON(body)
	assert.Equal(t, "validated_address", jsonField(result, "object"))
	assert.Equal(t, "true", jsonField(result, "is_valid"))
	assert.NotEmpty(t, jsonField(result, "formatted_address"))

	components := jsonObject(result, "components")
	require.NotNil(t, components, "components should be present for a valid address")
	assert.Equal(t, "address_components", jsonField(components, "object"))
	assert.NotEmpty(t, jsonField(components, "address_line_1"))
	assert.NotEmpty(t, jsonField(components, "city"))
	assert.NotEmpty(t, jsonField(components, "state"))
	assert.NotEmpty(t, jsonField(components, "postal_code"))
	assert.NotEmpty(t, jsonField(components, "country"))
	assert.NotEmpty(t, jsonField(components, "country_code"))
}

func TestAddressValidation_ValidAddressWithLine2(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(addressValidatePath, map[string]any{
		"address_line_1": "20 W 34th St",
		"address_line_2": "Suite 100",
		"city":           "New York",
		"state":          "NY",
		"postal_code":    "10001",
		"country":        "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	result := parseJSON(body)
	assert.Equal(t, "validated_address", jsonField(result, "object"))
	assert.NotEmpty(t, jsonField(result, "formatted_address"))

	components := jsonObject(result, "components")
	require.NotNil(t, components, "components should be present")
	assert.NotEmpty(t, jsonField(components, "address_line_1"))
}

func TestAddressValidation_InvalidAddress(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(addressValidatePath, map[string]any{
		"address_line_1": "99999 Nonexistent Blvd",
		"city":           "Faketown",
		"state":          "ZZ",
		"postal_code":    "00000",
		"country":        "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	result := parseJSON(body)
	assert.Equal(t, "validated_address", jsonField(result, "object"))
	isValid := jsonField(result, "is_valid")
	if isValid == "true" {
		msgs, ok := result["validation_messages"].([]any)
		assert.True(t, ok && len(msgs) > 0,
			"a clearly fake address should have validation messages if marked valid: %s", string(body))
	}
}

func TestAddressValidation_MissingRequiredFields(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(addressValidatePath, map[string]any{
		"city":        "Denver",
		"state":       "CO",
		"postal_code": "80202",
		"country":     "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Missing address_line_1 should return 400 or 422, got %d: %s", status, string(body))
}

func TestAddressValidation_MissingCity(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(addressValidatePath, map[string]any{
		"address_line_1": "123 Main St",
		"state":          "CO",
		"postal_code":    "80202",
		"country":        "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Missing city should return 400 or 422, got %d: %s", status, string(body))
}

func TestAddressValidation_MissingCountry(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(addressValidatePath, map[string]any{
		"address_line_1": "123 Main St",
		"city":           "Denver",
		"state":          "CO",
		"postal_code":    "80202",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Missing country should return 400 or 422, got %d: %s", status, string(body))
}

func TestAddressValidation_CountryNameNormalization(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(addressValidatePath, map[string]any{
		"address_line_1": "1600 Amphitheatre Parkway",
		"city":           "Mountain View",
		"state":          "CA",
		"postal_code":    "94043",
		"country":        "United States",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	result := parseJSON(body)
	assert.Equal(t, "validated_address", jsonField(result, "object"))

	components := jsonObject(result, "components")
	require.NotNil(t, components, "components should be present")
	assert.Equal(t, "US", jsonField(components, "country_code"))
}

// ---------------------------------------------------------------------------
// Autocomplete Address (GET /v1/core/addresses/autocomplete)
// ---------------------------------------------------------------------------

func TestAddressAutocomplete_BasicSearch(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(addressAutocompletePath, url.Values{
		"input": {"1600 Amphitheatre Parkway Mountain View"},
	})
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	require.GreaterOrEqual(t, len(list.Data), 1, "should return at least 1 suggestion")

	first := parseJSON(list.Data[0])
	assert.Equal(t, "address_suggestion", jsonField(first, "object"))
	assert.NotEmpty(t, jsonField(first, "id"), "suggestion should have a place ID")
	assert.NotEmpty(t, jsonField(first, "description"))
	assert.NotEmpty(t, jsonField(first, "main_text"))
	assert.NotEmpty(t, jsonField(first, "secondary_text"))
}

func TestAddressAutocomplete_WithSessionToken(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(addressAutocompletePath, url.Values{
		"input":         {"350 Fifth Avenue New York"},
		"session_token": {newIdempotencyKey()},
	})
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 1, "should return at least 1 suggestion")
}

func TestAddressAutocomplete_MissingInput(t *testing.T) {
	t.Parallel()
	statusCode, _, err := apiClient.GetListRaw(addressAutocompletePath, nil)
	require.NoError(t, err)
	assert.True(t, statusCode == 400 || statusCode == 422,
		"Missing input should return 400 or 422, got %d", statusCode)
}

func TestAddressAutocomplete_NonsenseInput(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(addressAutocompletePath, url.Values{
		"input": {"zzzzzzznotanaddress99999xxxxx"},
	})
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
}

// ---------------------------------------------------------------------------
// Get Address Details (GET /v1/core/addresses/details/{id})
// ---------------------------------------------------------------------------

func TestAddressDetails_FromAutocomplete(t *testing.T) {
	t.Parallel()

	// First, get a place ID from autocomplete
	list, _, err := apiClient.GetList(addressAutocompletePath, url.Values{
		"input":         {"1600 Amphitheatre Parkway Mountain View CA"},
		"session_token": {newIdempotencyKey()},
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1, "autocomplete should return at least 1 result")

	first := parseJSON(list.Data[0])
	placeID := jsonField(first, "id")
	require.NotEmpty(t, placeID, "place ID from autocomplete should not be empty")

	// Now look up details for that place ID
	detailsStatus, detailsBody, err := apiClient.GetListRaw(addressDetailsPath+"/"+placeID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, detailsStatus, detailsBody)

	result := parseJSON(detailsBody)
	assert.Equal(t, "address_details_result", jsonField(result, "object"))
	assert.NotEmpty(t, jsonField(result, "formatted_address"))

	address := jsonObject(result, "address")
	require.NotNil(t, address, "address components should be present")
	assert.Equal(t, "address_components", jsonField(address, "object"))
	assert.NotEmpty(t, jsonField(address, "address_line_1"))
	assert.NotEmpty(t, jsonField(address, "city"))
	assert.NotEmpty(t, jsonField(address, "state"))
	assert.NotEmpty(t, jsonField(address, "postal_code"))
	assert.NotEmpty(t, jsonField(address, "country"))
	assert.NotEmpty(t, jsonField(address, "country_code"))
}

func TestAddressDetails_InvalidPlaceID(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.GetListRaw(addressDetailsPath+"/invalid_place_id_zzzzz", nil)
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 404 || status == 500,
		"Invalid place ID should return an error status, got %d", status)
}
