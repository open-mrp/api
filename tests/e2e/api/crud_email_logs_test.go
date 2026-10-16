//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const emailLogsPath = "/v1/core/email-logs"

func TestEmailLogs_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(emailLogsPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 2, "should have at least 2 seeded email logs")
}

func TestEmailLogs_ListWithLimit(t *testing.T) {
	t.Parallel()
	assertListPageLen(t, emailLogsPath, url.Values{"limit": {"1"}}, 1)
}

func TestEmailLogs_ListPagination(t *testing.T) {
	t.Parallel()
	// Retry-bounded two-page fetch: parallel tests can delete the rows
	// behind the cursor between fetches on this shared list.
	assertCursorPaginationAdvances(t, emailLogsPath, nil)
}

func TestEmailLogs_ListSearch(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(emailLogsPath, url.Values{"q": {"Order Confirmation"}})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1, "search for 'Order Confirmation' should return at least 1 result")

	for _, item := range list.Data {
		subject := DataItemField(item, "subject")
		assert.Contains(t, subject, "Order Confirmation")
	}
}

func TestEmailLogs_ListSearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(emailLogsPath, url.Values{"q": {"zzzznotanemail99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

func TestEmailLogs_GetByID(t *testing.T) {
	t.Parallel()
	getStatus, getBody, err := apiClient.GetListRaw(emailLogsPath+"/"+SeedEmailLogID1, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	assert.Equal(t, SeedEmailLogID1, jsonField(got, "id"))
	assert.Equal(t, "email_log", jsonField(got, "object"))
	assert.Equal(t, "sent", jsonField(got, "send_status"))
	assert.NotEmpty(t, jsonField(got, "subject"))
	assert.NotEmpty(t, jsonField(got, "created_at"))
	assert.NotEmpty(t, jsonField(got, "updated_at"))

	// Recipients should be a non-empty array (seeded email_recipient rows)
	recipients, ok := got["recipients"]
	require.True(t, ok, "recipients field should be present")
	recipientList, ok := recipients.([]any)
	require.True(t, ok, "recipients should be an array")
	assert.GreaterOrEqual(t, len(recipientList), 1, "should have at least 1 recipient")

	sentBy := jsonObject(got, "sent_by")
	require.NotNil(t, sentBy, "sent_by is always present")
	assertActorRef(t, sentBy, "user")
	assert.Equal(t, SeedUserID, jsonField(sentBy, "id"))
	assert.NotEmpty(t, jsonField(sentBy, "name"))

	// Logged before emails recorded what they carried, so the related record is empty even when included.
	related := jsonObject(got, "related")
	require.NotNil(t, related, "related is always present")
	assert.Equal(t, map[string]any{"object": "email_log_related", "record": nil}, related)
}

func TestEmailLogs_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	getStatus, _, err := apiClient.GetListRaw(emailLogsPath+"/emlog_nonexistent000000", nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus)
}

// sent_by is always present now, so it is no longer an include.
func TestEmailLogs_SentByIsNotAnInclude(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(emailLogsPath+"/"+SeedEmailLogID1, url.Values{"include": {"sent_by"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

func TestEmailLogs_ListFilterBySendStatuses(t *testing.T) {
	t.Parallel()
	sent := emailLogIDs(t, url.Values{"send_statuses": {"sent"}, "q": {"Order Confirmation - ORD-001"}})
	assert.True(t, sent[SeedEmailLogID1], "a sent email matches send_statuses=sent")

	pending := emailLogIDs(t, url.Values{"send_statuses": {"pending"}, "q": {"Order Confirmation - ORD-001"}})
	assert.False(t, pending[SeedEmailLogID1], "a sent email is not pending")

	status, body, err := apiClient.GetListRaw(emailLogsPath, url.Values{"send_statuses": {"bounced"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

func TestEmailLogs_ListFilterByCreatedRange(t *testing.T) {
	t.Parallel()
	q := url.Values{"q": {"Order Confirmation - ORD-001"}}
	assert.True(t, emailLogIDs(t, withParams(q, "created_after", "2000-01-01T00:00:00Z"))[SeedEmailLogID1])
	assert.False(t, emailLogIDs(t, withParams(q, "created_after", "2099-01-01T00:00:00Z"))[SeedEmailLogID1])
	assert.False(t, emailLogIDs(t, withParams(q, "created_before", "2000-01-01T00:00:00Z"))[SeedEmailLogID1])

	status, body, err := apiClient.GetListRaw(emailLogsPath, url.Values{
		"created_after": {"2026-02-01T00:00:00Z"}, "created_before": {"2026-01-01T00:00:00Z"},
	})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

func TestEmailLogs_ListFilterByRelatedIDsImpossible(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(emailLogsPath, url.Values{"related_ids": {"inv_zzzzzzzzzzzzzzzz"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "no email carried a record that doesn't exist")
}

// emailLogIDs is the set of email log IDs a list returns for params.
func emailLogIDs(t *testing.T, params url.Values) map[string]bool {
	t.Helper()
	params.Set("limit", "100")
	list, status, err := apiClient.GetList(emailLogsPath, params)
	require.NoError(t, err)
	require.Equal(t, 200, status)
	ids := map[string]bool{}
	for _, raw := range list.Data {
		ids[DataItemField(raw, "id")] = true
	}
	return ids
}

// withParams copies params with key set to value.
func withParams(params url.Values, key, value string) url.Values {
	out := url.Values{}
	for k, v := range params {
		out[k] = append([]string(nil), v...)
	}
	out.Set(key, value)
	return out
}

// assertEmailLogRelated checks an email log fetched with related.record carried the given record.
func assertEmailLogRelated(t *testing.T, emailLog map[string]any, recordType, recordID string) {
	t.Helper()
	related := jsonObject(emailLog, "related")
	require.NotNil(t, related, "related is always present: %v", emailLog)
	assert.Equal(t, "email_log_related", jsonField(related, "object"))
	record := jsonObject(related, "record")
	require.NotNil(t, record, "related.record is set when included: %v", emailLog)
	assert.Equal(t, "record", jsonField(record, "object"))
	assert.Equal(t, recordType, jsonField(record, "type"))
	assert.Equal(t, recordID, jsonField(record, "id"))
}
