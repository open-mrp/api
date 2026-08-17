//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const unitsPath = "/v1/catalog/units"
const unitsValidatePath = "/v1/catalog/units/actions/validate"
const unitsBulkUpsertPath = "/v1/catalog/units/actions/bulk-upsert"

// upsertUnitInput builds a single unit entry for bulk-upsert payloads.
func upsertUnitInput(name, abbr, unitType string) map[string]any {
	return map[string]any{
		"name":               name,
		"abbreviation":       abbr,
		"type":               unitType,
		"ratio_numerator":    "1",
		"ratio_denominator":  "1",
		"offset_numerator":   "0",
		"offset_denominator": "1",
		"is_base_unit":       false,
	}
}

// cleanupUnitIDs deletes the given units by ID.
func cleanupUnitIDs(ids []string) {
	for _, id := range ids {
		if id != "" {
			apiClient.Delete(unitsPath + "/" + id)
		}
	}
}

// acceptBulkUpsertUnits posts a bulk upsert, requires the 202 job acknowledgment, and
// returns the job's ID to poll.
func acceptBulkUpsertUnits(t *testing.T, units ...map[string]any) string {
	t.Helper()
	rows := make([]any, len(units))
	for i, u := range units {
		rows[i] = u
	}
	status, body, err := apiClient.Post(unitsBulkUpsertPath, map[string]any{"units": rows}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, status, body)

	m := parseJSON(body)
	assert.Equal(t, "job", jsonField(m, "object"), "202 returns the canonical job resource")
	jobID := jsonField(m, "id")
	require.NotEmpty(t, jobID, "202 must name the job to poll")
	return jobID
}

// bulkUpsertUnits posts a bulk upsert, follows the job to completion, and returns the
// created/updated unit IDs from its results.
func bulkUpsertUnits(t *testing.T, units ...map[string]any) (createdIDs, updatedIDs []string) {
	t.Helper()
	job := bulkUpsertUnitsJob(t, units...)
	require.NotEmpty(t, jobResults(job), "a completed job must carry results")
	return jobResultIDs(job)
}

// bulkUpsertUnitsJob posts a bulk upsert and returns the completed job. Bulk upsert is
// partial-success: a row that fails validation against existing rows (dual-key conflict,
// immutability) is recorded in the job's `errors` field, not failed — the job completes.
func bulkUpsertUnitsJob(t *testing.T, units ...map[string]any) map[string]any {
	t.Helper()
	job := pollJobUntilTerminal(t, acceptBulkUpsertUnits(t, units...))
	require.Equal(t, "completed", jsonField(job, "status"), "job should complete: %v", job)
	return job
}

// --- List ---

func TestUnits_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(unitsPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Should have at least 1 seeded unit")

	// Paginate until found: seed rows are the oldest and fall off the
	// first page as repeated e2e runs accumulate data.
	assertListContainsID(t, unitsPath, nil, SeedUnitID)
}

