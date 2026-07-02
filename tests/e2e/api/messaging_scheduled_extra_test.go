//go:build e2e

package api_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Gap coverage for scheduled messages: input validation (empty body, bad time
// format), canceling an unknown id, and that a canceled message drops out of the schedule.

func TestScheduledMessages_ValidationErrors(t *testing.T) {
	t.Parallel()
	dane := chatUserClient(t)
	convID := jsonField(createDM(t, dane, SeedAccountUser2ID), "id")
	future := time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339)

	cases := []struct {
		name string
		body map[string]any
	}{
		{"empty-body", map[string]any{"body": "", "client_message_id": uniqueName("cmid"), "scheduled_at": future}},
		{"bad-time-format", map[string]any{"body": uniqueName("x"), "client_message_id": uniqueName("cmid"), "scheduled_at": "not-a-timestamp"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body, err := dane.Post(conversationsPath+"/"+convID+"/messages", tc.body, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 400, status, body)
		})
	}
}

func TestScheduledMessages_CancelUnknownIs404(t *testing.T) {
	t.Parallel()
	dane := chatUserClient(t)
	status, body, err := dane.Post("/v1/messaging/messages/mg_doesnotexist00000/actions/cancel", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, "canceling an unknown scheduled message should 404: %s", string(body))
}

func TestScheduledMessages_CanceledDropsFromSchedule(t *testing.T) {
	t.Parallel()
	dane := chatUserClient(t)
	convID := jsonField(createDM(t, dane, SeedAccountUser2ID), "id")

	resp, err := dane.PostFull(conversationsPath+"/"+convID+"/messages", map[string]any{
		"body":              uniqueName("cancel me"),
		"client_message_id": uniqueName("cmid"),
		"scheduled_at":      time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	id := jsonField(parseJSON(resp.Body), "id")

	// It is listed while scheduled.
	scheduled, _, err := dane.GetList(conversationsPath+"/"+convID+"/messages", url.Values{"status": {"scheduled"}})
	require.NoError(t, err)
	listed := false
	for _, raw := range scheduled.Data {
		if jsonField(parseJSON(raw), "id") == id {
			listed = true
		}
	}
	assert.True(t, listed, "a scheduled message is listed before cancellation")

	// Cancel it.
	cancelResp, err := dane.PostFull("/v1/messaging/messages/"+id+"/actions/cancel", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, cancelResp.StatusCode, cancelResp.Body)
	assert.Equal(t, "canceled", jsonField(parseJSON(cancelResp.Body), "status"))

	// A canceled message is no longer scheduled, so it drops out of the scheduled list.
	after, _, err := dane.GetList(conversationsPath+"/"+convID+"/messages", url.Values{"status": {"scheduled"}})
	require.NoError(t, err)
	for _, raw := range after.Data {
		assert.NotEqual(t, id, jsonField(parseJSON(raw), "id"), "a canceled message must not remain in the schedule")
	}
}
