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
	list, _, err := apiClient.GetList(emailLogsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	assert.Len(t, list.Data, 1)
}

func TestEmailLogs_ListPagination(t *testing.T) {
	t.Parallel()
	page1, _, err := apiClient.GetList(emailLogsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	require.Len(t, page1.Data, 1)
	require.True(t, page1.PageInfo.HasNextPage, "should have a next page")
	require.NotNil(t, page1.PageInfo.NextPageURL)

	page2, _, err := apiClient.GetListFromPageURL(page1.PageInfo.NextPageURL)
	require.NoError(t, err)
	require.Len(t, page2.Data, 1)

	id1 := DataItemField(page1.Data[0], "id")
	id2 := DataItemField(page2.Data[0], "id")
	assert.NotEqual(t, id1, id2, "pages should return different items")
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
	getStatus, getBody, err := apiClient.GetListRaw(emailLogsPath+"/"+SeedEmailLogID1, url.Values{"include": {"sent_by"}})
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

	// SentBy should be a sub-resource
	sentBy := jsonObject(got, "sent_by")
	require.NotNil(t, sentBy, "sent_by should be present for seeded email log")
	assert.Equal(t, SeedUserID, jsonField(sentBy, "id"))
	assert.Equal(t, "actor", jsonField(sentBy, "object"))
	assert.Equal(t, "user", jsonField(sentBy, "type"))
}

func TestEmailLogs_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	getStatus, _, err := apiClient.GetListRaw(emailLogsPath+"/emlog_nonexistent000000", nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus)
}

func TestEmailLogs_SentByNullWithoutInclude(t *testing.T) {
	t.Parallel()
	getStatus, getBody, err := apiClient.GetListRaw(emailLogsPath+"/"+SeedEmailLogID1, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	assert.Nil(t, got["sent_by"], "sent_by should be null without ?include=sent_by")
}
