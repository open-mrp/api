//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func serviceLevelsPath(carrierID string) string {
	return fmt.Sprintf("/v1/operations/carriers/%s/service-levels", carrierID)
}

func TestServiceLevels_OwnerIncludesOnGet(t *testing.T) {
	t.Parallel()
	path := serviceLevelsPath(SeedCarrierID) + "/" + SeedServiceLevelID
	status, body, err := apiClient.GetListRaw(path, url.Values{"include": {"owner"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, "service_level", jsonField(got, "object"))

	owner := jsonObject(got, "owner")
	require.NotNil(t, owner, "owner should be present with ?include=owner")
	assert.NotEmpty(t, jsonField(owner, "type"))
}

func TestServiceLevels_OwnerOmittedWithoutInclude(t *testing.T) {
	t.Parallel()
	path := serviceLevelsPath(SeedCarrierID) + "/" + SeedServiceLevelID
	status, body, err := apiClient.GetListRaw(path, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["owner"], "owner should be null without ?include=owner")
}

func TestServiceLevels_SystemOwned_UpdateForbidden(t *testing.T) {
	t.Parallel()
	path := serviceLevelsPath(SeedSystemCarrierID) + "/" + SeedSystemServiceLevelID
	status, body, err := apiClient.Patch(path, map[string]any{
		"name": uniqueName("e2e-sys-sl-upd"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 403, status, "System-owned service level update should return 403, got %d: %s", status, string(body))
}

func TestServiceLevels_SystemOwned_DeleteForbidden(t *testing.T) {
	t.Parallel()
	path := serviceLevelsPath(SeedSystemCarrierID) + "/" + SeedSystemServiceLevelID
	status, body, err := apiClient.Delete(path)
	require.NoError(t, err)
	assert.Equal(t, 403, status, "System-owned service level delete should return 403, got %d: %s", status, string(body))
}

func TestServiceLevels_AccountOwned_UpdateSucceeds(t *testing.T) {
	t.Parallel()
	basePath := serviceLevelsPath(SeedCarrierID)

	newName := uniqueName("e2e-sl-owned")
	createStatus, createBody, err := apiClient.Post(basePath, map[string]any{
		"name": newName,
		"code": uniqueName("sl_code"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	created := parseJSON(createBody)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(basePath + "/" + id)

	updatedName := uniqueName("e2e-sl-owned-upd")
	patchStatus, patchBody, err := apiClient.Patch(basePath+"/"+id, map[string]any{
		"name": updatedName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	assert.Equal(t, updatedName, jsonField(parseJSON(patchBody), "name"))
}
