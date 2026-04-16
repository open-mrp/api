//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func consumptionsPath() string {
	return "/v1/operations/production-steps/" + SeedProductionStepID + "/consumptions"
}

// ──────────────────────────────────────────────
// Consumption — Include Tests
// ──────────────────────────────────────────────
//
// Consumption GET endpoint whitelists: consumed_item.
// (quantity and waste_quantity are always populated as summaries on this endpoint.)

func TestConsumptions_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(consumptionsPath()+"/"+SeedConsumptionID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["consumed_item"], "consumed_item should be null without ?include=consumed_item")
}

func TestConsumptions_IncludeConsumedItem(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(consumptionsPath()+"/"+SeedConsumptionID, url.Values{"include": {"consumed_item"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	item := jsonObject(got, "consumed_item")
	require.NotNil(t, item, "consumed_item should be present with ?include=consumed_item")
	assert.Equal(t, "item", jsonField(item, "object"))
}
