//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCovCoreEmailLogs_GetByID_FilenameNullAndIDFormat closes the last unasserted
// response-struct field gap (filename, always null in seed data) and adds the
// missing emlog_ ID-prefix-format assertion for the GetEmailLog endpoint.
func TestCovCoreEmailLogs_GetByID_FilenameNullAndIDFormat(t *testing.T) {
	t.Parallel()
	getStatus, getBody, err := apiClient.GetListRaw(emailLogsPath+"/"+SeedEmailLogID1, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	assertIDFormat(t, jsonField(got, "id"), "emlog")
	assertObjectField(t, got, "email_log")
	assertNilField(t, got, "filename")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
}

// TestCovCoreEmailLogs_ListSearch_RecipientEmail proves the email_recipient EXISTS
// search branch works (not just subject substring search).
func TestCovCoreEmailLogs_ListSearch_RecipientEmail(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(emailLogsPath, url.Values{"q": {"warehouse@example.com"}})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1, "search by recipient email should return at least 1 result")

	found := false
	for _, item := range list.Data {
		if DataItemField(item, "id") == SeedEmailLogID2 {
			found = true
			assert.Contains(t, DataItemField(item, "subject"), "Shipment Notification")
		}
	}
	assert.True(t, found, "search q=warehouse@example.com should surface seeded log %s", SeedEmailLogID2)
}

// TestCovCoreEmailLogs_List_IncludeSentBy pairs with the existing
// TestIncludeIsolation_ListEndpointsReturnNullSubobjectsWithoutInclude negative
// case: with ?include=sent_by, list items should have a populated sent_by Actor
// whose type reflects the true sender. Emails sent by the seeded user resolve to
// a {us_, user} actor; emails the API sends on the caller's behalf (e.g. the
// account-user welcome email triggered under the seeded API key) resolve to an
// {apky_, api_key} actor. The sent_by type must match the sender's kind — it must
// never be blindly reported as "user".
func TestCovCoreEmailLogs_List_IncludeSentBy(t *testing.T) {
	t.Parallel()
	// Follow all pages so both seeded user-sent logs are inspected regardless of
	// how many runtime rows (with other senders) sort ahead of them.
	list, _, err := apiClient.GetList(emailLogsPath, url.Values{"include": {"sent_by"}})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 2, "should have at least 2 seeded email logs")

	validActorTypes := map[string]bool{"user": true, "api_key": true, "agent": true, "group": true}

	foundPopulated := false
	sawSeedUser := false
	for _, item := range list.Data {
		parsed := parseJSON(item)
		sentBy := jsonObject(parsed, "sent_by")
		if sentBy == nil {
			continue
		}
		foundPopulated = true

		actorID := jsonField(sentBy, "id")
		actorType := jsonField(sentBy, "type")
		assert.Equal(t, "actor", jsonField(sentBy, "object"))
		assert.True(t, validActorTypes[actorType], "sent_by.type %q is not a valid actor type (id %s)", actorType, actorID)

		// The actor's declared type must be consistent with its id kind, not
		// hardcoded — a user id is a user, an API-key id is an api_key.
		switch {
		case actorID == SeedUserID:
			sawSeedUser = true
			assert.Equal(t, "user", actorType, "the seeded user sender must resolve to a user actor")
		case actorID == SeedAPIKeyID:
			assert.Equal(t, "api_key", actorType, "an API-key sender must resolve to an api_key actor, not a user")
		}
	}
	assert.True(t, foundPopulated, "at least one email log in the list should have a populated sent_by with ?include=sent_by")
	assert.True(t, sawSeedUser, "the seeded user-sent email logs should expose a populated sent_by actor of type user")
}

// TestCovCoreEmailLogs_InvalidLimit_Zero validates limit=0 is rejected.
func TestCovCoreEmailLogs_InvalidLimit_Zero(t *testing.T) {
	t.Parallel()
	statusCode, body, err := apiClient.GetListRaw(emailLogsPath, url.Values{"limit": {"0"}})
	require.NoError(t, err)
	assert.Equal(t, 400, statusCode, "GET %s?limit=0 should return 400, got %d: %s", emailLogsPath, statusCode, string(body))
	if statusCode == 400 {
		requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	}
}

// TestCovCoreEmailLogs_InvalidLimit_Negative validates limit=-1 is rejected.
func TestCovCoreEmailLogs_InvalidLimit_Negative(t *testing.T) {
	t.Parallel()
	statusCode, body, err := apiClient.GetListRaw(emailLogsPath, url.Values{"limit": {"-1"}})
	require.NoError(t, err)
	assert.Equal(t, 400, statusCode, "GET %s?limit=-1 should return 400, got %d: %s", emailLogsPath, statusCode, string(body))
	if statusCode == 400 {
		requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	}
}

