//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	scheduleSettingsPath         = "/v1/operations/production-schedule-settings"
	scheduleResourceSettingsPath = scheduleSettingsPath + "/resources"
)

// readScheduleSettings returns the account's current planning assumptions.
func readScheduleSettings(t *testing.T) map[string]any {
	t.Helper()

	resp, err := apiClient.GetFull(scheduleSettingsPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	return parseJSON(resp.Body)
}

// writeScheduleSettings replaces the account's settings with `body`.
func writeScheduleSettings(t *testing.T, body map[string]any) map[string]any {
	t.Helper()

	resp, err := apiClient.PutFull(scheduleSettingsPath, body)
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	return parseJSON(resp.Body)
}

// settingsWriteBody turns a settings resource into the update request that reproduces it.
//
// The update is a whole-object replace, so a test that changes one assumption has to send back every other one. Deriving the body from a prior read is what lets the restore in claimScheduleSettings put the account back exactly as it was found.
func settingsWriteBody(current map[string]any) map[string]any {
	body := map[string]any{}
	// The response carries the constraint department as a sub-object; the request still takes its id.
	if dept := jsonObject(current, "constraint_department"); dept != nil {
		body["constraint_department_id"] = jsonField(dept, "id")
	}
	for _, key := range []string{
		"planning_horizon_weeks", "frozen_weeks", "week_start_day",
		"demand_window_months", "forecast_history_months", "forecast_months",
		"demand_basis", "forecast_z",
		"changeover_avg_minutes", "changeover_min_minutes", "changeover_max_minutes", "changeover_labor_rate",
		"holding_rate_pct", "service_level_z", "finish_lead_time_weeks",
		"default_constraint_lead_time_weeks", "max_weeks_supply", "max_flow_depth",
		"shifts_per_day", "hours_per_shift", "work_days_per_week", "weeks_per_year",
		"capacity_headroom_pct", "default_lot_units",
		"cadence_status", "generation_cron", "generation_timezone", "auto_publish_status",
	} {
		if value, ok := current[key]; ok && value != nil {
			body[key] = value
		}
	}
	return body
}

// claimScheduleSettings takes the account-wide settings write slot and restores whatever was there when the test finishes.
//
// Settings are singular per account and every solve reads them, so a test that changes the horizon while another test is generating would silently move that other test's plan. planningMu (see generateSchedule) is what keeps the two off the same clock.
func claimScheduleSettings(t *testing.T) map[string]any {
	t.Helper()

	planningMu.Lock()
	t.Cleanup(planningMu.Unlock)

	original := readScheduleSettings(t)
	restore := settingsWriteBody(original)
	t.Cleanup(func() {
		status, body, err := apiClient.Put(scheduleSettingsPath, restore)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
	})
	return original
}

// ──────────────────────────────────────────────
// Settings
// ──────────────────────────────────────────────

func TestScheduleSettings_ReadReturnsFullyPopulatedAssumptions(t *testing.T) {
	t.Parallel()

	settings := readScheduleSettings(t)

	assert.Equal(t, "production_schedule_settings", jsonField(settings, "object"))

	// An account that has never saved settings still gets a complete, usable set — the point of the endpoint is that a caller never has to know the solver's defaults.
	assert.Contains(t, []string{"stored", "default"}, jsonField(settings, "settings_status"))
	for _, key := range []string{
		"planning_horizon_weeks", "demand_window_months", "forecast_history_months",
		"forecast_months", "max_weeks_supply", "shifts_per_day", "hours_per_shift",
		"work_days_per_week", "weeks_per_year", "capacity_headroom_pct", "default_lot_units",
	} {
		value, ok := settings[key].(float64)
		require.True(t, ok, "%s must be present and numeric", key)
		assert.Greater(t, value, float64(0), "%s must never be advertised as zero", key)
	}

	assert.NotEmpty(t, jsonField(settings, "demand_basis"))
	assert.NotEmpty(t, jsonField(settings, "generation_timezone"))
	assert.Contains(t, []string{"active", "inactive"}, jsonField(settings, "cadence_status"))
	assert.Contains(t, []string{"active", "inactive"}, jsonField(settings, "auto_publish_status"))
}

func TestScheduleSettings_UpdatePersistsAndFlipsToStored(t *testing.T) {
	original := claimScheduleSettings(t)

	body := settingsWriteBody(original)
	body["planning_horizon_weeks"] = 9
	body["capacity_headroom_pct"] = 0.75
	body["default_lot_units"] = 48

	updated := writeScheduleSettings(t, body)

	assert.EqualValues(t, 9, updated["planning_horizon_weeks"])
	assert.EqualValues(t, 0.75, updated["capacity_headroom_pct"])
	assert.EqualValues(t, 48, updated["default_lot_units"])
	assert.Equal(t, "stored", jsonField(updated, "settings_status"),
		"once a merchant saves, the resource must stop reporting solver defaults")

	// The write is durable, not just echoed back.
	reread := readScheduleSettings(t)
	assert.EqualValues(t, 9, reread["planning_horizon_weeks"])
	assert.Equal(t, "stored", jsonField(reread, "settings_status"))
}

func TestScheduleSettings_UpdateRejectsUnworkableAssumptions(t *testing.T) {
	original := claimScheduleSettings(t)

	cases := []struct {
		name  string
		field string
		value any
	}{
		{"zero hour shifts", "hours_per_shift", 0},
		{"no shifts", "shifts_per_day", 0},
		{"headroom above one", "capacity_headroom_pct", 1.5},
		{"headroom of zero", "capacity_headroom_pct", 0},
		{"zero weeks of supply", "max_weeks_supply", 0},
		{"empty horizon", "planning_horizon_weeks", 0},
		{"eight day week", "work_days_per_week", 8},
		{"unknown demand basis", "demand_basis", "vibes"},
		{"unknown cadence status", "cadence_status", "sometimes"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := settingsWriteBody(original)
			body[tc.field] = tc.value

			status, respBody, err := apiClient.Put(scheduleSettingsPath, body)
			require.NoError(t, err)
			require.Less(t, status, 500, "a bad assumption must be a 400, not a crash: %s", string(respBody))
			assert.Equal(t, 400, status, "%s must be rejected: %s", tc.field, string(respBody))
		})
	}
}

