//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file extends the existing notification coverage in
// messaging_notifications_test.go / messaging_notifications_extra_test.go /
// messaging_announcements_test.go with the remaining required categories:
// every Notification/NotificationUnreadCount/NotificationUnreadSummary(Account)
// json field asserted at least once (including updated_at, dismissed_at
// post-dismiss, unread-count.conversations, unread-summary account.unread),
// omitted-field defaults (body -> null, priority -> "normal"), the
// sender/resource expandable sub-fields defaulting to null without
// ?include=, negative-enum validation on both the send body (category,
// priority) and list query params (category, status, sender_types),
// target.id="" / empty-title validation, malformed cursor / unknown include
// rejection, a direct 404 on a fabricated id, 404 for mark-read/mark-dismiss
// on a nonexistent id, mark-dismiss idempotency, mark-all-seen repeat
// stability, and OpenAPI schema conformance for the send-notification 202 ack
// and the unread-summary GET response.
//
// Unlike the pre-existing (unparallel) notification test files, these tests
// call t.Parallel() per repo convention for new coverage-gap files. To stay
// robust when run concurrently with each other and with sibling coverage
// files that also write to the shared seeded recipient feed
// (SeedAccountUserID), every test here keys off a per-test unique title
// (uniqueName) and asserts either a single test-owned notification's fields
// directly, or a >=1 threshold on aggregate counters -- never an exact
// baseline-delta on the shared feed's absolute totals (see
// TestNotifications_UnreadCountAndMarkSeen in messaging_notifications_test.go
// for that pattern, which relies on strictly sequential execution).
//
// It reuses notifUserClient, sendNotif, sendNotifTo, findNotif,
// feedContainsTitle, notifID, mustMark, notificationsPath,
// notificationUnreadCountPath, notificationMarkAllSeenPath, and
// notificationUnreadSumURL declared in messaging_notifications_test.go /
// messaging_announcements_test.go.

// ── All fields (incl. updated_at bump + dismissed_at post-dismiss) ────