// TestCovCoreEmailLogs_InvalidLimit_TooLarge validates limit=1001 is rejected
// (validate tag min=1,max=1000).
func TestCovCoreEmailLogs_InvalidLimit_TooLarge(t *testing.T) {
	t.Parallel()
	statusCode, body, err := apiClient.GetListRaw(emailLogsPath, url.Values{"limit": {"1001"}})
	require.NoError(t, err)
	assert.Equal(t, 400, statusCode, "GET %s?limit=1001 should return 400, got %d: %s", emailLogsPath, statusCode, string(body))
	if statusCode == 400 {
		requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	}
}

// TestCovCoreEmailLogs_InvalidCursor validates a malformed/opaque cursor value
// is rejected with 400, not silently accepted as an empty page.
func TestCovCoreEmailLogs_InvalidCursor(t *testing.T) {
	t.Parallel()
	statusCode, body, err := apiClient.GetListRaw(emailLogsPath, url.Values{"cursor": {"not-a-real-cursor"}})
	require.NoError(t, err)
	require.NotEqual(t, 200, statusCode, "GET %s?cursor=not-a-real-cursor should not silently succeed", emailLogsPath)
	assert.Equal(t, 400, statusCode, "GET %s?cursor=not-a-real-cursor should return 400, got %d: %s", emailLogsPath, statusCode, string(body))
	if statusCode == 400 {
		requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	}
}

// TestCovCoreEmailLogs_QueryTooLong validates q over 500 chars is rejected.
func TestCovCoreEmailLogs_QueryTooLong(t *testing.T) {
	t.Parallel()
	tooLongQuery := strings.Repeat("a", 501)
	statusCode, body, err := apiClient.GetListRaw(emailLogsPath, url.Values{"q": {tooLongQuery}})
	require.NoError(t, err)
	assert.Equal(t, 400, statusCode, "GET %s?q=<501 chars> should return 400, got %d: %s", emailLogsPath, statusCode, string(body))
	if statusCode == 400 {
		requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	}
}

// TestCovCoreEmailLogs_InvalidInclude_List validates an unknown ?include value
// on the list endpoint is rejected with 400.
func TestCovCoreEmailLogs_InvalidInclude_List(t *testing.T) {
	t.Parallel()
	statusCode, body, err := apiClient.GetListRaw(emailLogsPath, url.Values{"include": {"bogus_field"}})
	require.NoError(t, err)
	assert.Equal(t, 400, statusCode, "GET %s?include=bogus_field should return 400, got %d: %s", emailLogsPath, statusCode, string(body))
	if statusCode == 400 {
		requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	}
}

// TestCovCoreEmailLogs_InvalidInclude_Get validates an unknown ?include value
// on the get endpoint is rejected with 400.
func TestCovCoreEmailLogs_InvalidInclude_Get(t *testing.T) {
	t.Parallel()
	statusCode, body, err := apiClient.GetListRaw(emailLogsPath+"/"+SeedEmailLogID1, url.Values{"include": {"bogus_field"}})
	require.NoError(t, err)
	assert.Equal(t, 400, statusCode, "GET %s/%s?include=bogus_field should return 400, got %d: %s", emailLogsPath, SeedEmailLogID1, statusCode, string(body))
	if statusCode == 400 {
		requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	}
}

// TestCovCoreEmailLogs_UnknownQueryParam_List validates an unregistered query
// parameter on the list endpoint is rejected with 400.
func TestCovCoreEmailLogs_UnknownQueryParam_List(t *testing.T) {
	t.Parallel()
	statusCode, body, err := apiClient.GetListRaw(emailLogsPath, url.Values{bogusE2EQueryParam: {"1"}})
	require.NoError(t, err)
	assertUnknownQueryParamRejected(t, emailLogsPath, statusCode, body)
}

// TestCovCoreEmailLogs_UnknownQueryParam_Get validates an unregistered query
// parameter on the get endpoint is rejected with 400.
func TestCovCoreEmailLogs_UnknownQueryParam_Get(t *testing.T) {
	t.Parallel()
	path := emailLogsPath + "/" + SeedEmailLogID1
	statusCode, body, err := apiClient.GetListRaw(path, url.Values{bogusE2EQueryParam: {"1"}})
	require.NoError(t, err)
	assertUnknownQueryParamRejected(t, path, statusCode, body)
}
