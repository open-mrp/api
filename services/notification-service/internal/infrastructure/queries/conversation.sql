-- name: CreateConversation :exec
INSERT INTO conversation (
    id, account_id, type, audience, title, group_id, topic_resource_type, topic_resource_id,
    created_by_participant_id, next_sequence, metadata, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, NOW(3), NOW(3));

-- name: GetConversationByID :one
SELECT * FROM conversation
WHERE id = ? AND account_id = ?;

-- name: GetCustomerSupportConversation :one
-- The portal support case in a vendor account for a given customer account (dedup target for
-- "contact support"). Resolved via audience=customer and the customer participant's relation_account_id.
SELECT c.* FROM conversation c
JOIN conversation_participant p ON p.conversation_id = c.id
WHERE c.account_id = ? AND c.audience = 'customer'
  AND p.participant_type = 'customer' AND p.relation_account_id = ?
LIMIT 1;

-- name: CreateCustomerSupportConversation :exec
-- The topic anchors the case to the customer record (object=customer, id=customer account), so the
-- customer's name is derivable from the thread without a customer actor type.
INSERT INTO conversation (
    id, account_id, type, audience, workflow_status, topic_resource_type, topic_resource_id, next_sequence, created_at, updated_at
) VALUES (?, ?, 'group', 'customer', 'new', 'customer', ?, 1, NOW(3), NOW(3));

-- name: CreateCustomerParticipant :exec
-- relation_account_id is the routing/dedup key (one support thread per customer account); account_user_id
-- is the customer-account user who opened the case, surfaced as the participant's user actor.
INSERT INTO conversation_participant (
    id, conversation_id, account_id, participant_type, relation_account_id, account_user_id, role, membership, joined_at, created_at, updated_at
) VALUES (?, ?, ?, 'customer', ?, ?, 'member', 'active', NOW(3), NOW(3), NOW(3));

-- name: GetParticipantByRelationAccount :one
SELECT * FROM conversation_participant
WHERE conversation_id = ? AND relation_account_id = ?;

-- name: AdvanceReadCursorByID :exec
-- Advances a participant's read cursor by participant id (used for customer participants, which are
-- keyed/looked up by relation_account_id rather than account_user_id). Forward-only.
UPDATE conversation_participant
SET last_read_sequence = ?, last_read_message_id = ?, last_read_at = NOW(3), updated_at = NOW(3)
WHERE id = ? AND last_read_sequence < ?;

-- name: LockConversationSequence :one
-- Locks the conversation row and returns the next sequence to assign. Must run inside the
-- send transaction; the matching UPDATE increments next_sequence under the same lock.
SELECT next_sequence FROM conversation
WHERE id = ? AND account_id = ?
FOR UPDATE;

-- name: AdvanceConversationAfterMessage :exec
UPDATE conversation
SET next_sequence = next_sequence + 1,
    last_message_id = ?,
    last_message_at = ?,
    updated_at = NOW(3)
WHERE id = ?;

-- name: ListConversationsForUser :many
-- Lists the caller's active, non-hidden conversations (most-recently-active first), joined with
-- the caller's participant row so the service can compute per-conversation unread.
-- Keyset paginates on (c.last_message_at DESC, c.id DESC) over the parent `conversation`.
-- ACCEPTED RESIDUAL (performant-list-endpoint-patterns.md, residual-documentation):
-- the selective scope (membership: p.account_user_id + p.membership='active' + p.hidden_at IS NULL)
-- lives on the joined `conversation_participant`, while the keyset sort key lives on the
-- parent `conversation`. For a user whose memberships are sparse relative to the account's
-- conversation volume, the optimizer can drive the keyset off `conversation` and probe
-- participant per row (a rare-match stall) rather than driving membership-first. This is
-- accepted rather than denormalizing the keyset sort columns (last_message_at, id) onto
-- `conversation_participant` and driving from a membership-pinned index. See notes for the
-- membership-first subquery rewrite if this list becomes hot.
SELECT
    c.*,
    p.id AS participant_id,
    p.last_read_sequence AS participant_last_read_sequence,
    p.hidden_at AS participant_hidden_at
FROM conversation c
JOIN conversation_participant p
    ON p.conversation_id = c.id AND p.account_user_id = sqlc.arg('account_user_id')
