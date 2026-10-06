-- name: CreateAccountFollowup :exec
INSERT INTO account_followup (
        id,
        account_id,
        sandbox_account_id,
        user_id,
        account_name,
        registrant_name,
        registrant_email,
        registered_at,
        status,
        scheduled_for
    )
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: FindAccountFollowupByID :one
SELECT *
FROM account_followup
WHERE id = ?;

-- name: ListDueAccountFollowupIDs :many
-- Scheduled rows whose time has come, plus drafts abandoned mid-flight (a crashed consumer leaves the row in
-- drafting with no one to finish it). Both arms seek on (status, scheduled_for).
SELECT id
FROM account_followup
WHERE (
        status = 'scheduled'
        AND scheduled_for <= sqlc.arg(now)
    )
    OR (
        status = 'drafting'
        AND scheduled_for <= sqlc.arg(now)
        AND updated_at < sqlc.arg(stale_before)
    )
ORDER BY scheduled_for
LIMIT ?;

-- name: ClaimAccountFollowupForDraft :execrows
UPDATE account_followup
SET status = 'drafting',
    attempts = attempts + 1
WHERE id = sqlc.arg(id)
    AND (
        status = 'scheduled'
        OR (
            status = 'drafting'
            AND updated_at < sqlc.arg(stale_before)
        )
    );

-- name: SkipAccountFollowup :exec
UPDATE account_followup
SET status = 'skipped',
    skip_reason = ?
WHERE id = ?;

-- name: RescheduleAccountFollowup :exec
UPDATE account_followup
SET status = 'scheduled',
    scheduled_for = ?,
    last_error = ?
WHERE id = ?
    AND status = 'drafting';

-- name: FailAccountFollowup :exec
UPDATE account_followup
SET status = 'failed',
    last_error = ?
WHERE id = ?
    AND status = 'drafting';

-- name: SaveAccountFollowupDraft :execrows
UPDATE account_followup
SET status = 'pending_review',
    activity_summary = ?,
    llm_model = ?,
    prompt_version = ?,
    engagement = ?,
    internal_summary = ?,
    draft_subject = ?,
    draft_body = ?,
    review_token_hash = ?,
    review_token_expires_at = ?,
    drafted_at = ?,
    last_error = NULL
WHERE id = ?
    AND status = 'drafting';

-- name: ListAccountRequestActivity :many
-- Bounded by the window and LIMIT; seeks request_log_target_account_id_occurred_at_id_idx once per account.
SELECT target_account_id,
    method,
    normalized_route,
    status_code,
    occurred_at
FROM request_log
WHERE target_account_id IN (sqlc.arg(account_id), sqlc.arg(sandbox_account_id))
    AND occurred_at >= sqlc.arg(window_start)
    AND occurred_at < sqlc.arg(window_end)
ORDER BY occurred_at
LIMIT ?;

-- name: FindAccountFollowupByReviewTokenHash :one
SELECT *
FROM account_followup
WHERE review_token_hash = ?;

-- name: ApproveAccountFollowup :execrows
UPDATE account_followup
SET status = 'sent',
    final_subject = ?,
    final_body = ?,
    reviewed_at = sqlc.arg(now),
    sent_at = sqlc.arg(now)
WHERE id = sqlc.arg(id)
    AND status = 'pending_review'
    AND review_token_expires_at > sqlc.arg(now);

-- name: SkipReviewedAccountFollowup :execrows
UPDATE account_followup
SET status = 'skipped',
    skip_reason = ?,
    reviewed_at = sqlc.arg(now)
WHERE id = sqlc.arg(id)
    AND status = 'pending_review'
    AND review_token_expires_at > sqlc.arg(now);
