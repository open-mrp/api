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

// These tests exercise the Phase-1 in-app notification (bell) pipeline end to end:
//
//	POST send → outbox → enqueuer → RabbitMQ(notification.cmd.fanout)
//	  → notification-service fan-out consumer → notification rows
//	  → REST read/mark endpoints (source of truth)
//
// The send path is asynchronous (it enqueues a fan-out intent), so reads poll with
// `eventually` until the notification materializes — mirroring async_side_effects_test.go.
//
// Recipient is identity.Actor.ID. We act as the seeded admin user (dane → SeedAccountUserID)
// for both send and read (a valid "notify a specific user" / "note to self" scenario), and
// use the API-key client (a different actor) to prove recipient scoping. These tests do not
// call t.Parallel(): they share one recipient feed, so running sequentially keeps unread-count
// deltas deterministic. Assertions key off a per-test unique title rather than absolute counts,
// so they are robust to notifications left by prior e2e runs against the same database.

const (
	notificationsPath           = "/v1/messaging/notifications"
	notificationUnreadCountPath = "/v1/messaging/notifications/unread-count"
	notificationMarkAllSeenPath = "/v1/messaging/notifications/actions/mark-all-seen"
)

// notifUserClient logs in as the seeded admin user (dane = SeedAccountUserID) targeting the
// seeded account. Its actor id is the notification recipient for these tests.
func notifUserClient(t *testing.T) *Client {
	t.Helper()
	return loginAsUser(t, seedUserEmail, seedUserPassword, SeedAccountID)
}

// sendNotif sends a notification to SeedAccountUserID and asserts the 202 acknowledgement.
func sendNotif(t *testing.T, sender *Client, category, title string, extra map[string]any) {
	t.Helper()
	sendNotifTo(t, sender, SeedAccountUserID, category, title, extra)
}

// sendNotifTo sends a notification to a specific recipient and asserts the 202 ack.
func sendNotifTo(t *testing.T, sender *Client, target, category, title string, extra map[string]any) {
	t.Helper()
	body := map[string]any{
		"category": category,
		"target":   map[string]any{"type": "account_user", "id": target},
		"title":    title,
	}
	for k, v := range extra {
		body[k] = v
	}
	resp, err := sender.PostFull(notificationsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, resp.StatusCode, resp.Body)

	ack := parseJSON(resp.Body)
	assert.Equal(t, "notification_send_result", jsonField(ack, "object"))
	assert.Equal(t, "1", jsonField(ack, "enqueued"), "a targeted send enqueues one recipient")
}

// countNotifsWithTitle returns how many notifications in the reader's feed have the title.
func countNotifsWithTitle(t *testing.T, reader *Client, title string) int {
	t.Helper()
	list, _, err := reader.GetList(notificationsPath, url.Values{"limit": {"100"}})
	require.NoError(t, err)
	n := 0
	for _, raw := range list.Data {
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err == nil && jsonField(m, "title") == title {
			n++
		}
	}
	return n
}

// findNotif polls the reader's feed (optionally status-filtered) until a notification with
// the given title appears, returning it. Fails the test on timeout.
func findNotif(t *testing.T, reader *Client, title string, params url.Values) map[string]any {
	t.Helper()
	merged := url.Values{"limit": {"100"}}
	for k, v := range params {
		merged[k] = v
	}
	var found map[string]any
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		list, _, err := reader.GetList(notificationsPath, merged)
		if err != nil {
			return err
		}
		for _, raw := range list.Data {
			var m map[string]any
			if err := json.Unmarshal(raw, &m); err != nil {
				continue
			}
			if jsonField(m, "title") == title {
				found = m
				return nil
			}
		}
		return fmt.Errorf("notification %q not found yet", title)
	})
	return found
}

