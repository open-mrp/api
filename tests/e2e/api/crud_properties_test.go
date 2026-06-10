//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const propertiesPath = "/v1/catalog/properties"

func TestProperties_CRUD(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-prop")
	createResp, err := apiClient.PostFull(propertiesPath, map[string]any{"name": name}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	assert.Equal(t, "property", jsonField(created, "object"))
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	assert.Equal(t, name, jsonField(created, "name"))

	getStatus, getBody, err := apiClient.GetListRaw(propertiesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, name, jsonField(parseJSON(getBody), "name"))

	newName := uniqueName("e2e-prop-upd")
	patchStatus, patchBody, err := apiClient.Patch(propertiesPath+"/"+id, map[string]any{"name": newName}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	assert.Equal(t, newName, jsonField(parseJSON(patchBody), "name"))

	getStatus2, getBody2, err := apiClient.GetListRaw(propertiesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus2, getBody2)
	assert.Equal(t, newName, jsonField(parseJSON(getBody2), "name"))

	delStatus, delBody, err := apiClient.Delete(propertiesPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	getStatus3, _, err := apiClient.GetListRaw(propertiesPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus3)
}

func TestProperties_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()

	// ── CREATE ──
	name := uniqueName("e2e-prop-allf")
	createResp, err := apiClient.PostFull(propertiesPath, map[string]any{
		"name": name,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	got := parseJSON(createResp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	defer apiClient.Delete(propertiesPath + "/" + id)

	assert.Equal(t, "property", jsonField(got, "object"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.NotEmpty(t, jsonField(got, "created_at"))
	assert.NotEmpty(t, jsonField(got, "updated_at"))

	// ── UPDATE ──
	updatedName := uniqueName("e2e-prop-allf-u")
	patchStatus, patchBody, err := apiClient.Patch(propertiesPath+"/"+id, map[string]any{
		"name": updatedName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	assert.Equal(t, id, jsonField(updated, "id"), "ID must not change")
	assert.Equal(t, updatedName, jsonField(updated, "name"))
}

func TestProperties_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(propertiesPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 1)

	found := false
	for _, item := range list.Data {
		if DataItemField(item, "name") == SeedPropertyName {
			found = true
			break
		}
	}
	assert.True(t, found, "Seeded property %q should appear in list", SeedPropertyName)
}

func TestProperties_ListSearchByName(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(propertiesPath, url.Values{"q": {"Color"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Search for 'Color' should return at least 1 result")

	for _, item := range list.Data {
		name := DataItemField(item, "name")
		assert.True(t,
			strings.Contains(strings.ToLower(name), "color"),
			"Search result %q should contain 'color'", name,
		)
	}
}

func TestProperties_ListSearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(propertiesPath, url.Values{"q": {"zzzznotaprop99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

func TestProperties_IncludeAttributes(t *testing.T) {
	t.Parallel()
	getStatus, getBody, err := apiClient.GetListRaw(propertiesPath+"/"+SeedPropertyID, url.Values{"include": {"attributes"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	attrs := jsonObject(parseJSON(getBody), "attributes")
	require.NotNil(t, attrs)
	assert.Equal(t, "list", jsonField(attrs, "object"))
}

func TestProperties_CreateIdempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-idem-prop")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Post(propertiesPath, map[string]any{"name": name}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")

	status2, body2, err := apiClient.Post(propertiesPath, map[string]any{"name": name}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id1, jsonField(parseJSON(body2), "id"))

	apiClient.Delete(propertiesPath + "/" + id1)
}

func TestProperties_CreateValidation_EmptyName(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(propertiesPath, map[string]any{"name": ""}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Empty name should return 400 or 422, got %d: %s", status, string(body))
}
