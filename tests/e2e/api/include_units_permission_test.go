//go:build e2e

package api_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A role that may read production runs and their batches sees the units the batches are counted in
// without also holding units:read, as the dashboard always showed them; the units themselves stay
// behind units:read.
func TestIncludes_RunBatchesCarryTheirUnitsWithoutUnitsRead(t *testing.T) {
	t.Parallel()
	reader := customRoleClient(t, "production_runs:read", "batches:read")

	status, body, err := reader.GetListRaw(unitsPath, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusForbidden, status, "the role cannot list units on their own: %s", string(body))

	params := url.Values{"scope": {"run"}, "limit": {"100"}, "include": {"quantity.unit", "seconds.unit", "waste.unit"}}
	path := productionRunBatchesPath(SeedProductionRunID)
	got := parseJSON(mustGetAs(t, reader, path, params))
	want := parseJSON(mustGetAs(t, apiClient, path, params))
	assert.Equal(t, want, got, "the role reads the batches exactly as an admin does")

	batches := jsonArray(got, "data")
	require.NotEmpty(t, batches, "the seed run has batches")
	for _, raw := range batches {
		quantity := jsonObject(raw.(map[string]any), "quantity")
		require.NotNil(t, quantity)
		unit := jsonObject(quantity, "unit")
		require.NotNil(t, unit, "the batch quantity's unit is expanded")
		assert.NotEmpty(t, jsonField(unit, "abbreviation"))
	}
}
