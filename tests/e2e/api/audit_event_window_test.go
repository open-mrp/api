//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SeedAuditEventInfraAgentID was recorded in 2022 against request rqlog_01infraagent0, so a search for
// that request finds it only when the list reaches back that far.
const infraAuditRequestID = "rqlog_01infraagent0"

func auditEventIDsAs(t *testing.T, client *Client, params url.Values) map[string]bool {
	t.Helper()
	params.Set("limit", "100")
	list, status, err := client.GetList(auditEventsPath, params)
	require.NoError(t, err)
	require.Equal(t, 200, status)
	ids := map[string]bool{}
	for _, raw := range list.Data {
		ids[DataItemField(raw, "id")] = true
	}
	return ids
}

func TestAuditEvents_OmittedStartCoversTheLastDay(t *testing.T) {
	t.Parallel()

	recent := auditEventIDsAs(t, apiClient, url.Values{"q": {infraAuditRequestID}})
	assert.False(t, recent[SeedAuditEventInfraAgentID], "an event from 2022 is outside the default window")

	all := auditEventIDsAs(t, apiClient, url.Values{"q": {infraAuditRequestID}, "starts_at": {"2000-01-01T00:00:00Z"}})
	assert.True(t, all[SeedAuditEventInfraAgentID], "an explicit start reaches it")
}

// A record's timeline is its whole history, so naming the record lifts the window.
func TestAuditEvents_RecordHistoryIsWhole(t *testing.T) {
	t.Parallel()

	history := auditEventIDsAs(t, apiClient, url.Values{"resource_ids": {"ac_01seedcustomer2_acct0"}})
	assert.True(t, history[SeedAuditEventInfraAgentID], "a record's 2022 event stays in its history")
}

// preview.4 searched the whole history when starts_at was omitted.
func TestVersionCompat_AuditEvents_OmittedStartSearchesEverything(t *testing.T) {
	t.Parallel()

	ids := auditEventIDsAs(t, apiClient.WithAPIVersion(preview4APIVersion), url.Values{"q": {infraAuditRequestID}})
	assert.True(t, ids[SeedAuditEventInfraAgentID], "preview.4 still reaches events years old without a start")
}

func TestVersionCompat_InventoryChangeLogs_EndAloneReachesBackToTheStart(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.WithAPIVersion(preview4APIVersion).GetList(inventoryChangeLogsPath, url.Values{
		"ends_at": {"2100-04-10T00:00:00Z"},
		"limit":   {"100"},
	})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	found := false
	for _, raw := range list.Data {
		found = found || DataItemField(raw, "id") == SeedInventoryChangeLogID
	}
	assert.True(t, found, "preview.4 reads an end alone as everything up to it")
}
