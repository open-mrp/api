//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests cover the broadcast-announcement path, polymorphic sender attribution, free-text
// search / sender filtering, and the cross-account unread summary. Like the notification tests
// they avoid t.Parallel() and key off per-test unique titles for DB-state independence.

const (
	announcementsPath        = "/v1/messaging/announcements"
	notificationUnreadSumURL = "/v1/messaging/notifications/unread-summary"
)

// broadcastAnnouncement sends an account-scoped broadcast and asserts the 202 ack.
func broadcastAnnouncement(t *testing.T, sender *Client, title string) {
	t.Helper()
	resp, err := sender.PostFull(notificationsPath, map[string]any{
		"category": "system.broadcast",
		"target":   map[string]any{"type": "account", "id": SeedAccountID},
		"title":    title,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, resp.StatusCode, resp.Body)
}

// unreadCounts returns the reader's (notifications sub-count, total).
func unreadCounts(t *testing.T, reader *Client) (int, int) {
	t.Helper()
	resp, err := reader.GetFull(notificationUnreadCountPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	m := parseJSON(resp.Body)
	notifications, ok := m["notifications"].(float64)
	require.True(t, ok, "notifications should be a number: %s", string(resp.Body))
	total, ok := m["total"].(float64)
	require.True(t, ok, "total should be a number: %s", string(resp.Body))
	return int(notifications), int(total)
}

// ── Broadcast → announcement ───────────────────────────────────────

// findAnnouncement polls the announcements feed until one with the title appears.
func findAnnouncement(t *testing.T, reader *Client, title string) map[string]any {
	t.Helper()
	var found map[string]any
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		list, _, err := reader.GetList(announcementsPath, url.Values{"limit": {"100"}})
		if err != nil {
			return err
		}
		for _, raw := range list.Data {
			m := parseJSON(raw)
			if jsonField(m, "title") == title {
				found = m
				return nil
			}
		}
		return fmt.Errorf("announcement %q not found yet", title)
	})
	return found
}

