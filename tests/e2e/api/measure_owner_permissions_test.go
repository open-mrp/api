//go:build e2e

package api_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// A quantity or rate is edited under the update permission of the resource it belongs to, whatever the request names
// as its owner, and one the account does not hold reads as not found.

const (
	quantitiesPath = "/v1/operations/quantities"
	ratesPath      = "/v1/operations/rates"
)

func mustPostAs(t *testing.T, c *Client, path string, body map[string]any) []byte {
	t.Helper()
	status, resp, err := c.Post(path, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, status, resp)
	id := jsonField(parseJSON(resp), "id")
	base, _, _ := strings.Cut(path, "?")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(base + "/" + id) })
	return resp
}

func patchMeasure(t *testing.T, c *Client, path string, body map[string]any) (int, []byte) {
	t.Helper()
	status, resp, err := c.Patch(path, body, newIdempotencyKey())
	require.NoError(t, err)
	return status, resp
}

func TestRates_EditTakesThePermissionOfTheRatesOwner(t *testing.T) {
	t.Parallel()
	step := createEditableStep(t)
	_, itemID := parityMaterial(t)
	item := parseJSON(mustGetAs(t, apiClient, "/v1/catalog/items/"+itemID, url.Values{"include": {"unit_cost"}}))
	itemRate := jsonField(jsonObject(item, "unit_cost"), "id")
	require.NotEmpty(t, itemRate)
	stepRate := ratesPath + "/" + step.laborRateID

	itemsEditor := customRoleClient(t, "items:update")
	stepsEditor := customRoleClient(t, "production_steps:update")

	status, body := patchMeasure(t, itemsEditor, stepRate, map[string]any{"value": "25", "object_type": "item", "object_id": itemID})
	requirePermissionRefused(t, status, body, "production_steps:update")
	status, body = patchMeasure(t, stepsEditor, stepRate, map[string]any{"value": "25"})
	requireStatus(t, http.StatusOK, status, body)

	status, body = patchMeasure(t, stepsEditor, ratesPath+"/"+itemRate, map[string]any{"value": "0.5", "object_type": "production_step", "object_id": step.id})
	requirePermissionRefused(t, status, body, "items:update")
	status, body = patchMeasure(t, itemsEditor, ratesPath+"/"+itemRate, map[string]any{"value": "0.5"})
	requireStatus(t, http.StatusOK, status, body)

	status, body = patchMeasure(t, getTenantBClient(), stepRate, map[string]any{"value": "25"})
	requireStatus(t, http.StatusNotFound, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

func TestQuantities_EditTakesThePermissionOfTheQuantitysOwner(t *testing.T) {
	t.Parallel()
	step := createEditableStep(t)
	create := validMaterialBody(uniqueName("e2e-measure-mat"))
	create["order_point"] = map[string]any{"value": "10", "unit_id": SeedMaterialUnitID}
	material := parseJSON(mustPostAs(t, apiClient, materialsPath+"?include=item", create))
	itemID := jsonField(jsonObject(material, "item"), "id")
	orderPoint := quantitiesPath + "/" + jsonField(jsonObject(material, "order_point"), "id")
	require.NotEqual(t, quantitiesPath+"/", orderPoint, "the material reports its order point: %v", material)
	consumption := parseJSON(mustGetAs(t, apiClient, productionStepsPath+"/"+step.id+"/consumptions/"+step.consumption, url.Values{"include": {"quantity.unit"}}))
	consumed := quantitiesPath + "/" + jsonField(jsonObject(consumption, "quantity"), "id")

	itemsEditor := customRoleClient(t, "items:update")
	stepsEditor := customRoleClient(t, "production_steps:update")

	status, body := patchMeasure(t, stepsEditor, orderPoint, map[string]any{"value": "5", "object_type": "production_step", "object_id": step.id})
	requirePermissionRefused(t, status, body, "items:update")
	status, body = patchMeasure(t, itemsEditor, orderPoint, map[string]any{"value": "5"})
	requireStatus(t, http.StatusOK, status, body)

	status, body = patchMeasure(t, itemsEditor, consumed, map[string]any{"value": "1", "object_type": "item", "object_id": itemID})
	requirePermissionRefused(t, status, body, "production_steps:update")
	status, body = patchMeasure(t, stepsEditor, consumed, map[string]any{"value": "1"})
	requireStatus(t, http.StatusOK, status, body)

	status, body = patchMeasure(t, getTenantBClient(), orderPoint, map[string]any{"value": "5"})
	requireStatus(t, http.StatusNotFound, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}
