-- name: CreateMessagingGroup :exec
INSERT INTO messaging_group (
    id, account_id, name, created_by_account_user_id, created_at, updated_at
) VALUES (?, ?, ?, ?, NOW(3), NOW(3));

-- name: GetMessagingGroupByID :one
SELECT * FROM messaging_group
WHERE id = ? AND account_id = ?;

-- name: ListMessagingGroups :many
-- All rosters in an account, most-recently-updated first.
SELECT * FROM messaging_group
WHERE account_id = ?
ORDER BY updated_at DESC, id DESC;

-- name: UpdateMessagingGroupName :execrows
UPDATE messaging_group
SET name = ?, updated_at = NOW(3)
WHERE id = ? AND account_id = ?;

-- name: TouchMessagingGroup :exec
-- Bumps updated_at so the roster re-sorts to the top after a membership change.
UPDATE messaging_group
SET updated_at = NOW(3)
WHERE id = ? AND account_id = ?;

-- name: DeleteMessagingGroup :execrows
DELETE FROM messaging_group
WHERE id = ? AND account_id = ?;

-- name: CreateMessagingGroupMember :exec
INSERT INTO messaging_group_member (
    id, group_id, account_id, member_type, account_user_id, agent_config_id, created_at
) VALUES (?, ?, ?, ?, ?, ?, NOW(3));

-- name: ListMessagingGroupMembers :many
SELECT * FROM messaging_group_member
WHERE group_id = ?
ORDER BY created_at ASC, id ASC;

-- name: DeleteMessagingGroupMembers :exec
-- Removes every member of a roster (used before re-seeding, and on group delete).
DELETE FROM messaging_group_member
WHERE group_id = ?;

-- name: DeleteMessagingGroupMemberByID :execrows
DELETE FROM messaging_group_member
WHERE id = ? AND group_id = ?;

-- name: ClearConversationGroup :exec
-- Detaches every conversation that was seeded from a roster when that roster is deleted. Membership
-- is unaffected (snapshotted into conversation_participant); only the provenance link is nulled.
UPDATE conversation
SET group_id = NULL, updated_at = NOW(3)
WHERE account_id = ? AND group_id = ?;
