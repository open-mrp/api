-- name: CreateMessage :exec
-- Inserts a sent (timeline) message. status defaults to 'sent' (column default) since it is not in
-- the column list; draft/scheduled rows use CreateDraftMessage/CreateScheduledMessage instead.
INSERT INTO message (
    id, conversation_id, account_id, sequence, kind, visibility, channel,
    sender_participant_id, client_message_id,
    body, preview, subject, event_type, link_resource_type, link_resource_id,
    agent_run_id, reply_to_message_id, streaming_state, metadata, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(3), NOW(3));

-- name: CreateDraftMessage :exec
-- Inserts a customer-reply draft (status='draft', no timeline sequence yet). Held at visibility
-- 'internal' so it can never reach a customer payload before approval; promote-on-approve flips it to
-- 'external' and assigns a sequence. The conversation timeline is unaffected (no sequence advance).
INSERT INTO message (
    id, conversation_id, account_id, kind, status, visibility, channel, subject,
    sender_participant_id, agent_run_id, source_thread_message_id,
    body, preview, streaming_state, created_at, updated_at
) VALUES (?, ?, ?, 'chat', 'draft', 'internal', ?, ?, ?, ?, ?, ?, ?, 'complete', NOW(3), NOW(3));

-- name: CreateScheduledMessage :exec
-- Inserts a future-delivery message (status='scheduled', no timeline sequence yet). The leased worker
-- promotes it in place once scheduled_for arrives. visibility 'internal' (scheduled sends are team
-- messages); kept out of the timeline until promoted.
INSERT INTO message (
    id, conversation_id, account_id, kind, status, visibility,
    sender_participant_id, body, preview, channel, scheduled_for,
    streaming_state, created_at, updated_at
) VALUES (?, ?, ?, 'chat', 'scheduled', 'internal', ?, ?, ?, ?, ?, 'complete', NOW(3), NOW(3));

-- name: GetMessageByID :one
SELECT * FROM message
WHERE id = ? AND account_id = ?;

-- name: GetMessagesByIDs :many
-- Batch-loads messages by id, for hydrating last-message previews on the conversation
-- list without an N+1 fetch per row.
SELECT * FROM message
WHERE id IN (sqlc.slice('ids'));

-- name: GetMessageByClientID :one
-- Resolves an existing message by its client-supplied dedupe key (idempotent send).
SELECT * FROM message
WHERE conversation_id = ? AND client_message_id = ?;

-- name: UpdateMessageBody :exec
-- Edits a message body (author only — ownership is checked in the service).
UPDATE message
SET body = ?, preview = ?, edited_at = NOW(3), updated_at = NOW(3)
WHERE id = ? AND account_id = ?;

-- name: SetMessageStreamingBody :execrows
-- Streams a partial or final body into an in-flight agent reply, optionally flipping streaming_state to
-- 'complete'. Deliberately does NOT touch edited_at (streaming is not a human edit). Guarded on the row
-- still being 'streaming' so a stale/reordered patch can't clobber an already-completed or deleted
-- message — the rows-affected count lets the caller skip the realtime push when the update no-ops.
UPDATE message
SET body = ?, preview = ?, streaming_state = ?, updated_at = NOW(3)
WHERE id = ? AND account_id = ? AND streaming_state = 'streaming';

-- name: SoftDeleteMessage :exec
-- Tombstones a message: clears the body/preview and sets deleted_at.
UPDATE message
SET body = NULL, preview = NULL, deleted_at = NOW(3), updated_at = NOW(3)
WHERE id = ? AND account_id = ?;

-- name: ListMessages :many
-- Lists a conversation's sent (timeline) messages newest-first, keyset-paginated by sequence.
-- before_sequence loads older history (scroll up); after_sequence catches up after a reconnect.
-- status='sent' keeps draft/scheduled (NULL-sequence) rows out of the timeline.
SELECT * FROM message
WHERE conversation_id = sqlc.arg('conversation_id')
  AND status = 'sent'
  AND (sqlc.narg('before_sequence') IS NULL OR sequence < sqlc.narg('before_sequence'))
  AND (sqlc.narg('after_sequence') IS NULL OR sequence > sqlc.narg('after_sequence'))
ORDER BY sequence DESC
LIMIT ?;

-- name: ListMessagesVisible :many
-- Customer-viewer variant of ListMessages: excludes internal-only messages so internal team
-- discussion is never serialized into a customer payload. Served sort-free by message_conv_visibility_idx.
SELECT * FROM message
WHERE conversation_id = sqlc.arg('conversation_id')
  AND status = 'sent'
  AND visibility <> 'internal'
  AND (sqlc.narg('before_sequence') IS NULL OR sequence < sqlc.narg('before_sequence'))
  AND (sqlc.narg('after_sequence') IS NULL OR sequence > sqlc.narg('after_sequence'))
ORDER BY sequence DESC
LIMIT ?;

