//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A schedule's settings snapshot freezes the planning assumptions it was solved with, the changeover labor rate among them. That rate is cost data like the settings' own field, so it follows costs:read inside the snapshot too.

func TestCosts_ScheduleSettingsSnapshotLaborRateNeedsCostsRead(t *testing.T) {
	t.Parallel()
	id := jsonField(ownedSchedule(t, uniqueName("e2e-costs-snapshot")), "id")
	path := schedulePath(id)

	snapshotOf := func(client *Client) map[string]any {
		snapshot := jsonObject(parseJSON(mustGetAs(t, client, path, nil)), "settings_snapshot")
		require.NotNil(t, snapshot, "the schedule carries its settings snapshot")
		return snapshot
	}

	rate := snapshotOf(apiClient)["changeover_labor_rate"]
	require.NotNil(t, rate, "the admin reads the snapshot's changeover labor rate")
	assert.Equal(t, rate, snapshotOf(customRoleClient(t, "production_schedules:read", costsRead))["changeover_labor_rate"])

	reader := customRoleClient(t, "production_schedules:read")
	snapshot := snapshotOf(reader)
	require.Contains(t, snapshot, "changeover_labor_rate", "the key stays, so the snapshot keeps its shape")
	assert.Nil(t, snapshot["changeover_labor_rate"], "a reader without costs:read does not see the rate")
	assert.NotNil(t, snapshot["hours_per_shift"], "the rest of the snapshot is not cost data")

	list := parseJSON(mustGetAs(t, reader, productionSchedulesPath, url.Values{"limit": {"100"}}))
	found := false
	for _, row := range listRows(t, list) {
		listed := jsonObject(row, "settings_snapshot")
		if rate, ok := listed["changeover_labor_rate"]; ok {
			assert.Nil(t, rate, "listed schedule %s shows its labor rate to a reader without costs:read", jsonField(row, "id"))
		}
		found = found || jsonField(row, "id") == id
	}
	assert.True(t, found, "the schedule is on the reader's first page")
}
