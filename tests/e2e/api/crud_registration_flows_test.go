//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const registrationFlowsPath = "/v1/sales/registration-flows"

func TestRegistrationFlows_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(registrationFlowsPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
}

func TestRegistrationFlows_CRUDAndResponseShape(t *testing.T) {
	t.Parallel()

	createBody := map[string]any{
		"name": "E2E Test Registration Flow",
	}
	status, respBody, err := apiClient.Post(registrationFlowsPath, createBody, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)

	m := parseJSON(respBody)
	assert.Equal(t, "registration_flow", jsonField(m, "object"))
	flowID := jsonField(m, "id")
	assert.NotEmpty(t, flowID)
	assert.Equal(t, "E2E Test Registration Flow", jsonField(m, "name"))
	assert.NotEmpty(t, jsonField(m, "created_at"))
	assert.NotEmpty(t, jsonField(m, "updated_at"))

	cgOptions := jsonObject(m, "customer_group_options")
	require.NotNil(t, cgOptions, "customer_group_options should be present")
	assert.Equal(t, "list", jsonField(cgOptions, "object"))

	ptOptions := jsonObject(m, "payment_term_options")
	require.NotNil(t, ptOptions, "payment_term_options should be present")
	assert.Equal(t, "list", jsonField(ptOptions, "object"))

	stOptions := jsonObject(m, "shipping_term_options")
	require.NotNil(t, stOptions, "shipping_term_options should be present")
	assert.Equal(t, "list", jsonField(stOptions, "object"))

	flowPath := registrationFlowsPath + "/" + flowID
	status, respBody, err = apiClient.Do("GET", flowPath, nil, "")
	require.NoError(t, err)
	requireStatus(t, 200, status, respBody)

	retrieved := parseJSON(respBody)
	assert.Equal(t, flowID, jsonField(retrieved, "id"))
	assert.Equal(t, "registration_flow", jsonField(retrieved, "object"))

	updatedName := "Updated E2E Flow"
	updateBody := map[string]any{
		"name": updatedName,
	}
	status, respBody, err = apiClient.Patch(flowPath, updateBody, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, respBody)

	updated := parseJSON(respBody)
	assert.Equal(t, flowID, jsonField(updated, "id"))
	assert.Equal(t, updatedName, jsonField(updated, "name"))

	status, respBody, err = apiClient.Delete(flowPath)
	require.NoError(t, err)
	requireStatus(t, 200, status, respBody)
}

// optionIDs reads the ids of one of a registration flow's option lists.
func optionIDs(t *testing.T, flow map[string]any, list string) []string {
	t.Helper()
	require.NotNil(t, jsonObject(flow, list), "%s is always present: %v", list, flow)
	ids := []string{}
	for _, raw := range jsonListData(flow, list) {
		option, ok := raw.(map[string]any)
		require.True(t, ok)
		ids = append(ids, jsonField(option, "id"))
	}
	return ids
}

// createRegistrationFlow makes a flow offering the seeded payment term and deletes it afterwards.
func createRegistrationFlow(t *testing.T, client *Client) string {
	t.Helper()
	status, body, err := client.Post(registrationFlowsPath, map[string]any{
		"name":             uniqueName("e2e-flow"),
		"payment_term_ids": []string{SeedPaymentTermID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { _, _, _ = apiClient.Delete(registrationFlowsPath + "/" + id) })
	return id
}

func patchRegistrationFlow(t *testing.T, client *Client, id string, body map[string]any) (int, map[string]any, []byte) {
	t.Helper()
	status, respBody, err := client.Patch(registrationFlowsPath+"/"+id, body, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "updating a registration flow must not 5xx: %s", string(respBody))
	return status, parseJSON(respBody), respBody
}

// A list that is sent replaces the options, an omitted one is kept, and an empty one clears them.
func TestRegistrationFlows_UpdateReplacesKeepsAndClearsOptionLists(t *testing.T) {
	t.Parallel()
	id := createRegistrationFlow(t, apiClient)

	status, flow, body := patchRegistrationFlow(t, apiClient, id, map[string]any{"payment_term_ids": []string{SeedDefaultPaymentTermID}})
	requireStatus(t, 200, status, body)
	assert.Equal(t, []string{SeedDefaultPaymentTermID}, optionIDs(t, flow, "payment_term_options"), "a sent list replaces the options")

	status, flow, body = patchRegistrationFlow(t, apiClient, id, map[string]any{"customer_group_ids": []string{SeedCustomerGroupID}})
	requireStatus(t, 200, status, body)
	assert.Equal(t, []string{SeedDefaultPaymentTermID}, optionIDs(t, flow, "payment_term_options"), "an omitted list is unchanged")
	assert.Equal(t, []string{SeedCustomerGroupID}, optionIDs(t, flow, "customer_group_options"))

	status, flow, body = patchRegistrationFlow(t, apiClient, id, map[string]any{"payment_term_ids": []string{}})
	requireStatus(t, 200, status, body)
	assert.Empty(t, optionIDs(t, flow, "payment_term_options"), "an empty list clears the options")
	assert.Equal(t, []string{SeedCustomerGroupID}, optionIDs(t, flow, "customer_group_options"))
}

func TestRegistrationFlows_UpdateRejectsANullOptionList(t *testing.T) {
	t.Parallel()
	id := createRegistrationFlow(t, apiClient)

	status, _, body := patchRegistrationFlow(t, apiClient, id, map[string]any{"payment_term_ids": nil})
	assert.Equal(t, 400, status, "send [] to clear options, not null: %s", string(body))
}

// The replace flags belong to 1.0.forge-preview.5; the current version names them as unknown.
func TestRegistrationFlows_UpdateRejectsThePreview5Flags(t *testing.T) {
	t.Parallel()
	id := createRegistrationFlow(t, apiClient)

	status, _, body := patchRegistrationFlow(t, apiClient, id, map[string]any{"has_payment_term_ids": true, "payment_term_ids": []string{}})
	require.Equal(t, 400, status, "%s", string(body))
	errObj := requireErrorResponse(t, body, "parameter_unknown", "invalid_request_error")
	assertErrorParam(t, errObj, "has_payment_term_ids")
}
