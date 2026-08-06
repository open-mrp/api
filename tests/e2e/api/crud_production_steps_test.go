//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const productionStepsPath = "/v1/operations/production-steps"

func TestProductionSteps_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()
	id := SeedSewLargeProductionStepID

	status, body, err := apiClient.GetListRaw(productionStepsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["production"], "production should be null without include")
	assert.Nil(t, got["consumptions"], "consumptions should be null without include")
	assert.Nil(t, got["machines"], "machines should be null without include")
	assert.Nil(t, got["scanning_station"], "scanning_station should be null without include")
	assert.Nil(t, got["department"], "department should be null without include")
	assert.Nil(t, got["in_steps"], "in_steps should be null without include")
	assert.Nil(t, got["out_steps"], "out_steps should be null without include")
}

func TestProductionSteps_IncludeProduction(t *testing.T) {
	t.Parallel()
	id := SeedSewLargeProductionStepID

	status, body, err := apiClient.GetListRaw(productionStepsPath+"/"+id, url.Values{"include": {"production"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	production := jsonObject(got, "production")
	require.NotNil(t, production, "production should be present with ?include=production")
	assert.Equal(t, "production", jsonField(production, "object"))
}

func TestProductionSteps_IncludeConsumptions(t *testing.T) {
	t.Parallel()
	id := SeedSewLargeProductionStepID

	status, body, err := apiClient.GetListRaw(productionStepsPath+"/"+id, url.Values{"include": {"consumptions"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.NotNil(t, got["consumptions"], "consumptions should be present with ?include=consumptions")
}

func TestProductionSteps_IncludeMachines(t *testing.T) {
	t.Parallel()
	id := SeedSewLargeProductionStepID

	status, body, err := apiClient.GetListRaw(productionStepsPath+"/"+id, url.Values{"include": {"machines"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.NotNil(t, got["machines"], "machines should be present with ?include=machines")
}