-- name: GetLastVisibleMessage :one
-- The most recent customer-visible message in a conversation, for the customer's last-message
-- preview (internal notes must not surface as the customer's preview/unread anchor).
SELECT * FROM message
WHERE conversation_id = ? AND status = 'sent' AND visibility <> 'internal' AND deleted_at IS NULL
ORDER BY sequence DESC
LIMIT 1;

-- name: CountVisibleMessagesAfter :one
-- Counts customer-visible messages after a read cursor, for the customer's unread badge
-- (internal notes do not bump a customer's unread count).
SELECT COUNT(*) FROM message
WHERE conversation_id = ? AND status = 'sent' AND visibility <> 'internal' AND deleted_at IS NULL
  AND sequence > sqlc.arg('after_sequence');

-- name: ListDraftMessagesByConversation :many
-- A case's customer-reply drafts (draft-lifecycle rows only — never sent timeline or scheduled rows),
-- newest first; optionally filtered to a single status (e.g. open drafts only).
SELECT * FROM message
WHERE conversation_id = sqlc.arg('conversation_id') AND account_id = sqlc.arg('account_id')
  AND status IN ('draft', 'rejected', 'superseded')
  AND (sqlc.narg('status') IS NULL OR status = sqlc.narg('status'))
ORDER BY created_at DESC, id DESC;

-- name: UpdateDraftMessageContent :exec
-- Edits a still-open draft's body/subject (rejected on non-draft rows by the WHERE guard).
UPDATE message
SET body = sqlc.arg('body'),
    subject = COALESCE(sqlc.narg('subject'), subject),
    preview = sqlc.narg('preview'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id') AND status = 'draft';

-- name: SetDraftMessageStatus :execrows
-- Transitions an open draft (draft->rejected). Guarded on status='draft' so only an open draft can
-- transition; the rows-affected count tells the caller whether the transition applied.
UPDATE message
SET status = sqlc.arg('status'), updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id') AND status = 'draft';

-- name: SupersedeDraftMessagesForThread :exec
-- When a thread changes, mark its still-open drafts superseded so a stale draft can't be sent.
UPDATE message
SET status = 'superseded', updated_at = NOW(3)
WHERE conversation_id = ? AND source_thread_message_id = ? AND status = 'draft';

-- name: PromoteDraftMessage :execrows
-- Approve-and-send: promotes a draft to a sent, customer-visible timeline message IN PLACE. Assigns the
-- locked sequence, flips visibility to 'external', and records the approver. The customer-facing
-- "Customer Service" branding is applied at read time, not persisted. Guarded on status='draft'
-- (compare-and-set) so a concurrent double-approve sends only once.
UPDATE message
SET status = 'sent',
    sequence = sqlc.arg('sequence'),
    visibility = 'external',
    kind = sqlc.arg('kind'),
    approved_by_account_user_id = sqlc.narg('approved_by_account_user_id'),
    preview = sqlc.narg('preview'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id') AND status = 'draft';

-- name: ListScheduledMessagesByConversation :many
-- The caller's own scheduled messages in a single conversation (soonest first). Ownership is resolved
-- via the sender participant's account_user_id, since sender_participant_id is per-conversation.
SELECT m.* FROM message m
JOIN conversation_participant p ON p.id = m.sender_participant_id
WHERE m.account_id = sqlc.arg('account_id')
  AND m.conversation_id = sqlc.arg('conversation_id')
  AND p.account_user_id = sqlc.arg('account_user_id')
  AND m.status = 'scheduled'
  AND m.deleted_at IS NULL
ORDER BY m.scheduled_for ASC, m.id ASC
LIMIT ?;

-- name: CancelScheduledMessageForUser :execrows
-- Cancels a scheduled message owned by the caller (resolved via the sender participant's
-- account_user_id). Returns rows affected so the service can distinguish "canceled" from "not
-- cancelable" (already sent/canceled, not owned, or not found).
UPDATE message m
JOIN conversation_participant p ON p.id = m.sender_participant_id
SET m.status = 'canceled', m.updated_at = NOW(3)
WHERE m.id = ? AND m.account_id = ? AND p.account_user_id = ? AND m.status = 'scheduled' AND m.deleted_at IS NULL;

-- name: ListDueScheduledMessages :many
-- Scheduled messages whose time has arrived and that are not already claimed by a worker. Served by
-- message_status_sched_idx (status, scheduled_for).
SELECT * FROM message
WHERE status = 'scheduled'
  AND scheduled_for <= NOW(3)
  AND deleted_at IS NULL
  AND locked_at IS NULL
ORDER BY scheduled_for ASC
LIMIT ?;

-- name: ClaimScheduledMessage :execrows
-- Claims a due scheduled message for delivery (compare-and-set on locked_at IS NULL) so a second pod
-- between lease handoffs cannot also deliver it. rows-affected=0 means another worker already claimed it.
UPDATE message
SET locked_at = NOW(3), lock_owner = sqlc.arg('lock_owner'), updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND status = 'scheduled' AND locked_at IS NULL;

-- name: PromoteScheduledMessage :execrows
-- Promotes a claimed scheduled message to a sent timeline message IN PLACE: assigns the locked sequence
-- and flips status to 'sent'. Guarded on status='scheduled' (compare-and-set) — the single-delivery
-- idempotency guard; rows-affected=0 means a prior tick already delivered it.
UPDATE message
SET status = 'sent', sequence = sqlc.arg('sequence'), locked_at = NULL, lock_owner = NULL, updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND status = 'scheduled';

-- name: MarkScheduledMessageFailed :exec
-- Records a delivery failure: bumps the attempt counter, releases the worker claim, and sets the new
-- status ('scheduled' to retry next tick, or a terminal 'failed'/'canceled').
UPDATE message
SET status = sqlc.arg('status'),
    last_error = sqlc.narg('last_error'),
    scheduled_attempts = scheduled_attempts + 1,
    locked_at = NULL,
    lock_owner = NULL,
    updated_at = NOW(3)
WHERE id = sqlc.arg('id');
