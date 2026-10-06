//go:build e2e

package api_test

import (
	"net/http"
	"testing"
)

// Releasing a week edits the schedule and creates a production run, so it takes both permissions: either one alone is refused, naming the other.
func TestScheduleRelease_TakesTheScheduleAndRunPermissionsTogether(t *testing.T) {
	t.Parallel()
	path := schedulePath("pnsc_01cpmissing000000") + "/actions/release-week"
	body := map[string]any{"week_index": 0, "responsible_user_id": SeedAccountUserID}

	status, resp := counterpartyPost(t, customRoleClient(t, "production_schedules:update"), path, body)
	requirePermissionRefused(t, status, resp, "production_runs:create")
	status, resp = counterpartyPost(t, customRoleClient(t, "production_runs:create"), path, body)
	requirePermissionRefused(t, status, resp, "production_schedules:update")

	status, resp = counterpartyPost(t, customRoleClient(t, "production_schedules:update", "production_runs:create"), path, body)
	requireStatus(t, http.StatusNotFound, status, resp)
	requireErrorResponse(t, resp, "resource_not_found", "invalid_request_error")
}
