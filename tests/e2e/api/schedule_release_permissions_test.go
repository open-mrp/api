//go:build e2e

package api_test

import (
	"net/http"
	"testing"
)

// Releasing a week is an edit to the schedule. The production run it creates is part of that edit, so the schedule permission covers it and production_runs:create neither helps nor is needed.
func TestScheduleRelease_TakesOnlyTheSchedulePermission(t *testing.T) {
	t.Parallel()
	path := schedulePath("pnsc_01cpmissing000000") + "/actions/release-week"
	body := map[string]any{"week_index": 0, "responsible_user_id": SeedAccountUserID}

	status, resp := counterpartyPost(t, customRoleClient(t, "production_runs:create"), path, body)
	requirePermissionRefused(t, status, resp, "production_schedules:update")

	status, resp = counterpartyPost(t, customRoleClient(t, "production_schedules:update"), path, body)
	requireStatus(t, http.StatusNotFound, status, resp)
	requireErrorResponse(t, resp, "resource_not_found", "invalid_request_error")
}
