package versiontransforms

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/api/shared/apiversion"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/version"
)

const forgeWorkCenterJSON = `{"id":"wkcc_1","object":"work_center","name":"Knitting","location":null,"labor_rate":null,
	"machines":{"object":"list","data":[{"id":"mch_1","object":"machine","name":"Loom 1","serial_number":"L1","status":"active","work_center":null}]},
	"scanning_stations":null,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}`

func TestPreview9_WorkCenterReadsAsADepartment(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeWorkCenter, forgeWorkCenterJSON)
	assert.Equal(t, "department", got["object"])
	assert.Contains(t, got, "notes")
	assert.Nil(t, got["notes"])

	machine := got["machines"].(map[string]any)["data"].([]any)[0].(map[string]any)
	assert.Contains(t, machine, "department")
	assert.NotContains(t, machine, "work_center")
	assert.Nil(t, machine["notes"])
}

func TestPreview9_MachineAndStationNameTheirDepartment(t *testing.T) {
	t.Parallel()

	machine := downgrade(t, constants.ObjectTypeMachine, `{"id":"mch_1","object":"machine","status":"archived","work_center":{"id":"wkcc_1","object":"work_center","name":"Knitting"}}`)
	dept := machine["department"].(map[string]any)
	assert.Equal(t, "department", dept["object"])
	assert.Equal(t, "wkcc_1", dept["id"])
	assert.NotContains(t, machine, "work_center")
	assert.Equal(t, "archived", machine["status"])

	station := downgrade(t, constants.ObjectTypeScanningStation, `{"object":"list","data":[{"id":"ss_1","object":"scanning_station","work_center":null}]}`)
	row := station["data"].([]any)[0].(map[string]any)
	assert.Contains(t, row, "department")
	assert.Nil(t, row["department"])
	assert.Nil(t, row["notes"])
}

func TestPreview9_WorkCenterNestedInOtherResources(t *testing.T) {
	t.Parallel()

	user := downgrade(t, constants.ObjectTypeAccountUser, `{"id":"au_1","object":"account_user","department":{"id":"dp_1","object":"work_center","name":"Knitting"}}`)
	assert.Equal(t, "department", user["department"].(map[string]any)["object"])

	job := downgrade(t, constants.ObjectTypeAsyncJob, `{"id":"jb_1","object":"async_job","resource_type":"work_center","results":{"object":"list","data":[
		{"object":"async_job_result","resource":{"id":"wkcc_1","object":"work_center"}}]}}`)
	assert.Equal(t, "department", job["resource_type"])
	result := job["results"].(map[string]any)["data"].([]any)[0].(map[string]any)
	assert.Equal(t, "department", result["resource"].(map[string]any)["object"])

	event := downgrade(t, constants.ObjectTypeAuditEvent, `{"id":"ae_1","object":"audit_event","resource_type":"work_center"}`)
	assert.Equal(t, "department", event["resource_type"])
	other := downgrade(t, constants.ObjectTypeAuditEvent, `{"id":"ae_2","object":"audit_event","resource_type":"machine"}`)
	assert.Equal(t, "machine", other["resource_type"])
}

func TestPreview9_WorkCenterDeletedStubIsEmpty(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeWorkCenter, `{"id":"wkcc_1","object":"work_center","deleted":true}`)
	assert.Empty(t, got)
}

