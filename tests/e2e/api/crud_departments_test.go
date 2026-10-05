//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const departmentsPath = "/v1/operations/departments"

// ──────────────────────────────────────────────
// Department — Include Tests
// ──────────────────────────────────────────────

func TestDepartments_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(departmentsPath+"/"+SeedDepartmentID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["location"], "location should be null without ?include=location")
	assert.Nil(t, got["scanning_stations"], "scanning_stations should be null without ?include=scanning_stations")
	assert.Nil(t, got["machines"], "machines should be null without ?include=machines")
}

func TestDepartments_IncludeLocation(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(departmentsPath+"/"+SeedDepartmentID, url.Values{"include": {"location"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["location"]
	assert.True(t, ok, "location key should be present with ?include=location")
	if loc := jsonObject(got, "location"); loc != nil {
		assert.Equal(t, "location", jsonField(loc, "object"))
	}
}

func TestDepartments_IncludeScanningStations(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(departmentsPath+"/"+SeedDepartmentID, url.Values{"include": {"scanning_stations"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	ss := jsonObject(got, "scanning_stations")
	require.NotNil(t, ss, "scanning_stations should be present with ?include=scanning_stations")
	assert.Equal(t, "list", jsonField(ss, "object"))
}

func TestDepartments_IncludeMachines(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(departmentsPath+"/"+SeedDepartmentID, url.Values{"include": {"machines"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	ms := jsonObject(got, "machines")
	require.NotNil(t, ms, "machines should be present with ?include=machines")
	assert.Equal(t, "list", jsonField(ms, "object"))
}

func TestDepartments_UpdateNotes_OmittedKeepsAndNullClears(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(departmentsPath, map[string]any{
		"name":  uniqueName("e2e-dp-notes"),
		"notes": "Keep me",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(departmentsPath + "/" + id)

	renamed := uniqueName("e2e-dp-notes-r")
	status, body, err = apiClient.Patch(departmentsPath+"/"+id, map[string]any{"name": renamed}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	assert.Equal(t, renamed, jsonField(got, "name"))
	assert.Equal(t, "Keep me", jsonField(got, "notes"), "omitting notes must leave them unchanged")

	status, body, err = apiClient.Patch(departmentsPath+"/"+id, map[string]any{"notes": nil}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assertNilField(t, parseJSON(body), "notes")

	getStatus, getBody, err := apiClient.GetListRaw(departmentsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assertNilField(t, parseJSON(getBody), "notes")
}
