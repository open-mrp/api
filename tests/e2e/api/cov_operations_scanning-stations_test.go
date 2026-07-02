//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file extends the existing scanning-stations coverage in
// crud_scanning_stations_test.go with the gaps identified during review:
//
//   - `label_size`/`label_type` are asserted with real (non-null) values on
//     both create and update, using the correct JSON keys (the existing
//     suite's "nil" checks target the wrong keys, `label_size_code` /
//     `label_type_code`, and are vacuous no-ops).
//   - A dedicated create-response-shape test with assertIDFormat.
//   - An update-idempotency test mirroring the existing create-idempotency
//     one.
//   - Validation coverage for name >255 chars, invalid `type` enum, invalid
//     `operator_requirement`/`label_size`/`label_type` enums (on both create
//     and update), a duplicate name on update (409), and 404s for
//     PATCH/DELETE of a nonexistent id.
//   - A nonexistent-but-well-formed `department_id` case.
//
// It reuses scanningStationsPath (declared in crud_scanning_stations_test.go)
// and SeedDepartmentID (declared in seed_test.go). No new seed rows are
// needed. All new package-level identifiers are prefixed with
// covOperationsScanningStations to avoid collisions with other test files.

// covOperationsScanningStationsCreateBody returns a map with the 4 required
// create fields. Tests can override/add individual fields by mutating the
// returned map before posting.
func covOperationsScanningStationsCreateBody(name string) map[string]any {
	return map[string]any{
		"name":                 name,
		"type":                 "init_batch",
		"operator_requirement": "none",
		"department_id":        SeedDepartmentID,
	}
}

// covOperationsScanningStationsCreate creates a scanning station with the
// given name and returns its id. Fails the test on error.
func covOperationsScanningStationsCreate(t *testing.T, name string) string {
	t.Helper()
	status, body, err := apiClient.Post(scanningStationsPath, covOperationsScanningStationsCreateBody(name), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, id)
	return id
}

// --- Response shape ---

