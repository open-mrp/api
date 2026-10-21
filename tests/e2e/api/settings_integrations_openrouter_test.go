//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const openRouterUsagePath = "/v1/settings/integrations/openrouter/usage"

// The main account's only OpenRouter slot: every step runs in order on the one integration.
func TestSettingsIntegrations_OpenRouter(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(covSettingsIntegrationsPath, map[string]any{
		"name":                uniqueName("e2e-openrouter"),
		"provider":            "openrouter",
		"credentials":         `{"api_key":"sk-or-v1-e2e"}`,
		"monthly_spend_limit": map[string]any{"amount": "25.50", "currency": "usd"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	created := parseJSON(body)
	id := jsonField(created, "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(covSettingsIntegrationsPath + "/" + id) })
	assert.Equal(t, "openrouter", jsonField(created, "provider"))
	assert.Equal(t, map[string]any{"amount": "25.5", "currency": "usd"}, created["monthly_spend_limit"])

	t.Run("UsageReportsTheLimit", func(t *testing.T) {
		status, body, err := apiClient.GetListRaw(openRouterUsagePath, nil)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		usage := parseJSON(body)
		assertObjectField(t, usage, "openrouter_usage")
		assert.Equal(t, map[string]any{"amount": "25.5", "currency": "usd"}, usage["monthly_spend_limit"])
		assert.Equal(t, map[string]any{"amount": "0", "currency": "usd"}, usage["spent_this_month"])
		assert.Equal(t, map[string]any{"amount": "25.5", "currency": "usd"}, usage["monthly_spend_remaining"])
	})

	t.Run("RejectsABadLimit", func(t *testing.T) {
		for name, limit := range map[string]map[string]any{
			"zero":        {"amount": "0", "currency": "usd"},
			"negative":    {"amount": "-1", "currency": "usd"},
			"not dollars": {"amount": "10", "currency": "eur"},
		} {
			t.Run(name, func(t *testing.T) {
				status, body, err := apiClient.Put(covSettingsIntegrationsPath+"/"+id, map[string]any{"monthly_spend_limit": limit})
				require.NoError(t, err)
				requireStatus(t, 400, status, body)
				requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
			})
		}
	})

	t.Run("Preview9ListLeavesItOut", func(t *testing.T) {
		assertPreview9IntegrationListOmits(t, id)
	})

	t.Run("NullRemovesTheLimit", func(t *testing.T) {
		status, body, err := apiClient.Put(covSettingsIntegrationsPath+"/"+id, map[string]any{"monthly_spend_limit": nil})
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		assert.Nil(t, parseJSON(body)["monthly_spend_limit"])
	})
}

// Only OpenRouter spends on the account's behalf, so no other provider takes a limit.
func TestSettingsIntegrations_SpendLimitIsOpenRouterOnly(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(covSettingsIntegrationsPath, map[string]any{
		"name":                uniqueName("e2e-hubspot-limit"),
		"provider":            "hubspot",
		"credentials":         covSettingsIntegrationsHubspotCredentials("limit"),
		"monthly_spend_limit": map[string]any{"amount": "10", "currency": "usd"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
}
