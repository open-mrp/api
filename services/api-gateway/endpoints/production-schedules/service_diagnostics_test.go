package productionscheduleep

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeScheduleDiagnostics_LegacyPascalCase(t *testing.T) {
	t.Parallel()

	raw := `{
		"AppliedOverrides": null,
		"AverageInputsAdded": 1.5,
		"CapacityStarvedSKUs": null,
		"ChangeoverSlopeMinutes": 2.25,
		"EOQCappedSKUs": ["SKU-A"],
		"ExcludedItemCount": 3,
		"ItemsWithoutRunRate": ["SKU-B"],
		"UnschedulableSKUs": []
	}`

	got := decodeScheduleDiagnostics(raw)
	assert.Equal(t, []string{"SKU-A"}, got.EOQCappedSKUs)
	assert.Equal(t, []string{}, got.UnschedulableSKUs)
	assert.Equal(t, []string{}, got.CapacityStarvedSKUs)
	assert.Equal(t, []string{"SKU-B"}, got.ItemsWithoutRunRate)
	assert.EqualValues(t, 3, got.ExcludedItemCount)
	assert.Equal(t, 2.25, got.ChangeoverSlopeMinutes)
	assert.Equal(t, 1.5, got.AverageInputsAdded)
	require.NotNil(t, got.AppliedOverrides)
	assert.Empty(t, got.AppliedOverrides.Data)

	encoded, err := json.Marshal(got)
	require.NoError(t, err)
	var asMap map[string]any
	require.NoError(t, json.Unmarshal(encoded, &asMap))
	for _, field := range []string{
		"eoq_capped_skus", "unschedulable_skus", "capacity_starved_skus",
		"items_without_run_rate", "applied_overrides", "excluded_item_count",
		"changeover_slope_minutes", "average_inputs_added",
	} {
		_, ok := asMap[field]
		assert.True(t, ok, "expected snake_case field %s", field)
	}
	for _, field := range []string{"EOQCappedSKUs", "AppliedOverrides", "AverageInputsAdded"} {
		_, ok := asMap[field]
		assert.False(t, ok, "legacy PascalCase field %s must not appear", field)
	}
}

func TestDecodeScheduleDiagnostics_SnakeCase(t *testing.T) {
	t.Parallel()

	raw := `{
		"eoq_capped_skus": ["A"],
		"unschedulable_skus": [],
		"capacity_starved_skus": ["B"],
		"items_without_run_rate": [],
		"excluded_item_count": 1,
		"changeover_slope_minutes": 4,
		"average_inputs_added": 2,
		"applied_overrides": [{
			"override_id": "dov_1",
			"item_id": "it_1",
			"month_start": "2026-01-01T00:00:00Z",
			"before": 10,
			"after": 20,
			"adjustment": "absolute",
			"reason": "promotion"
		}]
	}`

	got := decodeScheduleDiagnostics(raw)
	assert.Equal(t, []string{"A"}, got.EOQCappedSKUs)
	assert.Equal(t, []string{"B"}, got.CapacityStarvedSKUs)
	assert.EqualValues(t, 1, got.ExcludedItemCount)
	require.NotNil(t, got.AppliedOverrides)
	require.Len(t, got.AppliedOverrides.Data, 1)
	applied := got.AppliedOverrides.Data[0]
	require.NotNil(t, applied.Override)
	assert.Equal(t, "dov_1", applied.Override.ID)
	require.NotNil(t, applied.Item)
	assert.Equal(t, "it_1", applied.Item.ID)
}
