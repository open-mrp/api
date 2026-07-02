//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Gap coverage for notifications: single retrieve + recipient scoping, mark-read
// idempotency, and the unread-summary shape. These share the seeded recipient
// feed, so they run sequentially (no t.Parallel) like messaging_notifications_test.go.

func TestNotifications_RetrieveSingleAndScoping(t *testing.T) {
	user := notifUserClient(t)
	title := uniqueName("retrieve-single")
	sendNotif(t, user, "order.updated", title, nil)
	notif := findNotif(t, user, title, nil)
	id := notifID(t, notif)

	// The recipient can retrieve the single notification.
	resp, err := user.GetFull(notificationsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	got := parseJSON(resp.Body)
	assert.Equal(t, "notification", jsonField(got, "object"))
	assert.Equal(t, id, jsonField(got, "id"))

	// A different actor (the API key, not the recipient) must not retrieve it.
	resp2, err := apiClient.GetFull(notificationsPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Contains(t, []int{403, 404}, resp2.StatusCode,
		"a non-recipient must not retrieve another user's notification, got %d: %s", resp2.StatusCode, string(resp2.Body))
}

func TestNotifications_MarkReadIsIdempotent(t *testing.T) {
	user := notifUserClient(t)
	title := uniqueName("mark-read-idem")
	sendNotif(t, user, "order.updated", title, nil)
	id := notifID(t, findNotif(t, user, title, nil))

	// Marking read twice is allowed and stable.
	mustMark(t, user, id, "read")
	mustMark(t, user, id, "read")

	resp, err := user.GetFull(notificationsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	assert.Equal(t, "read", jsonField(parseJSON(resp.Body), "status"), "the notification stays read after a repeat mark")
}
