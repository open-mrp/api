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

// The department and scanning_station includes must carry the full resource, not a stub built from the
// production-step projection: that projection hardcoded the department name to the literal "Department"
// and dropped the station's label type/size, so a stub returns those wrong or null even though the
// station and department store real values. See TestIncludes_HydratedToOneMatchesCanonical for the
// cross-endpoint guard.
func TestProductionSteps_IncludeDepartmentAndScanningStationHydrated(t *testing.T) {
	t.Parallel()
	id := SeedSewLargeProductionStepID

	status, body, err := apiClient.GetListRaw(productionStepsPath+"/"+id, url.Values{"include": {"department", "scanning_station"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)

	dept := jsonObject(got, "department")
	require.NotNil(t, dept, "department should be present with ?include=department")
	assert.Equal(t, "department", jsonField(dept, "object"))
	assert.NotEmpty(t, jsonField(dept, "id"))
	assert.Equal(t, "Sewing", jsonField(dept, "name"),
		"the included department carries its real name, not the hardcoded \"Department\" stub")

	station := jsonObject(got, "scanning_station")
	require.NotNil(t, station, "scanning_station should be present with ?include=scanning_station")
	assert.Equal(t, "scanning_station", jsonField(station, "object"))
	assert.NotEmpty(t, jsonField(station, "id"))
	assert.Equal(t, "move_batch", jsonField(station, "type"),
		"the included station carries its real type, not the init_batch stub default")
	assert.Equal(t, "tag", jsonField(station, "label_type"),
		"the included station carries its label_type, not a partial stub")
	assert.Equal(t, "1x4", jsonField(station, "label_size"),
		"the included station carries its label_size, not a partial stub")
}