func TestPreview9_WorkCenterRequestBodies(t *testing.T) {
	t.Parallel()

	create := upgradeBody(constants.ObjectTypeWorkCenter, http.MethodPost, workCentersRoute, map[string]any{"name": "Knitting", "notes": "old"})
	assert.Equal(t, map[string]any{"name": "Knitting"}, create)
	patch := upgradeBody(constants.ObjectTypeWorkCenter, http.MethodPatch, workCenterRoute, map[string]any{"notes": nil})
	assert.Empty(t, patch)

	bulk := upgradeBody(constants.ObjectTypeAsyncJob, http.MethodPost, workCentersBulkUpsertRoute, map[string]any{
		"departments": []any{map[string]any{"name": "Knitting", "notes": "x", "location": map[string]any{"name": "Plant"}}}})
	assert.Equal(t, map[string]any{"work_centers": []any{map[string]any{"name": "Knitting", "location": map[string]any{"name": "Plant"}}}}, bulk)

	machine := upgradeBody(constants.ObjectTypeMachine, http.MethodPost, machinesRoute, map[string]any{"name": "Loom", "department_id": "dp_1", "notes": "x"})
	assert.Equal(t, map[string]any{"name": "Loom", "work_center_id": "dp_1"}, machine)
	station := upgradeBody(constants.ObjectTypeScanningStation, http.MethodPost, stationsRoute, map[string]any{"name": "S1", "department_id": "dp_1"})
	assert.Equal(t, map[string]any{"name": "S1", "work_center_id": "dp_1"}, station)
	machinePatch := upgradeBody(constants.ObjectTypeMachine, http.MethodPatch, machineRoute, map[string]any{"name": "Loom 2", "notes": nil})
	assert.Equal(t, map[string]any{"name": "Loom 2"}, machinePatch)

	machines := upgradeBody(constants.ObjectTypeAsyncJob, http.MethodPost, machinesBulkUpsertRoute, map[string]any{
		"machines": []any{map[string]any{"name": "Loom", "department": map[string]any{"name": "Knitting"}, "notes": "x"}}})
	assert.Equal(t, map[string]any{"machines": []any{map[string]any{"name": "Loom", "work_center": map[string]any{"name": "Knitting"}}}}, machines)
	stations := upgradeBody(constants.ObjectTypeAsyncJob, http.MethodPost, stationsBulkUpsertRoute, map[string]any{
		"scanning_stations": []any{map[string]any{"name": "S1", "department": map[string]any{"id": "dp_1"}}}})
	assert.Equal(t, map[string]any{"scanning_stations": []any{map[string]any{"name": "S1", "work_center": map[string]any{"id": "dp_1"}}}}, stations)

	other := upgradeBody(constants.ObjectTypeAccountUser, http.MethodPost, "/v1/identity/account-users", map[string]any{"department_id": "dp_1"})
	assert.Equal(t, "dp_1", other["department_id"], "bodies of other endpoints are left alone")
}

func TestPreview9_WorkCenterIncludeKeys(t *testing.T) {
	t.Parallel()

	const route = "/v1/test/wc/machines"
	RegisterRouteIncludes(route, []string{"work_center", "work_center.location"})
	q := version.TransformQuery(apiversion.V1_0_Forge_Preview9, apiversion.V1_0_Forge, constants.ObjectTypeMachine, route,
		url.Values{"include[]": {"department", "department.location"}})
	assert.Equal(t, []string{"work_center", "work_center.location"}, q["include[]"])

	const stepRoute = "/v1/test/wc/steps"
	RegisterRouteIncludes(stepRoute, []string{"department", "machines", "machines.work_center"})
	q = version.TransformQuery(apiversion.V1_0_Forge_Preview9, apiversion.V1_0_Forge, constants.ObjectTypeProductionStep, stepRoute,
		url.Values{"include": {"department,machines.department"}})
	assert.Equal(t, []string{"department,machines.work_center"}, q["include"], "an offered department key is kept")
}

func TestPreview9_WorkCenterRoutes(t *testing.T) {
	t.Parallel()

	var path string
	h := RoutePreview9WorkCenters(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { path = r.URL.Path }))
	serve := func(p string, v version.APIVersion) {
		req := httptest.NewRequest(http.MethodGet, p, nil)
		req.Header.Set(apiversion.Header, v.String())
		h.ServeHTTP(httptest.NewRecorder(), req)
	}

	serve("/v1/operations/departments", apiversion.V1_0_Forge_Preview9)
	assert.Equal(t, "/v1/operations/work-centers", path)
	serve("/v1/operations/departments/dp_1", apiversion.V1_0_Forge_Preview9)
	assert.Equal(t, "/v1/operations/work-centers/dp_1", path)
	serve("/v1/operations/departments/actions/bulk-upsert", apiversion.V1_0_Forge_Preview9)
	assert.Equal(t, "/v1/operations/work-centers/actions/bulk-upsert", path)
	serve("/v1/operations/departments-other", apiversion.V1_0_Forge_Preview9)
	assert.Equal(t, "/v1/operations/departments-other", path, "only the departments path itself moves")

	serve("/v1/operations/departments", apiversion.V1_0_Forge)
	assert.Equal(t, "/v1/operations/departments", path, "forge callers are not rerouted")
}
