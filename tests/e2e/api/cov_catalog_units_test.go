//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file extends the existing catalog/units coverage in crud_units_test.go
// (list + validate-action only) with the CRUD lifecycle, all-fields,
// omitted-fields, response-shape, idempotency, and validation categories that
// had zero e2e coverage. It reuses unitsPath (declared in crud_units_test.go)
// and SeedUnitID / SeedSystemUnitID / SeedUnitGroupID (declared in
// seed_test.go). No new seed rows or package-level constants are needed.

// covCatalogUnitsCreateBody returns a map with all 7 required create fields.
// Tests can override individual fields by writing to the returned map before
// posting.
func covCatalogUnitsCreateBody(name, abbreviation string) map[string]any {
	return map[string]any{
		"name":               name,
		"abbreviation":       abbreviation,
		"type":               "mass",
		"ratio_numerator":    "5",
		"ratio_denominator":  "2",
		"offset_numerator":   "1",
		"offset_denominator": "3",
	}
}

// covCatalogUnitsCreate creates a unit with the given name/abbreviation and
// returns its id. Fails the test on error.
func covCatalogUnitsCreate(t *testing.T, name, abbreviation string) string {
	t.Helper()
	status, body, err := apiClient.Post(unitsPath, covCatalogUnitsCreateBody(name, abbreviation), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, id)
	return id
}

// --- CRUD lifecycle ---

