//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file extends the existing announcements coverage in messaging_announcements_test.go (broadcast materialization, seen/read/dismiss transitions, retrieve, 404-on-unknown-seen, cursor pagination, unread-summary cross-feature counting) with the remaining required categories: every apiresource.Announcement json field asserted at least once, omitted-field defaults, the `resource` expandable sub-field (nil-without-include and populated-with-include, on both retrieve/mark and list), list query-param validation (`limit` bounds, `q` length bound), documentation of the `q` no-op prod-bug-suspect, mark-action idempotency, 404 for `read`/`dismiss` on an unknown id (existing coverage only exercises `seen`), malformed-id 404, unknown-include rejection, unauthenticated 401, and cross-account isolation.
//
// This group has no create/update/delete endpoints of its own (crudLifecycle is `na` per the task spec — announcements only materialize as a side effect of POST /v1/messaging/notifications with target.type=account), so lifecycle coverage lives in the seen/read/dismiss action tests instead. It reuses notifUserClient, broadcastAnnouncement, findAnnouncement, notificationsPath, and announcementsPath declared in messaging_announcements_test.go / messaging_notifications_test.go.

// ── All fields ──────────────────────────────────────────────────────

func TestCovMessagingAnnouncements_AllFieldsPopulated(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)
	title := uniqueName("e2e-cov-announce-full")

	resp, err := user.PostFull(notificationsPath, map[string]any{
		"category":           "system.broadcast",
		"target":             map[string]any{"type": "account", "id": SeedAccountID},
		"title":              title,
		"body":               "All hands at 3pm.",
		"priority":           "high",
		"link_resource_type": "sales_order",
		"link_resource_id":   SeedSalesOrderID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, resp.StatusCode, resp.Body)

	ann := findAnnouncement(t, user, title)
	id := jsonField(ann, "id")
	assertIDFormat(t, id, "an")
	assertObjectField(t, ann, "announcement")
	assert.Equal(t, "account", jsonField(ann, "scope"))
	assert.Equal(t, "system.broadcast", jsonField(ann, "category"))
	assert.Equal(t, title, jsonField(ann, "title"))
	assert.Equal(t, "All hands at 3pm.", jsonField(ann, "body"))
	assert.Equal(t, "unseen", jsonField(ann, "status"))
	assert.Equal(t, "high", jsonField(ann, "priority"))
	// resource is expandable: nil without ?include=resource even though a link was set.
	assertNilField(t, ann, "resource")
	assertValidTimestamp(t, jsonField(ann, "publish_at"), "publish_at")
	// expires_at is unreachable via the public send API today (no field on SendNotificationRequest) — always null for API-created announcements.
	assertNilField(t, ann, "expires_at")
	assertNilField(t, ann, "seen_at")
	assertNilField(t, ann, "read_at")
	assertNilField(t, ann, "dismissed_at")
	assertValidTimestamp(t, jsonField(ann, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(ann, "updated_at"), "updated_at")

	// Mark seen with ?include=resource: the resource expandable populates on a mark action too.
	seen, err := user.PostFull(announcementsPath+"/"+id+"/actions/seen?include=resource", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, seen.StatusCode, seen.Body)
	seenM := parseJSON(seen.Body)
	assert.Equal(t, "seen", jsonField(seenM, "status"))
	assertValidTimestamp(t, jsonField(seenM, "seen_at"), "seen_at")
	resourceObj := jsonObject(seenM, "resource")
	require.NotNil(t, resourceObj, "resource should populate with ?include=resource: %s", string(seen.Body))
	assertObjectField(t, resourceObj, "entity")
	assert.Equal(t, "sales_order", jsonField(resourceObj, "type"))
	assert.Equal(t, SeedSalesOrderID, jsonField(resourceObj, "id"))
	assertNilField(t, resourceObj, "name")
	assertNilField(t, resourceObj, "handle")

	// Mark read (no include this time).
	read, err := user.PostFull(announcementsPath+"/"+id+"/actions/read", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, read.StatusCode, read.Body)
	readM := parseJSON(read.Body)
	assert.Equal(t, "read", jsonField(readM, "status"))
	assertValidTimestamp(t, jsonField(readM, "read_at"), "read_at")

	// Mark dismissed.
	dismiss, err := user.PostFull(announcementsPath+"/"+id+"/actions/dismiss", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, dismiss.StatusCode, dismiss.Body)
	dismissM := parseJSON(dismiss.Body)
	assert.Equal(t, "dismissed", jsonField(dismissM, "status"))
	assertValidTimestamp(t, jsonField(dismissM, "dismissed_at"), "dismissed_at")
	// Prior receipts are preserved, not cleared, by later transitions.
	assertValidTimestamp(t, jsonField(dismissM, "seen_at"), "seen_at")
	assertValidTimestamp(t, jsonField(dismissM, "read_at"), "read_at")

	// A dismissed announcement is still directly retrievable by id (only the active *list* excludes it); include=resource still gates the sub-field.
	getResp, err := user.GetFull(announcementsPath+"/"+id+"?include=resource", nil)
	require.NoError(t, err)
	requireStatus(t, 200, getResp.StatusCode, getResp.Body)
	getM := parseJSON(getResp.Body)
	assert.Equal(t, "dismissed", jsonField(getM, "status"))
	getResource := jsonObject(getM, "resource")
	require.NotNil(t, getResource)
	assert.Equal(t, SeedSalesOrderID, jsonField(getResource, "id"))
}

// ── Omitted fields ──────────────────────────────────────────────────

func TestCovMessagingAnnouncements_OmittedFieldsDefaults(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)
	title := uniqueName("e2e-cov-announce-omit")

	// No body, no priority, no link_resource_type/id.
	resp, err := user.PostFull(notificationsPath, map[string]any{
		"category": "system.broadcast",
		"target":   map[string]any{"type": "account", "id": SeedAccountID},
		"title":    title,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, resp.StatusCode, resp.Body)

	ann := findAnnouncement(t, user, title)
	id := jsonField(ann, "id")

	assertNilField(t, ann, "body")
	assert.Equal(t, "normal", jsonField(ann, "priority"), "priority should default to normal when omitted")
	assertNilField(t, ann, "resource")
	assertNilField(t, ann, "expires_at")
	assertNilField(t, ann, "seen_at")
	assertNilField(t, ann, "read_at")
	assertNilField(t, ann, "dismissed_at")
	assert.Equal(t, "unseen", jsonField(ann, "status"))

	// Even with ?include=resource requested, resource stays null when no link_resource_type/id were set at send time — there is nothing stashed for resourceloaders to populate.
	getResp, err := user.GetFull(announcementsPath+"/"+id+"?include=resource", nil)
	require.NoError(t, err)
	requireStatus(t, 200, getResp.StatusCode, getResp.Body)
	assertNilField(t, parseJSON(getResp.Body), "resource")
}

// ── Expandable resource via list ────────────────────────────────────

func TestCovMessagingAnnouncements_ResourceExpandableViaList(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)
	title := uniqueName("e2e-cov-announce-listres")

	resp, err := user.PostFull(notificationsPath, map[string]any{
		"category":           "system.broadcast",
		"target":             map[string]any{"type": "account", "id": SeedAccountID},
		"title":              title,
		"link_resource_type": "sales_order",
		"link_resource_id":   SeedSalesOrderID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, resp.StatusCode, resp.Body)

	// Without ?include=resource: nil on the list row too.
	ann := findAnnouncement(t, user, title)
	assertNilField(t, ann, "resource")

	// With ?include=resource: populated on the list row.
	list, _, err := user.GetList(announcementsPath, url.Values{"limit": {"100"}, "include": {"resource"}})
	require.NoError(t, err)
	var found map[string]any
	for _, raw := range list.Data {
		m := parseJSON(raw)
		if jsonField(m, "title") == title {
			found = m
			break
		}
	}
	require.NotNil(t, found, "announcement %q should appear in the include=resource list", title)
	resourceObj := jsonObject(found, "resource")
	require.NotNil(t, resourceObj, "resource should populate on the list row with ?include=resource")
	assertObjectField(t, resourceObj, "entity")
	assert.Equal(t, "sales_order", jsonField(resourceObj, "type"))
	assert.Equal(t, SeedSalesOrderID, jsonField(resourceObj, "id"))
}

// ── List: basic shape + query no-op documentation ──────────────────

func TestCovMessagingAnnouncements_ListBasicSmoke(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)
	title := uniqueName("e2e-cov-announce-smoke")
	broadcastAnnouncement(t, user, title)
	findAnnouncement(t, user, title) // ensure at least one row is materialized

	list, status, err := user.GetList(announcementsPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	assert.Equal(t, "list", list.Object)
	assert.NotEmpty(t, list.Data, "the announcements list should hold at least one row")
}

func TestCovMessagingAnnouncements_ListQueryIsNoOp(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)
	title := uniqueName("e2e-cov-announce-qnoop")
	broadcastAnnouncement(t, user, title)
	findAnnouncement(t, user, title)

	// KNOWN GAP (do not fix): ListAnnouncementsRequest.Query is validated (max=500) and documented, but AnnouncementSvc.ListAnnouncements never forwards it into pb.ListAnnouncementsRequest, which has no query field at all. A syntactically valid but non-matching q must NOT filter the announcement out of the results — asserting otherwise would assert behavior the backend does not implement.
	list, _, err := user.GetList(announcementsPath, url.Values{"limit": {"100"}, "q": {uniqueName("nonmatching-token")}})
	require.NoError(t, err)
	var sawTitle bool
	for _, raw := range list.Data {
		if jsonField(parseJSON(raw), "title") == title {
			sawTitle = true
			break
		}
	}
	assert.True(t, sawTitle, "q is a documented-but-inert filter on this endpoint; a non-matching q must not remove the row")
}

