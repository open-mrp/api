//go:build e2e

package api_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// This file extends coverage for `/v1/core/addresses` (address-validation
// endpoints: PUT actions/validate, GET suggestions, GET details/{id}).
//
// Path constants (addressAutocompletePath, addressDetailsPath,
// addressValidatePath) already exist in crud_address_validation_test.go —
// reused here, not redeclared. That file also already covers the core
// happy-path + missing-required-field cases; this file closes the remaining
// gaps: address_line_2 null/populated assertions, validation_messages
// empty-array assertion, blank-string and explicit-null variants, unknown
// field/query-param rejection, page_info shape, and a tightened
// InvalidPlaceID assertion that no longer tolerates a 5xx.
// ---------------------------------------------------------------------------

// TestCovCoreAddresses_ValidateAddress_Line2NullWhenOmitted asserts that
// components.address_line_2 is null when address_line_2 is not supplied,
// and that validation_messages is an empty array (not null) for a clean
// valid address.
func TestCovCoreAddresses_ValidateAddress_Line2NullWhenOmitted(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Put(addressValidatePath, map[string]any{
		"address_line_1": "1600 Amphitheatre Parkway",
		"city":           "Mountain View",
		"state":          "CA",
		"postal_code":    "94043",
		"country":        "US",
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	result := parseJSON(body)
	assertObjectField(t, result, "validated_address")
	assert.Equal(t, "valid", jsonField(result, "status"))

	msgs, ok := result["validation_messages"].([]any)
	require.True(t, ok, "validation_messages should be present as a JSON array: %s", string(body))
	assert.Empty(t, msgs, "validation_messages should be empty for a clean valid address")

	components := jsonObject(result, "components")
	require.NotNil(t, components)
	assertObjectField(t, components, "address_components")
	assertNilField(t, components, "address_line_2")
}

// TestCovCoreAddresses_ValidateAddress_Line2PopulatedWhenProvided asserts
// that a supplied address_line_2 round-trips into components.address_line_2.
func TestCovCoreAddresses_ValidateAddress_Line2PopulatedWhenProvided(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Put(addressValidatePath, map[string]any{
		"address_line_1": "20 W 34th St",
		"address_line_2": "Suite 100",
		"city":           "New York",
		"state":          "NY",
		"postal_code":    "10001",
		"country":        "US",
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	result := parseJSON(body)
	components := jsonObject(result, "components")
	require.NotNil(t, components)
	assert.Equal(t, "Suite 100", jsonField(components, "address_line_2"))
}

// TestCovCoreAddresses_ValidateAddress_Line2Null asserts that an explicit
// JSON null for the optional address_line_2 field is rejected with 400
// ("cannot be null"), per field.Optional semantics.
func TestCovCoreAddresses_ValidateAddress_Line2Null(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Put(addressValidatePath, map[string]any{
		"address_line_1": "1600 Amphitheatre Parkway",
		"address_line_2": nil,
		"city":           "Mountain View",
		"state":          "CA",
		"postal_code":    "94043",
		"country":        "US",
	})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assertErrorParam(t, errObj, "address_line_2")
}

// TestCovCoreAddresses_ValidateAddress_BlankAddressLine1 asserts a blank
// (empty-string) address_line_1 is rejected with 400, distinct from the
// omitted-key case already covered by TestAddressValidation_MissingRequiredFields.
func TestCovCoreAddresses_ValidateAddress_BlankAddressLine1(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Put(addressValidatePath, map[string]any{
		"address_line_1": "",
		"city":           "Denver",
		"state":          "CO",
		"postal_code":    "80202",
		"country":        "US",
	})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
	assertErrorParam(t, errObj, "address_line_1")
}

// TestCovCoreAddresses_ValidateAddress_BlankCity asserts a blank city is
// rejected with 400.
func TestCovCoreAddresses_ValidateAddress_BlankCity(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Put(addressValidatePath, map[string]any{
		"address_line_1": "123 Main St",
		"city":           "",
		"state":          "CO",
		"postal_code":    "80202",
		"country":        "US",
	})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
	assertErrorParam(t, errObj, "city")
}

// TestCovCoreAddresses_ValidateAddress_BlankState asserts a blank state is
// rejected with 400.
func TestCovCoreAddresses_ValidateAddress_BlankState(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Put(addressValidatePath, map[string]any{
		"address_line_1": "123 Main St",
		"city":           "Denver",
		"state":          "",
		"postal_code":    "80202",
		"country":        "US",
	})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
	assertErrorParam(t, errObj, "state")
}

// TestCovCoreAddresses_ValidateAddress_BlankPostalCode asserts a blank
// postal_code is rejected with 400.
func TestCovCoreAddresses_ValidateAddress_BlankPostalCode(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Put(addressValidatePath, map[string]any{
		"address_line_1": "123 Main St",
		"city":           "Denver",
		"state":          "CO",
		"postal_code":    "",
		"country":        "US",
	})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
	assertErrorParam(t, errObj, "postal_code")
}

// TestCovCoreAddresses_ValidateAddress_BlankCountry asserts a blank country
// is rejected with 400.
func TestCovCoreAddresses_ValidateAddress_BlankCountry(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Put(addressValidatePath, map[string]any{
		"address_line_1": "123 Main St",
		"city":           "Denver",
		"state":          "CO",
		"postal_code":    "80202",
		"country":        "",
	})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
	assertErrorParam(t, errObj, "country")
}

// TestCovCoreAddresses_ValidateAddress_GarbageCountry submits a clearly
// bogus country value and asserts a non-5xx outcome. Per §10.2 of the task
// spec this was a suspected 500 in the real Google-backed implementation,
// but the live e2e stack proxies through core-service's
// AddressValidationSvc stub (services/core-service/internal/infrastructure/stub/address_validation.go),
// which never calls out to Google and derives a region code purely from
// local string logic, so a garbage country currently degrades gracefully
// to a 200 "valid" response rather than an error. Assert the actual
// current (and correct, non-5xx) behavior; do not fabricate a 4xx that the
// live stack does not return.
func TestCovCoreAddresses_ValidateAddress_GarbageCountry(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Put(addressValidatePath, map[string]any{
		"address_line_1": "1600 Amphitheatre Parkway",
		"city":           "Mountain View",
		"state":          "CA",
		"postal_code":    "94043",
		"country":        "Nonexistent Countryland",
	})
	require.NoError(t, err)
	require.NotEqual(t, 500, status, "garbage country must not 500: %s", string(body))
	requireStatus(t, 200, status, body)

	result := parseJSON(body)
	assertObjectField(t, result, "validated_address")
	components := jsonObject(result, "components")
	require.NotNil(t, components)
	assert.Equal(t, "Nonexistent Countryland", jsonField(components, "country"))
	assert.NotEmpty(t, jsonField(components, "country_code"))
}

// TestCovCoreAddresses_ValidateAddress_UnknownFieldRejected asserts PUT
// actions/validate rejects a body containing only an unrecognized field.
func TestCovCoreAddresses_ValidateAddress_UnknownFieldRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Put(addressValidatePath, map[string]any{
		bogusE2EJSONField: "should_be_rejected",
	})
	require.NoError(t, err)
	assertJSONUnknownFieldRejected(t, http.MethodPut, addressValidatePath, status, body)
}

// TestCovCoreAddresses_Suggestions_UnknownQueryParamRejected asserts GET
// suggestions rejects an undeclared query parameter.
func TestCovCoreAddresses_Suggestions_UnknownQueryParamRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(addressAutocompletePath, url.Values{
		"input":            {"1600 Amphitheatre Parkway"},
		bogusE2EQueryParam: {"should_be_rejected"},
	})
	require.NoError(t, err)
	assertUnknownQueryParamRejected(t, addressAutocompletePath, status, body)
}

// TestCovCoreAddresses_Suggestions_UnsupportedPaginationParamsRejected
// asserts that limit/cursor (unsupported by this endpoint — it always
// returns apiresource.NewList(suggestions, apiresource.PageInfo{})) are
// rejected as unknown query params rather than silently accepted.
func TestCovCoreAddresses_Suggestions_UnsupportedPaginationParamsRejected(t *testing.T) {
	t.Parallel()

	limitStatus, limitBody, err := apiClient.GetListRaw(addressAutocompletePath, url.Values{
		"input": {"1600 Amphitheatre Parkway"},
		"limit": {"5"},
	})
	require.NoError(t, err)
	requireStatus(t, 400, limitStatus, limitBody)
	errObj := requireErrorResponse(t, limitBody, "parameter_unknown", "invalid_request_error")
	assertErrorParam(t, errObj, "limit")

	cursorStatus, cursorBody, err := apiClient.GetListRaw(addressAutocompletePath, url.Values{
		"input":  {"1600 Amphitheatre Parkway"},
		"cursor": {"abc"},
	})
	require.NoError(t, err)
	requireStatus(t, 400, cursorStatus, cursorBody)
	errObj2 := requireErrorResponse(t, cursorBody, "parameter_unknown", "invalid_request_error")
	assertErrorParam(t, errObj2, "cursor")
}

// TestCovCoreAddresses_Suggestions_PageInfoShape asserts the always-empty
// PageInfo shape this endpoint returns (apiresource.NewList(suggestions,
// apiresource.PageInfo{})): has_next_page/has_prev_page false and both
// page URLs null, regardless of how many suggestions come back.
func TestCovCoreAddresses_Suggestions_PageInfoShape(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(addressAutocompletePath, url.Values{
		"input": {"1600 Amphitheatre Parkway Mountain View"},
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	assert.False(t, list.PageInfo.HasNextPage, "suggestions list should never report has_next_page")
	assert.False(t, list.PageInfo.HasPrevPage, "suggestions list should never report has_prev_page")
	assert.Nil(t, list.PageInfo.NextPageURL, "suggestions list should never populate next_page_url")
	assert.Nil(t, list.PageInfo.PreviousPageURL, "suggestions list should never populate previous_page_url")
}

// TestCovCoreAddresses_Suggestions_BlankInputRejected asserts a blank
// (empty-string) input query param is rejected with 400, distinct from the
// omitted-key case already covered by TestAddressAutocomplete_MissingInput.
func TestCovCoreAddresses_Suggestions_BlankInputRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(addressAutocompletePath, url.Values{
		"input": {""},
	})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assertErrorParam(t, errObj, "input")
}

// TestCovCoreAddresses_Details_FromAutocomplete_AllFields chains
// autocomplete -> details (the required pattern for this stateless group,
// since Google/stub place IDs are not seedable) and asserts every
// AddressDetailsResult and nested AddressComponents json field, including
// the never-before-asserted address_line_2 (null, since neither stub place
// entry has a second line — see
// services/core-service/internal/infrastructure/stub/address_validation.go).
func TestCovCoreAddresses_Details_FromAutocomplete_AllFields(t *testing.T) {
	t.Parallel()

	list, _, err := apiClient.GetList(addressAutocompletePath, url.Values{
		"input":         {"350 Fifth Avenue New York"},
		"session_token": {newIdempotencyKey()},
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	first := parseJSON(list.Data[0])
	placeID := jsonField(first, "id")
	require.NotEmpty(t, placeID)

	status, body, err := apiClient.GetListRaw(addressDetailsPath+"/"+placeID, url.Values{
		"session_token": {newIdempotencyKey()},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	result := parseJSON(body)
	assertObjectField(t, result, "address_details_result")
	assert.NotEmpty(t, jsonField(result, "formatted_address"))

	address := jsonObject(result, "address")
	require.NotNil(t, address)
	assertObjectField(t, address, "address_components")
	assert.NotEmpty(t, jsonField(address, "address_line_1"))
	assertNilField(t, address, "address_line_2")
	assert.NotEmpty(t, jsonField(address, "city"))
	assert.NotEmpty(t, jsonField(address, "state"))
	assert.NotEmpty(t, jsonField(address, "postal_code"))
	assert.NotEmpty(t, jsonField(address, "country"))
	assert.NotEmpty(t, jsonField(address, "country_code"))
}

// TestCovCoreAddresses_Details_InvalidPlaceID_No5xx supersedes the
// coverage gap flagged in the task spec (§10.1): the pre-existing
// TestAddressDetails_InvalidPlaceID in crud_address_validation_test.go
// tolerates a 500 (`status == 400 || 404 || 500`), which conflicts with
// the repo rule "never skip 5xx in e2e". Per this task's hard rules this
// new file must not modify that pre-existing test, so this test asserts
// the tightened, correct expectation independently: an invalid/unknown
// place ID must return 400 or 404, never 5xx. On the live e2e stack this
// currently passes cleanly (404 "Place not found." via the
// AddressValidationSvc stub's GetPlaceDetails, which uses a plain map
// lookup with no upstream call) — no 500 was observed, so no backend bug
// is being flagged here for this particular path; the 500-tolerant
// assertion in the pre-existing test is simply stale/over-broad and should
// be narrowed by whoever next touches that file.
func TestCovCoreAddresses_Details_InvalidPlaceID_No5xx(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(addressDetailsPath+"/invalid_place_id_zzzzz", nil)
	require.NoError(t, err)
	require.NotEqual(t, 500, status, "invalid place ID must not 500: %s", string(body))
	assert.True(t, status == 400 || status == 404,
		"Invalid place ID should return 400 or 404, got %d: %s", status, string(body))
	if status == 404 {
		requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
	}
}

// TestCovCoreAddresses_Suggestions_ListItemAllFields is a focused
// single-suggestion check asserting every AddressSuggestion json field
// (id, object, description, main_text, secondary_text) with exact expected
// stub values, complementing the looser NotEmpty assertions in
// TestAddressAutocomplete_BasicSearch.
func TestCovCoreAddresses_Suggestions_ListItemAllFields(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(addressAutocompletePath, url.Values{
		"input": {"1600 Amphitheatre Parkway Mountain View"},
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	first := parseJSON(list.Data[0])
	assertObjectField(t, first, "address_suggestion")
	assert.Equal(t, "stub_place_1600_amphitheatre", jsonField(first, "id"))
	assert.Equal(t, "1600 Amphitheatre Parkway, Mountain View, CA, USA", jsonField(first, "description"))
	assert.Equal(t, "1600 Amphitheatre Parkway", jsonField(first, "main_text"))
	assert.Equal(t, "Mountain View, CA, USA", jsonField(first, "secondary_text"))
}
