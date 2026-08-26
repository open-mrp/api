//go:build e2e

package api_test

import (
	"fmt"
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

	// The recompute runs off the consumption transaction via the outbox, so the rate lands a beat after the PATCH returns.
	var burnRate map[string]any
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		status, body, getErr := apiClient.GetListRaw(materialsPath+"/"+materialID, url.Values{
			"include": {"item.burn_rate"},
		})
		if getErr != nil {
			return getErr
		}
		if status != 200 {
			return fmt.Errorf("retrieve material: status %d: %s", status, body)
		}

		gotItem := jsonObject(parseJSON(body), "item")
		if gotItem == nil {
			return fmt.Errorf("item should be present with include=item.burn_rate: %s", body)
		}
		rate := jsonObject(gotItem, "burn_rate")
		if rate == nil {
			return fmt.Errorf("item.burn_rate should be present: %s", body)
		}
		valueStr := jsonField(rate, "value")
		measure, parseErr := strconv.ParseFloat(valueStr, 64)
		if parseErr != nil {
			return fmt.Errorf("burn rate value %q: %w", valueStr, parseErr)
		}
		if measure <= 0 {
			return fmt.Errorf("burn rate %q should reflect consumption history", valueStr)
		}

		burnRate = rate
		return nil
	})

	denUnit := jsonObject(burnRate, "denominator_unit")
	require.NotNil(t, denUnit, "burn rate denominator_unit should be present")
	assert.Equal(t, "day", jsonField(denUnit, "id"))
}