// ── List query-param validation ─────────────────────────────────────

func TestCovMessagingAnnouncements_ListLimitValidation(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)

	status, body, err := user.GetListRaw(announcementsPath, url.Values{"limit": {"0"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assertErrorParam(t, errObj, "Limit")

	status, body, err = user.GetListRaw(announcementsPath, url.Values{"limit": {"1001"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj = requireErrorResponse(t, body, "", "invalid_request_error")
	assertErrorParam(t, errObj, "Limit")

	// Boundary values remain valid.
	status, _, err = user.GetListRaw(announcementsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)

	status, _, err = user.GetListRaw(announcementsPath, url.Values{"limit": {"1000"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
}

func TestCovMessagingAnnouncements_ListQueryLengthValidation(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)

	tooLong := ""
	for i := 0; i < 501; i++ {
		tooLong += "a"
	}
	status, body, err := user.GetListRaw(announcementsPath, url.Values{"q": {tooLong}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assertErrorParam(t, errObj, "Query")

	// Exactly at the boundary (500 chars) remains valid, even though (per the no-op gap above) it does not actually filter anything.
	atLimit := tooLong[:500]
	status, _, err = user.GetListRaw(announcementsPath, url.Values{"q": {atLimit}})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
}

func TestCovMessagingAnnouncements_UnknownIncludeRejected(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)

	status, body, err := user.GetListRaw(announcementsPath, url.Values{"limit": {"1"}, "include": {"bogus"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "include[]")
}

// ── Mark-action idempotency ─────────────────────────────────────────

func TestCovMessagingAnnouncements_MarkActionsIdempotent(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)
	title := uniqueName("e2e-cov-announce-idem")
	broadcastAnnouncement(t, user, title)
	id := jsonField(findAnnouncement(t, user, title), "id")

	// seen: calling twice (distinct idempotency keys, same underlying receipt upsert) must not error, and seen_at must not move on the repeat call — the repo layer's ON DUPLICATE KEY ... COALESCE makes this first-write-wins.
	seen1, err := user.PostFull(announcementsPath+"/"+id+"/actions/seen", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, seen1.StatusCode, seen1.Body)
	seenAt1 := jsonField(parseJSON(seen1.Body), "seen_at")
	require.NotEmpty(t, seenAt1)

	seen2, err := user.PostFull(announcementsPath+"/"+id+"/actions/seen", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, seen2.StatusCode, seen2.Body)
	seen2M := parseJSON(seen2.Body)
	assert.Equal(t, "seen", jsonField(seen2M, "status"))
	assert.Equal(t, seenAt1, jsonField(seen2M, "seen_at"), "repeat mark-seen must not move seen_at")

	// read: same pattern.
	read1, err := user.PostFull(announcementsPath+"/"+id+"/actions/read", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, read1.StatusCode, read1.Body)
	readAt1 := jsonField(parseJSON(read1.Body), "read_at")
	require.NotEmpty(t, readAt1)

	read2, err := user.PostFull(announcementsPath+"/"+id+"/actions/read", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, read2.StatusCode, read2.Body)
	read2M := parseJSON(read2.Body)
	assert.Equal(t, "read", jsonField(read2M, "status"))
	assert.Equal(t, readAt1, jsonField(read2M, "read_at"), "repeat mark-read must not move read_at")

	// dismiss: same pattern.
	dismiss1, err := user.PostFull(announcementsPath+"/"+id+"/actions/dismiss", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, dismiss1.StatusCode, dismiss1.Body)
	dismissedAt1 := jsonField(parseJSON(dismiss1.Body), "dismissed_at")
	require.NotEmpty(t, dismissedAt1)

	dismiss2, err := user.PostFull(announcementsPath+"/"+id+"/actions/dismiss", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, dismiss2.StatusCode, dismiss2.Body)
	dismiss2M := parseJSON(dismiss2.Body)
	assert.Equal(t, "dismissed", jsonField(dismiss2M, "status"))
	assert.Equal(t, dismissedAt1, jsonField(dismiss2M, "dismissed_at"), "repeat mark-dismiss must not move dismissed_at")
	// Earlier receipts survive across the repeat calls too.
	assert.Equal(t, seenAt1, jsonField(dismiss2M, "seen_at"))
	assert.Equal(t, readAt1, jsonField(dismiss2M, "read_at"))
}

// ── 404s ─────────────────────────────────────────────────────────────

// TestCovMessagingAnnouncements_MarkReadDismissNonexistentReturns404 extends the existing TestAnnouncements_MarkNonexistentReturns404 (which only covers the `seen` action) to the `read` and `dismiss` actions.
func TestCovMessagingAnnouncements_MarkReadDismissNonexistentReturns404(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)
	const bogusID = "an_doesnotexist0000000000"

	status, body, err := user.Post(announcementsPath+"/"+bogusID+"/actions/read", nil, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, "marking read on a nonexistent announcement should 404: %s", string(body))

	status, body, err = user.Post(announcementsPath+"/"+bogusID+"/actions/dismiss", nil, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, "marking dismissed on a nonexistent announcement should 404: %s", string(body))
}

func TestCovMessagingAnnouncements_RetrieveMalformedIDReturns404(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)

	// A well-formed id with the wrong resource-type prefix should 404, not 500 or leak a mismatched row.
	resp, err := user.GetFull(announcementsPath+"/xx_wrongprefix0000000000", nil)
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode, "a wrong-prefix id should 404: %s", string(resp.Body))
}

// ── Auth ─────────────────────────────────────────────────────────────

func TestCovMessagingAnnouncements_Unauthenticated(t *testing.T) {
	t.Parallel()
	anon := apiClient.WithBearerToken("", SeedAccountID)

	status, body, err := anon.GetListRaw(announcementsPath, nil)
	require.NoError(t, err)
	requireStatus(t, 401, status, body)
	requireErrorResponse(t, body, "invalid_credentials", "invalid_request_error")

	getResp, err := anon.GetFull(announcementsPath+"/an_doesnotexist0000000000", nil)
	require.NoError(t, err)
	assert.Equal(t, 401, getResp.StatusCode, "unauthenticated retrieve should 401: %s", string(getResp.Body))

	postResp, err := anon.PostFull(announcementsPath+"/an_doesnotexist0000000000/actions/seen", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 401, postResp.StatusCode, "unauthenticated mark-seen should 401: %s", string(postResp.Body))
}

// ── Cross-account isolation ─────────────────────────────────────────

// TestCovMessagingAnnouncements_CrossAccountIsolation proves the DB-level account_id filter in GetActiveAnnouncementByID actually holds end to end: a user authenticated against a completely different account cannot read or mark-seen a scope=account announcement that belongs to SeedAccountID. It should 404 (not found), never leak the row or its content.
func TestCovMessagingAnnouncements_CrossAccountIsolation(t *testing.T) {
	t.Parallel()
	owner := notifUserClient(t)
	title := uniqueName("e2e-cov-announce-xacct")
	broadcastAnnouncement(t, owner, title)
	id := jsonField(findAnnouncement(t, owner, title), "id")

	tenantB := apiClient.WithBearerToken(SeedTenantBAPIKey, SeedTenantBAccountID)

	getResp, err := tenantB.GetFull(announcementsPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getResp.StatusCode,
		"a different account must not be able to retrieve another account's announcement: %s", string(getResp.Body))

	postResp, err := tenantB.PostFull(announcementsPath+"/"+id+"/actions/seen", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, postResp.StatusCode,
		"a different account must not be able to mark-seen another account's announcement: %s", string(postResp.Body))

	// It also must not surface in tenant B's own list.
	list, _, err := tenantB.GetList(announcementsPath, url.Values{"limit": {"100"}})
	require.NoError(t, err)
	for _, raw := range list.Data {
		assert.NotEqual(t, id, jsonField(parseJSON(raw), "id"), "another account's announcement must not leak into tenant B's list")
	}
}

// ── Bootstrap request-body validation (POST /v1/messaging/notifications) ──

// TestCovMessagingAnnouncements_BroadcastFieldValidation covers the remaining per-field validation of the bootstrap send endpoint used to materialize announcements: invalid priority enum, missing required title, missing required category, and a blank (as opposed to omitted) optional body.
func TestCovMessagingAnnouncements_BroadcastFieldValidation(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)

	// priority not in the enum.
	status, body, err := user.Post(notificationsPath, map[string]any{
		"category": "system.broadcast",
		"target":   map[string]any{"type": "account", "id": SeedAccountID},
		"title":    uniqueName("e2e-cov-announce-badprio"),
		"priority": "critical",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	// title omitted.
	status, body, err = user.Post(notificationsPath, map[string]any{
		"category": "system.broadcast",
		"target":   map[string]any{"type": "account", "id": SeedAccountID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	// category omitted.
	status, body, err = user.Post(notificationsPath, map[string]any{
		"target": map[string]any{"type": "account", "id": SeedAccountID},
		"title":  uniqueName("e2e-cov-announce-nocat"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	// body explicitly blank (as opposed to omitted) — field.Optional[string] rejects a present-but-empty value.
	status, body, err = user.Post(notificationsPath, map[string]any{
		"category": "system.broadcast",
		"target":   map[string]any{"type": "account", "id": SeedAccountID},
		"title":    uniqueName("e2e-cov-announce-blankbody"),
		"body":     "",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}
