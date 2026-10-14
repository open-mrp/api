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
