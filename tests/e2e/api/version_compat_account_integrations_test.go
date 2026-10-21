//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const preview9APIVersion = "1.0.forge-preview.9"

// preview.10 adds the `openrouter` provider. A client pinned to preview.9 lists integrations without it,
// while the latest list carries it. Called with an OpenRouter integration the caller owns.
func assertPreview9IntegrationListOmits(t *testing.T, openRouterIntegrationID string) {
	t.Helper()
	pinned := apiClient.WithAPIVersion(preview9APIVersion)
	list, _, err := pinned.GetList(covSettingsIntegrationsPath, nil)
	require.NoError(t, err)
	for _, item := range list.Data {
		assert.NotEqual(t, "openrouter", DataItemField(item, "provider"), "a preview.9 list must not carry an OpenRouter integration")
		assert.NotEqual(t, openRouterIntegrationID, DataItemField(item, "id"))
	}
	assertListContainsID(t, covSettingsIntegrationsPath, nil, openRouterIntegrationID)
}