// feedContainsTitle reports whether the reader's feed (with params) currently contains a
// notification with the given title.
func feedContainsTitle(t *testing.T, reader *Client, title string, params url.Values) bool {
	t.Helper()
	merged := url.Values{"limit": {"100"}}
	for k, v := range params {
		merged[k] = v
	}
	list, _, err := reader.GetList(notificationsPath, merged)
	require.NoError(t, err)
	for _, raw := range list.Data {
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			continue
		}
		if jsonField(m, "title") == title {
			return true
		}
	}
	return false
}

// unreadNotifTotal returns the reader's unread notification count.
func unreadNotifTotal(t *testing.T, reader *Client) int {
	t.Helper()
	resp, err := reader.GetFull(notificationUnreadCountPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	m := parseJSON(resp.Body)
	assert.Equal(t, "notification_unread_count", jsonField(m, "object"))
	n, ok := m["notifications"].(float64)
	require.True(t, ok, "unread-count.notifications should be a number: %s", string(resp.Body))
	return int(n)
}

func uniqueTitle(t *testing.T) string {
	t.Helper()
	return uniqueName("e2e-notif")
}

func notifID(t *testing.T, m map[string]any) string {
	t.Helper()
	id := jsonField(m, "id")
	require.NotEmpty(t, id, "notification should have an id")
	assertIDFormat(t, id, "nf")
	return id
}

// ── Send + feed shape ──────────────────────────────────────────────

func TestNotifications_SendAppearsInFeedWithCorrectShape(t *testing.T) {
	user := notifUserClient(t)
	title := uniqueTitle(t)

	sendNotif(t, user, "order.updated", title, map[string]any{
		"body":     "Order #1024 was updated.",
		"priority": "high",
	})

	n := findNotif(t, user, title, nil)
	assertObjectField(t, n, "notification")
	assert.Equal(t, "order.updated", jsonField(n, "category"))
	assert.Equal(t, "high", jsonField(n, "priority"))
	assert.Equal(t, "Order #1024 was updated.", jsonField(n, "body"))
	assert.Equal(t, "unseen", jsonField(n, "status"), "a fresh notification is unseen")
	assertValidTimestamp(t, jsonField(n, "created_at"), "created_at")
	assertNilField(t, n, "seen_at")
	assertNilField(t, n, "read_at")
	assertNilField(t, n, "dismissed_at")
}

// ── Unread count + mark seen ───────────────────────────────────────

func TestNotifications_UnreadCountAndMarkSeen(t *testing.T) {
	user := notifUserClient(t)
	title := uniqueTitle(t)

	baseline := unreadNotifTotal(t, user)

	sendNotif(t, user, "system.broadcast", title, nil)

	// The async fan-out increments the unread count by exactly one.
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		if got := unreadNotifTotal(t, user); got != baseline+1 {
			return fmt.Errorf("unread count = %d, want %d", got, baseline+1)
		}
		return nil
	})

	n := findNotif(t, user, title, nil)
	id := notifID(t, n)

	// Mark seen → status transitions, seen_at set, count returns to baseline.
	status, body, err := user.Post(notificationsPath+"/"+id+"/actions/seen", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	marked := parseJSON(body)
	assert.Equal(t, "seen", jsonField(marked, "status"))
	assertValidTimestamp(t, jsonField(marked, "seen_at"), "seen_at")

	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		if got := unreadNotifTotal(t, user); got != baseline {
			return fmt.Errorf("unread count after mark-seen = %d, want %d", got, baseline)
		}
		return nil
	})

	// Mark-seen is idempotent.
	status, body, err = user.Post(notificationsPath+"/"+id+"/actions/seen", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, "seen", jsonField(parseJSON(body), "status"))
}

// ── Mark read ──────────────────────────────────────────────────────

