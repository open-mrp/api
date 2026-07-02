//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCovSalesAddresses_CreateResponseShape confirms the create response has
// the expected id prefix, object type, and valid RFC3339 timestamps.
func TestCovSalesAddresses_CreateResponseShape(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-addr-shape")

	resp, err := apiClient.PostFull(addressesPath, map[string]any{
		"name":    name,
		"country": "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	created := parseJSON(resp.Body)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(addressesPath + "/" + id)

	assertIDFormat(t, id, "ad")
	assertObjectField(t, created, "address")
	assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")

	geo := jsonObject(created, "geolocation")
	require.NotNil(t, geo, "geolocation should be present")
	assertIDFormat(t, jsonField(geo, "id"), "gl")
	assertObjectField(t, geo, "geolocation")
}

// TestCovSalesAddresses_CreateOmittedFieldsDefault confirms that creating an
// address with only the required fields (name, country) leaves every
// optional field null and defaults type to "standard".
func TestCovSalesAddresses_CreateOmittedFieldsDefault(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-addr-omit")

	status, body, err := apiClient.Post(addressesPath, map[string]any{
		"name":    name,
		"country": "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	created := parseJSON(body)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(addressesPath + "/" + id)

	assert.Equal(t, name, jsonField(created, "name"))
	assertNilField(t, created, "phone")
	assertNilField(t, created, "email")
	assert.Equal(t, "standard", jsonField(created, "type"), "type should default to standard when omitted")

	geo := jsonObject(created, "geolocation")
	require.NotNil(t, geo, "geolocation should always be present")
	assertNilField(t, geo, "street_line_1")
	assertNilField(t, geo, "street_line_2")
	assertNilField(t, geo, "locality")
	assertNilField(t, geo, "state")
	assertNilField(t, geo, "postal_code")
	assert.Equal(t, "US", jsonField(geo, "country"))
}

// TestCovSalesAddresses_UpdateClearNullableFields confirms the highest-risk
// untested path: patching phone/email/street_line_2 with JSON null clears
// each field back to null in the response, exercising the non-pointer
// field.Clearable[T] update path (prodBugSuspect #3 in the task doc).
func TestCovSalesAddresses_UpdateClearNullableFields(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-addr-clear")

	createStatus, createBody, err := apiClient.Post(addressesPath, map[string]any{
		"name":    name,
		"country": "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(addressesPath + "/" + id)

	// First set the three clearable fields to non-null values.
	setStatus, setBody, err := apiClient.Patch(addressesPath+"/"+id, map[string]any{
		"phone":         "555-000-0000",
		"email":         name + "@e2e-test.augno.com",
		"street_line_2": "Suite 5",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, setStatus, setBody)
	setResp := parseJSON(setBody)
	assert.Equal(t, "555-000-0000", jsonField(setResp, "phone"))
	assert.Equal(t, name+"@e2e-test.augno.com", jsonField(setResp, "email"))
	setGeo := jsonObject(setResp, "geolocation")
	require.NotNil(t, setGeo)
	assert.Equal(t, "Suite 5", jsonField(setGeo, "street_line_2"))

	// Now clear all three via explicit JSON null.
	clearStatus, clearBody, err := apiClient.Patch(addressesPath+"/"+id, map[string]any{
		"phone":         nil,
		"email":         nil,
		"street_line_2": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, clearStatus, clearBody)

	cleared := parseJSON(clearBody)
	assertNilField(t, cleared, "phone")
	assertNilField(t, cleared, "email")
	clearedGeo := jsonObject(cleared, "geolocation")
	require.NotNil(t, clearedGeo)
	assertNilField(t, clearedGeo, "street_line_2")

	// Name should be unaffected by the clear.
	assert.Equal(t, name, jsonField(cleared, "name"))
}

// TestCovSalesAddresses_UpdateClearPhoneOnly confirms an individual clearable
// field (phone) can be nulled out independently without touching email.
func TestCovSalesAddresses_UpdateClearPhoneOnly(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-addr-clear-phone")
	email := name + "@e2e-test.augno.com"

	createStatus, createBody, err := apiClient.Post(addressesPath, map[string]any{
		"name":    name,
		"country": "US",
		"phone":   "555-111-1111",
		"email":   email,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(addressesPath + "/" + id)

	patchStatus, patchBody, err := apiClient.Patch(addressesPath+"/"+id, map[string]any{
		"phone": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	assertNilField(t, updated, "phone")
	assert.Equal(t, email, jsonField(updated, "email"), "email should be untouched by clearing phone")
}

// TestCovSalesAddresses_UpdateIdempotent confirms a PATCH replayed with the
// same idempotency key returns the same resulting state without erroring.
func TestCovSalesAddresses_UpdateIdempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-addr-patch-idem")

	createStatus, createBody, err := apiClient.Post(addressesPath, map[string]any{
		"name":    name,
		"country": "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(addressesPath + "/" + id)

	updatedName := uniqueName("e2e-addr-patch-idem-v2")
	idemKey := newIdempotencyKey()
	payload := map[string]any{"name": updatedName}

	status1, body1, err := apiClient.Patch(addressesPath+"/"+id, payload, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)
	first := parseJSON(body1)

	status2, body2, err := apiClient.Patch(addressesPath+"/"+id, payload, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)
	second := parseJSON(body2)

	assert.Equal(t, jsonField(first, "id"), jsonField(second, "id"))
	assert.Equal(t, updatedName, jsonField(second, "name"))
	assert.Equal(t, jsonField(first, "updated_at"), jsonField(second, "updated_at"),
		"replayed idempotent PATCH should return the identical cached response, not perform a second update")
}

// TestCovSalesAddresses_CreateValidation_InvalidEmail confirms an
// obviously-malformed email is rejected on create via the custom_email
// validator.
func TestCovSalesAddresses_CreateValidation_InvalidEmail(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(addressesPath, map[string]any{
		"name":    uniqueName("e2e-addr-bademail"),
		"country": "US",
		"email":   "not-an-email",
	}, newIdempotencyKey())
	require.NoError(t, err)

	if status == 201 {
		if created := parseJSON(body); created != nil {
			if id := jsonField(created, "id"); id != "" {
				apiClient.Delete(addressesPath + "/" + id)
			}
		}
	}

	require.True(t, status == 400 || status == 422,
		"Invalid email on create should return 400 or 422, got %d: %s", status, string(body))
}

// TestCovSalesAddresses_CreateValidation_CountryTooLong confirms a 3-character
// country code is rejected (max=2).
func TestCovSalesAddresses_CreateValidation_CountryTooLong(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(addressesPath, map[string]any{
		"name":    uniqueName("e2e-addr-badcountry"),
		"country": "USA",
	}, newIdempotencyKey())
	require.NoError(t, err)

	if status == 201 {
		if created := parseJSON(body); created != nil {
			if id := jsonField(created, "id"); id != "" {
				apiClient.Delete(addressesPath + "/" + id)
			}
		}
	}

	require.True(t, status == 400 || status == 422,
		"Country longer than 2 chars should return 400 or 422, got %d: %s", status, string(body))
}

// TestCovSalesAddresses_CreateValidation_PhoneTooLong confirms an over-max
// length (256 chars) phone value is rejected on create.
func TestCovSalesAddresses_CreateValidation_PhoneTooLong(t *testing.T) {
	t.Parallel()
	longPhone := make([]byte, 256)
	for i := range longPhone {
		longPhone[i] = 'a'
	}

	status, body, err := apiClient.Post(addressesPath, map[string]any{
		"name":    uniqueName("e2e-addr-longphone"),
		"country": "US",
		"phone":   string(longPhone),
	}, newIdempotencyKey())
	require.NoError(t, err)

	if status == 201 {
		if created := parseJSON(body); created != nil {
			if id := jsonField(created, "id"); id != "" {
				apiClient.Delete(addressesPath + "/" + id)
			}
		}
	}

	require.True(t, status == 400 || status == 422,
		"Phone over 255 chars should return 400 or 422, got %d: %s", status, string(body))
}

// TestCovSalesAddresses_CreateValidation_NameTooLong confirms an over-max
// length (256 chars) name value is rejected on create.
func TestCovSalesAddresses_CreateValidation_NameTooLong(t *testing.T) {
	t.Parallel()
	longName := make([]byte, 256)
	for i := range longName {
		longName[i] = 'a'
	}

	status, body, err := apiClient.Post(addressesPath, map[string]any{
		"name":    string(longName),
		"country": "US",
	}, newIdempotencyKey())
	require.NoError(t, err)

	if status == 201 {
		if created := parseJSON(body); created != nil {
			if id := jsonField(created, "id"); id != "" {
				apiClient.Delete(addressesPath + "/" + id)
			}
		}
	}

	require.True(t, status == 400 || status == 422,
		"Name over 255 chars should return 400 or 422, got %d: %s", status, string(body))
}

// TestCovSalesAddresses_CreateValidation_InvalidType confirms the "type"
// field rejects a value outside the standard/drop_ship enum on create.
// Live verification shows this IS validated (400), contradicting the
// task doc's prodBugSuspect #1 which flagged it as a suspected silent-accept.
func TestCovSalesAddresses_CreateValidation_InvalidType(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(addressesPath, map[string]any{
		"name":    uniqueName("e2e-addr-badtype"),
		"country": "US",
		"type":    "bogus",
	}, newIdempotencyKey())
	require.NoError(t, err)

	if status == 201 {
		if created := parseJSON(body); created != nil {
			if id := jsonField(created, "id"); id != "" {
				apiClient.Delete(addressesPath + "/" + id)
			}
		}
	}

	require.True(t, status == 400 || status == 422,
		"Invalid type enum on create should return 400 or 422, got %d: %s", status, string(body))
}

// TestCovSalesAddresses_UpdateValidation_InvalidType confirms the "type"
// field rejects a value outside the standard/drop_ship enum on update, same
// as create.
func TestCovSalesAddresses_UpdateValidation_InvalidType(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-addr-badtype-upd")

	createStatus, createBody, err := apiClient.Post(addressesPath, map[string]any{
		"name":    name,
		"country": "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(addressesPath + "/" + id)

	status, body, err := apiClient.Patch(addressesPath+"/"+id, map[string]any{
		"type": "bogus",
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.True(t, status == 400 || status == 422,
		"Invalid type enum on update should return 400 or 422, got %d: %s", status, string(body))
}

// TestCovSalesAddresses_UpdateValidation_InvalidEmail documents
// prodBugSuspect #2: AddressInput.Email (create) has a custom_email
// validator but UpdateAddressRequest.Email does not, so this asserts the
// CORRECT/desired behavior (reject, matching create's contract) which the
// live stack currently fails (accepts with 200). This test is intentionally
// red until the backend closes the validator gap — see confirmedBugs.
func TestCovSalesAddresses_UpdateValidation_InvalidEmail(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-addr-bademail-upd")

	createStatus, createBody, err := apiClient.Post(addressesPath, map[string]any{
		"name":    name,
		"country": "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(addressesPath + "/" + id)

	status, body, err := apiClient.Patch(addressesPath+"/"+id, map[string]any{
		"email": "not-an-email",
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.True(t, status == 400 || status == 422,
		"Invalid email on update should return 400 or 422 to match create's validation contract, got %d: %s", status, string(body))
}

// TestCovSalesAddresses_GetNotFound_Patch confirms PATCH on an unknown id
// returns 404 (the existing suite only covers GET 404).
func TestCovSalesAddresses_GetNotFound_Patch(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.Patch(addressesPath+"/ad_000000000000000000000000", map[string]any{
		"name": "does-not-matter",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status)
}

// TestCovSalesAddresses_GetNotFound_Delete confirms DELETE on an unknown id
// returns 404 (the existing suite only covers GET 404).
func TestCovSalesAddresses_GetNotFound_Delete(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.Delete(addressesPath + "/ad_000000000000000000000000")
	require.NoError(t, err)
	assert.Equal(t, 404, status)
}

// TestCovSalesAddresses_ListLimitTooLow confirms limit=0 is rejected.
func TestCovSalesAddresses_ListLimitTooLow(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(addressesPath, url.Values{"limit": {"0"}})
	require.NoError(t, err)
	require.True(t, status == 400 || status == 422,
		"limit=0 should return 400 or 422, got %d: %s", status, string(body))
}

// TestCovSalesAddresses_ListLimitTooHigh confirms limit=1001 is rejected
// (max=1000).
func TestCovSalesAddresses_ListLimitTooHigh(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(addressesPath, url.Values{"limit": {"1001"}})
	require.NoError(t, err)
	require.True(t, status == 400 || status == 422,
		"limit=1001 should return 400 or 422, got %d: %s", status, string(body))
}

// TestCovSalesAddresses_ListLimitNegative confirms a negative limit is
// rejected.
func TestCovSalesAddresses_ListLimitNegative(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(addressesPath, url.Values{"limit": {"-1"}})
	require.NoError(t, err)
	require.True(t, status == 400 || status == 422,
		"limit=-1 should return 400 or 422, got %d: %s", status, string(body))
}

// TestCovSalesAddresses_ListInvalidTypeFilter confirms an out-of-enum "type"
// query filter is rejected rather than silently ignored.
func TestCovSalesAddresses_ListInvalidTypeFilter(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(addressesPath, url.Values{"type": {"bogus_type"}})
	require.NoError(t, err)
	require.True(t, status == 400 || status == 422,
		"type=bogus_type should return 400 or 422, got %d: %s", status, string(body))
}

// TestCovSalesAddresses_ListInvalidCursor confirms a malformed cursor is
// rejected with 400 rather than silently returning an empty/garbage page.
func TestCovSalesAddresses_ListInvalidCursor(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(addressesPath, url.Values{"cursor": {"garbage_not_a_cursor"}})
	require.NoError(t, err)
	require.True(t, status == 400 || status == 422,
		"malformed cursor should return 400 or 422, got %d: %s", status, string(body))
}