func TestCovCatalogUnits_CRUD(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cov-unit")
	abbr := uniqueName("e2ecu")

	// Create
	createResp, err := apiClient.PostFull(unitsPath, covCatalogUnitsCreateBody(name, abbr), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	assert.Equal(t, "unit", jsonField(created, "object"))
	assert.Equal(t, name, jsonField(created, "name"))
	assert.Equal(t, abbr, jsonField(created, "abbreviation"))
	assert.Equal(t, "mass", jsonField(created, "type"))

	// Get
	getStatus, getBody, err := apiClient.GetListRaw(unitsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, name, jsonField(parseJSON(getBody), "name"))

	// Update
	newName := uniqueName("e2e-cov-unit-upd")
	patchStatus, patchBody, err := apiClient.Patch(unitsPath+"/"+id, map[string]any{
		"name": newName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	assert.Equal(t, newName, jsonField(parseJSON(patchBody), "name"))

	// Verify update
	getStatus2, getBody2, err := apiClient.GetListRaw(unitsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus2, getBody2)
	assert.Equal(t, newName, jsonField(parseJSON(getBody2), "name"))

	// Delete
	delStatus, delBody, err := apiClient.Delete(unitsPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// Verify deletion
	getStatus3, _, err := apiClient.GetListRaw(unitsPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus3)
}

// --- All fields ---

func TestCovCatalogUnits_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-cov-unit-allf")
	abbr := uniqueName("e2ecuaf")

	createResp, err := apiClient.PostFull(unitsPath+"?include=owner", map[string]any{
		"name":               name,
		"abbreviation":       abbr,
		"type":               "length",
		"ratio_numerator":    "7",
		"ratio_denominator":  "4",
		"offset_numerator":   "2",
		"offset_denominator": "9",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	got := parseJSON(createResp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	defer apiClient.Delete(unitsPath + "/" + id)

	// All 12 response-struct fields.
	assert.Equal(t, "unit", jsonField(got, "object"))
	assert.Equal(t, id, jsonField(got, "id"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, abbr, jsonField(got, "abbreviation"))
	assert.Equal(t, "length", jsonField(got, "type"))
	assert.Equal(t, "7", jsonField(got, "ratio_numerator"))
	assert.Equal(t, "4", jsonField(got, "ratio_denominator"))
	assert.Equal(t, "2", jsonField(got, "offset_numerator"))
	assert.Equal(t, "9", jsonField(got, "offset_denominator"))
	assert.Equal(t, false, got["is_base_unit"], "account-created units always have is_base_unit=false")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	owner := jsonObject(got, "owner")
	require.NotNil(t, owner, "owner should be present with ?include=owner")
	assert.Equal(t, "owner", jsonField(owner, "object"))

	// Update a subset of fields.
	updatedName := uniqueName("e2e-cov-unit-allf-u")
	patchStatus, patchBody, err := apiClient.Patch(unitsPath+"/"+id+"?include=owner", map[string]any{
		"name":              updatedName,
		"ratio_numerator":   "11",
		"ratio_denominator": "5",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	assert.Equal(t, id, jsonField(updated, "id"), "ID must not change")
	assert.Equal(t, updatedName, jsonField(updated, "name"))
	assert.Equal(t, "11", jsonField(updated, "ratio_numerator"))
	assert.Equal(t, "5", jsonField(updated, "ratio_denominator"))

	// Fields not touched by the patch should be preserved.
	assert.Equal(t, abbr, jsonField(updated, "abbreviation"), "abbreviation should be preserved")
	assert.Equal(t, "length", jsonField(updated, "type"), "type should be preserved (not updatable)")
	assert.Equal(t, "2", jsonField(updated, "offset_numerator"), "offset_numerator should be preserved")
	assert.Equal(t, "9", jsonField(updated, "offset_denominator"), "offset_denominator should be preserved")
	assert.Equal(t, false, updated["is_base_unit"])

	updOwner := jsonObject(updated, "owner")
	require.NotNil(t, updOwner, "owner should still be present with ?include=owner")
	assert.Equal(t, "owner", jsonField(updOwner, "object"))
}

// --- Response shape ---

func TestCovCatalogUnits_CreateResponseShape(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cov-unit-shape")
	abbr := uniqueName("e2ecush")

	createResp, err := apiClient.PostFull(unitsPath, covCatalogUnitsCreateBody(name, abbr), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assertIDFormat(t, id, "un")
	assertCreatedLocation(t, createResp.Header, id)
	assert.Equal(t, "unit", jsonField(created, "object"))
	assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")
	// owner should be null without ?include=owner
	assertNilField(t, created, "owner")

	apiClient.Delete(unitsPath + "/" + id)
}

// --- Omitted fields ---

func TestCovCatalogUnits_OmittedFields(t *testing.T) {
	t.Parallel()

	t.Run("CreateDefaults", func(t *testing.T) {
		t.Parallel()
		name := uniqueName("e2e-cov-unit-def")
		abbr := uniqueName("e2ecudef")

		status, body, err := apiClient.Post(unitsPath, covCatalogUnitsCreateBody(name, abbr), newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)

		created := parseJSON(body)
		id := jsonField(created, "id")
		defer apiClient.Delete(unitsPath + "/" + id)

		// Every create field is required; there is nothing optional to omit
		// besides the always-server-derived is_base_unit, which must default
		// to false for account-created units.
		assert.Equal(t, false, created["is_base_unit"])
		assertNilField(t, created, "owner")
	})

	t.Run("MissingRequiredFields", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name         string
			field        string
			expectedCode string
		}{
			{"name", "name", "missing_field"},
			{"abbreviation", "abbreviation", "missing_field"},
			{"type", "type", "parameter_invalid"},
			{"ratio_numerator", "ratio_numerator", "missing_field"},
			{"ratio_denominator", "ratio_denominator", "missing_field"},
			{"offset_numerator", "offset_numerator", "missing_field"},
			{"offset_denominator", "offset_denominator", "missing_field"},
		}
		for _, tc := range cases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				body := covCatalogUnitsCreateBody(uniqueName("e2e-cov-unit-miss"), uniqueName("e2ecumiss"))
				delete(body, tc.field)

				status, respBody, err := apiClient.Post(unitsPath, body, newIdempotencyKey())
				require.NoError(t, err)
				requireStatus(t, 400, status, respBody)
				errObj := requireErrorResponse(t, respBody, tc.expectedCode, "invalid_request_error")
				assertErrorParam(t, errObj, tc.field)
			})
		}
	})

	t.Run("UpdatePreservesOmittedFields", func(t *testing.T) {
		t.Parallel()
		name := uniqueName("e2e-cov-unit-pres")
		abbr := uniqueName("e2ecupres")
		id := covCatalogUnitsCreate(t, name, abbr)
		defer apiClient.Delete(unitsPath + "/" + id)

		newName := uniqueName("e2e-cov-unit-pres-upd")
		patchStatus, patchBody, err := apiClient.Patch(unitsPath+"/"+id, map[string]any{
			"name": newName,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, patchStatus, patchBody)

		patched := parseJSON(patchBody)
		assert.Equal(t, newName, jsonField(patched, "name"))
		assert.Equal(t, abbr, jsonField(patched, "abbreviation"), "abbreviation should be preserved when omitted from patch")
		assert.Equal(t, "mass", jsonField(patched, "type"), "type should be preserved when omitted from patch")
		assert.Equal(t, "5", jsonField(patched, "ratio_numerator"), "ratio_numerator should be preserved when omitted from patch")
		assert.Equal(t, "2", jsonField(patched, "ratio_denominator"), "ratio_denominator should be preserved when omitted from patch")
		assert.Equal(t, "1", jsonField(patched, "offset_numerator"), "offset_numerator should be preserved when omitted from patch")
		assert.Equal(t, "3", jsonField(patched, "offset_denominator"), "offset_denominator should be preserved when omitted from patch")
	})
}

// --- Idempotency ---

func TestCovCatalogUnits_CreateIdempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cov-unit-idem")
	abbr := uniqueName("e2ecuidem")
	idemKey := newIdempotencyKey()
	body := covCatalogUnitsCreateBody(name, abbr)

	status1, body1, err := apiClient.Post(unitsPath, body, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")
	defer apiClient.Delete(unitsPath + "/" + id1)

	status2, body2, err := apiClient.Post(unitsPath, body, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id1, jsonField(parseJSON(body2), "id"))
	assert.Equal(t, name, jsonField(parseJSON(body2), "name"))
}

func TestCovCatalogUnits_UpdateIdempotent(t *testing.T) {
	t.Parallel()
	id := covCatalogUnitsCreate(t, uniqueName("e2e-cov-unit-uidem"), uniqueName("e2ecuuidem"))
	defer apiClient.Delete(unitsPath + "/" + id)

	newName := uniqueName("e2e-cov-unit-uidem2")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Patch(unitsPath+"/"+id, map[string]any{"name": newName}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)

	status2, body2, err := apiClient.Patch(unitsPath+"/"+id, map[string]any{"name": newName}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)

	assert.Equal(t, jsonField(parseJSON(body1), "id"), jsonField(parseJSON(body2), "id"))
	assert.Equal(t, jsonField(parseJSON(body1), "name"), jsonField(parseJSON(body2), "name"))
	assert.Equal(t, jsonField(parseJSON(body1), "updated_at"), jsonField(parseJSON(body2), "updated_at"))
}

// --- Validation: create ---

func TestCovCatalogUnits_CreateBlankStringFields(t *testing.T) {
	t.Parallel()
	cases := []string{"name", "abbreviation"}
	for _, field := range cases {
		field := field
		t.Run(field, func(t *testing.T) {
			t.Parallel()
			body := covCatalogUnitsCreateBody(uniqueName("e2e-cov-unit-blank"), uniqueName("e2ecublank"))
			body[field] = ""

			status, respBody, err := apiClient.Post(unitsPath, body, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 400, status, respBody)
			errObj := requireErrorResponse(t, respBody, "missing_field", "invalid_request_error")
			assertErrorParam(t, errObj, field)
		})
	}
}

func TestCovCatalogUnits_CreateInvalidTypeEnum(t *testing.T) {
	t.Parallel()
	body := covCatalogUnitsCreateBody(uniqueName("e2e-cov-unit-badtype"), uniqueName("e2ecubadtype"))
	body["type"] = "bogus_type"

	status, respBody, err := apiClient.Post(unitsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)
	errObj := requireErrorResponse(t, respBody, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "type")
}

func TestCovCatalogUnits_CreateZeroRatioDenominator(t *testing.T) {
	t.Parallel()
	body := covCatalogUnitsCreateBody(uniqueName("e2e-cov-unit-zrd"), uniqueName("e2ecuzrd"))
	body["ratio_denominator"] = "0"

	status, respBody, err := apiClient.Post(unitsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)
	errObj := requireErrorResponse(t, respBody, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "ratio_denominator")
}

func TestCovCatalogUnits_CreateZeroOffsetDenominator(t *testing.T) {
	t.Parallel()
	body := covCatalogUnitsCreateBody(uniqueName("e2e-cov-unit-zod"), uniqueName("e2ecuzod"))
	body["offset_denominator"] = "0"

	status, respBody, err := apiClient.Post(unitsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)
	errObj := requireErrorResponse(t, respBody, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "offset_denominator")
}

// TestCovCatalogUnits_CreateNonNumericRatioNumerator documents a suspected
// backend bug (see confirmedBugs): CreateUnitRequest.RatioNumerator only has
// validate:"required" (non-empty), not a decimal-format check, so a
// non-numeric string reaches core-service's DECIMAL column unguarded and the
// live stack currently 500s instead of returning a 400 validation error.
// This assertion intentionally encodes the CORRECT desired behavior (400) and
// will fail red against the current build until the backend adds decimal
// parsing validation for ratio_numerator/offset_numerator.
func TestCovCatalogUnits_CreateNonNumericRatioNumerator(t *testing.T) {
	t.Parallel()
	body := covCatalogUnitsCreateBody(uniqueName("e2e-cov-unit-nonnum"), uniqueName("e2ecunonnum"))
	body["ratio_numerator"] = "abc"

	status, respBody, err := apiClient.Post(unitsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status,
		"non-numeric ratio_numerator should be rejected with 400, got %d: %s (suspected backend bug: missing decimal-format validation)",
		status, string(respBody))
}

// --- Validation: update ---

func TestCovCatalogUnits_UpdateNullFieldsRejected(t *testing.T) {
	t.Parallel()
	fields := []string{"name", "abbreviation", "ratio_numerator", "ratio_denominator", "offset_numerator", "offset_denominator"}
	for _, field := range fields {
		field := field
		t.Run(field, func(t *testing.T) {
			t.Parallel()
			id := covCatalogUnitsCreate(t, uniqueName("e2e-cov-unit-null"), uniqueName("e2ecunull"))
			defer apiClient.Delete(unitsPath + "/" + id)

			status, body, err := apiClient.Patch(unitsPath+"/"+id, map[string]any{field: nil}, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 400, status, body)
			errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
			assertErrorParam(t, errObj, field)
		})
	}
}

func TestCovCatalogUnits_UpdateBlankStringFieldsRejected(t *testing.T) {
	t.Parallel()
	fields := []string{"name", "abbreviation"}
	for _, field := range fields {
		field := field
		t.Run(field, func(t *testing.T) {
			t.Parallel()
			id := covCatalogUnitsCreate(t, uniqueName("e2e-cov-unit-blk"), uniqueName("e2ecublk"))
			defer apiClient.Delete(unitsPath + "/" + id)

			status, body, err := apiClient.Patch(unitsPath+"/"+id, map[string]any{field: ""}, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 400, status, body)
			errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
			assertErrorParam(t, errObj, field)
		})
	}
}

func TestCovCatalogUnits_UpdateZeroRatioDenominator(t *testing.T) {
	t.Parallel()
	id := covCatalogUnitsCreate(t, uniqueName("e2e-cov-unit-uzrd"), uniqueName("e2ecuuzrd"))
	defer apiClient.Delete(unitsPath + "/" + id)

	status, body, err := apiClient.Patch(unitsPath+"/"+id, map[string]any{"ratio_denominator": "0"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "ratio_denominator")
}

func TestCovCatalogUnits_UpdateZeroOffsetDenominator(t *testing.T) {
	t.Parallel()
	id := covCatalogUnitsCreate(t, uniqueName("e2e-cov-unit-uzod"), uniqueName("e2ecuuzod"))
	defer apiClient.Delete(unitsPath + "/" + id)

	status, body, err := apiClient.Patch(unitsPath+"/"+id, map[string]any{"offset_denominator": "0"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "offset_denominator")
}

// TestCovCatalogUnits_UpdateNonNumericRatioNumerator is the PATCH-side
// counterpart of TestCovCatalogUnits_CreateNonNumericRatioNumerator (see that
// test's doc comment and confirmedBugs). UpdateUnitRequest's RatioNumerator
// has no validate tag at all, so this is expected to reach core-service
// unguarded the same way. Asserts the CORRECT desired behavior (400).
func TestCovCatalogUnits_UpdateNonNumericRatioNumerator(t *testing.T) {
	t.Parallel()
	id := covCatalogUnitsCreate(t, uniqueName("e2e-cov-unit-unonnum"), uniqueName("e2ecuunonnum"))
	defer apiClient.Delete(unitsPath + "/" + id)

	status, body, err := apiClient.Patch(unitsPath+"/"+id, map[string]any{"ratio_numerator": "abc"}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status,
		"non-numeric ratio_numerator on update should be rejected with 400, got %d: %s (suspected backend bug: missing decimal-format validation)",
		status, string(body))
}

// --- Conflict (409) ---

func TestCovCatalogUnits_CreateDuplicateName(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cov-unit-dupname")
	id := covCatalogUnitsCreate(t, name, uniqueName("e2ecudupname1"))
	defer apiClient.Delete(unitsPath + "/" + id)

	body := covCatalogUnitsCreateBody(name, uniqueName("e2ecudupname2"))
	status, respBody, err := apiClient.Post(unitsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, respBody)
	errObj := requireErrorResponse(t, respBody, "resource_conflict", "invalid_request_error")
	assertErrorParam(t, errObj, "name")
}

func TestCovCatalogUnits_CreateDuplicateAbbreviation(t *testing.T) {
	t.Parallel()
	abbr := uniqueName("e2ecudupabbr")
	id := covCatalogUnitsCreate(t, uniqueName("e2e-cov-unit-dupabbr1"), abbr)
	defer apiClient.Delete(unitsPath + "/" + id)

	body := covCatalogUnitsCreateBody(uniqueName("e2e-cov-unit-dupabbr2"), abbr)
	status, respBody, err := apiClient.Post(unitsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, respBody)
	errObj := requireErrorResponse(t, respBody, "resource_conflict", "invalid_request_error")
	assertErrorParam(t, errObj, "abbreviation")
}

// TestCovCatalogUnits_CreateDuplicateAbbreviationAgainstSystemUnit locks in
// the cross-scope uniqueness rule: account-owned units are checked against
// both their own account's units AND global system units (account_id IS
// NULL), e.g. the seeded "each" unit's abbreviation "ea".
func TestCovCatalogUnits_CreateDuplicateAbbreviationAgainstSystemUnit(t *testing.T) {
	t.Parallel()
	body := covCatalogUnitsCreateBody(uniqueName("e2e-cov-unit-sysabbr"), "ea")
	status, respBody, err := apiClient.Post(unitsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, respBody)
	errObj := requireErrorResponse(t, respBody, "resource_conflict", "invalid_request_error")
	assertErrorParam(t, errObj, "abbreviation")
}

func TestCovCatalogUnits_UpdateDuplicateName(t *testing.T) {
	t.Parallel()
	name1 := uniqueName("e2e-cov-unit-updname1")
	id1 := covCatalogUnitsCreate(t, name1, uniqueName("e2ecuupdname1"))
	defer apiClient.Delete(unitsPath + "/" + id1)

	id2 := covCatalogUnitsCreate(t, uniqueName("e2e-cov-unit-updname2"), uniqueName("e2ecuupdname2"))
	defer apiClient.Delete(unitsPath + "/" + id2)

	status, body, err := apiClient.Patch(unitsPath+"/"+id2, map[string]any{"name": name1}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
	errObj := requireErrorResponse(t, body, "resource_conflict", "invalid_request_error")
	assertErrorParam(t, errObj, "name")
}

func TestCovCatalogUnits_UpdateDuplicateAbbreviation(t *testing.T) {
	t.Parallel()
	abbr1 := uniqueName("e2ecuupdabbr1")
	id1 := covCatalogUnitsCreate(t, uniqueName("e2e-cov-unit-updabbr1"), abbr1)
	defer apiClient.Delete(unitsPath + "/" + id1)

	id2 := covCatalogUnitsCreate(t, uniqueName("e2e-cov-unit-updabbr2"), uniqueName("e2ecuupdabbr2"))
	defer apiClient.Delete(unitsPath + "/" + id2)

	status, body, err := apiClient.Patch(unitsPath+"/"+id2, map[string]any{"abbreviation": abbr1}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
	errObj := requireErrorResponse(t, body, "resource_conflict", "invalid_request_error")
	assertErrorParam(t, errObj, "abbreviation")
}

// --- System unit guard (400, not 403/404) ---

func TestCovCatalogUnits_UpdateSystemUnitFails(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Patch(unitsPath+"/"+SeedSystemUnitID, map[string]any{
		"name": uniqueName("e2e-cov-sys-upd"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
}

func TestCovCatalogUnits_DeleteSystemUnitFails(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Delete(unitsPath + "/" + SeedSystemUnitID)
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "validation_failed", "invalid_request_error")

	// The system unit must remain retrievable and unmodified after the
	// rejected mutation attempts above.
	getStatus, getBody, err := apiClient.GetListRaw(unitsPath+"/"+SeedSystemUnitID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, "ea", jsonField(parseJSON(getBody), "abbreviation"))
}

// --- Not found (404) ---

func TestCovCatalogUnits_GetNotFound(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.GetListRaw(unitsPath+"/un_00000000000000000000", nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status)
}

func TestCovCatalogUnits_UpdateNotFound(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.Patch(unitsPath+"/un_00000000000000000000", map[string]any{
		"name": uniqueName("e2e-cov-unit-404upd"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status)
}

func TestCovCatalogUnits_DeleteNotFound(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.Delete(unitsPath + "/un_00000000000000000000")
	require.NoError(t, err)
	assert.Equal(t, 404, status)
}

// --- Already deleted (410) ---

func TestCovCatalogUnits_DeleteAlreadyDeleted(t *testing.T) {
	t.Parallel()
	id := covCatalogUnitsCreate(t, uniqueName("e2e-cov-unit-deldel"), uniqueName("e2ecudeldel"))

	delStatus, delBody, err := apiClient.Delete(unitsPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	status2, body2, err := apiClient.Delete(unitsPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 410, status2, body2)
	requireErrorResponse(t, body2, "resource_gone", "invalid_request_error")
}

// --- List query params ---

// TestCovCatalogUnits_ListInvalidTypeFilter documents the observed behavior
// of an invalid `type` query filter value: it is rejected with 400 via the
// same reflection-based enum validation used for request bodies. Note the
// error.param is the Go struct field name ("Type"), not the query tag
// ("type") — this matches the established convention for query-param enum
// validation elsewhere in the codebase (e.g. TestCovFinanceTransactionMethods
// asserts "Limit"/"Query" for its query params), so it is not flagged as a
// units-specific bug here.
func TestCovCatalogUnits_ListInvalidTypeFilter(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(unitsPath, url.Values{"type": {"not_a_type"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "Type")
}

// TestCovCatalogUnits_ListUnitGroupIDsNonexistent verifies a nonexistent
// unit_group_ids value simply yields an empty match rather than an error.
func TestCovCatalogUnits_ListUnitGroupIDsNonexistent(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(unitsPath, url.Values{"unit_group_ids": {"ungp_nonexistent00000000"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

// TestCovCatalogUnits_ListUnitGroupIDsSeeded verifies the seeded unit group
// filter returns matches whose id appears in the result set (union-exclusion
// semantics are covered generically by array_filter_union_test.go's
// units/unit_group_ids entry — this just confirms the single-value case).
func TestCovCatalogUnits_ListUnitGroupIDsSeeded(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(unitsPath, url.Values{"unit_group_ids": {SeedUnitGroupID}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Should have at least 1 unit in the seeded unit group")

	found := false
	for _, item := range list.Data {
		if DataItemField(item, "id") == SeedUnitID {
			found = true
			break
		}
	}
	assert.True(t, found, "SeedUnitID should be in the unit_group_ids=%s filtered list", SeedUnitGroupID)
}