func TestAnnouncements_BroadcastCreatesAnnouncement(t *testing.T) {
	user := notifUserClient(t)
	title := uniqueName("e2e-announce")

	// A broadcast send (target type=account) is acknowledged like any send but materializes an announcement.
	resp, err := user.PostFull(notificationsPath, map[string]any{
		"category": "system.broadcast",
		"target":   map[string]any{"type": "account", "id": SeedAccountID},
		"title":    title,
		"body":     "All hands at 3pm.",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, resp.StatusCode, resp.Body)

	ann := findAnnouncement(t, user, title)
	assert.Equal(t, "announcement", jsonField(ann, "object"))
	assert.Equal(t, "account", jsonField(ann, "scope"))
	assert.Equal(t, "unseen", jsonField(ann, "status"))
	id := jsonField(ann, "id")
	assertIDFormat(t, id, "an")
}

func TestAnnouncements_MarkSeenReadDismiss(t *testing.T) {
	user := notifUserClient(t)
	title := uniqueName("e2e-announce-mark")

	resp, err := user.PostFull(notificationsPath, map[string]any{
		"category": "system.broadcast",
		"target":   map[string]any{"type": "account", "id": SeedAccountID},
		"title":    title,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, resp.StatusCode, resp.Body)

	ann := findAnnouncement(t, user, title)
	id := jsonField(ann, "id")

	seen, err := user.PostFull(announcementsPath+"/"+id+"/actions/seen", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, seen.StatusCode, seen.Body)
	assert.Equal(t, "seen", jsonField(parseJSON(seen.Body), "status"))

	read, err := user.PostFull(announcementsPath+"/"+id+"/actions/read", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, read.StatusCode, read.Body)
	assert.Equal(t, "read", jsonField(parseJSON(read.Body), "status"))

	dismiss, err := user.PostFull(announcementsPath+"/"+id+"/actions/dismiss", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, dismiss.StatusCode, dismiss.Body)
	assert.Equal(t, "dismissed", jsonField(parseJSON(dismiss.Body), "status"))

	// A dismissed announcement drops out of the active feed.
	list, _, err := user.GetList(announcementsPath, url.Values{"limit": {"100"}})
	require.NoError(t, err)
	for _, raw := range list.Data {
		assert.NotEqual(t, title, jsonField(parseJSON(raw), "title"), "dismissed announcement should be hidden")
	}
}

func TestAnnouncements_BroadcastReachesAllUsersInAccount(t *testing.T) {
	sender := notifUserClient(t)
	title := uniqueName("e2e-announce-broadcast-all")
	broadcastAnnouncement(t, sender, title)

	// A second user in the same account sees the same broadcast (per-user receipt, unseen).
	other := loginAsUser(t, seedUser2Email, seedUserPassword, SeedAccountID)
	ann := findAnnouncement(t, other, title)
	assert.Equal(t, "unseen", jsonField(ann, "status"), "a fresh broadcast is unseen for each recipient")
	assert.Equal(t, "account", jsonField(ann, "scope"))
}

func TestAnnouncements_RetrieveByIDAndNotFound(t *testing.T) {
	user := notifUserClient(t)
	title := uniqueName("e2e-announce-get")
	broadcastAnnouncement(t, user, title)
	id := jsonField(findAnnouncement(t, user, title), "id")

	resp, err := user.GetFull(announcementsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	assert.Equal(t, id, jsonField(parseJSON(resp.Body), "id"))

	missing, err := user.GetFull(announcementsPath+"/an_doesnotexist0000000000", nil)
	require.NoError(t, err)
	assert.Equal(t, 404, missing.StatusCode, "an unknown announcement should 404: %s", string(missing.Body))
}

func TestAnnouncements_MarkNonexistentReturns404(t *testing.T) {
	user := notifUserClient(t)
	status, body, err := user.Post(announcementsPath+"/an_doesnotexist0000000000/actions/seen", nil, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, "marking a nonexistent announcement should 404: %s", string(body))
}

func TestAnnouncements_CursorPaginationAdvances(t *testing.T) {
	user := notifUserClient(t)
	broadcastAnnouncement(t, user, uniqueName("e2e-announce-page"))
	broadcastAnnouncement(t, user, uniqueName("e2e-announce-page"))

	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		list, _, err := user.GetList(announcementsPath, url.Values{"limit": {"100"}})
		if err != nil {
			return err
		}
		if len(list.Data) < 2 {
			return fmt.Errorf("need ≥2 announcements, have %d", len(list.Data))
		}
		return nil
	})

	page1, _, err := user.GetList(announcementsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	require.Len(t, page1.Data, 1, "limit=1 should return one row")
	require.True(t, page1.PageInfo.HasNextPage, "first page should have a next page")
	require.NotNil(t, page1.PageInfo.NextPageURL)

	page2, _, err := user.GetListFromPageURL(page1.PageInfo.NextPageURL)
	require.NoError(t, err)
	require.Len(t, page2.Data, 1, "second page should return one row")
	assert.NotEqual(t, DataItemField(page1.Data[0], "id"), DataItemField(page2.Data[0], "id"),
		"consecutive pages should return different announcements")
}

func TestAnnouncements_CountedInUnreadTotalNotSubcount(t *testing.T) {
	user := notifUserClient(t)
	title := uniqueName("e2e-announce-unread")

	nBefore, tBefore := unreadCounts(t, user)
	broadcastAnnouncement(t, user, title)
	id := jsonField(findAnnouncement(t, user, title), "id")

	// The announcement raises the total but NOT the per-user notifications sub-count.
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		n, total := unreadCounts(t, user)
		if n != nBefore {
			return fmt.Errorf("notifications sub-count changed: %d → %d (announcements must not count here)", nBefore, n)
		}
		if total != tBefore+1 {
			return fmt.Errorf("total = %d, want %d (announcement should add to total)", total, tBefore+1)
		}
		return nil
	})

	// Dismissing the announcement clears it from the total again.
	dismiss, err := user.PostFull(announcementsPath+"/"+id+"/actions/dismiss", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, dismiss.StatusCode, dismiss.Body)
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		if _, total := unreadCounts(t, user); total != tBefore {
			return fmt.Errorf("total after dismiss = %d, want %d", total, tBefore)
		}
		return nil
	})
}

// ── Target validation ──────────────────────────────────────────────

