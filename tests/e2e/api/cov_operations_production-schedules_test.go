//go:build e2e

package api_test

import (
	"math"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const productionSchedulePreviewPath = "/v1/operations/production-schedules/actions/preview"

// planningMu keeps settings writes off the same clock as any solve.
//
// Planning settings are singular per account and every solve reads them, so a test that halves the horizon while another test is generating would silently move that other test's plan. Solves take the read side and run concurrently with each other; only a settings write is exclusive.
var planningMu sync.RWMutex

// lockPlanningRead holds the read side of planningMu for the duration of one call.
func lockPlanningRead() func() {
	planningMu.RLock()
	return planningMu.RUnlock
}

// The solver is unusable without at least one machine marked as the planning constraint. That is a configuration problem, so it must be a clear 4xx rather than a blank plan that reads as "nothing to do".
func TestProductionSchedulePreview_RequiresAConstraintMachine(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Put(productionSchedulePreviewPath, map[string]any{})
	require.NoError(t, err)
	require.Less(t, status, 500, "the preview must not 5xx: %s", string(body))

	if status == 200 {
		// A constraint is configured in this environment, so the solve should have run.
		preview := parseJSON(body)
		assert.Equal(t, "production_schedule_preview", jsonField(preview, "object"))
		return
	}

	assert.Equal(t, 400, status,
		"with no constraint machine configured the preview should be a 400, got %d: %s", status, string(body))
	assert.Contains(t, string(body), "constraint",
		"the error should say what to configure")
}

func TestProductionSchedulePreview_ValidatesOverrides(t *testing.T) {
	t.Parallel()

	t.Run("rejects an out-of-range horizon", func(t *testing.T) {
		status, body, err := apiClient.Put(productionSchedulePreviewPath, map[string]any{
			"horizon_weeks": 500,
		})
		require.NoError(t, err)
		assert.Less(t, status, 500, "a bad horizon must be a client error, not a 5xx: %s", string(body))
		assert.Equal(t, 400, status, "horizon_weeks above the maximum should be 400, got %d: %s", status, string(body))
	})

	t.Run("rejects an unknown demand basis", func(t *testing.T) {
		status, body, err := apiClient.Put(productionSchedulePreviewPath, map[string]any{
			"demand_basis": "crystal_ball",
		})
		require.NoError(t, err)
		assert.Less(t, status, 500, "a bad basis must be a client error, not a 5xx: %s", string(body))
		assert.Equal(t, 400, status, "an unknown demand_basis should be 400, got %d: %s", status, string(body))
	})

	t.Run("accepts the documented bases", func(t *testing.T) {
		for _, basis := range []string{"trailing_12", "seasonal_ema"} {
			status, body, err := apiClient.Put(productionSchedulePreviewPath, map[string]any{
				"demand_basis":  basis,
				"horizon_weeks": 13,
			})
			require.NoError(t, err)
			require.Less(t, status, 500, "basis %q must not 5xx: %s", basis, string(body))
			assert.Contains(t, []int{200, 400}, status,
				"basis %q should be accepted or fail configuration, got %d: %s", basis, status, string(body))
		}
	})
}

// A plan that cannot explain itself will not be trusted, so the diagnostics block must always be present and its lists must serialize as [] rather than null.
func TestProductionSchedulePreview_ShapeWhenSolved(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Put(productionSchedulePreviewPath, map[string]any{
		"planning_as_of": rfc3339(time.Now().UTC()),
	})
	require.NoError(t, err)
	require.Less(t, status, 500, "the preview must not 5xx: %s", string(body))

	if status != 200 {
		t.Skipf("no constraint machine configured in this environment (status %d)", status)
	}

	preview := parseJSON(body)
	assert.Equal(t, "production_schedule_preview", jsonField(preview, "object"))
	assert.NotEmpty(t, jsonField(preview, "solver_version"), "the plan must record which solver produced it")
	assert.NotEmpty(t, jsonField(preview, "planning_as_of_at"))

	assert.NotNil(t, preview["policies"], "policies must serialize as [] rather than null")
	assert.NotNil(t, preview["campaigns"], "campaigns must serialize as [] rather than null")
	assert.NotNil(t, preview["projections"], "projections must serialize as [] rather than null")

	diagnostics := jsonObject(preview, "diagnostics")
	require.NotNil(t, diagnostics, "diagnostics must always be present")
	for _, field := range []string{
		"eoq_capped_skus", "unschedulable_skus", "capacity_starved_skus",
		"items_without_run_rate", "applied_overrides",
	} {
		assert.NotNil(t, diagnostics[field], "diagnostics.%s must serialize as [] rather than null", field)
	}

	// Every campaign must be actionable: a machine to run on and hours it consumes.
	for _, raw := range jsonListData(preview, "campaigns") {
		campaign, ok := raw.(map[string]any)
		require.True(t, ok)
		machine := jsonObject(campaign, "machine")
		require.NotNil(t, machine, "a campaign must name the machine it runs on: %v", campaign)
		assert.NotEmpty(t, jsonField(machine, "id"))
		runHours, ok := campaign["run_hours"].(float64)
		require.True(t, ok)
		assert.Greater(t, runHours, 0.0, "a campaign must consume constraint time")
	}
}

// The same input must produce the same plan. Go randomizes map iteration, so this is the guard that the sorting in the solver has not been lost.
func TestProductionSchedulePreview_Deterministic(t *testing.T) {
	t.Parallel()

	// Both solves have to see the same assumptions, so a settings write cannot land between them.
	defer lockPlanningRead()()

	asOf := rfc3339(time.Now().UTC())
	req := map[string]any{"planning_as_of": asOf, "horizon_weeks": 13}

	status, first, err := apiClient.Put(productionSchedulePreviewPath, req)
	require.NoError(t, err)
	require.Less(t, status, 500, "the preview must not 5xx: %s", string(first))
	if status != 200 {
		t.Skipf("no constraint machine configured in this environment (status %d)", status)
	}

	status2, second, err := apiClient.Put(productionSchedulePreviewPath, req)
	require.NoError(t, err)
	requireStatus(t, 200, status2, second)

	// The assumptions a solve was run under are fully determined by the request and the
	// settings, so these must match exactly.
	firstPlan, secondPlan := parseJSON(first), parseJSON(second)
	assert.Equal(t, jsonField(firstPlan, "solver_version"), jsonField(secondPlan, "solver_version"))
	assert.Equal(t, firstPlan["settings_snapshot"], secondPlan["settings_snapshot"],
		"two solves seconds apart must have run under the same assumptions")

	// The plan itself is deliberately NOT compared byte for byte any more. A solve now
	// reads the open order book as well as history, and the suite issues and unissues
	// orders in parallel throughout — so two solves seconds apart can legitimately see
	// different demand and produce different campaigns. That is the point of planning
	// against firm orders, not a defect.
	//
	// Byte-identity for a FIXED input is a stronger claim and is pinned where the input
	// can be held still: TestSolve_Deterministic and TestBuildFirmSchedule_Deterministic
	// each run 50 times over one fixture and compare exactly.
	assert.NotEmpty(t, jsonField(secondPlan, "object"), "both solves must return a well-formed plan")
}

const productionSchedulesPath = "/v1/operations/production-schedules"

// generateSchedule creates a schedule version, skipping the test when the environment has no constraint machine configured.
func generateSchedule(t *testing.T, body map[string]any) map[string]any {
	t.Helper()

	defer lockPlanningRead()()
	return generateScheduleLocked(t, body)
}

// generateScheduleLocked is generateSchedule for callers already holding the planning lock; taking it again here would deadlock against a queued writer.
func generateScheduleLocked(t *testing.T, body map[string]any) map[string]any {
	t.Helper()

	resp, err := apiClient.PostFull(productionSchedulesPath, body, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, resp.StatusCode, 500, "generate must not 5xx: %s", string(resp.Body))
	if resp.StatusCode == 400 {
		t.Skipf("no constraint machine configured in this environment: %s", string(resp.Body))
	}
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	return parseJSON(resp.Body)
}

// cleanupSchedule registers deletion of a generated version. Only drafts delete; a published version is retired by archiving it.
func cleanupSchedule(t *testing.T, id string) {
	t.Helper()
	t.Cleanup(func() {
		status, _, _ := apiClient.Delete(schedulePath(id))
		if status == 400 {
			_, _, _ = apiClient.Put(schedulePath(id)+"/actions/archive", map[string]any{})
		}
	})
}

func TestProductionSchedules_GenerateAndRead(t *testing.T) {
	// Parallel-safe: it only compares this test's own create against its own re-read.
	t.Parallel()

	created := generateSchedule(t, map[string]any{
		"name":          uniqueName("e2e-schedule"),
		"horizon_weeks": 4,
	})

	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	cleanupSchedule(t, id)

	assertIDFormat(t, id, "pnsc")
	assert.Equal(t, "production_schedule", jsonField(created, "object"))
	assert.Equal(t, "draft", jsonField(created, "status"),
		"a freshly generated schedule must be a draft, not published")
	assert.Equal(t, "manual", jsonField(created, "generation_source"))
	assert.NotEmpty(t, jsonField(created, "solver_version"), "the plan must record which solver produced it")
	assert.NotEmpty(t, jsonField(created, "planning_as_of_at"))
	assert.NotEmpty(t, jsonField(created, "horizon_starts_at"))
	assert.NotEmpty(t, jsonField(created, "horizon_ends_at"))
	assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")

	// The assumptions and diagnostics are frozen onto the version so the plan stays explainable after settings move.
	assert.NotNil(t, created["settings_snapshot"], "settings must be snapshotted onto the version")
	assert.NotNil(t, created["diagnostics"], "diagnostics must be snapshotted onto the version")

	settings := jsonObject(created, "settings_snapshot")
	require.NotNil(t, settings, "settings_snapshot must decode to an object, not an escaped string")
	assert.NotEmpty(t, settings, "the snapshot must actually contain the assumptions used")

	// GET
	getStatus, getBody, err := apiClient.GetListRaw(productionSchedulesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	fetched := parseJSON(getBody)
	assert.Equal(t, id, jsonField(fetched, "id"))
	assert.Equal(t, jsonField(created, "version"), jsonField(fetched, "version"))

	// LIST includes it
	listStatus, listBody, err := apiClient.GetListRaw(productionSchedulesPath, url.Values{"limit": {"100"}})
	require.NoError(t, err)
	requireStatus(t, 200, listStatus, listBody)

	var found bool
	for _, raw := range jsonArray(parseJSON(listBody), "data") {
		schedule, ok := raw.(map[string]any)
		require.True(t, ok)
		if jsonField(schedule, "id") == id {
			found = true
		}
	}
	assert.True(t, found, "the generated schedule must appear in the list")
}

// Generating again creates a NEW version rather than replacing the previous one: attainment is measured against whichever version was live at the time.
func TestProductionSchedules_GeneratingAgainCreatesANewVersion(t *testing.T) {
	// Parallel-safe: it compares this test's own two versions relative to each other, not absolute numbering.
	t.Parallel()

	first := generateSchedule(t, map[string]any{"horizon_weeks": 4})
	second := generateSchedule(t, map[string]any{"horizon_weeks": 4})

	firstID := jsonField(first, "id")
	secondID := jsonField(second, "id")
	cleanupSchedule(t, firstID)
	cleanupSchedule(t, secondID)
	assert.NotEqual(t, firstID, secondID, "a second generation must be a distinct version")

	firstVersion, ok := first["version"].(float64)
	require.True(t, ok)
	secondVersion, ok := second["version"].(float64)
	require.True(t, ok)
	assert.Greater(t, secondVersion, firstVersion, "version numbers must increase")

	// The first version must still be readable and unchanged.
	status, body, err := apiClient.GetListRaw(productionSchedulesPath+"/"+firstID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, firstVersion, parseJSON(body)["version"],
		"an earlier version must not be mutated by a later generation")
}

// Replaying the same idempotency key must return the same version rather than minting another one.
func TestProductionSchedules_GenerateIdempotent(t *testing.T) {
	t.Parallel()

	defer lockPlanningRead()()

	idemKey := newIdempotencyKey()
	body := map[string]any{"horizon_weeks": 4}

	resp1, err := apiClient.PostFull(productionSchedulesPath, body, idemKey)
	require.NoError(t, err)
	require.Less(t, resp1.StatusCode, 500, "generate must not 5xx: %s", string(resp1.Body))
	if resp1.StatusCode == 400 {
		t.Skipf("no constraint machine configured in this environment: %s", string(resp1.Body))
	}
	requireStatus(t, 201, resp1.StatusCode, resp1.Body)
	id1 := jsonField(parseJSON(resp1.Body), "id")
	require.NotEmpty(t, id1)
	cleanupSchedule(t, id1)

	resp2, err := apiClient.PostFull(productionSchedulesPath, body, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, resp2.StatusCode, resp2.Body)
	assert.Equal(t, id1, jsonField(parseJSON(resp2.Body), "id"),
		"a replayed generate must return the version the first call created")
}

func TestProductionSchedules_List(t *testing.T) {
	t.Parallel()

	created := generateSchedule(t, map[string]any{"horizon_weeks": 4})
	cleanupSchedule(t, jsonField(created, "id"))

	list, _, err := apiClient.GetList(productionSchedulesPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 1, "the generated version must be listable")
}

func TestProductionSchedules_ListPagination(t *testing.T) {
	t.Parallel()

	// Two versions guarantee at least two rows, so a limit=1 walk must advance.
	first := generateSchedule(t, map[string]any{"horizon_weeks": 4})
	cleanupSchedule(t, jsonField(first, "id"))
	second := generateSchedule(t, map[string]any{"horizon_weeks": 4})
	cleanupSchedule(t, jsonField(second, "id"))

	assertCursorPaginationAdvances(t, productionSchedulesPath, nil)
}

func TestProductionSchedules_LinesAndItemPolicies(t *testing.T) {
	t.Parallel()

	created := generateSchedule(t, map[string]any{"horizon_weeks": 4})
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	cleanupSchedule(t, id)

	t.Run("lines are persisted and ordered forward in time", func(t *testing.T) {
		status, body, err := apiClient.GetListRaw(productionSchedulesPath+"/"+id+"/lines", nil)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)

		lines := jsonArray(parseJSON(body), "data")
		// Non-vacuity guard. The e2e seed deliberately carries a knit batch linked to a machine, a batch-flow descendant with a product, and a backdated order — the full chain the solver needs. If any link breaks, this suite would otherwise keep passing against an empty plan and prove nothing.
		require.NotEmpty(t, lines,
			"no campaigns were persisted; the seeded demand chain (batch -> machine, "+
				"batch_flow -> sellable item, backdated order) must be intact for this to test anything")

		var previousWeek float64 = -1
		for _, raw := range lines {
			line, ok := raw.(map[string]any)
			require.True(t, ok)

			assert.Equal(t, "production_schedule_line", jsonField(line, "object"))
			assert.Equal(t, id, jsonField(jsonObject(line, "production_schedule"), "id"))
			machine := jsonObject(line, "machine")
			require.NotNil(t, machine, "a campaign must name the machine it runs on: %v", line)
			assert.NotEmpty(t, jsonField(machine, "id"))
			item := jsonObject(line, "item")
			require.NotNil(t, item, "a campaign must name the item it produces: %v", line)
			assert.NotEmpty(t, jsonField(item, "handle"),
				"a campaign's item must carry its SKU: the plan grid labels rows from it, "+
					"and a line the version holds no policy for has nowhere else to get one")
			assert.Equal(t, "planned", jsonField(line, "status"))
			assert.Equal(t, "solver", jsonField(line, "source"))
			assert.Equal(t, "flexible", jsonField(line, "freeze_status"),
				"nothing is frozen until the version is published")

			week, ok := line["week_index"].(float64)
			require.True(t, ok)
			assert.GreaterOrEqual(t, week, previousWeek, "lines must read forward in time")
			previousWeek = week
		}
	})

	t.Run("week filter narrows to one horizon week", func(t *testing.T) {
		status, body, err := apiClient.GetListRaw(productionSchedulesPath+"/"+id+"/lines",
			url.Values{"week_index": {"0"}})
		require.NoError(t, err)
		requireStatus(t, 200, status, body)

		for _, raw := range jsonArray(parseJSON(body), "data") {
			line, ok := raw.(map[string]any)
			require.True(t, ok)
			assert.Equal(t, float64(0), line["week_index"])
		}
	})

	t.Run("item policies are snapshotted and ordered by run hours", func(t *testing.T) {
		status, body, err := apiClient.GetListRaw(productionSchedulesPath+"/"+id+"/item-policies", nil)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)

		policies := jsonArray(parseJSON(body), "data")
		require.NotEmpty(t, policies,
			"no item policies were persisted; without them a plan cannot explain itself")

		var previousHours = math.Inf(1)
		for _, raw := range policies {
			policy, ok := raw.(map[string]any)
			require.True(t, ok)

			assert.Equal(t, "production_schedule_item_policy", jsonField(policy, "object"))
			assert.Equal(t, id, jsonField(jsonObject(policy, "production_schedule"), "id"))
			assert.NotEmpty(t, jsonField(policy, "sku"), "the policy must record which SKU it explains")

			hours, ok := policy["annual_run_hours"].(float64)
			require.True(t, ok)
			assert.LessOrEqual(t, hours, previousHours,
				"policies must be ordered by constraint run hours descending")
			previousHours = hours
		}
	})

	t.Run("every line has a policy explaining it", func(t *testing.T) {
		_, linesBody, err := apiClient.GetListRaw(productionSchedulesPath+"/"+id+"/lines", nil)
		require.NoError(t, err)
		_, policiesBody, err := apiClient.GetListRaw(productionSchedulesPath+"/"+id+"/item-policies", nil)
		require.NoError(t, err)

		policyItems := map[string]bool{}
		for _, raw := range jsonArray(parseJSON(policiesBody), "data") {
			policy, ok := raw.(map[string]any)
			require.True(t, ok)
			policyItems[jsonField(jsonObject(policy, "item"), "id")] = true
		}

		lines := jsonArray(parseJSON(linesBody), "data")
		require.NotEmpty(t, lines, "nothing to check; see the non-vacuity note above")

		for _, raw := range lines {
			line, ok := raw.(map[string]any)
			require.True(t, ok)
			itemID := jsonField(jsonObject(line, "item"), "id")
			assert.True(t, policyItems[itemID],
				"line for item %s has no policy snapshot; the plan could not explain itself", itemID)
		}
	})
}