func TestNotifications_MarkRead(t *testing.T) {
	user := notifUserClient(t)
	title := uniqueTitle(t)
	sendNotif(t, user, "order.updated", title, nil)

	id := notifID(t, findNotif(t, user, title, nil))

	status, body, err := user.Post(notificationsPath+"/"+id+"/actions/read", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	n := parseJSON(body)
	assert.Equal(t, "read", jsonField(n, "status"))
	assertValidTimestamp(t, jsonField(n, "read_at"), "read_at")
	// Reading implies seen.
	assertValidTimestamp(t, jsonField(n, "seen_at"), "seen_at")
}

// ── Dismiss ────────────────────────────────────────────────────────

func TestNotifications_MarkDismissedRemovesFromDefaultFeed(t *testing.T) {
	user := notifUserClient(t)
	title := uniqueTitle(t)
	sendNotif(t, user, "order.updated", title, nil)

	id := notifID(t, findNotif(t, user, title, nil))

	status, body, err := user.Post(notificationsPath+"/"+id+"/actions/dismiss", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, "dismissed", jsonField(parseJSON(body), "status"))

	// Dismissed notifications are excluded from the default (active) feed...
	assert.False(t, feedContainsTitle(t, user, title, nil),
		"dismissed notification should not appear in the default feed")
	// ...but are returned when explicitly filtering by status=dismissed.
	assert.True(t, feedContainsTitle(t, user, title, url.Values{"status": {"dismissed"}}),
		"dismissed notification should appear when status=dismissed")
}

// ── Mark all seen ──────────────────────────────────────────────────

func TestNotifications_MarkAllSeen(t *testing.T) {
	user := notifUserClient(t)
	titleA := uniqueTitle(t)
	titleB := uniqueTitle(t)
	sendNotif(t, user, "order.updated", titleA, nil)
	sendNotif(t, user, "order.updated", titleB, nil)

	// Wait for both to land.
	findNotif(t, user, titleA, nil)
	findNotif(t, user, titleB, nil)

	status, body, err := user.Post(notificationMarkAllSeenPath, nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	// Both notifications are now seen, and the unread total is zero.
	for _, title := range []string{titleA, titleB} {
		n := findNotif(t, user, title, nil)
		assert.NotEqual(t, "unseen", jsonField(n, "status"), "%s should be seen after mark-all-seen", title)
	}
	assert.Equal(t, 0, unreadNotifTotal(t, user))
}

// ── Category filter ────────────────────────────────────────────────

func TestNotifications_FilterByCategory(t *testing.T) {
	user := notifUserClient(t)
	orderTitle := uniqueTitle(t)
	systemTitle := uniqueTitle(t)
	sendNotif(t, user, "order.updated", orderTitle, nil)
	sendNotif(t, user, "system.broadcast", systemTitle, nil)

	findNotif(t, user, orderTitle, nil)
	findNotif(t, user, systemTitle, nil)

	params := url.Values{"category": {"order.updated"}}
	assert.True(t, feedContainsTitle(t, user, orderTitle, params),
		"order.updated notification should appear when filtering category=order.updated")
	assert.False(t, feedContainsTitle(t, user, systemTitle, params),
		"system.broadcast notification should be filtered out by category=order.updated")
}

// ── Recipient scoping ──────────────────────────────────────────────

func TestNotifications_RecipientScoping(t *testing.T) {
	user := notifUserClient(t)
	title := uniqueTitle(t)
	sendNotif(t, user, "order.updated", title, nil)

	id := notifID(t, findNotif(t, user, title, nil))

	// A different user (same account, different account_user) must not see the recipient's feed.
	other := loginAsUser(t, seedUser2Email, seedUserPassword, SeedAccountID)
	assert.False(t, feedContainsTitle(t, other, title, nil),
		"a different user must not see another user's notifications")

	// Direct retrieval by a non-recipient must 404 (ownership is enforced by the resolved recipient).
	resp, err := other.GetFull(notificationsPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode,
		"retrieving another user's notification should 404, got %d: %s", resp.StatusCode, string(resp.Body))
}

// ── Cursor pagination ──────────────────────────────────────────────

func TestNotifications_CursorPaginationAdvances(t *testing.T) {
	user := notifUserClient(t)
	sendNotif(t, user, "order.updated", uniqueTitle(t), nil)
	sendNotif(t, user, "order.updated", uniqueTitle(t), nil)

	// Ensure at least two are present before walking pages of one.
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		list, _, err := user.GetList(notificationsPath, url.Values{"limit": {"100"}})
		if err != nil {
			return err
		}
		if len(list.Data) < 2 {
			return fmt.Errorf("need ≥2 notifications, have %d", len(list.Data))
		}
		return nil
	})

	page1, _, err := user.GetList(notificationsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	require.Len(t, page1.Data, 1, "limit=1 should return one row")
	require.True(t, page1.PageInfo.HasNextPage, "first page should have a next page")
	require.NotNil(t, page1.PageInfo.NextPageURL)

	page2, _, err := user.GetListFromPageURL(page1.PageInfo.NextPageURL)
	require.NoError(t, err)
	require.Len(t, page2.Data, 1, "second page should return one row")

	assert.NotEqual(t,
		DataItemField(page1.Data[0], "id"), DataItemField(page2.Data[0], "id"),
		"consecutive pages should return different notifications (newest-first, no overlap)")
}

// ── Send validation ────────────────────────────────────────────────

func TestNotifications_SendValidation(t *testing.T) {
	user := notifUserClient(t)

	// Missing title.
	status, body, err := user.Post(notificationsPath, map[string]any{
		"category": "order.updated",
		"target":   map[string]any{"type": "account_user", "id": SeedAccountUserID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	// Missing target.
	status, body, err = user.Post(notificationsPath, map[string]any{
		"category": "order.updated",
		"title":    "no target",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

// ── Marking a nonexistent / non-owned notification ─────────────────

func TestNotifications_MarkNonexistentReturns404(t *testing.T) {
	user := notifUserClient(t)
	status, body, err := user.Post(notificationsPath+"/nf_doesnotexist000000000000/actions/seen", nil, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, "marking a nonexistent notification should 404: %s", string(body))
}

// ── Response schema conformance ────────────────────────────────────

func TestNotifications_ResponsesConformToSpec(t *testing.T) {
	user := notifUserClient(t)
	title := uniqueTitle(t)
	sendNotif(t, user, "order.updated", title, map[string]any{"body": "b", "priority": "high"})
	id := notifID(t, findNotif(t, user, title, nil))

	// Single resource, list envelope, and unread-count must all conform to the OpenAPI schema.
	getResp, err := user.GetFull(notificationsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getResp.StatusCode, getResp.Body)
	AssertResponseBodyValid(t, getResp.Body)

	_, listBody, err := user.GetListRaw(notificationsPath, url.Values{"limit": {"5"}})
	require.NoError(t, err)
	AssertResponseBodyValid(t, listBody)

	ucResp, err := user.GetFull(notificationUnreadCountPath, nil)
	require.NoError(t, err)
	AssertResponseBodyValid(t, ucResp.Body)
}

// ── Send to another user (positive cross-user) ─────────────────────

func TestNotifications_SendToAnotherUser(t *testing.T) {
	admin := notifUserClient(t) // dane, holds alerts:create
	title := uniqueTitle(t)
	sendNotifTo(t, admin, SeedAccountUser2ID, "order.updated", title, nil)

	// The targeted user (user2) sees it in her feed.
	other := loginAsUser(t, seedUser2Email, seedUserPassword, SeedAccountID)
	n := findNotif(t, other, title, nil)
	assert.Equal(t, "order.updated", jsonField(n, "category"))

	// The sender does not (it was targeted at someone else).
	assert.False(t, feedContainsTitle(t, admin, title, nil),
		"a notification targeted at another user must not appear in the sender's own feed")
}

// ── Linked notification round-trip ─────────────────────────────────

func TestNotifications_LinkedRoundTrip(t *testing.T) {
	user := notifUserClient(t)
	title := uniqueTitle(t)
	sendNotif(t, user, "order.updated", title, map[string]any{
		"link_resource_type": "sales_order",
		"link_resource_id":   "so_e2e_1024",
	})

	n := findNotif(t, user, title, url.Values{"include": {"resource"}})

	res := jsonObject(n, "resource")
	require.NotNil(t, res, "a linked notification should expose a resource entity when included")
	assert.Equal(t, "entity", jsonField(res, "object"))
	assert.Equal(t, "sales_order", jsonField(res, "type"))
	assert.Equal(t, "so_e2e_1024", jsonField(res, "id"))
}

// ── Status filter exhaustiveness ───────────────────────────────────

func TestNotifications_StatusFilters(t *testing.T) {
	user := notifUserClient(t)
	unseenTitle := uniqueTitle(t)
	seenTitle := uniqueTitle(t)
	readTitle := uniqueTitle(t)
	sendNotif(t, user, "order.updated", unseenTitle, nil)
	sendNotif(t, user, "order.updated", seenTitle, nil)
	sendNotif(t, user, "order.updated", readTitle, nil)

	findNotif(t, user, unseenTitle, nil)
	seenID := notifID(t, findNotif(t, user, seenTitle, nil))
	readID := notifID(t, findNotif(t, user, readTitle, nil))

	mustMark(t, user, seenID, "seen")
	mustMark(t, user, readID, "read")

	// status=unseen → only the untouched one.
	assert.True(t, feedContainsTitle(t, user, unseenTitle, url.Values{"status": {"unseen"}}))
	assert.False(t, feedContainsTitle(t, user, seenTitle, url.Values{"status": {"unseen"}}))
	assert.False(t, feedContainsTitle(t, user, readTitle, url.Values{"status": {"unseen"}}))

	// status=seen → seen-but-not-read only (the read one is "read", not "seen").
	assert.True(t, feedContainsTitle(t, user, seenTitle, url.Values{"status": {"seen"}}))
	assert.False(t, feedContainsTitle(t, user, unseenTitle, url.Values{"status": {"seen"}}))
	assert.False(t, feedContainsTitle(t, user, readTitle, url.Values{"status": {"seen"}}))

	// status=read → only the read one.
	assert.True(t, feedContainsTitle(t, user, readTitle, url.Values{"status": {"read"}}))
	assert.False(t, feedContainsTitle(t, user, seenTitle, url.Values{"status": {"read"}}))
}

func mustMark(t *testing.T, c *Client, id, action string) {
	t.Helper()
	status, body, err := c.Post(notificationsPath+"/"+id+"/actions/"+action, nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

// ── Send idempotency (same Idempotency-Key) ────────────────────────

func TestNotifications_SendIdempotency(t *testing.T) {
	user := notifUserClient(t)
	title := uniqueTitle(t)
	key := newIdempotencyKey()
	body := map[string]any{
		"category": "order.updated",
		"target":   map[string]any{"type": "account_user", "id": SeedAccountUserID},
		"title":    title,
	}

	// Two POSTs with the SAME idempotency key must collapse to a single notification.
	for i := 0; i < 2; i++ {
		resp, err := user.PostFull(notificationsPath, body, key)
		require.NoError(t, err)
		requireStatus(t, 202, resp.StatusCode, resp.Body)
	}

	findNotif(t, user, title, nil) // ensure it materialized
	// Give any (incorrect) duplicate a chance to appear, then assert exactly one exists.
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		if c := countNotifsWithTitle(t, user, title); c != 1 {
			return fmt.Errorf("expected exactly 1 notification for the idempotency key, found %d", c)
		}
		return nil
	})
}

// ── Send permission enforcement ────────────────────────────────────

func TestNotifications_SendRequiresPermission(t *testing.T) {
	// The customer API key is a relation actor with no alerts permission; it must not
	// be able to send notifications.
	customer := apiClient.WithBearerToken(SeedCustomerAPIKey, SeedAccountID)
	resp, err := customer.PostFull(notificationsPath, map[string]any{
		"category": "order.updated",
		"target":   map[string]any{"type": "account_user", "id": SeedAccountUserID},
		"title":    uniqueTitle(t),
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode,
		"an actor without alerts:create must be forbidden from sending, got %d: %s", resp.StatusCode, string(resp.Body))
}