func TestCovOperationsScanningStations_CreateResponseShape(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cov-stn-shape")

	createResp, err := apiClient.PostFull(scanningStationsPath, covOperationsScanningStationsCreateBody(name), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(scanningStationsPath + "/" + id)

	assertIDFormat(t, id, "sgsn")
	assertCreatedLocation(t, createResp.Header, id)
	assertObjectField(t, created, "scanning_station")
	assert.Equal(t, name, jsonField(created, "name"))
	assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")

	// Expandable fields are null unless explicitly included.
	assertNilField(t, created, "department")
	assertNilField(t, created, "production_steps")
}

// --- label_size / label_type: default-null with correct keys ---

func TestCovOperationsScanningStations_LabelFieldsDefaultNil(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cov-stn-labelnil")
	id := covOperationsScanningStationsCreate(t, name)
	defer apiClient.Delete(scanningStationsPath + "/" + id)

	getStatus, getBody, err := apiClient.GetListRaw(scanningStationsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	// The existing suite asserts the wrong keys (label_size_code /
	// label_type_code), which are always absent and thus vacuously nil. The
	// actual resource JSON keys are label_size / label_type.
	assertNilField(t, got, "label_size")
	assertNilField(t, got, "label_type")
}

// --- label_size / label_type: create round-trip ---
//
// KNOWN BUG: services/core-service/internal/infrastructure/grpc/grpc_scanning_station_handler.go's
// CreateScanningStation handler never copies req.LabelSizeCode/req.LabelTypeCode
// into domain.CreateScanningStationParams (which doesn't even declare those
// fields), so label_size/label_type sent on create are silently dropped and
// the response always has them null, even though the request schema
// (CreateScanningStationRequest) declares and documents both fields.
// Confirmed via curl: POST with label_size="1x1", label_type="tag" returns
// "label_size": null, "label_type": null in a 201 response.
func TestCovOperationsScanningStations_CreateWithLabelFields(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cov-stn-labelcreate")

	body := covOperationsScanningStationsCreateBody(name)
	body["label_size"] = "1x1"
	body["label_type"] = "tag"

	createResp, err := apiClient.PostFull(scanningStationsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(scanningStationsPath + "/" + id)

	// EXPECTED/CORRECT behavior: label_size and label_type set on create
	// should round-trip. This currently fails against the live stack (a
	// confirmed backend bug, see comment above) — do not weaken this
	// assertion.
	assert.Equal(t, "1x1", jsonField(created, "label_size"), "label_size sent on create should round-trip (known backend bug: dropped silently)")
	assert.Equal(t, "tag", jsonField(created, "label_type"), "label_type sent on create should round-trip (known backend bug: dropped silently)")
}

// --- label_size / label_type: update round-trip + preservation ---

func TestCovOperationsScanningStations_UpdateLabelFields(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cov-stn-labelupd")
	id := covOperationsScanningStationsCreate(t, name)
	defer apiClient.Delete(scanningStationsPath + "/" + id)

	// Set label_size/label_type via PATCH.
	patchStatus, patchBody, err := apiClient.Patch(scanningStationsPath+"/"+id, map[string]any{
		"label_size": "1x3",
		"label_type": "traveler",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	assert.Equal(t, "1x3", jsonField(updated, "label_size"))
	assert.Equal(t, "traveler", jsonField(updated, "label_type"))

	// Round-trip to different valid values.
	patchStatus2, patchBody2, err := apiClient.Patch(scanningStationsPath+"/"+id, map[string]any{
		"label_size": "2x4",
		"label_type": "tag",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus2, patchBody2)

	updated2 := parseJSON(patchBody2)
	assert.Equal(t, "2x4", jsonField(updated2, "label_size"))
	assert.Equal(t, "tag", jsonField(updated2, "label_type"))

	// A subsequent partial update that doesn't touch label_size/label_type
	// should preserve them.
	newName := uniqueName("e2e-cov-stn-labelupd-2")
	patchStatus3, patchBody3, err := apiClient.Patch(scanningStationsPath+"/"+id, map[string]any{
		"name": newName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus3, patchBody3)

	updated3 := parseJSON(patchBody3)
	assert.Equal(t, newName, jsonField(updated3, "name"))
	assert.Equal(t, "2x4", jsonField(updated3, "label_size"), "label_size should be preserved by an unrelated partial update")
	assert.Equal(t, "tag", jsonField(updated3, "label_type"), "label_type should be preserved by an unrelated partial update")
}

// --- Idempotency ---

func TestCovOperationsScanningStations_UpdateIdempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cov-stn-uidem")
	id := covOperationsScanningStationsCreate(t, name)
	defer apiClient.Delete(scanningStationsPath + "/" + id)

	newName := uniqueName("e2e-cov-stn-uidem-2")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Patch(scanningStationsPath+"/"+id, map[string]any{
		"name":       newName,
		"label_size": "1x4",
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)

	status2, body2, err := apiClient.Patch(scanningStationsPath+"/"+id, map[string]any{
		"name":       newName,
		"label_size": "1x4",
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)

	got1 := parseJSON(body1)
	got2 := parseJSON(body2)
	assert.Equal(t, jsonField(got1, "id"), jsonField(got2, "id"))
	assert.Equal(t, jsonField(got1, "name"), jsonField(got2, "name"))
	assert.Equal(t, jsonField(got1, "label_size"), jsonField(got2, "label_size"))
	assert.Equal(t, jsonField(got1, "updated_at"), jsonField(got2, "updated_at"),
		"replaying the same idempotency key should not perform a second update")
}

// --- 404s ---

func TestCovOperationsScanningStations_NotFound_UpdateAndDelete(t *testing.T) {
	t.Parallel()

	patchStatus, patchBody, err := apiClient.Patch(scanningStationsPath+"/sgsn_nonexistent000000000", map[string]any{
		"name": uniqueName("e2e-cov-stn-404"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, patchStatus, "PATCH of a nonexistent scanning station should 404: %s", string(patchBody))

	delStatus, delBody, err := apiClient.Delete(scanningStationsPath + "/sgsn_nonexistent000000000")
	require.NoError(t, err)
	assert.Equal(t, 404, delStatus, "DELETE of a nonexistent scanning station should 404: %s", string(delBody))
}

// --- 409 duplicate name on update ---

func TestCovOperationsScanningStations_UpdateDuplicateName(t *testing.T) {
	t.Parallel()
	nameA := uniqueName("e2e-cov-stn-dupA")
	nameB := uniqueName("e2e-cov-stn-dupB")
	idA := covOperationsScanningStationsCreate(t, nameA)
	defer apiClient.Delete(scanningStationsPath + "/" + idA)
	idB := covOperationsScanningStationsCreate(t, nameB)
	defer apiClient.Delete(scanningStationsPath + "/" + idB)

	status, body, err := apiClient.Patch(scanningStationsPath+"/"+idB, map[string]any{
		"name": nameA,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 409, status, "Updating a scanning station's name to a duplicate should return 409: %s", string(body))
}

// --- Validation: create ---

func TestCovOperationsScanningStations_CreateValidation_NameTooLong(t *testing.T) {
	t.Parallel()
	body := covOperationsScanningStationsCreateBody(strings.Repeat("a", 256))

	status, respBody, err := apiClient.Post(scanningStationsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"name >255 chars should return 400 or 422, got %d: %s", status, string(respBody))
}

func TestCovOperationsScanningStations_CreateValidation_InvalidTypeEnum(t *testing.T) {
	t.Parallel()
	body := covOperationsScanningStationsCreateBody(uniqueName("e2e-cov-stn-badtype"))
	body["type"] = "bogus_type"

	status, respBody, err := apiClient.Post(scanningStationsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"invalid type enum should return 400 or 422, got %d: %s", status, string(respBody))
}

func TestCovOperationsScanningStations_CreateValidation_InvalidEnums(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		field string
		value string
	}{
		{"OperatorRequirement", "operator_requirement", "bogus_requirement"},
		{"LabelSize", "label_size", "9x9"},
		{"LabelType", "label_type", "poster"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body := covOperationsScanningStationsCreateBody(uniqueName("e2e-cov-stn-badenum-" + tc.field))
			body[tc.field] = tc.value

			status, respBody, err := apiClient.Post(scanningStationsPath, body, newIdempotencyKey())
			require.NoError(t, err)
			// Confirmed live (not just "suspected"): the gateway rejects
			// these at the schema-validation layer before reaching the
			// service. Per feedback_no_skip_5xx_in_e2e, assert the correct
			// behavior outright rather than accepting a loose status range.
			assert.True(t, status == 400 || status == 422,
				"invalid %s value should return 400 or 422, got %d: %s", tc.field, status, string(respBody))
		})
	}
}

func TestCovOperationsScanningStations_CreateNonexistentDepartment(t *testing.T) {
	t.Parallel()
	body := covOperationsScanningStationsCreateBody(uniqueName("e2e-cov-stn-baddept"))
	body["department_id"] = "dp_nonexistent000000000"

	status, respBody, err := apiClient.Post(scanningStationsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	if status == 201 {
		// Clean up regardless of the assertion outcome below.
		id := jsonField(parseJSON(respBody), "id")
		defer apiClient.Delete(scanningStationsPath + "/" + id)
	}

	// EXPECTED/CORRECT behavior: creating a scanning station with a
	// well-formed but nonexistent department_id should be rejected.
	// KNOWN BUG (confirmed live): the create service never validates
	// department_id existence (no FK constraint on scanning_station.department_id,
	// no lookup in CreateScanningStation), so this currently returns 201 with
	// a dangling reference instead of 400/404 — do not weaken this
	// assertion to accept 201.
	assert.True(t, status == 400 || status == 404,
		"create with a nonexistent department_id should be rejected (400/404), got %d: %s (known backend bug: dangling department_id silently accepted)", status, string(respBody))
}

// --- Validation: update ---

func TestCovOperationsScanningStations_UpdateValidation_NameTooLong(t *testing.T) {
	t.Parallel()
	id := covOperationsScanningStationsCreate(t, uniqueName("e2e-cov-stn-updnametoolong"))
	defer apiClient.Delete(scanningStationsPath + "/" + id)

	status, respBody, err := apiClient.Patch(scanningStationsPath+"/"+id, map[string]any{
		"name": strings.Repeat("b", 256),
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"name >255 chars should return 400 or 422 on update, got %d: %s", status, string(respBody))
}

func TestCovOperationsScanningStations_UpdateValidation_InvalidEnums(t *testing.T) {
	t.Parallel()
	id := covOperationsScanningStationsCreate(t, uniqueName("e2e-cov-stn-updbadenum"))
	defer apiClient.Delete(scanningStationsPath + "/" + id)

	cases := []struct {
		name  string
		field string
		value string
	}{
		{"OperatorRequirement", "operator_requirement", "bogus_requirement"},
		{"LabelSize", "label_size", "9x9"},
		{"LabelType", "label_type", "poster"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			status, respBody, err := apiClient.Patch(scanningStationsPath+"/"+id, map[string]any{
				tc.field: tc.value,
			}, newIdempotencyKey())
			require.NoError(t, err)
			assert.True(t, status == 400 || status == 422,
				"invalid %s value should return 400 or 422 on update, got %d: %s", tc.field, status, string(respBody))
		})
	}
}

// --- Query params: include ---

func TestCovOperationsScanningStations_GetByID_InvalidInclude(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(scanningStationsPath+"/"+SeedScanningStationID, url.Values{"include": {"bogus_field"}})
	require.NoError(t, err)
	assert.Equal(t, 400, status, "invalid include value should return 400: %s", string(body))
}
