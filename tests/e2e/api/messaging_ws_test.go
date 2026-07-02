//go:build e2e

package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/require"
)

// These tests exercise the realtime path end to end: a REST action enqueues an outbox event
// that flows through RabbitMQ to the api-gateway WS consumer, which fans it out to the right
// Hub topic and onto a connected client's socket. The REST tests cover the persisted source of
// truth; these cover the live push that the bell/inbox UI relies on.

const wsPath = "/v1/ws"

// wsLoginToken logs in and returns the raw access-token cookie value (the WS handler
// authenticates via cookie, not the Authorization header).
func wsLoginToken(t *testing.T, identifier, password string) string {
	t.Helper()
	resp, err := apiClient.PostFull(loginPath, map[string]any{
		"identifier": identifier,
		"password":   password,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	token := cookieValue(resp.Header["Set-Cookie"], accessTokenCookie)
	require.NotEmpty(t, token, "login should set the access-token cookie")
	return token
}

// dialNotificationWS opens an authenticated WebSocket to the gateway for the seeded admin user
// (dane) targeting accountID, and registers cleanup. The connection auto-subscribes to the
// user, account, and userglobal topics server-side.
func dialNotificationWS(t *testing.T, accountID string) *websocket.Conn {
	t.Helper()
	return dialNotificationWSAs(t, seedUserEmail, seedUserPassword, accountID)
}

// dialNotificationWSAs opens an authenticated WebSocket for the given user credentials.
func dialNotificationWSAs(t *testing.T, identifier, password, accountID string) *websocket.Conn {
	t.Helper()
	token := wsLoginToken(t, identifier, password)

	baseURL := envOr("E2E_BASE_URL", defaultBaseURL)
	wsURL := strings.Replace(baseURL, "http", "ws", 1) + wsPath + "?accountId=" + accountID

	header := http.Header{}
	header.Set("Cookie", accessTokenCookie+"="+token)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{HTTPHeader: header})
	require.NoError(t, err, "WebSocket dial should succeed")
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "") })

	// The server subscribes the connection to its topics synchronously after the upgrade; give
	// that a beat so an immediately-triggered push isn't missed (live pushes aren't buffered).
	time.Sleep(300 * time.Millisecond)
	return conn
}

// wsFrame is the decoded envelope of a server-sent WS message.
type wsFrame struct {
	Type string         `json:"type"`
	Data map[string]any `json:"data"`
}

// readWSUntil reads frames until match returns true or the timeout elapses.
func readWSUntil(t *testing.T, conn *websocket.Conn, timeout time.Duration, match func(wsFrame) bool) wsFrame {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Until(deadline))
		_, raw, err := conn.Read(ctx)
		cancel()
		if err != nil {
			t.Fatalf("WebSocket read failed before a matching frame arrived: %v", err)
		}
		var frame wsFrame
		if err := json.Unmarshal(raw, &frame); err != nil {
			continue // ignore non-JSON / unexpected frames
		}
		if match(frame) {
			return frame
		}
	}
	t.Fatalf("no matching WebSocket frame within %s", timeout)
	return wsFrame{}
}

func frameEvent(f wsFrame) string {
	if f.Data == nil {
		return ""
	}
	if e, ok := f.Data["event"].(string); ok {
		return e
	}
	return ""
}

func TestWS_TargetedNotificationPush(t *testing.T) {
	conn := dialNotificationWS(t, SeedAccountID)
	user := notifUserClient(t)
	title := uniqueName("e2e-ws-notif")

	sendNotif(t, user, "order.updated", title, nil)

	frame := readWSUntil(t, conn, e2eAsyncWaitTimeout, func(f wsFrame) bool {
		return f.Type == "notification" && frameEvent(f) == "notification.created"
	})
	require.NotEmpty(t, frame.Data["notification_id"], "the push should reference the created notification")
}

func TestWS_BroadcastAnnouncementPush(t *testing.T) {
	conn := dialNotificationWS(t, SeedAccountID)
	user := notifUserClient(t)
	title := uniqueName("e2e-ws-announce")

	broadcastAnnouncement(t, user, title)

	frame := readWSUntil(t, conn, e2eAsyncWaitTimeout, func(f wsFrame) bool {
		return frameEvent(f) == "announcement.created"
	})
	require.NotEmpty(t, frame.Data["announcement_id"], "the broadcast push should reference the announcement")
}

func TestWS_UnreadChangedOnMarkSeen(t *testing.T) {
	user := notifUserClient(t)
	title := uniqueName("e2e-ws-unread")
	sendNotif(t, user, "order.updated", title, nil)
	id := notifID(t, findNotif(t, user, title, nil))

	// Connect only after the notification exists, then mark it seen and expect the live badge update.
	conn := dialNotificationWS(t, SeedAccountID)
	resp, err := user.PostFull(notificationsPath+"/"+id+"/actions/seen", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)

	readWSUntil(t, conn, e2eAsyncWaitTimeout, func(f wsFrame) bool {
		return f.Type == "unread" && frameEvent(f) == "unread.changed"
	})
}

func TestWS_ChatMessagePushToRecipient(t *testing.T) {
	// dane DMs user2; user2's live connection should receive the message on their
	// per-user topic (no explicit conversation subscribe needed — the server fans to it).
	user := chatUserClient(t)
	convID := jsonField(createDM(t, user, SeedAccountUser2ID), "id")

	conn := dialNotificationWSAs(t, seedUser2Email, seedUserPassword, SeedAccountID)
	sendMessage(t, user, convID, uniqueName("ws-chat"), newIdempotencyKey())

	frame := readWSUntil(t, conn, e2eAsyncWaitTimeout, func(f wsFrame) bool {
		return f.Type == "message" && frameEvent(f) == "message.created"
	})
	require.Equal(t, convID, fmt.Sprint(frame.Data["conversation_id"]), "the push identifies the conversation")
	require.NotEmpty(t, frame.Data["message_id"])
}

func TestWS_CrossAccountHintOnUserGlobal(t *testing.T) {
	conn := dialNotificationWS(t, SeedAccountID)
	user := notifUserClient(t)
	title := uniqueName("e2e-ws-hint")

	sendNotif(t, user, "order.updated", title, nil)

	// The same notification also fans an account-agnostic hint to the user's global topic.
	frame := readWSUntil(t, conn, e2eAsyncWaitTimeout, func(f wsFrame) bool {
		return f.Type == "account_unread_hint" && frameEvent(f) == "account.unread_hint"
	})
	require.Equal(t, SeedAccountID, fmt.Sprint(frame.Data["account_id"]), "the hint identifies which account has unread")
}