func TestNotifications_SendTargetValidation(t *testing.T) {
	user := notifUserClient(t)

	// Unsupported target type.
	status, body, err := user.Post(notificationsPath, map[string]any{
		"category": "order.updated",
		"target":   map[string]any{"type": "wormhole", "id": SeedAccountUserID},
		"title":    "bad type",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	// A broadcast may only target the caller's own account.
	status, body, err = user.Post(notificationsPath, map[string]any{
		"category": "system.broadcast",
		"target":   map[string]any{"type": "account", "id": "ac_some_other_account"},
		"title":    "cross-account broadcast",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

// ── Sender attribution + filters ───────────────────────────────────

func TestNotifications_SenderDerivedFromIdentity(t *testing.T) {
	user := notifUserClient(t)
	title := uniqueName("e2e-sender")
	sendNotif(t, user, "order.updated", title, nil)

	// sender is an expandable Actor — request it explicitly.
	n := findNotif(t, user, title, url.Values{"include": {"sender"}})
	sender, ok := n["sender"].(map[string]any)
	require.True(t, ok, "a user-initiated notification carries a sender object: %v", n["sender"])
	assert.Equal(t, "actor", jsonField(sender, "object"))
	assert.Equal(t, "user", jsonField(sender, "type"))
	assert.NotEmpty(t, jsonField(sender, "id"), "user sender carries the account_user id")
}

func TestNotifications_FilterBySenderType(t *testing.T) {
	user := notifUserClient(t)
	title := uniqueName("e2e-senderfilter")
	sendNotif(t, user, "order.updated", title, nil)
	findNotif(t, user, title, nil) // ensure materialized

	// Filtering by the user sender type still returns it.
	assert.True(t, feedContainsTitle(t, user, title, url.Values{"sender_types": {"user"}}),
		"notification should match sender_types=user")
	// Filtering by a different sender type excludes it.
	assert.False(t, feedContainsTitle(t, user, title, url.Values{"sender_types": {"agent"}}),
		"user notification should not match sender_types=agent")
}

func TestNotifications_FilterBySenderID(t *testing.T) {
	user := notifUserClient(t)
	title := uniqueName("e2e-senderidfilter")
	sendNotif(t, user, "order.updated", title, nil)
	findNotif(t, user, title, nil)

	// The sender id for a user is their account_user id (dane = SeedAccountUserID).
	assert.True(t, feedContainsTitle(t, user, title, url.Values{"sender_ids": {SeedAccountUserID}}),
		"notification should match its own sender id")
	assert.False(t, feedContainsTitle(t, user, title, url.Values{"sender_ids": {SeedAccountUser2ID}}),
		"notification should not match a different sender id")
}

func TestNotifications_SearchByTitle(t *testing.T) {
	user := notifUserClient(t)
	needle := uniqueName("zzqsearch")
	title := "Quarterly report " + needle
	sendNotif(t, user, "order.updated", title, nil)
	findNotif(t, user, title, nil)

	assert.True(t, feedContainsTitle(t, user, title, url.Values{"q": {needle}}),
		"search for the unique token should find the notification")
	assert.False(t, feedContainsTitle(t, user, title, url.Values{"q": {"nonmatchingtoken-" + needle}}),
		"search for an absent token should not find it")
}

// ── Cross-account unread summary ───────────────────────────────────

func TestNotifications_UnreadSummaryShape(t *testing.T) {
	user := notifUserClient(t)
	title := uniqueName("e2e-summary")
	sendNotif(t, user, "order.updated", title, nil)
	findNotif(t, user, title, nil)

	resp, err := user.GetFull(notificationUnreadSumURL, nil)
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	m := parseJSON(resp.Body)
	assert.Equal(t, "notification_unread_summary", jsonField(m, "object"))

	accounts, ok := listData(m, "accounts")
	require.True(t, ok, "summary carries an accounts list: %s", string(resp.Body))

	var sawSeedAccount bool
	for _, raw := range accounts {
		entry, ok := raw.(map[string]any)
		require.True(t, ok)
		acct, ok := entry["account"].(map[string]any)
		require.True(t, ok, "each entry has an account sub-object")
		assert.Equal(t, "entity", jsonField(acct, "object"))
		assert.Equal(t, "account", jsonField(acct, "type"))
		if jsonField(acct, "id") == SeedAccountID {
			sawSeedAccount = true
		}
	}
	assert.True(t, sawSeedAccount, "the seeded account should appear in the cross-account summary")
}
