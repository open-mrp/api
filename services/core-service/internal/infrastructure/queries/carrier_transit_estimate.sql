-- Resolves both transit sources for one lane in a single round trip, so stamping a commitment adds one seek rather than two. The service level always exists (it is the join root); the lane estimate is the optional half, and a NULL lane_transit_days means it has never been warmed.
-- Staleness is judged in Go rather than filtered here: the cutoff is a policy constant, and a query that hid stale rows would make "expired" indistinguishable from "never warmed" when deciding whether to fall back.
-- name: ResolveCarrierTransit :one
SELECT
    carrier_option.default_transit_days,
    carrier_transit_estimate.transit_days AS lane_transit_days,
    carrier_transit_estimate.source_code AS lane_source_code,
    carrier_transit_estimate.refreshed_at AS lane_refreshed_at
FROM carrier_option
LEFT JOIN carrier_transit_estimate
    ON carrier_transit_estimate.carrier_option_id = carrier_option.id
    AND carrier_transit_estimate.account_id = sqlc.arg('account_id')
    AND carrier_transit_estimate.origin_country = sqlc.arg('origin_country')
    AND carrier_transit_estimate.origin_postal = sqlc.arg('origin_postal')
    AND carrier_transit_estimate.dest_country = sqlc.arg('dest_country')
    AND carrier_transit_estimate.dest_postal = sqlc.arg('dest_postal')
WHERE carrier_option.id = sqlc.arg('carrier_option_id')
-- Same value as account_id above, under its own name because carrier_option.account_id is nullable (system-owned service levels) and sqlc will not infer one param as both nullable and not.
AND (carrier_option.account_id = sqlc.narg('option_account_id') OR carrier_option.account_id IS NULL);

-- Writes a harvested lane estimate, leaving operator-entered rows alone. The guard is in the statement rather than a read-then-write so two warms racing on the same lane cannot interleave and clobber a manual answer between them.
-- name: UpsertCarrierTransitEstimate :exec
INSERT INTO carrier_transit_estimate (
    id,
    account_id,
    carrier_option_id,
    origin_country,
    origin_postal,
    dest_country,
    dest_postal,
    transit_days,
    source_code,
    refreshed_at,
    created_at,
    updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('account_id'),
    sqlc.arg('carrier_option_id'),
    sqlc.arg('origin_country'),
    sqlc.arg('origin_postal'),
    sqlc.arg('dest_country'),
    sqlc.arg('dest_postal'),
    sqlc.arg('transit_days'),
    sqlc.arg('source_code'),
    NOW(3),
    NOW(3),
    NOW(3)
)
ON DUPLICATE KEY UPDATE
    transit_days = IF(carrier_transit_estimate.source_code = 'manual', carrier_transit_estimate.transit_days, VALUES(transit_days)),
    refreshed_at = IF(carrier_transit_estimate.source_code = 'manual', carrier_transit_estimate.refreshed_at, NOW(3)),
    updated_at = NOW(3);
