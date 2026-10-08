//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Production schedule settings pinned to 1.0.forge-preview.5, which typed changeover_labor_rate as a number. A caller without costs:read reads it as null on preview.6 and as 0 on preview.5; a caller with costs:read reads the rate on both.

func TestVersionCompat_ScheduleSettings_Retrieve(t *testing.T) {
	t.Parallel()
	// Both versions have to read the same stored settings, so no settings write may land between them.
	defer lockPlanningRead()()

	t.Run("without costs:read preview.5 reads 0", func(t *testing.T) {
		planner := customRoleClient(t, "production_schedules:read")
		latest := parseJSON(mustGetAs(t, planner, scheduleSettingsPath, nil))
		old := parseJSON(mustGetAs(t, planner.WithAPIVersion(preview5APIVersion), scheduleSettingsPath, nil))

		assertNilField(t, latest, "changeover_labor_rate")
		assert.Equal(t, float64(0), old["changeover_labor_rate"], "preview.5 typed the rate as a number")
		assert.Equal(t, latest["holding_rate_pct"], old["holding_rate_pct"], "the rest of the settings are the same on both")
	})

	t.Run("with costs:read both versions read the rate", func(t *testing.T) {
		reader := customRoleClient(t, "production_schedules:read", costsRead)
		latest := parseJSON(mustGetAs(t, reader, scheduleSettingsPath, nil))
		old := parseJSON(mustGetAs(t, reader.WithAPIVersion(preview5APIVersion), scheduleSettingsPath, nil))

		require.NotNil(t, latest["changeover_labor_rate"], "a costs:read holder reads the rate")
		assert.Equal(t, latest["changeover_labor_rate"], old["changeover_labor_rate"])
	})
}

// The save answers with the settings it stored, in the caller's version.
func TestVersionCompat_ScheduleSettings_Update(t *testing.T) {
	// Not parallel: rewrites the account-wide schedule settings.
	original := claimScheduleSettings(t)
	rate := original["changeover_labor_rate"]
	require.NotNil(t, rate, "the admin reads the changeover labor rate")
	body := settingsWriteBody(original)
	save := func(client *Client) map[string]any {
		t.Helper()
		status, raw, err := client.Put(scheduleSettingsPath, body)
		require.NoError(t, err)
		requireStatus(t, 200, status, raw)
		return parseJSON(raw)
	}

	t.Run("without costs:read preview.5 reads 0", func(t *testing.T) {
		planner := customRoleClient(t, "production_schedules:read", "production_schedules:update")
		assertNilField(t, save(planner), "changeover_labor_rate")
		assert.Equal(t, float64(0), save(planner.WithAPIVersion(preview5APIVersion))["changeover_labor_rate"], "preview.5 typed the rate as a number")
	})

	t.Run("with costs:read both versions read the rate", func(t *testing.T) {
		planner := customRoleClient(t, "production_schedules:read", "production_schedules:update", costsRead)
		assert.Equal(t, rate, save(planner)["changeover_labor_rate"])
		assert.Equal(t, rate, save(planner.WithAPIVersion(preview5APIVersion))["changeover_labor_rate"])
	})
}
