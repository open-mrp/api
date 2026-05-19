-- Production step DAG edges: table _parent_child_production_steps(A,B) uses A=downstream, B=upstream (Prisma).
-- ConnectSteps persists (target_id, source_id). Do not reinterpret as A=parent. docs/patterns/production-step-graph-patterns.md

-- name: GetConsumptionPartItemIDs :many
SELECT ci.id AS item_id
FROM consumption c
JOIN item ci ON c.item_id = ci.id
JOIN production_step ps ON c.production_step_id = ps.id
WHERE ps.id = sqlc.arg('step_id')
AND ci.item_type_code = 'part';

-- name: FlowGetProducedItemIDByStep :one
SELECT p.item_id FROM production p
WHERE p.production_step_id = sqlc.arg('step_id')
LIMIT 1;

-- name: FindStepsByProducedItem :many
SELECT ps.id
FROM production_step ps
JOIN production p ON p.production_step_id = ps.id
JOIN item i ON p.item_id = i.id
WHERE ps.account_id = sqlc.arg('account_id')
AND p.item_id = sqlc.arg('item_id')
AND i.deleted_at IS NULL;

-- name: FindStepsThatConsumeItem :many
SELECT ps.id
FROM production_step ps
JOIN consumption c ON c.production_step_id = ps.id
JOIN item i ON c.item_id = i.id
WHERE ps.account_id = sqlc.arg('account_id')
AND c.item_id = sqlc.arg('item_id')
AND i.deleted_at IS NULL;

-- name: ClearStepLinks :exec
DELETE FROM _parent_child_production_steps
WHERE A = sqlc.arg('step_id') OR B = sqlc.arg('step_id');

-- name: ConnectSteps :exec
-- Prisma _parent_child_production_steps: A = downstream step, B = upstream (parent) step (flow B → A).
INSERT INTO _parent_child_production_steps (A, B) VALUES (sqlc.arg('target_id'), sqlc.arg('source_id'));

-- name: FlowDisconnectSteps :exec
DELETE FROM _parent_child_production_steps
WHERE A = sqlc.arg('target_id') AND B = sqlc.arg('source_id');

-- name: FindSourceStepsByConsumption :many
SELECT ps.id
FROM production_step ps
JOIN _parent_child_production_steps pcps ON pcps.B = ps.id
WHERE pcps.A = sqlc.arg('target_step_id')
AND ps.account_id = sqlc.arg('account_id')
AND EXISTS (
    SELECT 1 FROM consumption c
    WHERE c.id = sqlc.arg('consumption_id')
    AND c.production_step_id = pcps.A
);

-- name: FindDownstreamStepByItem :one
SELECT ps.id
FROM production_step ps
JOIN _parent_child_production_steps pcps ON pcps.B = sqlc.arg('source_step_id') AND pcps.A = ps.id
JOIN production p ON p.production_step_id = ps.id
WHERE ps.account_id = sqlc.arg('account_id')
AND p.item_id = sqlc.arg('item_id')
LIMIT 1;

-- name: GetAllStepEdgesForAccount :many
-- Parent→child graph: upstream parent is B, downstream child is A.
SELECT pcps.B AS parent_step_id, pcps.A AS child_step_id
FROM _parent_child_production_steps pcps
JOIN production_step ps ON pcps.B = ps.id
WHERE ps.account_id = sqlc.arg('account_id');

-- name: ConnectStepsIdempotent :exec
-- Inserts a step connection, ignoring if it already exists.
INSERT IGNORE INTO _parent_child_production_steps (A, B)
VALUES (sqlc.arg('target_id'), sqlc.arg('source_id'));