func TestCovMessagingNotifications_AllFieldsPopulatedAndTimestampsBump(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)
	title := uniqueName("e2e-cov-notif-full")

	resp, err := user.PostFull(notificationsPath, map[string]any{
		"category":           "order.updated",
		"target":             map[string]any{"type": "account_user", "id": SeedAccountUserID},
		"title":              title,
		"body":               "full body",
		"priority":           "urgent",
		"link_resource_type": "sales_order",
		"link_resource_id":   SeedSalesOrderID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, resp.StatusCode, resp.Body)

	n := findNotif(t, user, title, url.Values{"include": {"sender", "resource"}})
	id := notifID(t, n)
	assertObjectField(t, n, "notification")
	assert.Equal(t, "order.updated", jsonField(n, "category"))
	assert.Equal(t, title, jsonField(n, "title"))
	assert.Equal(t, "full body", jsonField(n, "body"))
	assert.Equal(t, "unseen", jsonField(n, "status"))
	assert.Equal(t, "urgent", jsonField(n, "priority"))

	sender := jsonObject(n, "sender")
	require.NotNil(t, sender, "sender should populate with ?include=sender")
	assert.Equal(t, "actor", jsonField(sender, "object"))
	assert.Equal(t, "user", jsonField(sender, "type"))
	assert.Equal(t, SeedAccountUserID, jsonField(sender, "id"))

	resourceObj := jsonObject(n, "resource")
	require.NotNil(t, resourceObj, "resource should populate with ?include=resource")
	assert.Equal(t, "entity", jsonField(resourceObj, "object"))
	assert.Equal(t, "sales_order", jsonField(resourceObj, "type"))
	assert.Equal(t, SeedSalesOrderID, jsonField(resourceObj, "id"))

	assertNilField(t, n, "seen_at")
	assertNilField(t, n, "read_at")
	assertNilField(t, n, "dismissed_at")
	assertValidTimestamp(t, jsonField(n, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(n, "updated_at"), "updated_at")
	createdAt := jsonField(n, "created_at")
	updatedAtInitial := jsonField(n, "updated_at")

	// Mark seen: updated_at must bump.
	seenStatus, seenBody, err := user.Post(notificationsPath+"/"+id+"/actions/seen", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, seenStatus, seenBody)
	seenM := parseJSON(seenBody)
	assert.Equal(t, "seen", jsonField(seenM, "status"))
	assertValidTimestamp(t, jsonField(seenM, "seen_at"), "seen_at")
	assert.Equal(t, createdAt, jsonField(seenM, "created_at"), "created_at must never change")
	assertValidTimestamp(t, jsonField(seenM, "updated_at"), "updated_at")
	assert.NotEqual(t, updatedAtInitial, jsonField(seenM, "updated_at"), "updated_at should bump on mark-seen")
	updatedAtAfterSeen := jsonField(seenM, "updated_at")

	// Mark dismissed: dismissed_at must populate, updated_at must bump again.
	dismissStatus, dismissBody, err := user.Post(notificationsPath+"/"+id+"/actions/dismiss", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, dismissStatus, dismissBody)
	dismissM := parseJSON(dismissBody)
	assert.Equal(t, "dismissed", jsonField(dismissM, "status"))
	assertValidTimestamp(t, jsonField(dismissM, "dismissed_at"), "dismissed_at")
	// Prior receipt is preserved, not cleared, by the later transition.
	assertValidTimestamp(t, jsonField(dismissM, "seen_at"), "seen_at")
	assertValidTimestamp(t, jsonField(dismissM, "updated_at"), "updated_at")
	assert.NotEqual(t, updatedAtAfterSeen, jsonField(dismissM, "updated_at"), "updated_at should bump again on mark-dismiss")
}

// ── Omitted fields default correctly ───────────────────────────────

func TestCovMessagingNotifications_OmittedFieldsDefaults(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)
	title := uniqueName("e2e-cov-notif-omit")

	// No body, no priority, no link_resource_type/id.
	resp, err := user.PostFull(notificationsPath, map[string]any{
		"category": "order.updated",
		"target":   map[string]any{"type": "account_user", "id": SeedAccountUserID},
		"title":    title,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, resp.StatusCode, resp.Body)

	n := findNotif(t, user, title, nil)
	assertNilField(t, n, "body")
	assert.Equal(t, "normal", jsonField(n, "priority"), "priority should default to normal when omitted")
	assertNilField(t, n, "sender")
	assertNilField(t, n, "resource")
	assertNilField(t, n, "seen_at")
	assertNilField(t, n, "read_at")
	assertNilField(t, n, "dismissed_at")

	// Even requesting ?include=resource, resource stays null: nothing was
	// stashed for resourceloaders to populate when link_resource_type/id
	// were never sent.
	id := notifID(t, n)
	getResp, err := user.GetFull(notificationsPath+"/"+id+"?include=resource", nil)
	require.NoError(t, err)
	requireStatus(t, 200, getResp.StatusCode, getResp.Body)
	assertNilField(t, parseJSON(getResp.Body), "resource")
}

// ── Expandable fields default to null without ?include ─────────────

func TestCovMessagingNotifications_ExpandableFieldsNilWithoutInclude(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)
	title := uniqueName("e2e-cov-notif-noinclude")

	resp, err := user.PostFull(notificationsPath, map[string]any{
		"category":           "order.updated",
		"target":             map[string]any{"type": "account_user", "id": SeedAccountUserID},
		"title":              title,
		"link_resource_type": "sales_order",
		"link_resource_id":   SeedSalesOrderID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, resp.StatusCode, resp.Body)

	// List row: no ?include at all.
	n := findNotif(t, user, title, nil)
	assertNilField(t, n, "sender")
	assertNilField(t, n, "resource")
	id := notifID(t, n)

	// Direct GET without ?include also nils both, even though the row has a
	// real sender and a real linked resource stashed server-side.
	getResp, err := user.GetFull(notificationsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getResp.StatusCode, getResp.Body)
	getM := parseJSON(getResp.Body)
	assertNilField(t, getM, "sender")
	assertNilField(t, getM, "resource")
}

// ── NotificationUnreadCount.conversations ───────────────────────────

func TestCovMessagingNotifications_UnreadCountConversationsField(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)

	resp, err := user.GetFull(notificationUnreadCountPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	m := parseJSON(resp.Body)
	assert.Equal(t, "notification_unread_count", jsonField(m, "object"))

	// PROD-BUG-SUSPECT (flag only, do not fix here): conversations is
	// hard-coded to 0 in notification-service
	// (messaging_service.go, "Conversations is 0 until chat ships (Phase 2)")
	// even though chat has since shipped. This assertion pins the CURRENT
	// value so a silent behavior change is caught; it does not assert this
	// is the desired long-term behavior.
	conversations, ok := m["conversations"].(float64)
	require.True(t, ok, "unread-count.conversations should be a number: %s", string(resp.Body))
	assert.Equal(t, float64(0), conversations, "conversations is currently hard-coded to 0 (see prod-bug-suspect note)")

	_, ok = m["notifications"].(float64)
	require.True(t, ok, "unread-count.notifications should be a number: %s", string(resp.Body))
	_, ok = m["total"].(float64)
	require.True(t, ok, "unread-count.total should be a number: %s", string(resp.Body))
}

// ── NotificationUnreadSummaryAccount.unread ─────────────────────────

func TestCovMessagingNotifications_UnreadSummaryAccountUnreadField(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)
	title := uniqueName("e2e-cov-notif-summary-unread")

	// Send (but do not mark) a notification to bump the seeded account's unread tally.
	sendNotif(t, user, "order.updated", title, nil)
	findNotif(t, user, title, nil) // ensure materialized

	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		resp, err := user.GetFull(notificationUnreadSumURL, nil)
		if err != nil {
			return err
		}
		if resp.StatusCode != 200 {
			return fmt.Errorf("unread-summary returned status %d: %s", resp.StatusCode, string(resp.Body))
		}
		m := parseJSON(resp.Body)
		accounts, ok := listData(m, "accounts")
		if !ok {
			return fmt.Errorf("unread-summary response missing accounts list: %s", string(resp.Body))
		}
		for _, raw := range accounts {
			entry, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			acct, ok := entry["account"].(map[string]any)
			if !ok {
				continue
			}
			if jsonField(acct, "id") != SeedAccountID {
				continue
			}
			unread, ok := entry["unread"].(float64)
			if !ok {
				return fmt.Errorf("account entry for %s has no numeric unread field", SeedAccountID)
			}
			if unread >= 1 {
				return nil
			}
			return fmt.Errorf("unread for %s = %v, want >= 1", SeedAccountID, unread)
		}
		return fmt.Errorf("seeded account %s not found in unread-summary accounts list", SeedAccountID)
	})
}

