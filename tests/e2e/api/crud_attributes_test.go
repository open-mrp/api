//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func attributesPath(propertyID string) string {
	return "/v1/catalog/properties/" + propertyID + "/attributes"
}

func attributePath(propertyID, attributeID string) string {
	return attributesPath(propertyID) + "/" + attributeID
}

func TestAttributes_CRUD(t *testing.T) {
	t.Parallel()
	value := uniqueName("e2e-attr")
	createResp, err := apiClient.PostFull(attributesPath(SeedPropertyID), map[string]any{
		"value": value,
		"color": "blue",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	assert.Equal(t, "attribute", jsonField(created, "object"))
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assert.Equal(t, value, jsonField(created, "value"))
	assert.Equal(t, "blue", jsonField(created, "color"))

	getStatus, getBody, err := apiClient.GetListRaw(attributePath(SeedPropertyID, id), nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, value, jsonField(parseJSON(getBody), "value"))

	newValue := uniqueName("e2e-attr-upd")
	patchStatus, patchBody, err := apiClient.Patch(attributePath(SeedPropertyID, id), map[string]any{
		"value": newValue,
		"color": "red",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	assert.Equal(t, newValue, jsonField(parseJSON(patchBody), "value"))
	assert.Equal(t, "red", jsonField(parseJSON(patchBody), "color"))

	delStatus, delBody, err := apiClient.Delete(attributePath(SeedPropertyID, id))
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	getStatus2, _, err := apiClient.GetListRaw(attributePath(SeedPropertyID, id), nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus2)
}

func TestAttributes_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()

	// ── CREATE with all fields ──
	value := uniqueName("e2e-attr-allf")
	createResp, err := apiClient.PostFull(attributesPath(SeedPropertyID), map[string]any{
		"value": value,
		"color": "blue",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	got := parseJSON(createResp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(attributePath(SeedPropertyID, id))

	assert.Equal(t, "attribute", jsonField(got, "object"))
	assert.Equal(t, value, jsonField(got, "value"))
	assert.Equal(t, "blue", jsonField(got, "color"))
	assert.NotEmpty(t, jsonField(got, "sort_order"))
	assert.NotEmpty(t, jsonField(got, "created_at"))
	assert.NotEmpty(t, jsonField(got, "updated_at"))

	// ── UPDATE with different values ──
	updatedValue := uniqueName("e2e-attr-allf-u")
	patchStatus, patchBody, err := apiClient.Patch(attributePath(SeedPropertyID, id), map[string]any{
		"value":      updatedValue,
		"color":      "red",
		"sort_order": 1,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	assert.Equal(t, id, jsonField(updated, "id"), "ID must not change")
	assert.Equal(t, updatedValue, jsonField(updated, "value"))
	assert.Equal(t, "red", jsonField(updated, "color"))
	assert.Equal(t, "1", jsonField(updated, "sort_order"))
}

func TestAttributes_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(attributesPath(SeedPropertyID), nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 1)

	found := false
	for _, item := range list.Data {
		if DataItemField(item, "id") == SeedAttributeID {
			found = true
			break
		}
	}
	assert.True(t, found, "Seeded attribute should appear in list")
}

func TestAttributes_ListSearchByValue(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(attributesPath(SeedPropertyID), url.Values{"q": {"Beige"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Search for 'Beige' should return at least 1 result")
}

func TestAttributes_ListSearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(attributesPath(SeedPropertyID), url.Values{"q": {"zzzznotanattr99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}
