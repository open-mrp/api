//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Scheduled-message coverage: a message is the single resource for sent, scheduled, and draft
// content — scheduling is a send with a future scheduled_at (status "scheduled"), listed via the
// conversation messages endpoint with status=scheduled, canceled via the message cancel action, and
// materialized into a sent timeline message by the lease-guarded worker (the e2e worker polls every ~2s).

func TestScheduledMessages_ScheduleListCancel(t *testing.T) {
	t.Parallel()
	dane := chatUserClient(t)
	dm := createDM(t, dane, SeedAccountUser2ID)
	convID := jsonField(dm, "id")

	body := uniqueName("scheduled note")
	resp, err := dane.PostFull(withQuery(conversationsPath+"/"+convID+"/messages", url.Values{"include": {"conversation"}}), map[string]any{
		"body":              body,
		"client_message_id": uniqueName("cmid"),
		"scheduled_at":      time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	sm := parseJSON(resp.Body)
	assert.Equal(t, "chat_message", jsonField(sm, "object"))
	assert.Equal(t, "scheduled", jsonField(sm, "status"))
	id := jsonField(sm, "id")
	assertIDFormat(t, id, "mg")
	conv, _ := sm["conversation"].(map[string]any)
	require.NotNil(t, conv)
	assert.Equal(t, convID, jsonField(conv, "id"))

	// It appears in the conversation's scheduled-message list.
	list, _, err := dane.GetList(conversationsPath+"/"+convID+"/messages", url.Values{"status": {"scheduled"}})
	require.NoError(t, err)
	found := false
	for _, raw := range list.Data {
		if jsonField(parseJSON(raw), "id") == id {
			found = true
		}
	}
	assert.True(t, found, "the scheduled message is listed while scheduled")

	// Cancel it.
	cancelResp, err := dane.PostFull("/v1/messaging/messages/"+id+"/actions/cancel", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, cancelResp.StatusCode, cancelResp.Body)
	assert.Equal(t, "canceled", jsonField(parseJSON(cancelResp.Body), "status"))

	// Canceling again is no longer allowed (not scheduled).
	cancelResp2, err := dane.PostFull("/v1/messaging/messages/"+id+"/actions/cancel", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, cancelResp2.StatusCode, cancelResp2.Body)
}

func TestScheduledMessages_PastTimeRejected(t *testing.T) {
	t.Parallel()
	dane := chatUserClient(t)
	dm := createDM(t, dane, SeedAccountUser2ID)
	convID := jsonField(dm, "id")

	resp, err := dane.PostFull(conversationsPath+"/"+convID+"/messages", map[string]any{
		"body":              uniqueName("too late"),
		"client_message_id": uniqueName("cmid"),
		"scheduled_at":      time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, resp.StatusCode, resp.Body)
}

func TestScheduledMessages_DeliveredByWorker(t *testing.T) {
	t.Parallel()
	dane := chatUserClient(t)
	dm := createDM(t, dane, SeedAccountUser2ID)
	convID := jsonField(dm, "id")

	body := uniqueName("deliver me")
	// A few seconds out: far enough to clear validation despite second-precision truncation and any
	// host/container clock skew, near enough that the ~2s-poll worker delivers within the wait window.
	resp, err := dane.PostFull(conversationsPath+"/"+convID+"/messages", map[string]any{
		"body":              body,
		"client_message_id": uniqueName("cmid"),
		"scheduled_at":      time.Now().Add(5 * time.Second).UTC().Format(time.RFC3339),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	// The worker delivers it: the scheduled message is materialized into a real timeline message that
	// appears in the thread.
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		msgs, _, err := dane.GetList(conversationsPath+"/"+convID+"/messages", url.Values{"limit": {"50"}})
		if err != nil {
			return err
		}
		for _, raw := range msgs.Data {
			if jsonField(parseJSON(raw), "body") == body {
				return nil
			}
		}
		return fmt.Errorf("scheduled message not delivered yet")
	})
}
