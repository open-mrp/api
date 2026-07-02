-- name: InsertHubspotCompanyReview :exec
INSERT INTO hubspot_company_review (
    id,
    job_id,
    account_id,
    augno_customer_id,
    customer_name,
    candidate_matches,
    status,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('job_id'),
    sqlc.arg('account_id'),
    sqlc.arg('augno_customer_id'),
    sqlc.arg('customer_name'),
    sqlc.narg('candidate_matches'),
    sqlc.arg('status'),
    NOW(3),
    NOW(3)
);

-- name: GetHubspotCompanyReview :one
SELECT
    id,
    job_id,
    account_id,
    augno_customer_id,
    customer_name,
    candidate_matches,
    status,
    resolution,
    resolved_hubspot_id,
    created_at,
    updated_at
FROM hubspot_company_review
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: ListHubspotCompanyReviewsForJob :many
SELECT
    id,
    job_id,
    account_id,
    augno_customer_id,
    customer_name,
    candidate_matches,
    status,
    resolution,
    resolved_hubspot_id,
    created_at,
    updated_at
FROM hubspot_company_review
WHERE job_id = sqlc.arg('job_id')
AND (
    sqlc.narg('status') IS NULL
    OR status = sqlc.narg('status')
)
ORDER BY created_at ASC, id ASC;

-- name: CountPendingHubspotCompanyReviews :one
SELECT COUNT(*)
FROM hubspot_company_review
WHERE job_id = sqlc.arg('job_id')
AND status = 'pending';

-- name: ResolveHubspotCompanyReview :execresult
UPDATE hubspot_company_review SET
    status = sqlc.arg('status'),
    resolution = sqlc.narg('resolution'),
    resolved_hubspot_id = sqlc.narg('resolved_hubspot_id'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');