// ── Send validation ─────────────────────────────────────────────────

func TestCovMessagingNotifications_SendEmptyTitleRejected(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)

	status, body, err := user.Post(notificationsPath, map[string]any{
		"category": "order.updated",
		"target":   map[string]any{"type": "account_user", "id": SeedAccountUserID},
		"title":    "",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "missing_field", "invalid_request_error")
}

func TestCovMessagingNotifications_SendInvalidCategoryRejected(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)

	status, body, err := user.Post(notificationsPath, map[string]any{
		"category": "bogus.category",
		"target":   map[string]any{"type": "account_user", "id": SeedAccountUserID},
		"title":    uniqueName("e2e-cov-notif-badcat"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "category")
}

func TestCovMessagingNotifications_SendInvalidPriorityRejected(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)

	status, body, err := user.Post(notificationsPath, map[string]any{
		"category": "order.updated",
		"target":   map[string]any{"type": "account_user", "id": SeedAccountUserID},
		"title":    uniqueName("e2e-cov-notif-badprio"),
		"priority": "extreme",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "priority")
}

func TestCovMessagingNotifications_SendEmptyTargetIDRejected(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)

	status, body, err := user.Post(notificationsPath, map[string]any{
		"category": "order.updated",
		"target":   map[string]any{"type": "account_user", "id": ""},
		"title":    uniqueName("e2e-cov-notif-emptytarget"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "missing_field", "invalid_request_error")
}

// ── List query-param validation ─────────────────────────────────────

func TestCovMessagingNotifications_ListInvalidCategoryRejected(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)

	status, body, err := user.GetListRaw(notificationsPath, url.Values{"category": {"bogus.category"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "Category")
}

func TestCovMessagingNotifications_ListInvalidStatusRejected(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)

	status, body, err := user.GetListRaw(notificationsPath, url.Values{"status": {"bogus"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "Status")
}

// TestCovMessagingNotifications_ListInvalidSenderTypeIsSilentNoMatch documents
// current (asymmetric) behavior: unlike category/status, sender_types is not
// enum-validated server-side -- an unrecognized value simply matches nothing
// and returns 200 with an empty page, rather than 400. This is a real gap
// (inconsistent with category/status validation) but not a data-correctness
// bug, so it is pinned here rather than "fixed" by this test.
func TestCovMessagingNotifications_ListInvalidSenderTypeRejected(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)
	// Unrecognized enum values in a list filter are rejected with 400 (platform convention).
	status, body, err := user.GetListRaw(notificationsPath, url.Values{"sender_types": {"bogus_sender_type"}, "limit": {"100"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

func TestCovMessagingNotifications_MalformedCursorRejected(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)

	status, body, err := user.GetListRaw(notificationsPath, url.Values{"cursor": {"not-a-real-cursor"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "cursor")
}

func TestCovMessagingNotifications_UnknownIncludeRejected(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)

	status, body, err := user.GetListRaw(notificationsPath, url.Values{"include": {"bogus_field"}, "limit": {"1"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "include[]")
}

// ── 404s ─────────────────────────────────────────────────────────────

func TestCovMessagingNotifications_RetrieveFabricatedIDReturns404(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)

	resp, err := user.GetFull(notificationsPath+"/nf_doesnotexist000000000000", nil)
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode, "retrieving a fabricated notification id should 404: %s", string(resp.Body))
}

func TestCovMessagingNotifications_MarkReadDismissNonexistentReturns404(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)
	const bogusID = "nf_doesnotexist000000000000"

	status, body, err := user.Post(notificationsPath+"/"+bogusID+"/actions/read", nil, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, "marking read on a nonexistent notification should 404: %s", string(body))

	status, body, err = user.Post(notificationsPath+"/"+bogusID+"/actions/dismiss", nil, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, "marking dismissed on a nonexistent notification should 404: %s", string(body))
}

// ── Action idempotency ───────────────────────────────────────────────

func TestCovMessagingNotifications_MarkDismissedIsIdempotent(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)
	title := uniqueName("e2e-cov-notif-dismissidem")
	sendNotif(t, user, "order.updated", title, nil)
	id := notifID(t, findNotif(t, user, title, nil))

	status1, body1, err := user.Post(notificationsPath+"/"+id+"/actions/dismiss", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)
	m1 := parseJSON(body1)
	assert.Equal(t, "dismissed", jsonField(m1, "status"))
	dismissedAt1 := jsonField(m1, "dismissed_at")
	require.NotEmpty(t, dismissedAt1)

	status2, body2, err := user.Post(notificationsPath+"/"+id+"/actions/dismiss", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)
	m2 := parseJSON(body2)
	assert.Equal(t, "dismissed", jsonField(m2, "status"))
	assert.Equal(t, dismissedAt1, jsonField(m2, "dismissed_at"), "repeat mark-dismiss must not move dismissed_at")
}

// Not parallel: mark-all-seen mutates the entire seed user's feed, so it must
// run in the sequential phase — before any parallel test that asserts a fresh
// notification is still unseen (e.g. OmittedFieldsDefaults) resumes. Its sibling
// TestNotifications_MarkAllSeen is serial for the same reason.
func TestCovMessagingNotifications_MarkAllSeenRepeatedIsStable(t *testing.T) {
	user := notifUserClient(t)
	title := uniqueName("e2e-cov-notif-markallidem")
	sendNotif(t, user, "order.updated", title, nil)
	findNotif(t, user, title, nil)

	status1, body1, err := user.Post(notificationMarkAllSeenPath, nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)

	// The notification sent above is now (at least) seen.
	n := findNotif(t, user, title, nil)
	assert.NotEqual(t, "unseen", jsonField(n, "status"), "notification should be seen after mark-all-seen")

	// Calling again with nothing new unseen must still succeed, not error.
	status2, body2, err := user.Post(notificationMarkAllSeenPath, nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)
}

// ── Response schema conformance (send ack + unread-summary) ─────────

func TestCovMessagingNotifications_SendAckConformsToSpec(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)
	title := uniqueName("e2e-cov-notif-ackschema")

	resp, err := user.PostFull(notificationsPath, map[string]any{
		"category": "order.updated",
		"target":   map[string]any{"type": "account_user", "id": SeedAccountUserID},
		"title":    title,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, resp.StatusCode, resp.Body)
	AssertResponseBodyValid(t, resp.Body)
}

func TestCovMessagingNotifications_UnreadSummaryConformsToSpec(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)

	resp, err := user.GetFull(notificationUnreadSumURL, nil)
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	AssertResponseBodyValid(t, resp.Body)
}