WHERE c.account_id = sqlc.arg('account_id')
  AND p.membership = 'active'
  -- Internal conversations only. External customer-facing cases (audience='customer': portal support
  -- and email-bridged threads) live in the support inbox, not a staff member's Messages list.
  AND c.audience = 'internal'
  AND (
    (sqlc.narg('status') IS NULL OR sqlc.narg('status') = 'active') AND p.hidden_at IS NULL
    OR sqlc.narg('status') = 'hidden' AND p.hidden_at IS NOT NULL
  )
  AND (sqlc.narg('type') IS NULL OR c.type = sqlc.narg('type'))
  AND (
    sqlc.narg('cursor_last_message_at') IS NULL
    OR c.last_message_at < sqlc.narg('cursor_last_message_at')
    OR (c.last_message_at = sqlc.narg('cursor_last_message_at') AND c.id < sqlc.narg('cursor_id'))
  )
ORDER BY c.last_message_at DESC, c.id DESC
LIMIT ?;

-- name: GetDMConversationID :one
SELECT conversation_id FROM conversation_dm_key
WHERE account_id = ? AND dm_key = ?;

-- name: CreateDMKey :exec
INSERT INTO conversation_dm_key (account_id, dm_key, conversation_id)
VALUES (?, ?, ?);

-- name: CreateParticipant :exec
INSERT INTO conversation_participant (
    id, conversation_id, account_id, participant_type, account_user_id, agent_config_id,
    role, membership, joined_at, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, 'active', NOW(3), NOW(3), NOW(3));

-- name: GetParticipant :one
SELECT * FROM conversation_participant
WHERE conversation_id = ? AND account_user_id = ?;

-- name: ListParticipants :many
SELECT * FROM conversation_participant
WHERE conversation_id = ? AND membership = 'active';

-- name: ListAllParticipants :many
-- Every participant regardless of state. Used to resolve historical message authorship: a member
-- who has left or been removed still authored their past messages, so their participant row must
-- remain resolvable (active-only ListParticipants would drop them and surface "Unknown").
SELECT * FROM conversation_participant
WHERE conversation_id = ?;

-- name: AdvanceReadCursor :exec
-- Advances the caller's read cursor forward only (never rewinds).
UPDATE conversation_participant
SET last_read_sequence = ?, last_read_message_id = ?, last_read_at = NOW(3), updated_at = NOW(3)
WHERE conversation_id = ? AND account_user_id = ? AND last_read_sequence < ?;

-- name: UpdateConversation :exec
-- Partial update: only the provided (non-null) fields change.
UPDATE conversation
SET title = CASE WHEN sqlc.arg('clear_title') = true THEN NULL ELSE COALESCE(sqlc.narg('title'), title) END,
    is_archived = COALESCE(sqlc.narg('is_archived'), is_archived),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: SetConversationLegalHold :exec
-- Sets or clears the legal-hold flag, which exempts the conversation from the reaper and redaction.
UPDATE conversation
SET legal_hold = sqlc.arg('legal_hold'), updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: RedactConversationMessages :execrows
-- GDPR redaction: clears the body/preview of every non-deleted message in the conversation while
-- keeping the row as an audit shell. The row is left non-tombstoned so the reaper retains the shell.
UPDATE message
SET body = NULL, preview = NULL, edited_at = NOW(3), updated_at = NOW(3)
WHERE conversation_id = sqlc.arg('conversation_id') AND deleted_at IS NULL;

-- name: ListConversationAttachments :many
-- All live attachments across a conversation's messages, for S3 deletion during redaction.
SELECT a.id AS id, a.s3_key AS s3_key
FROM message_attachment a
JOIN message m ON m.id = a.message_id
WHERE m.conversation_id = ? AND a.deleted_at IS NULL;

-- name: GetParticipantByID :one
SELECT * FROM conversation_participant
WHERE id = ? AND conversation_id = ?;

-- name: CreateAgentParticipant :exec
INSERT INTO conversation_participant (
    id, conversation_id, account_id, participant_type, agent_config_id,
    role, membership, agent_trigger_policy, agent_trigger_keywords, joined_at, created_at, updated_at
) VALUES (?, ?, ?, 'agent', ?, 'member', 'active', ?, ?, NOW(3), NOW(3), NOW(3));

-- name: GetParticipantByAgentConfigID :one
SELECT * FROM conversation_participant
WHERE conversation_id = ? AND agent_config_id = ?;

-- name: ReactivateAgentParticipant :exec
UPDATE conversation_participant
SET membership = 'active', agent_trigger_policy = ?, agent_trigger_keywords = ?, updated_at = NOW(3)
WHERE id = ?;

-- name: SetParticipantMembershipByID :exec
UPDATE conversation_participant
SET membership = ?, updated_at = NOW(3)
WHERE id = ? AND conversation_id = ?;

-- name: SetParticipantRole :exec
UPDATE conversation_participant
SET role = ?, updated_at = NOW(3)
WHERE conversation_id = ? AND account_user_id = ?;

-- name: SetParticipantMembership :exec
UPDATE conversation_participant
SET membership = ?, updated_at = NOW(3)
WHERE conversation_id = ? AND account_user_id = ?;

