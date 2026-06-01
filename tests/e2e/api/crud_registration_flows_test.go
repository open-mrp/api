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
