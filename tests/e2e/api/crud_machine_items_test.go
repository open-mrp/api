//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func machineItemsPath(machineID string) string {
	return machinesPath + "/" + machineID + "/items"
}

func TestMachineItems_ReplaceListSuggest(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-mc-items")
	createResp, err := apiClient.PostFull(machinesPath, map[string]any{
		"name":           name,
		"serial_number":  uniqueName("SN-MI"),
		"work_center_id": SeedDepartmentID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)
	machineID := jsonField(parseJSON(createResp.Body), "id")
	t.Cleanup(func() {
		_, _, _ = apiClient.Delete(machinesPath + "/" + machineID)
	})

	listStatus, listBody, err := apiClient.GetListRaw(machineItemsPath(machineID), nil)
	require.NoError(t, err)
	requireStatus(t, 200, listStatus, listBody)
	assert.Empty(t, jsonArray(parseJSON(listBody), "data"))

	suggestStatus, suggestBody, err := apiClient.Post(machineItemsPath(machineID)+"/actions/suggest-from-scans", map[string]any{
		"window_months": 24,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, suggestStatus, suggestBody)
	assert.NotNil(t, jsonArray(parseJSON(suggestBody), "data"))

	listStatus, listBody, err = apiClient.GetListRaw(machineItemsPath(machineID), nil)
	require.NoError(t, err)
	requireStatus(t, 200, listStatus, listBody)
	assert.Empty(t, jsonArray(parseJSON(listBody), "data"),
		"suggest-from-scans must not write machine_item rows")

	putStatus, putBody, err := apiClient.Put(machineItemsPath(machineID), map[string]any{
		"item_ids": []any{SeedItemID},
	})
	require.NoError(t, err)
	requireStatus(t, 200, putStatus, putBody)
	data := jsonArray(parseJSON(putBody), "data")
	require.Len(t, data, 1)
	assert.Equal(t, SeedItemID, jsonField(data[0].(map[string]any), "id"))
	assert.Equal(t, "item", jsonField(data[0].(map[string]any), "object"))

	listStatus, listBody, err = apiClient.GetListRaw(machineItemsPath(machineID), nil)
	require.NoError(t, err)
	requireStatus(t, 200, listStatus, listBody)
	require.Len(t, jsonArray(parseJSON(listBody), "data"), 1)

	clearStatus, clearBody, err := apiClient.Put(machineItemsPath(machineID), map[string]any{
		"item_ids": []any{},
	})
	require.NoError(t, err)
	requireStatus(t, 200, clearStatus, clearBody)
	assert.Empty(t, jsonArray(parseJSON(clearBody), "data"))
}
