//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sandboxesPath = "/v1/core/sandboxes"

func TestSandboxes_CreateBlank(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-sandbox")
	status, body, err := apiClient.Post(sandboxesPath, map[string]any{
		"name": name,
		"mode": "blank",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	m := parseJSON(body)
	assert.Equal(t, "sandbox", jsonField(m, "object"))
	assert.NotEmpty(t, jsonField(m, "id"))
	assert.Equal(t, name, jsonField(m, "name"))
	assert.NotEmpty(t, jsonField(m, "created_at"))
	assert.NotEmpty(t, jsonField(m, "updated_at"))

	apiClient.Delete(sandboxesPath + "/" + jsonField(m, "id"))
}

func TestSandboxes_CreateSeeded(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-seeded")
	status, body, err := apiClient.Post(sandboxesPath, map[string]any{
		"name": name,
		"mode": "seeded",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	m := parseJSON(body)
	assert.Equal(t, "sandbox", jsonField(m, "object"))
	assert.NotEmpty(t, jsonField(m, "id"))

	apiClient.Delete(sandboxesPath + "/" + jsonField(m, "id"))
}

func TestSandboxes_Get(t *testing.T) {
	t.Parallel()
	createStatus, createBody, err := apiClient.Post(sandboxesPath, map[string]any{
		"name": uniqueName("e2e-get"),
		"mode": "blank",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	status, body, err := apiClient.GetListRaw(sandboxesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	m := parseJSON(body)
	assert.Equal(t, "sandbox", jsonField(m, "object"))
	assert.Equal(t, id, jsonField(m, "id"))

	apiClient.Delete(sandboxesPath + "/" + id)
}

func TestSandboxes_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(sandboxesPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.NotNil(t, list.Data)
}

func TestSandboxes_ListSearchByName(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(sandboxesPath, url.Values{"q": {"Sandbox"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Search for 'Sandbox' should return at least 1 result")
}

func TestSandboxes_ListSearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(sandboxesPath, url.Values{"q": {"zzzznotasandbox99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

func TestSandboxes_Delete(t *testing.T) {
	t.Parallel()
	createStatus, createBody, err := apiClient.Post(sandboxesPath, map[string]any{
		"name": uniqueName("e2e-del"),
		"mode": "blank",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	delStatus, delBody, err := apiClient.Delete(sandboxesPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 202, delStatus, delBody)

	getStatus, _, err := apiClient.GetListRaw(sandboxesPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus)
}

func TestSandboxes_Idempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-idem")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Post(sandboxesPath, map[string]any{"name": name, "mode": "blank"}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")

	status2, body2, err := apiClient.Post(sandboxesPath, map[string]any{"name": name, "mode": "blank"}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id1, jsonField(parseJSON(body2), "id"))

	apiClient.Delete(sandboxesPath + "/" + id1)
}

func TestSandboxes_CreateValidation_EmptyName(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(sandboxesPath, map[string]any{
		"name": "",
		"mode": "blank",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Empty name should return 400 or 422, got %d: %s", status, string(body))
}