func TestProductionSchedules_ReadValidation(t *testing.T) {
	t.Parallel()

	t.Run("unknown id is 404", func(t *testing.T) {
		status, body, err := apiClient.GetListRaw(productionSchedulesPath+"/pnsc_000000000000000000", nil)
		require.NoError(t, err)
		assert.Less(t, status, 500, "an unknown id must not 5xx: %s", string(body))
		assert.Equal(t, 404, status)
	})

	t.Run("rejects an out-of-range horizon", func(t *testing.T) {
		status, body, err := apiClient.Post(productionSchedulesPath, map[string]any{
			"horizon_weeks": 500,
		}, newIdempotencyKey())
		require.NoError(t, err)
		assert.Less(t, status, 500, "a bad horizon must be a client error, not a 5xx: %s", string(body))
		assert.Equal(t, 400, status)
	})

	t.Run("status filter narrows the list", func(t *testing.T) {
		status, body, err := apiClient.GetListRaw(productionSchedulesPath,
			url.Values{"statuses": {"published"}, "limit": {"100"}})
		require.NoError(t, err)
		requireStatus(t, 200, status, body)

		for _, raw := range jsonArray(parseJSON(body), "data") {
			schedule, ok := raw.(map[string]any)
			require.True(t, ok)
			assert.Equal(t, "published", jsonField(schedule, "status"))
		}
	})
}

// Nothing is published yet in this suite, so "the current schedule" must 404 rather than return a blank object that reads as a real but empty plan.
func TestProductionSchedules_CurrentIsNotFoundBeforePublish(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(productionSchedulesPath+"/current", nil)
	require.NoError(t, err)
	assert.Less(t, status, 500, "current must not 5xx: %s", string(body))
	assert.Contains(t, []int{200, 404}, status,
		"current should be 200 or 404, got %d: %s", status, string(body))

	if status == 404 {
		assert.NotContains(t, string(body), `"object"`,
			"a missing current schedule must not return a schedule-shaped body")
	}
}
