//go:build e2e

package api_test

import (
	"encoding/json"
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase-4 slice-2 coverage: notification-preference CRUD plus the chat→bell fan-out honoring the
// in-app channel preference and per-conversation mute.

const preferencesPath = "/v1/messaging/preferences"

func upsertPreference(t *testing.T, c *Client, body map[string]any) map[string]any {
	t.Helper()
	resp, err := c.PutFull(preferencesPath, body)
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	return parseJSON(resp.Body)
}

// chatFeedHasBody reports whether the reader's chat.message feed currently contains a notification
// whose body matches the given text.
func chatFeedHasBody(t *testing.T, reader *Client, body string) bool {
	t.Helper()
	list, _, err := reader.GetList(notificationsPath, url.Values{"limit": {"100"}, "category": {"chat.message"}})
	require.NoError(t, err)
	for _, raw := range list.Data {
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			continue
		}
		if jsonField(m, "body") == body {
			return true
		}
	}
	return false
}

func TestNotificationPreferences_UpsertAndList(t *testing.T) {
	user := chatUser2Client(t)

	// Global default (null category).
	global := upsertPreference(t, user, map[string]any{
		"in_app_enabled": true,
		"email_enabled":  false,
		"push_enabled":   false,
		"digest":         "daily",
	})
	assert.Equal(t, "notification_preference", jsonField(global, "object"))
	assertIDFormat(t, jsonField(global, "id"), "nfpf")
	assert.Nil(t, global["category"], "an omitted category is the global default (null)")
	assert.Equal(t, "daily", jsonField(global, "digest"))

	// Category-specific override.
	cat := upsertPreference(t, user, map[string]any{
		"category":       "chat.message",
		"in_app_enabled": false,
		"email_enabled":  false,
		"digest":         "instant",
	})
	assert.Equal(t, "chat.message", jsonField(cat, "category"))
	assert.Equal(t, "false", jsonField(cat, "in_app_enabled"))
	catID := jsonField(cat, "id")

	// Re-upsert the same category replaces (no duplicate row, id stable per (user, category)).
	cat2 := upsertPreference(t, user, map[string]any{
		"category":       "chat.message",
		"in_app_enabled": true,
		"email_enabled":  true,
		"digest":         "off",
	})
	assert.Equal(t, catID, jsonField(cat2, "id"), "re-upsert replaces the same row")
	assert.Equal(t, "true", jsonField(cat2, "in_app_enabled"))
	assert.Equal(t, "off", jsonField(cat2, "digest"))

	// List reflects both (global + the chat.message override).
	list, _, err := user.GetList(preferencesPath, nil)
	require.NoError(t, err)
	categories := map[string]bool{}
	for _, raw := range list.Data {
		categories[jsonField(parseJSON(raw), "category")] = true
	}
	assert.True(t, categories["chat.message"], "the category override is listed")
	assert.True(t, categories[""], "the global default is listed")
}

func TestNotificationPreferences_InvalidDigestRejected(t *testing.T) {
	user := chatUser2Client(t)
	resp, err := user.PutFull(preferencesPath, map[string]any{
		"in_app_enabled": true,
		"digest":         "fortnightly",
	})
	require.NoError(t, err)
	requireStatus(t, 400, resp.StatusCode, resp.Body)
}

func TestChatBell_MessageCreatesBellNotification(t *testing.T) {
	dane := chatUserClient(t)
	user2 := chatUser2Client(t)

	// Default preferences for the recipient (no in-app opt-out).
	upsertPreference(t, user2, map[string]any{
		"category":       "chat.message",
		"in_app_enabled": true,
		"email_enabled":  false,
		"digest":         "off",
	})

	dm := createDM(t, dane, SeedAccountUser2ID)
	convID := jsonField(dm, "id")
	body := uniqueName("bell ping")
	sendMessage(t, dane, convID, body, uniqueName("cmid"))

	// The recipient's bell feed gains a chat.message notification carrying the message body.
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		if chatFeedHasBody(t, user2, body) {
			return nil
		}
		return fmt.Errorf("chat bell notification not visible yet")
	})

	// The sender does not notify themselves.
	assert.False(t, chatFeedHasBody(t, dane, body), "the sender gets no bell notification")
}

func TestChatBell_InAppDisabledSuppressesBell(t *testing.T) {
	dane := chatUserClient(t)
	user2 := chatUser2Client(t)

	// Recipient opts out of in-app chat notifications.
	upsertPreference(t, user2, map[string]any{
		"category":       "chat.message",
		"in_app_enabled": false,
		"email_enabled":  false,
		"digest":         "off",
	})

	dm := createDM(t, dane, SeedAccountUser2ID)
	convID := jsonField(dm, "id")
	body := uniqueName("suppressed ping")
	sendMessage(t, dane, convID, body, uniqueName("cmid"))

	// The bell row is written synchronously with the send, so its absence is stable to assert.
	assert.False(t, chatFeedHasBody(t, user2, body), "in-app opt-out suppresses the bell notification")

	// Restore the default so other tests are unaffected.
	upsertPreference(t, user2, map[string]any{
		"category":       "chat.message",
		"in_app_enabled": true,
		"email_enabled":  false,
		"digest":         "off",
	})
}

func TestChatBell_MuteSuppressesBellButKeepsUnread(t *testing.T) {
	dane := chatUserClient(t)
	user2 := chatUser2Client(t)

	upsertPreference(t, user2, map[string]any{
		"category":       "chat.message",
		"in_app_enabled": true,
		"email_enabled":  false,
		"digest":         "off",
	})

	dm := createDM(t, dane, SeedAccountUser2ID)
	convID := jsonField(dm, "id")

	// The recipient mutes the conversation.
	muteResp, err := user2.PostFull(conversationsPath+"/"+convID+"/actions/mute", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, muteResp.StatusCode, muteResp.Body)
	// The dane↔user2 DM is deduped/shared with the other chat-bell tests, so the
	// mute must be undone or it persists in the DB and suppresses their bells on
	// the next run.
	t.Cleanup(func() {
		_, _ = user2.PostFull(conversationsPath+"/"+convID+"/actions/unmute", map[string]any{}, newIdempotencyKey())
	})

	body := uniqueName("muted ping")
	sendMessage(t, dane, convID, body, uniqueName("cmid"))

	// Muted: no bell notification.
	assert.False(t, chatFeedHasBody(t, user2, body), "a muted conversation produces no bell notification")

	// But the conversation itself still counts as unread for the recipient.
	getResp, err := user2.GetFull(conversationsPath+"/"+convID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getResp.StatusCode, getResp.Body)
	conv := parseJSON(getResp.Body)
	unread, _ := conv["unread"].(float64)
	assert.GreaterOrEqual(t, unread, float64(1), "a muted conversation still tracks unread messages")
}