-- name: LeaveConversation :exec
UPDATE conversation_participant
SET membership = 'left', hidden_at = NOW(3), updated_at = NOW(3)
WHERE conversation_id = ? AND account_user_id = ?;

-- name: HideConversation :exec
UPDATE conversation_participant
SET hidden_at = NOW(3), updated_at = NOW(3)
WHERE conversation_id = ? AND account_user_id = ?;

-- name: UnhideConversation :exec
UPDATE conversation_participant
SET hidden_at = NULL, updated_at = NOW(3)
WHERE conversation_id = ? AND account_user_id = ? AND membership = 'active';

-- name: ReactivateParticipant :exec
UPDATE conversation_participant
SET membership = 'active', hidden_at = NULL, role = ?, updated_at = NOW(3)
WHERE conversation_id = ? AND account_user_id = ?;

-- name: SetParticipantNotifications :exec
UPDATE conversation_participant
SET notifications = ?, muted_until = sqlc.narg('muted_until'), updated_at = NOW(3)
WHERE conversation_id = ? AND account_user_id = ?;

-- name: BindConversationInbox :exec
-- Marks a conversation as bridged to an email inbox: inbound mail threads onto it and outbound
-- replies route through the inbox identity.
UPDATE conversation
SET email_inbox_id = ?, email_external_address = ?, updated_at = NOW(3)
WHERE id = ? AND account_id = ?;

-- name: SetConversationAudienceCustomer :exec
-- Promotes a conversation to an external customer-facing case (used when an inbound email opens a
-- new thread). Seeds the triage status if it was not already an external case.
UPDATE conversation
SET audience = 'customer',
    workflow_status = COALESCE(workflow_status, 'new'),
    updated_at = NOW(3)
WHERE id = ? AND account_id = ?;

-- name: UpdateConversationWorkflowStatus :exec
-- Sets the customer-service triage lane (new|open|waiting_internal|waiting_external|needs_approval|resolved).
UPDATE conversation
SET workflow_status = sqlc.arg('workflow_status'), updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: AssignConversation :exec
-- Assigns (or clears, with NULLs) the owning user or team for a customer-service case. The assignee is
-- a single polymorphic reference (resource_type, resource_id) — an account_user or an account_group.
UPDATE conversation
SET assignee_resource_type = sqlc.narg('assignee_resource_type'),
    assignee_resource_id = sqlc.narg('assignee_resource_id'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: ListSupportInbox :many
-- The customer-service inbox: external (audience='customer') cases for triage, filtered by status,
-- assignee (a single polymorphic user-or-team reference), and archived state, keyset-paginated by
-- (last_message_at DESC, id DESC). Served sort-free by conversation_inbox_idx.
SELECT * FROM conversation
WHERE account_id = sqlc.arg('account_id')
  AND audience = 'customer'
  AND is_archived = sqlc.arg('is_archived')
  AND (sqlc.narg('workflow_status') IS NULL OR workflow_status = sqlc.narg('workflow_status'))
  -- Resolved cases are done: keep them out of the default triage view. The repository sets hide_resolved
  -- only when no explicit status is requested and the archived view is off, so the "Resolved" lane
  -- (workflow_status='resolved') and the archived view still surface them.
  AND (sqlc.arg('hide_resolved') = FALSE OR workflow_status <> 'resolved')
  AND (sqlc.narg('assignee_resource_id') IS NULL OR assignee_resource_id = sqlc.narg('assignee_resource_id'))
  AND (sqlc.narg('unassigned') IS NULL OR assignee_resource_id IS NULL)
  AND (
    sqlc.narg('cursor_last_message_at') IS NULL
    OR last_message_at < sqlc.narg('cursor_last_message_at')
    OR (last_message_at = sqlc.narg('cursor_last_message_at') AND id < sqlc.narg('cursor_id'))
  )
ORDER BY last_message_at DESC, id DESC
LIMIT ?;

-- name: ListConversationsByResource :many
-- Every conversation linked to a business record, via either the primary topic anchor or a secondary
-- conversation_link row. Drives the "conversations on this object" panel from a record's page.
SELECT c.* FROM conversation c
WHERE c.account_id = sqlc.arg('account_id')
  AND (
    (c.topic_resource_type = sqlc.arg('topic_resource_type') AND c.topic_resource_id = sqlc.arg('topic_resource_id'))
    OR EXISTS (
      SELECT 1 FROM conversation_link l
      WHERE l.conversation_id = c.id
        AND l.resource_type = sqlc.arg('link_resource_type')
        AND l.resource_id = sqlc.arg('link_resource_id')
    )
  )
ORDER BY c.last_message_at DESC, c.id DESC
LIMIT ?;
