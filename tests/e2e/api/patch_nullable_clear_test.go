//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddresses_UpdateClearNullableFields(t *testing.T) {
	t.Parallel()
	id := SeedAddressID

	setStatus, setBody, err := apiClient.Patch(addressesPath+"/"+id, map[string]any{
		"email":         "clear-me@e2e.openmrp.ai",
		"phone":         "555-111-2222",
		"street_line_2": "Suite 500",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, setStatus, setBody)

	clearStatus, clearBody, err := apiClient.Patch(addressesPath+"/"+id, map[string]any{
		"email":         nil,
		"phone":         nil,
		"street_line_2": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, clearStatus, clearBody)

	getStatus, getBody, err := apiClient.GetListRaw(addressesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assertNilField(t, got, "email")
	assertNilField(t, got, "phone")
	geo := jsonObject(got, "geolocation")
	require.NotNil(t, geo)
	assertNilField(t, geo, "street_line_2")
}

func TestParts_UpdateClearDescriptionAndNotes(t *testing.T) {
	t.Parallel()
	id := SeedPartID

	setStatus, setBody, err := apiClient.Patch(partsPath+"/"+id, map[string]any{
		"description": "temp part description",
		"notes":       "temp part notes",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, setStatus, setBody)

	clearStatus, clearBody, err := apiClient.Patch(partsPath+"/"+id, map[string]any{
		"description": nil,
		"notes":       nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, clearStatus, clearBody)

	getStatus, getBody, err := apiClient.GetListRaw(partsPath+"/"+id, url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	item := jsonObject(parseJSON(getBody), "item")
	require.NotNil(t, item)
	assertNilField(t, item, "description")
	assertNilField(t, item, "notes")
}

func TestScanningStations_UpdateClearNotes(t *testing.T) {
	t.Parallel()
	id := SeedScanningStationID

	setStatus, setBody, err := apiClient.Patch("/v1/operations/scanning-stations/"+id, map[string]any{
		"notes": "station notes to clear",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, setStatus, setBody)

	clearStatus, clearBody, err := apiClient.Patch("/v1/operations/scanning-stations/"+id, map[string]any{
		"notes": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, clearStatus, clearBody)
	assert.Nil(t, parseJSON(clearBody)["notes"])
}

func TestUnitGroups_UpdateClearNotes(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-ug-clear-notes")
	createStatus, createBody, err := apiClient.Post(unitGroupsPath, map[string]any{
		"name":         name,
		"type":         "quantity",
		"base_unit_id": "each",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	t.Cleanup(func() { apiClient.Delete(unitGroupsPath + "/" + id) })

	setStatus, setBody, err := apiClient.Patch(unitGroupsPath+"/"+id, map[string]any{
		"notes": "notes to clear",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, setStatus, setBody)

	clearStatus, clearBody, err := apiClient.Patch(unitGroupsPath+"/"+id, map[string]any{
		"notes": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, clearStatus, clearBody)
	assert.Nil(t, parseJSON(clearBody)["notes"])
}
