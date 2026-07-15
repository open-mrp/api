//go:build e2e

package api_test

import (
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Coverage for the messaging_messages group: the 4 message-lifecycle endpoints mounted under
// /v1/messaging/messages/{id} — UpdateDraft (PATCH), ApproveSendDraft, RejectDraft, and
// CancelScheduled (the three POST .../actions/*). This group has no create/list/get/delete of its
// own; every fixture is minted elsewhere (a customer-case draft via SendMessage mode=draft, or a
// scheduled message via SendMessage scheduled_at) and then driven through these 4 endpoints.

// covMessagingMessagesCreateDraft proposes a customer-reply draft on a customer case and returns the
// created (status=draft) Message resource.
func covMessagingMessagesCreateDraft(t *testing.T, c *Client, conversationID, body string, subject *string) map[string]any {
	t.Helper()
	payload := map[string]any{
		"mode":    "draft",
		"channel": "message",
		"body":    body,
	}
	if subject != nil {
		payload["subject"] = *subject
	}
	resp, err := c.PostFull(conversationsPath+"/"+conversationID+"/messages", payload, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	return parseJSON(resp.Body)
}

// covMessagingMessagesScheduleMessage schedules a message an hour out on the given conversation and
// returns the created (status=scheduled) Message resource.
func covMessagingMessagesScheduleMessage(t *testing.T, c *Client, conversationID string) map[string]any {
	t.Helper()
	resp, err := c.PostFull(conversationsPath+"/"+conversationID+"/messages", map[string]any{
		"body":              uniqueName("cov scheduled"),
		"client_message_id": uniqueName("cmid"),
		"scheduled_at":      time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	return parseJSON(resp.Body)
}

// ──────────────────────────────────────────────
// UpdateDraft (PATCH /v1/messaging/messages/{id})
// ──────────────────────────────────────────────

// Full-field coverage of UpdateDraft's response, plus the expandable-with/without-?include pair
// (conversation, attachments) — the only endpoint in this group with a request body rich enough to
// change both body and subject in one call.
func TestCovMessagingMessages_UpdateDraftAllFields(t *testing.T) {
	t.Parallel()
	seedSupportRoute(t)
	customer := getCustomerPortalClient()
	dane := chatUserClient(t)
	convID := openCustomerCase(t, customer)

	origSubject := uniqueName("original subject")
	draft := covMessagingMessagesCreateDraft(t, dane, convID, uniqueName("original body"), &origSubject)
	draftID := jsonField(draft, "id")
	assertIDFormat(t, draftID, "mg")

	newBody := uniqueName("revised body")
	newSubject := uniqueName("revised subject")
	resp, err := dane.PatchFull(withQuery("/v1/messaging/messages/"+draftID, url.Values{"include": {"conversation", "attachments"}}), map[string]any{
		"body":    newBody,
		"subject": newSubject,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)

	got := parseJSON(resp.Body)
	assert.Equal(t, draftID, jsonField(got, "id"))
	assertObjectField(t, got, "chat_message")
	assert.Equal(t, "chat", jsonField(got, "kind"), "a customer-reply draft is a chat-kind message")
	assert.Equal(t, "draft", jsonField(got, "status"), "editing a draft does not change its status")
	assert.Equal(t, "internal", jsonField(got, "visibility"), "an unsent draft is never customer-visible")

	conv := jsonObject(got, "conversation")
	require.NotNil(t, conv, "conversation should be present with ?include=conversation")
	assert.Equal(t, convID, jsonField(conv, "id"))
	assert.Equal(t, "conversation", jsonField(conv, "object"))

	assert.Equal(t, "0", jsonField(got, "sequence"), "a draft never carries a timeline sequence")
	assert.Equal(t, newBody, jsonField(got, "body"))
	assert.Equal(t, newSubject, jsonField(got, "subject"))
	assertNilField(t, got, "sender")
	assertNilField(t, got, "author")

	attachments := jsonObject(got, "attachments")
	require.NotNil(t, attachments, "attachments should be present (as an empty list envelope) with ?include=attachments")
	assertObjectField(t, attachments, "list")

	assertNilField(t, got, "reply_to")
	assertNilField(t, got, "resource")
	assert.Equal(t, "message", jsonField(got, "channel"))
	assertNilField(t, got, "scheduled_at")
	assertNilField(t, got, "agent_run")
	assert.Equal(t, "complete", jsonField(got, "streaming_state"))
	assertNilField(t, got, "client_message_id")
	assertNilField(t, got, "edited_at")
	assertNilField(t, got, "deleted_at")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	// Without ?include, the expandable sub-objects fall back to null.
	resp2, err := dane.PatchFull("/v1/messaging/messages/"+draftID, map[string]any{"body": newBody + " v2"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp2.StatusCode, resp2.Body)
	got2 := parseJSON(resp2.Body)
	assertNilField(t, got2, "conversation")
	assertNilField(t, got2, "attachments")
}

// CONFIRMED BUG (see prodBugSuspects #2/#5 in the task audit): UpdateDraftRequest.Subject is a bare
// field.Optional[string] with documented "PATCH omit = leave unchanged" semantics (per
// nullable-field-patterns.md), but UpdateReplyDraft passes the unset pointer straight through to a
// plain (non-COALESCE) SQL `SET subject = ?` — so omitting `subject` on a PATCH silently NULLs out a
// previously-set subject instead of preserving it. This asserts the CORRECT (documented) behavior and
// is expected to fail today.
func TestCovMessagingMessages_UpdateDraftOmittedSubjectNotPreserved(t *testing.T) {
	t.Parallel()
	seedSupportRoute(t)
	customer := getCustomerPortalClient()
	dane := chatUserClient(t)
	convID := openCustomerCase(t, customer)

	subject := uniqueName("keep me")
	draft := covMessagingMessagesCreateDraft(t, dane, convID, uniqueName("body v1"), &subject)
	draftID := jsonField(draft, "id")
	require.Equal(t, subject, jsonField(draft, "subject"), "sanity: the draft was created with a subject")

	status, body, err := dane.Patch("/v1/messaging/messages/"+draftID, map[string]any{"body": uniqueName("body v2")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, subject, jsonField(got, "subject"), "PATCH omitting subject must preserve the previously-set subject, not clear it")
}

// Every documented UpdateDraft failure mode: missing/empty body, an explicit null subject
// (field.Optional rejects null), an unknown message id, and the two wrong-state 409s (already-sent,
// scheduled-not-draft).
func TestCovMessagingMessages_UpdateDraftValidation(t *testing.T) {
	t.Parallel()
	seedSupportRoute(t)
	customer := getCustomerPortalClient()
	dane := chatUserClient(t)
	convID := openCustomerCase(t, customer)
	draft := covMessagingMessagesCreateDraft(t, dane, convID, uniqueName("validation body"), nil)
	draftID := jsonField(draft, "id")

	t.Run("missing body", func(t *testing.T) {
		status, body, err := dane.Patch("/v1/messaging/messages/"+draftID, map[string]any{}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
	})

	t.Run("empty body", func(t *testing.T) {
		status, body, err := dane.Patch("/v1/messaging/messages/"+draftID, map[string]any{"body": ""}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
	})

	t.Run("subject explicit null rejected", func(t *testing.T) {
		status, body, err := dane.Patch("/v1/messaging/messages/"+draftID, map[string]any{"body": uniqueName("x"), "subject": nil}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
		assertErrorParam(t, errObj, "subject")
	})

	t.Run("unknown message id 404s", func(t *testing.T) {
		status, body, err := dane.Patch("/v1/messaging/messages/mg_doesnotexist00000", map[string]any{"body": uniqueName("x")}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 404, status, body)
	})

	t.Run("already-sent message can no longer be edited", func(t *testing.T) {
		dm := createDM(t, dane, SeedAccountUser2ID)
		sent := sendMessage(t, dane, jsonField(dm, "id"), uniqueName("already sent"), uniqueName("cmid"))
		status, body, err := dane.Patch("/v1/messaging/messages/"+jsonField(sent, "id"), map[string]any{"body": uniqueName("x")}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 409, status, body)
	})

	t.Run("scheduled (not draft) message can no longer be edited", func(t *testing.T) {
		dm := createDM(t, dane, SeedAccountUser2ID)
		scheduled := covMessagingMessagesScheduleMessage(t, dane, jsonField(dm, "id"))
		status, body, err := dane.Patch("/v1/messaging/messages/"+jsonField(scheduled, "id"), map[string]any{"body": uniqueName("x")}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 409, status, body)
	})
}

// A relation actor (customer) can never reach this account-admin-only endpoint.
func TestCovMessagingMessages_UpdateDraftCustomerForbidden(t *testing.T) {
	t.Parallel()
	customer := getCustomerPortalClient()
	status, body, err := customer.Patch("/v1/messaging/messages/mg_doesnotexist00000", map[string]any{"body": uniqueName("x")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
}

func TestCovMessagingMessages_UpdateDraftUnknownFieldRejected(t *testing.T) {
	t.Parallel()
	seedSupportRoute(t)
	customer := getCustomerPortalClient()
	dane := chatUserClient(t)
	convID := openCustomerCase(t, customer)
	draft := covMessagingMessagesCreateDraft(t, dane, convID, uniqueName("unknown field body"), nil)
	draftID := jsonField(draft, "id")
	path := "/v1/messaging/messages/" + draftID

	status, body, err := dane.Patch(path, map[string]any{
		"body":            uniqueName("x"),
		bogusE2EJSONField: "should be rejected",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assertJSONUnknownFieldRejected(t, "PATCH", path, status, body)
}

// Transport-level Idempotency-Key replay: the same key with a different PATCH body conflicts, matching
// the pattern in idempotency_test.go.
func TestCovMessagingMessages_UpdateDraftIdempotencyKeyConflict(t *testing.T) {
	t.Parallel()
	seedSupportRoute(t)
	customer := getCustomerPortalClient()
	dane := chatUserClient(t)
	convID := openCustomerCase(t, customer)
	draft := covMessagingMessagesCreateDraft(t, dane, convID, uniqueName("idem body"), nil)
	draftID := jsonField(draft, "id")
	path := "/v1/messaging/messages/" + draftID

	idemKey := newIdempotencyKey()
	status1, body1, err := dane.Patch(path, map[string]any{"body": uniqueName("idem body A")}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)

	status2, body2, err := dane.Patch(path, map[string]any{"body": uniqueName("idem body B")}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 400, status2, body2)
	requireErrorResponse(t, body2, "validation_failed", "idempotency_error")
}

// ──────────────────────────────────────────────
// ApproveSendDraft (POST .../actions/approve-send)
// ──────────────────────────────────────────────

// Full-field coverage of the happy path (with ?include=conversation), plus the highest-value new test
// in this group: a sequential retry with the SAME client_message_id after the first approve-send
// already completed does NOT idempotently return the sent message — it 409s (see prodBugSuspects #3,
// the doc comment's "Idempotent on client_message_id" claim only holds for a concurrent double-approve,
// not a sequential retry).
func TestCovMessagingMessages_ApproveSendDraftAllFieldsAndSequentialRetryConflict(t *testing.T) {
	t.Parallel()
	seedSupportRoute(t)
	customer := getCustomerPortalClient()
	dane := chatUserClient(t)
	convID := openCustomerCase(t, customer)

	draft := covMessagingMessagesCreateDraft(t, dane, convID, uniqueName("approve me"), nil)
	draftID := jsonField(draft, "id")

	cmid := uniqueName("cmid-approve")
	resp, err := dane.PostFull(withQuery("/v1/messaging/messages/"+draftID+"/actions/approve-send", url.Values{"include": {"conversation"}}), map[string]any{
		"client_message_id": cmid,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)

	got := parseJSON(resp.Body)
	assert.Equal(t, draftID, jsonField(got, "id"), "approve-send promotes the draft in place, it does not mint a new id")
	assertObjectField(t, got, "chat_message")
	assert.Equal(t, "chat", jsonField(got, "kind"))
	assert.Equal(t, "sent", jsonField(got, "status"))
	assert.Equal(t, "external", jsonField(got, "visibility"), "an approved draft is promoted to a customer-visible message")

	conv := jsonObject(got, "conversation")
	require.NotNil(t, conv, "conversation should be present with ?include=conversation")
	assert.Equal(t, convID, jsonField(conv, "id"))
	// NOTE: we deliberately do NOT assert conv.workflow_status here. Every cov test in this file
	// resolves openCustomerCase to the SAME conversation (support is deduped per customer, and the
	// portal client is a package singleton), so workflow_status is a globally-shared, mutable field.
	// approve-send correctly sets it to "waiting_external" (autoSetCaseWorkflow, unconditional), but
	// the gateway re-fetches the conversation for ?include= AFTER that write, and sibling t.Parallel()
	// tests (e.g. RejectDraft -> "waiting_internal") legitimately clobber it in that window. The
	// deterministic "approving a reply hands the ball to the customer" transition is covered by the
	// non-parallel TestExternalCase_AutoStatusFromActivity.

	seq, convErr := strconv.Atoi(jsonField(got, "sequence"))
	require.NoError(t, convErr)
	assert.Greater(t, seq, 0, "a sent (promoted) message carries a real timeline sequence")
	assert.Equal(t, "message", jsonField(got, "channel"))
	assert.Equal(t, "complete", jsonField(got, "streaming_state"))
	// CONFIRMED BUG (prodBugSuspects #3): the response never echoes the request's client_message_id
	// back, and ApproveAndSendReplyDraft never persists or looks it up by it either — the parameter is
	// required but functionally dead beyond the request struct, despite the endpoint doc comment
	// describing the operation as "Idempotent on client_message_id."
	assertNilField(t, got, "client_message_id")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	status2, body2, err := dane.Post("/v1/messaging/messages/"+draftID+"/actions/approve-send", map[string]any{
		"client_message_id": cmid,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status2, body2)
	requireErrorResponse(t, body2, "resource_conflict", "invalid_request_error")
}

func TestCovMessagingMessages_ApproveSendDraftValidation(t *testing.T) {
	t.Parallel()
	seedSupportRoute(t)
	customer := getCustomerPortalClient()
	dane := chatUserClient(t)
	convID := openCustomerCase(t, customer)
	draft := covMessagingMessagesCreateDraft(t, dane, convID, uniqueName("validation approve"), nil)
	draftID := jsonField(draft, "id")

	t.Run("missing client_message_id", func(t *testing.T) {
		status, body, err := dane.Post("/v1/messaging/messages/"+draftID+"/actions/approve-send", map[string]any{}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
	})

	t.Run("unknown message id 404s", func(t *testing.T) {
		status, body, err := dane.Post("/v1/messaging/messages/mg_doesnotexist00000/actions/approve-send", map[string]any{
			"client_message_id": uniqueName("cmid"),
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 404, status, body)
	})

	t.Run("an already-rejected draft can no longer be approved", func(t *testing.T) {
		other := covMessagingMessagesCreateDraft(t, dane, convID, uniqueName("reject then approve"), nil)
		otherID := jsonField(other, "id")
		rstatus, rbody, err := dane.Post("/v1/messaging/messages/"+otherID+"/actions/reject", map[string]any{}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, rstatus, rbody)

		status, body, err := dane.Post("/v1/messaging/messages/"+otherID+"/actions/approve-send", map[string]any{
			"client_message_id": uniqueName("cmid"),
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 409, status, body)
	})
}

func TestCovMessagingMessages_ApproveSendDraftCustomerForbidden(t *testing.T) {
	t.Parallel()
	customer := getCustomerPortalClient()
	status, body, err := customer.Post("/v1/messaging/messages/mg_doesnotexist00000/actions/approve-send", map[string]any{
		"client_message_id": uniqueName("cmid"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
}

// ──────────────────────────────────────────────
// RejectDraft (POST .../actions/reject)
// ──────────────────────────────────────────────

func TestCovMessagingMessages_RejectDraftHappyPathAndRepeatConflict(t *testing.T) {
	t.Parallel()
	seedSupportRoute(t)
	customer := getCustomerPortalClient()
	dane := chatUserClient(t)
	convID := openCustomerCase(t, customer)
	draft := covMessagingMessagesCreateDraft(t, dane, convID, uniqueName("reject happy"), nil)
	draftID := jsonField(draft, "id")

	resp, err := dane.PostFull(withQuery("/v1/messaging/messages/"+draftID+"/actions/reject", url.Values{"include": {"conversation"}}), map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	got := parseJSON(resp.Body)
	assert.Equal(t, draftID, jsonField(got, "id"))
	assertObjectField(t, got, "chat_message")
	assert.Equal(t, "rejected", jsonField(got, "status"))
	assert.Equal(t, "internal", jsonField(got, "visibility"), "a rejected draft was never sent, so it stays internal")
	assert.Equal(t, "chat", jsonField(got, "kind"))
	assert.Equal(t, "complete", jsonField(got, "streaming_state"))
	conv := jsonObject(got, "conversation")
	require.NotNil(t, conv, "conversation should be present with ?include=conversation")
	assert.Equal(t, convID, jsonField(conv, "id"))
	// NOTE: we deliberately do NOT assert conv.workflow_status here — see the same note on
	// ApproveSendDraftAllFields above. openCustomerCase dedups to one shared support conversation and
	// this file's cov tests run t.Parallel(), so workflow_status is a globally-mutable field a sibling
	// test can clobber in the window between reject's write and the gateway's ?include= re-fetch. Reject
	// correctly sets it to "waiting_internal" (autoSetCaseWorkflow, unconditional); the deterministic
	// "rejecting the only draft returns the case to the team's queue" transition is covered by the
	// non-parallel TestExternalCase_AutoStatusFromActivity.

	status2, body2, err := dane.Post("/v1/messaging/messages/"+draftID+"/actions/reject", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status2, body2)
	requireErrorResponse(t, body2, "resource_conflict", "invalid_request_error")
}

// CONFIRMED BUG (prodBugSuspects #4): unlike UpdateDraft and ApproveSendDraft — which both do an
// explicit GetByID 404 check before touching status — RejectDraft's SetDraftStatus is a bare
// compare-and-set with no distinct not-found signal, so an unknown message id currently returns 409
// ("This draft is no longer open."), not 404. This pins down the actual (inconsistent, surprising)
// ground truth rather than assuming the more-consistent 404.
func TestCovMessagingMessages_RejectDraftUnknownIDIs409(t *testing.T) {
	t.Parallel()
	dane := chatUserClient(t)
	status, body, err := dane.Post("/v1/messaging/messages/mg_doesnotexist00000/actions/reject", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
	requireErrorResponse(t, body, "resource_conflict", "invalid_request_error")
}

func TestCovMessagingMessages_RejectDraftCustomerForbidden(t *testing.T) {
	t.Parallel()
	customer := getCustomerPortalClient()
	status, body, err := customer.Post("/v1/messaging/messages/mg_doesnotexist00000/actions/reject", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
}

// ──────────────────────────────────────────────
// CancelScheduled (POST .../actions/cancel)
// ──────────────────────────────────────────────

// Full-field coverage of the cancel response (with and without ?include=conversation) — the sibling
// existing tests (messaging_scheduled_test.go / messaging_scheduled_extra_test.go) only ever check
// id/status.
func TestCovMessagingMessages_CancelScheduledAllFields(t *testing.T) {
	t.Parallel()
	dane := chatUserClient(t)
	dm := createDM(t, dane, SeedAccountUser2ID)
	convID := jsonField(dm, "id")

	scheduled := covMessagingMessagesScheduleMessage(t, dane, convID)
	id := jsonField(scheduled, "id")
	scheduledAt := jsonField(scheduled, "scheduled_at")
	require.NotEmpty(t, scheduledAt)

	resp, err := dane.PostFull(withQuery("/v1/messaging/messages/"+id+"/actions/cancel", url.Values{"include": {"conversation"}}), map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	got := parseJSON(resp.Body)

	assert.Equal(t, id, jsonField(got, "id"))
	assertObjectField(t, got, "chat_message")
	assert.Equal(t, "chat", jsonField(got, "kind"))
	assert.Equal(t, "canceled", jsonField(got, "status"))
	assert.Equal(t, "internal", jsonField(got, "visibility"))
	conv := jsonObject(got, "conversation")
	require.NotNil(t, conv, "conversation should be present with ?include=conversation")
	assert.Equal(t, convID, jsonField(conv, "id"))
	assert.Equal(t, "0", jsonField(got, "sequence"), "a canceled message never reaches the timeline")
	assertNilField(t, got, "subject")
	assert.Equal(t, "message", jsonField(got, "channel"))
	assert.Equal(t, scheduledAt, jsonField(got, "scheduled_at"), "scheduled_at round-trips unchanged through cancel")
	assertNilField(t, got, "agent_run")
	assert.Equal(t, "complete", jsonField(got, "streaming_state"))
	assertNilField(t, got, "client_message_id")
	assertNilField(t, got, "edited_at")
	assertNilField(t, got, "deleted_at")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	// Without ?include, conversation falls back to null.
	scheduled2 := covMessagingMessagesScheduleMessage(t, dane, convID)
	id2 := jsonField(scheduled2, "id")
	status2, body2, err := dane.Post("/v1/messaging/messages/"+id2+"/actions/cancel", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)
	assertNilField(t, parseJSON(body2), "conversation")
}

// Canceling a message that is not scheduled at all (a normal sent timeline message) is a 400
// conflict, not a 404 or 409.
func TestCovMessagingMessages_CancelScheduledNonScheduledConflict(t *testing.T) {
	t.Parallel()
	dane := chatUserClient(t)
	dm := createDM(t, dane, SeedAccountUser2ID)
	convID := jsonField(dm, "id")
	sent := sendMessage(t, dane, convID, uniqueName("not scheduled"), uniqueName("cmid"))
	sentID := jsonField(sent, "id")

	status, body, err := dane.Post("/v1/messaging/messages/"+sentID+"/actions/cancel", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
}

// CONFIRMED BUG (prodBugSuspects #1): canceling a scheduled message that exists in the caller's own
// account but is owned by a DIFFERENT account_user returns 400 ("...can no longer be canceled (status:
// scheduled).") rather than a 403/404 — and the message actively lies (the message plainly could be
// canceled, by its owner). This also leaks the existence and status of another user's scheduled
// message to a non-owner. Pinning down actual (surprising) behavior, not the arguably-more-correct one.
func TestCovMessagingMessages_CancelScheduledCrossUserConflict(t *testing.T) {
	t.Parallel()
	dane := chatUserClient(t)
	user2 := chatUser2Client(t)
	dm := createDM(t, dane, SeedAccountUser2ID)
	convID := jsonField(dm, "id")

	scheduled := covMessagingMessagesScheduleMessage(t, user2, convID)
	id := jsonField(scheduled, "id")

	status, body, err := dane.Post("/v1/messaging/messages/"+id+"/actions/cancel", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assert.Contains(t, errObj["message"], "status: scheduled", "the conflict message misleadingly reports the message as still cancelable")
}

// CancelScheduled does not call requireMessagingAdmin like the other 3 endpoints (it uses s.caller
// directly), but the gateway-level messaging:update permission check still blocks a customer-portal
// key before the service is ever reached.
func TestCovMessagingMessages_CancelScheduledCustomerForbidden(t *testing.T) {
	t.Parallel()
	customer := getCustomerPortalClient()
	status, body, err := customer.Post("/v1/messaging/messages/mg_doesnotexist00000/actions/cancel", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
}