func TestScheduleSettings_CadenceRequiresParseableCron(t *testing.T) {
	original := claimScheduleSettings(t)

	// A cadence that cannot be parsed would simply never fire, and the merchant would have no way to tell that from "nothing was due yet".
	body := settingsWriteBody(original)
	body["cadence_status"] = "active"
	body["generation_cron"] = "every other tuesday"

	status, respBody, err := apiClient.Put(scheduleSettingsPath, body)
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(respBody))
	assert.Equal(t, 400, status, "an unparseable cron must be rejected: %s", string(respBody))

	// A real expression turns the cadence on.
	valid := settingsWriteBody(original)
	valid["cadence_status"] = "active"
	valid["generation_cron"] = "0 6 * * 1"
	accepted := writeScheduleSettings(t, valid)
	assert.Equal(t, "active", jsonField(accepted, "cadence_status"))
	assert.Equal(t, "0 6 * * 1", jsonField(accepted, "generation_cron"))
}

// ──────────────────────────────────────────────
// Resource settings
// ──────────────────────────────────────────────

func upsertResourceSetting(t *testing.T, body map[string]any) map[string]any {
	t.Helper()

	resp, err := apiClient.PutFull(scheduleResourceSettingsPath, body)
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	return parseJSON(resp.Body)
}