func TestUnits_ListResponseShape(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(unitsPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	AssertResponseBodyValid(t, body)

	list := parseJSON(body)
	data, ok := list["data"].([]any)
	require.True(t, ok, "units list data should be an array")

	for _, item := range data {
		m, ok := item.(map[string]any)
		require.True(t, ok, "unit list item should be an object")
		assert.Equal(t, "unit", jsonField(m, "object"))
		assert.NotEmpty(t, jsonField(m, "id"))
		assert.NotEmpty(t, jsonField(m, "name"))
		assert.NotEmpty(t, jsonField(m, "abbreviation"))
		assert.NotEmpty(t, jsonField(m, "type"))
		assert.NotEmpty(t, jsonField(m, "created_at"))
		assert.NotEmpty(t, jsonField(m, "updated_at"))
	}
}

func TestUnits_ListPagination(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(unitsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	assert.Len(t, list.Data, 1)
}

func TestUnits_ListCursorPagination(t *testing.T) {
	t.Parallel()
	// Retry-bounded two-page fetch: parallel tests can delete the rows
	// behind the cursor between fetches on this shared list.
	assertCursorPaginationAdvances(t, unitsPath, nil)
}

func TestUnits_ListSearchByName(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(unitsPath, url.Values{"q": {"Pair"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Search for 'Pair' should return at least 1 result")

	for _, item := range list.Data {
		m := parseJSON(item)
		name := strings.ToLower(jsonField(m, "name"))
		abbr := strings.ToLower(jsonField(m, "abbreviation"))
		assert.True(t,
			strings.Contains(name, "pair") || strings.Contains(abbr, "pair"),
			"Search result (name=%q, abbreviation=%q) should contain 'pair'",
			jsonField(m, "name"), jsonField(m, "abbreviation"),
		)
	}
}

func TestUnits_ListSearchByAbbreviation(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(unitsPath, url.Values{"q": {"pr"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Search for 'pr' should return at least 1 result")

	for _, item := range list.Data {
		m := parseJSON(item)
		name := strings.ToLower(jsonField(m, "name"))
		abbr := strings.ToLower(jsonField(m, "abbreviation"))
		assert.True(t,
			strings.Contains(name, "pr") || strings.Contains(abbr, "pr"),
			"Search result (name=%q, abbreviation=%q) should contain 'pr'",
			jsonField(m, "name"), jsonField(m, "abbreviation"),
		)
	}
}

func TestUnits_ListSearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(unitsPath, url.Values{"q": {"zzzznotaunit99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

func TestUnits_ListFilterByType(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(unitsPath, url.Values{"type": {"quantity"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Should have at least 1 quantity unit")

	for _, item := range list.Data {
		m := parseJSON(item)
		assert.Equal(t, "quantity", jsonField(m, "type"), "All results should have type=quantity")
	}
}

// --- Validate ---

func TestUnits_ValidateKnown(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Put(unitsValidatePath, map[string]any{
		"unit_map": map[string]string{
			"quantity": "ea",
			"weight":   "dz",
		},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	m := parseJSON(body)
	assert.Equal(t, "map", jsonField(m, "object"))

	units, ok := m["units"].(map[string]any)
	require.True(t, ok, "units should be a map")
	assert.Len(t, units, 2)

	for key, val := range units {
		unit, ok := val.(map[string]any)
		require.True(t, ok, "unit %s should be an object", key)
		assert.Equal(t, "unit", jsonField(unit, "object"))
		assert.NotEmpty(t, jsonField(unit, "id"))
		assert.NotEmpty(t, jsonField(unit, "name"))
		assert.NotEmpty(t, jsonField(unit, "abbreviation"))
	}
}

func TestUnits_ValidateUnknown(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Put(unitsValidatePath, map[string]any{
		"unit_map": map[string]string{
			"thing": "zzz_nonexistent",
		},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	m := parseJSON(body)
	units, ok := m["units"].(map[string]any)
	require.True(t, ok, "units should be a map")
	assert.Nil(t, units["thing"], "Unknown abbreviation should map to null")
}

func TestUnits_ValidateMixed(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Put(unitsValidatePath, map[string]any{
		"unit_map": map[string]string{
			"known":   "ea",
			"unknown": "zzz_fake",
		},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	m := parseJSON(body)
	units, ok := m["units"].(map[string]any)
	require.True(t, ok, "units should be a map")

	known, ok := units["known"].(map[string]any)
	assert.True(t, ok, "known abbreviation should resolve to a unit object")
	assert.Equal(t, "unit", jsonField(known, "object"))

	assert.Nil(t, units["unknown"], "Unknown abbreviation should be null")
}

// --- Bulk Upsert ---

func TestUnits_BulkUpsert_AllCreates(t *testing.T) {
	t.Parallel()

	name1, abbr1 := uniqueName("e2e-bu"), uniqueName("ebu")
	name2, abbr2 := uniqueName("e2e-bu"), uniqueName("ebu")

	createdIDs, updatedIDs := bulkUpsertUnits(t,
		upsertUnitInput(name1, abbr1, "quantity"),
		upsertUnitInput(name2, abbr2, "quantity"),
	)
	defer cleanupUnitIDs(createdIDs)

	require.Len(t, createdIDs, 2, "should have 2 created IDs")
	for _, id := range createdIDs {
		assertIDFormat(t, id, "un")
	}
	assert.Empty(t, updatedIDs, "no updates expected")
}

func TestUnits_BulkUpsert_AllUpdates(t *testing.T) {
	t.Parallel()

	name, abbr := uniqueName("e2e-bu"), uniqueName("ebu")

	// Create first
	createdIDs, _ := bulkUpsertUnits(t, upsertUnitInput(name, abbr, "quantity"))
	defer cleanupUnitIDs(createdIDs)

	// Upsert again with a different ratio — should update
	created, updated := bulkUpsertUnits(t, map[string]any{
		"name":               name,
		"abbreviation":       abbr,
		"type":               "quantity",
		"ratio_numerator":    "12",
		"ratio_denominator":  "1",
		"offset_numerator":   "0",
		"offset_denominator": "1",
		"is_base_unit":       false,
	})

	assert.Empty(t, created, "no creates expected")
	assert.Len(t, updated, 1, "should have 1 updated ID")
}

func TestUnits_BulkUpsert_MixedCreateAndUpdate(t *testing.T) {
	t.Parallel()

	existingName, existingAbbr := uniqueName("e2e-bu"), uniqueName("ebu")
	newName, newAbbr := uniqueName("e2e-bu"), uniqueName("ebu")

	// Seed the existing unit
	seeded, _ := bulkUpsertUnits(t, upsertUnitInput(existingName, existingAbbr, "quantity"))
	defer cleanupUnitIDs(seeded)

	// Mix: update existing + create new
	created, updated := bulkUpsertUnits(t,
		upsertUnitInput(existingName, existingAbbr, "quantity"),
		upsertUnitInput(newName, newAbbr, "quantity"),
	)
	defer cleanupUnitIDs(created)

	assert.Len(t, created, 1, "one new unit created")
	assert.Len(t, updated, 1, "one existing unit updated")
}

func TestUnits_BulkUpsert_ResponseShape(t *testing.T) {
	t.Parallel()

	name, abbr := uniqueName("e2e-bu"), uniqueName("ebu")

	// 202 returns the canonical job resource.
	status, body, err := apiClient.Post(unitsBulkUpsertPath, map[string]any{
		"units": []any{upsertUnitInput(name, abbr, "quantity")},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, status, body)

	m := parseJSON(body)
	require.NotNil(t, m, "response body should parse as JSON")
	assertObjectField(t, m, "job")
	jobID := jsonField(m, "id")
	require.NotEmpty(t, jobID)

	// The completed job carries a row-indexed result per created/updated unit.
	job := pollJobUntilTerminal(t, jobID)
	require.Equal(t, "completed", jsonField(job, "status"))
	createdIDs, _ := jobResultIDs(job)
	require.Len(t, createdIDs, 1)
	entry := jobResults(job)[0]
	assert.Equal(t, float64(0), entry["index"], "the result names request row 0")
	assert.Equal(t, "created", entry["status"])
	assertIDFormat(t, createdIDs[0], "un")
	defer cleanupUnitIDs([]string{createdIDs[0]})
}

func TestUnits_BulkUpsert_Idempotent(t *testing.T) {
	t.Parallel()

	name, abbr := uniqueName("e2e-bu"), uniqueName("ebu")
	idemKey := newIdempotencyKey()
	payload := map[string]any{
		"units": []any{upsertUnitInput(name, abbr, "quantity")},
	}

	status1, body1, err := apiClient.Post(unitsBulkUpsertPath, payload, idemKey)
	require.NoError(t, err)
	requireStatus(t, 202, status1, body1)

	status2, body2, err := apiClient.Post(unitsBulkUpsertPath, payload, idemKey)
	require.NoError(t, err)
	requireStatus(t, 202, status2, body2)

	// Replay must return the identical job acknowledgment, not raise a second one.
	assert.JSONEq(t, string(body1), string(body2), "idempotent replay must return the same job")

	jobID := jsonField(parseJSON(body1), "id")
	job := pollJobUntilTerminal(t, jobID)
	require.Equal(t, "completed", jsonField(job, "status"))
	created, _ := jobResultIDs(job)
	require.Len(t, created, 1)
	defer cleanupUnitIDs([]string{created[0]})
}

func TestUnits_BulkUpsert_CreatedUnitAppearsInList(t *testing.T) {
	t.Parallel()

	name, abbr := uniqueName("e2e-bu"), uniqueName("ebu")

	createdIDs, _ := bulkUpsertUnits(t, upsertUnitInput(name, abbr, "quantity"))
	defer cleanupUnitIDs(createdIDs)
	require.Len(t, createdIDs, 1)
	createdID := createdIDs[0]

	// Verify it's reachable via GET
	getStatus, getBody, err := apiClient.GetListRaw(unitsPath+"/"+createdID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	assert.Equal(t, createdID, jsonField(got, "id"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, abbr, jsonField(got, "abbreviation"))
}

// --- Bulk Upsert Validation ---

func TestUnits_BulkUpsert_EmptyUnits(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(unitsBulkUpsertPath, map[string]any{
		"units": []any{},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"empty units should return 400 or 422, got %d: %s", status, string(body))
}

func TestUnits_BulkUpsert_TooManyUnits(t *testing.T) {
	t.Parallel()
	units := make([]any, 1001)
	for i := range units {
		units[i] = upsertUnitInput(uniqueName("e2e-bu"), uniqueName("ebu"), "quantity")
	}
	status, body, err := apiClient.Post(unitsBulkUpsertPath, map[string]any{"units": units}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"1001 units should return 400 or 422, got %d: %s", status, string(body))
}

func TestUnits_BulkUpsert_DuplicateNameInRequest(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-bu")
	status, body, err := apiClient.Post(unitsBulkUpsertPath, map[string]any{
		"units": []any{
			upsertUnitInput(name, uniqueName("ebu"), "quantity"),
			upsertUnitInput(name, uniqueName("ebu"), "quantity"),
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"duplicate name in request should return 400 or 422, got %d: %s", status, string(body))
	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assert.Equal(t, "units[1].name", errObj["param"])
	assert.Contains(t, errObj["message"], "duplicate name")
}

func TestUnits_BulkUpsert_DuplicateAbbreviationInRequest(t *testing.T) {
	t.Parallel()
	abbr := uniqueName("ebu")
	status, body, err := apiClient.Post(unitsBulkUpsertPath, map[string]any{
		"units": []any{
			upsertUnitInput(uniqueName("e2e-bu"), abbr, "quantity"),
			upsertUnitInput(uniqueName("e2e-bu"), abbr, "quantity"),
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"duplicate abbreviation in request should return 400 or 422, got %d: %s", status, string(body))
	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assert.Equal(t, "units[1].abbreviation", errObj["param"])
	assert.Contains(t, errObj["message"], "duplicate abbreviation")
}

func TestUnits_BulkUpsert_ZeroRatioDenominator(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(unitsBulkUpsertPath, map[string]any{
		"units": []any{
			map[string]any{
				"name":               uniqueName("e2e-bu"),
				"abbreviation":       uniqueName("ebu"),
				"type":               "quantity",
				"ratio_numerator":    "1",
				"ratio_denominator":  "0",
				"offset_numerator":   "0",
				"offset_denominator": "1",
				"is_base_unit":       false,
			},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"zero ratio_denominator should return 400 or 422, got %d: %s", status, string(body))
	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assertErrorParam(t, errObj, "units[0].ratio_denominator")
}

func TestUnits_BulkUpsert_ZeroOffsetDenominator(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(unitsBulkUpsertPath, map[string]any{
		"units": []any{
			map[string]any{
				"name":               uniqueName("e2e-bu"),
				"abbreviation":       uniqueName("ebu"),
				"type":               "quantity",
				"ratio_numerator":    "1",
				"ratio_denominator":  "1",
				"offset_numerator":   "0",
				"offset_denominator": "0",
				"is_base_unit":       false,
			},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"zero offset_denominator should return 400 or 422, got %d: %s", status, string(body))
	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assertErrorParam(t, errObj, "units[0].offset_denominator")
}

func TestUnits_BulkUpsert_NameAndAbbrConflict(t *testing.T) {
	t.Parallel()

	name1, abbr1 := uniqueName("e2e-bu"), uniqueName("ebu")
	name2, abbr2 := uniqueName("e2e-bu"), uniqueName("ebu")

	// Create two distinct units
	seeded, _ := bulkUpsertUnits(t,
		upsertUnitInput(name1, abbr1, "quantity"),
		upsertUnitInput(name2, abbr2, "quantity"),
	)
	defer cleanupUnitIDs(seeded)

	// A row whose name matches unit1 but whose abbreviation matches unit2 crosses two
	// existing units — a conflict decidable only against live rows, so it is accepted
	// (202) and recorded as a per-row failure on the otherwise-completed job.
	job := bulkUpsertUnitsJob(t, map[string]any{
		"name":               name1,
		"abbreviation":       abbr2, // crosses into the other unit
		"type":               "quantity",
		"ratio_numerator":    "1",
		"ratio_denominator":  "1",
		"offset_numerator":   "0",
		"offset_denominator": "1",
		"is_base_unit":       false,
	})

	rowErrs := jobErrors(job)
	require.Len(t, rowErrs, 1, "the conflicting row is recorded in errors")
	msg := jobRowErrorMessage(rowErrs[0])
	assert.Contains(t, msg, name1)
	assert.Contains(t, msg, abbr2)
}

func TestUnits_BulkUpsert_DimensionCodeImmutable(t *testing.T) {
	t.Parallel()

	name, abbr := uniqueName("e2e-bu"), uniqueName("ebu")

	// Create as "quantity"
	seeded, _ := bulkUpsertUnits(t, upsertUnitInput(name, abbr, "quantity"))
	defer cleanupUnitIDs(seeded)

	// Changing the dimension code is rejected against the existing row: accepted (202),
	// recorded as a per-row failure on the completed job.
	job := bulkUpsertUnitsJob(t, upsertUnitInput(name, abbr, "mass"))
	rowErrs := jobErrors(job)
	require.Len(t, rowErrs, 1)
	msg := jobRowErrorMessage(rowErrs[0])
	assert.Contains(t, msg, "immutable")
}

func TestUnits_BulkUpsert_IsBaseUnitImmutable(t *testing.T) {
	t.Parallel()

	name, abbr := uniqueName("e2e-bu"), uniqueName("ebu")

	// Create with is_base_unit=false
	seeded, _ := bulkUpsertUnits(t, upsertUnitInput(name, abbr, "quantity"))
	defer cleanupUnitIDs(seeded)

	// Flipping is_base_unit is rejected against the existing row: accepted (202),
	// recorded as a per-row failure on the completed job.
	job := bulkUpsertUnitsJob(t, map[string]any{
		"name":               name,
		"abbreviation":       abbr,
		"type":               "quantity",
		"ratio_numerator":    "1",
		"ratio_denominator":  "1",
		"offset_numerator":   "0",
		"offset_denominator": "1",
		"is_base_unit":       true,
	})
	rowErrs := jobErrors(job)
	require.Len(t, rowErrs, 1)
	msg := jobRowErrorMessage(rowErrs[0])
	assert.Contains(t, msg, "IsBaseUnit")
}

// Partial success: a batch mixing a valid new unit with a row that fails against an
// existing unit creates the good one and records the bad one — the job completes.
func TestUnits_BulkUpsert_PartialSuccess(t *testing.T) {
	t.Parallel()

	existingName, existingAbbr := uniqueName("e2e-bu"), uniqueName("ebu")
	goodName, goodAbbr := uniqueName("e2e-bu"), uniqueName("ebu")

	// Seed a unit whose dimension the bad row will try to change.
	seeded, _ := bulkUpsertUnits(t, upsertUnitInput(existingName, existingAbbr, "quantity"))
	defer cleanupUnitIDs(seeded)

	// Row 0: a valid new unit. Row 1: an immutable dimension change on the seeded unit.
	job := bulkUpsertUnitsJob(t,
		upsertUnitInput(goodName, goodAbbr, "quantity"),
		upsertUnitInput(existingName, existingAbbr, "mass"),
	)

	created, _ := jobResultIDs(job)
	require.Len(t, created, 1, "the valid row is still created")
	defer cleanupUnitIDs([]string{created[0]})
	assert.Equal(t, float64(0), jobResults(job)[0]["index"], "the surviving result names request row 0")

	rowErrs := jobErrors(job)
	require.Len(t, rowErrs, 1, "the bad row is recorded")
	assert.Equal(t, float64(1), rowErrs[0]["index"], "the failure names the second row")
	code, _ := jobRowError(rowErrs[0])["code"].(string)
	assert.NotEmpty(t, code, "the failure carries the canonical error object with a machine-readable code")
}
