//go:build e2e

package api_test

import (
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaterials_BurnRate_FromConsumptionHistory(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-mat-burn")
	body := validMaterialBody(sku)
	createStatus, createBody, err := apiClient.Post(materialsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	created := parseJSON(createBody)
	materialID := jsonField(created, "id")
	require.NotEmpty(t, materialID)
	t.Cleanup(func() { apiClient.Delete(materialsPath + "/" + materialID) })

	getStatus, getBody, err := apiClient.GetListRaw(materialsPath+"/"+materialID, url.Values{"include": {"item"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	item := jsonObject(parseJSON(getBody), "item")
	require.NotNil(t, item, "item should be present with ?include=item")
	itemID := jsonField(item, "id")
	require.NotEmpty(t, itemID)

	adjust := func(delta string) {
		t.Helper()
		patchBody := map[string]any{
			"quantity":  map[string]any{"value": delta, "unit_id": nonCurrencyUnitID},
			"operation": "adjust",
		}
		status, resp, patchErr := apiClient.Patch(itemsPath+"/"+itemID+"/inventory", patchBody, newIdempotencyKey())
		require.NoError(t, patchErr)
		requireStatus(t, 200, status, resp)
	}

	adjust("-10")
	adjust("-5")

	status, getBody, err := apiClient.GetListRaw(materialsPath+"/"+materialID, url.Values{
		"include": {"item.burn_rate"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, getBody)

	got := parseJSON(getBody)
	item = jsonObject(got, "item")
	require.NotNil(t, item, "item should be present with include=item.burn_rate")

	burnRate := jsonObject(item, "burn_rate")
	require.NotNil(t, burnRate, "item.burn_rate should be present")

	valueStr := jsonField(burnRate, "value")
	require.NotEmpty(t, valueStr)
	measure, parseErr := strconv.ParseFloat(valueStr, 64)
	require.NoError(t, parseErr)
	assert.Greater(t, measure, 0.0, "burn rate should reflect consumption history")

	denUnit := jsonObject(burnRate, "denominator_unit")
	require.NotNil(t, denUnit, "burn rate denominator_unit should be present")
	assert.Equal(t, "day", jsonField(denUnit, "id"))
}