func listResourceSettings(t *testing.T) []map[string]any {
	t.Helper()

	status, body, err := apiClient.GetListRaw(scheduleResourceSettingsPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	out := []map[string]any{}
	for _, raw := range jsonArray(parseJSON(body), "data") {
		if row, ok := raw.(map[string]any); ok {
			out = append(out, row)
		}
	}
	return out
}

func TestScheduleResourceSettings_ListReturnsConstraintMachines(t *testing.T) {
	t.Parallel()

	settings := listResourceSettings(t)

	for _, setting := range settings {
		assert.Equal(t, "production_schedule_resource_setting", jsonField(setting, "object"))
		assert.Contains(t, []string{"machine", "department", "production_step"}, jsonField(setting, "scope_type"))
		assert.Contains(t, []string{"included", "excluded"}, jsonField(setting, "participation_status"))
		scope := jsonObject(setting, "scope")
		require.NotNil(t, scope, "every override must name the resource it overrides: %v", setting)
		assert.Equal(t, "entity", jsonField(scope, "object"))
		assert.NotEmpty(t, jsonField(scope, "id"))
	}
}

func TestScheduleResourceSettings_UpsertReplacesRatherThanDuplicates(t *testing.T) {
	// Not parallel: resource settings decide which machines the solver plans on, so changing one mid-solve would move another test's plan.
	planningMu.Lock()
	t.Cleanup(planningMu.Unlock)

	before := listResourceSettings(t)
	countFor := func(rows []map[string]any, refID string) int {
		n := 0
		for _, row := range rows {
			if jsonField(jsonObject(row, "scope"), "id") == refID && jsonField(row, "scope_type") == "production_step" {
				n++
			}
		}
		return n
	}

	// A production step is used rather than a machine so the upsert cannot accidentally add or remove a constraint machine and change what any other test's plan contains.
	body := map[string]any{
		"scope_type":             "production_step",
		"scope_ref_id":           SeedProductionStepID,
		"participation_status":   "excluded",
		"lead_time_offset_weeks": 2,
	}

	created := upsertResourceSetting(t, body)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() {
		status, respBody, err := apiClient.Delete(scheduleResourceSettingsPath + "/" + id)
		require.NoError(t, err)
		if status != 200 && status != 204 && status != 404 {
			t.Fatalf("cleanup delete %s returned %d: %s", id, status, string(respBody))
		}
	})

	assert.EqualValues(t, 2, created["lead_time_offset_weeks"])

	// Writing the same scope again must overwrite, not accumulate: the solver resolves one override per resource, and a second row would make which one wins arbitrary.
	body["lead_time_offset_weeks"] = 4
	updated := upsertResourceSetting(t, body)
	assert.Equal(t, id, jsonField(updated, "id"), "the same scope must keep its identity")
	assert.EqualValues(t, 4, updated["lead_time_offset_weeks"])

	after := listResourceSettings(t)
	assert.Equal(t, countFor(before, SeedProductionStepID)+1, countFor(after, SeedProductionStepID),
		"upserting twice must leave exactly one row for the scope")
}

func TestScheduleResourceSettings_DeleteReturnsResourceToDefaults(t *testing.T) {
	planningMu.Lock()
	t.Cleanup(planningMu.Unlock)

	created := upsertResourceSetting(t, map[string]any{
		"scope_type":             "production_step",
		"scope_ref_id":           SeedProductionStepID,
		"participation_status":   "excluded",
		"lead_time_offset_weeks": 3,
	})
	id := jsonField(created, "id")
	require.NotEmpty(t, id)

	status, body, err := apiClient.Delete(scheduleResourceSettingsPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	for _, row := range listResourceSettings(t) {
		assert.NotEqual(t, id, jsonField(row, "id"), "a deleted override must stop being returned")
	}

	// Deleting again is a 404, not a crash.
	status, body, err = apiClient.Delete(scheduleResourceSettingsPath + "/" + id)
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 404, status)
}

func TestScheduleResourceSettings_UpsertRejectsUnknownScope(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Put(scheduleResourceSettingsPath, map[string]any{
		"scope_type":           "spaceship",
		"scope_ref_id":         SeedProductionStepID,
		"participation_status": "excluded",
	})
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 400, status)
}
