-- Parent/child step links: _parent_child_production_steps has A=downstream, B=upstream (see docs/patterns/production-step-graph-patterns.md).

-- name: GetProductionStep :one
SELECT
    ps.id,
    ps.name,
    p.id AS production_id,
    pi.id AS produced_item_id,
    pi.sku AS produced_item_sku,
    pq.id AS produced_quantity_id,
    pq.value AS produced_quantity_value,
    pu.id AS produced_unit_id,
    pu.abbreviation AS produced_unit_abbreviation,
    pu.unit_dimension_code AS produced_unit_type
FROM production_step ps
JOIN production p ON p.production_step_id = ps.id
JOIN item pi ON p.item_id = pi.id
JOIN quantity pq ON p.quantity_id = pq.id
JOIN unit pu ON pq.unit_id = pu.id
WHERE ps.id = sqlc.arg('id')
AND ps.account_id = sqlc.arg('account_id');

-- name: GetProductionStepConsumptions :many
SELECT
    c.id,
    ci.id AS consumed_item_id,
    ci.sku AS consumed_item_sku,
    ci.description AS consumed_item_description,
    ci.item_type_code AS consumed_item_type_code,
    cq.id AS consumption_quantity_id,
    cq.value AS consumption_quantity_value,
    cu.id AS consumption_unit_id,
    cu.abbreviation AS consumption_unit_abbreviation,
    cu.unit_dimension_code AS consumption_unit_type,
    wq.id AS waste_quantity_id,
    wq.value AS waste_quantity_value,
    wu.id AS waste_unit_id,
    wu.abbreviation AS waste_unit_abbreviation,
    wu.unit_dimension_code AS waste_unit_type,
    c.instructions,
    c.created_at,
    c.updated_at
FROM consumption c
JOIN item ci ON c.item_id = ci.id
JOIN quantity cq ON c.quantity_id = cq.id
JOIN unit cu ON cq.unit_id = cu.id
JOIN quantity wq ON c.waste_quantity_id = wq.id
JOIN unit wu ON wq.unit_id = wu.id
WHERE c.production_step_id = sqlc.arg('production_step_id');

-- name: IsProductionStepInAccount :one
SELECT COUNT(*) FROM production_step
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: CountProductionStepConsumptions :one
SELECT COUNT(*) FROM consumption WHERE production_step_id = sqlc.arg('production_step_id');

-- name: CountProductionStepPartConsumptions :one
SELECT COUNT(*) FROM consumption c
JOIN item i ON c.item_id = i.id
WHERE c.production_step_id = sqlc.arg('production_step_id')
AND i.item_type_code = 'part';

-- name: IsLastProductionStep :one
-- A = downstream; leaf steps have no outgoing edge where they are the upstream B.
SELECT CASE
    WHEN NOT EXISTS (
        SELECT 1 FROM _parent_child_production_steps WHERE B = sqlc.arg('id')
    ) THEN 1
    ELSE 0
END AS is_last;

-- name: IsInputOfProductionStep :one
-- Row (A,B) = (downstream, upstream); current receives from input when A = current and B = input.
SELECT COUNT(*) FROM _parent_child_production_steps
WHERE A = sqlc.arg('current_step_id') AND B = sqlc.arg('input_step_id');

-- name: FindProducedItemIDByStep :one
SELECT p.item_id FROM production p
WHERE p.production_step_id = sqlc.arg('production_step_id');

-- name: FindProducedUnitByStep :one
SELECT u.id, u.abbreviation, u.unit_dimension_code AS type
FROM production p
JOIN quantity q ON p.quantity_id = q.id
JOIN unit u ON q.unit_id = u.id
WHERE p.production_step_id = sqlc.arg('production_step_id');

-- name: FindStepIDByScanningStationAndItem :one
SELECT ps.id
FROM production_step ps
JOIN production p ON p.production_step_id = ps.id
WHERE ps.scanning_station_id = sqlc.arg('scanning_station_id')
AND ps.account_id = sqlc.arg('account_id')
AND p.item_id = sqlc.arg('item_id')
LIMIT 1;

-- name: FindStepByScanningStationAndItem :one
SELECT
    ps.id,
    ps.name,
    p.id AS production_id,
    pi.id AS produced_item_id,
    pi.sku AS produced_item_sku,
    pq.id AS produced_quantity_id,
    pq.value AS produced_quantity_value,
    pu.id AS produced_unit_id,
    pu.abbreviation AS produced_unit_abbreviation,
    pu.unit_dimension_code AS produced_unit_type
FROM production_step ps
JOIN production p ON p.production_step_id = ps.id
JOIN item pi ON p.item_id = pi.id
JOIN quantity pq ON p.quantity_id = pq.id
JOIN unit pu ON pq.unit_id = pu.id
WHERE ps.scanning_station_id = sqlc.arg('scanning_station_id')
AND ps.account_id = sqlc.arg('account_id')
AND p.item_id = sqlc.arg('item_id')
LIMIT 1;

-- name: GetProductionStepChildSteps :many
SELECT A AS child_step_id FROM _parent_child_production_steps
WHERE B = sqlc.arg('parent_step_id');

-- name: GetProductionStepScanningStationID :one
SELECT scanning_station_id FROM production_step
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: ListProductionStepsForward :many
SELECT
    ps.id,
    ps.name,
    ps.notes,
    ps.leveling_factor,
    ps.allowances,
    ps.department_id,
    ps.created_at,
    ps.updated_at,
    p.id AS production_id,
    pi.id AS produced_item_id,
    pi.sku AS produced_item_sku,
    pi.description AS produced_item_description,
    pi.item_type_code AS produced_item_type_code,
    pq.id AS produced_quantity_id,
    pq.value AS produced_quantity_value,
    pu.id AS produced_unit_id,
    pu.abbreviation AS produced_unit_abbreviation,
    pu.unit_dimension_code AS produced_unit_type,
    p.created_at AS production_created_at,
    p.updated_at AS production_updated_at,
    ss.id AS scanning_station_id,
    ss.name AS scanning_station_name,
    lr.id AS labor_rate_id, lr.value AS labor_rate_value,
    lrnu.id AS labor_rate_num_unit_id, lrnu.abbreviation AS labor_rate_num_unit_abbr, lrnu.unit_dimension_code AS labor_rate_num_unit_type,
    lrdu.id AS labor_rate_den_unit_id, lrdu.abbreviation AS labor_rate_den_unit_abbr, lrdu.unit_dimension_code AS labor_rate_den_unit_type,
    lt.id AS labor_time_id, lt.value AS labor_time_value,
    ltnu.id AS labor_time_num_unit_id, ltnu.abbreviation AS labor_time_num_unit_abbr, ltnu.unit_dimension_code AS labor_time_num_unit_type,
    ltdu.id AS labor_time_den_unit_id, ltdu.abbreviation AS labor_time_den_unit_abbr, ltdu.unit_dimension_code AS labor_time_den_unit_type,
    ohr.id AS overhead_rate_id, ohr.value AS overhead_rate_value,
    ohrnu.id AS overhead_rate_num_unit_id, ohrnu.abbreviation AS overhead_rate_num_unit_abbr, ohrnu.unit_dimension_code AS overhead_rate_num_unit_type,
    ohrdu.id AS overhead_rate_den_unit_id, ohrdu.abbreviation AS overhead_rate_den_unit_abbr, ohrdu.unit_dimension_code AS overhead_rate_den_unit_type
FROM production_step ps
JOIN production p ON p.production_step_id = ps.id
JOIN item pi ON p.item_id = pi.id
JOIN quantity pq ON p.quantity_id = pq.id
JOIN unit pu ON pq.unit_id = pu.id
LEFT JOIN scanning_station ss ON ps.scanning_station_id = ss.id
LEFT JOIN rate lr ON ps.labor_rate_id = lr.id
LEFT JOIN unit lrnu ON lr.numerator_unit_id = lrnu.id
LEFT JOIN unit lrdu ON lr.denominator_unit_id = lrdu.id
LEFT JOIN rate lt ON ps.labor_time_id = lt.id
LEFT JOIN unit ltnu ON lt.numerator_unit_id = ltnu.id
LEFT JOIN unit ltdu ON lt.denominator_unit_id = ltdu.id
LEFT JOIN rate ohr ON ps.overhead_rate_id = ohr.id
LEFT JOIN unit ohrnu ON ohr.numerator_unit_id = ohrnu.id
LEFT JOIN unit ohrdu ON ohr.denominator_unit_id = ohrdu.id
WHERE ps.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR MATCH(ps.name) AGAINST(sqlc.narg('search_query') IN BOOLEAN MODE)
)
AND (
    sqlc.arg('include_item_filter') = false
    OR p.item_id IN (sqlc.slice('item_ids'))
    OR EXISTS (
        SELECT 1 FROM consumption c
        WHERE c.production_step_id = ps.id
        AND c.item_id IN (sqlc.slice('item_ids'))
    )
)
AND (
    sqlc.arg('include_machine_filter') = false
    OR EXISTS (
        SELECT 1 FROM machine m
        WHERE m.production_step_id = ps.id
        AND m.id IN (sqlc.slice('machine_ids'))
    )
)
AND (
    sqlc.arg('include_scanning_station_filter') = false
    OR ps.scanning_station_id IN (sqlc.slice('scanning_station_ids'))
)
AND (
    sqlc.arg('include_input_step_filter') = false
    OR EXISTS (
        SELECT 1 FROM _parent_child_production_steps pcps
        WHERE pcps.A = ps.id
        AND pcps.B IN (sqlc.slice('input_step_ids'))
    )
)
AND (
    sqlc.arg('include_output_step_filter') = false
    OR EXISTS (
        SELECT 1 FROM _parent_child_production_steps pcps
        WHERE pcps.B = ps.id
        AND pcps.A IN (sqlc.slice('output_step_ids'))
    )
)
AND (sqlc.narg('start_date') IS NULL OR ps.created_at >= sqlc.narg('start_date'))
AND (sqlc.narg('end_date') IS NULL OR ps.created_at <= sqlc.narg('end_date'))
AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR ps.created_at < sqlc.narg('cursor_created_at')
    OR (ps.created_at = sqlc.narg('cursor_created_at') AND ps.id < sqlc.narg('cursor_id'))
)
ORDER BY ps.created_at DESC, ps.id DESC
LIMIT ?;

-- name: ListProductionStepsBackward :many
SELECT
    ps.id,
    ps.name,
    ps.notes,
    ps.leveling_factor,
    ps.allowances,
    ps.department_id,
    ps.created_at,
    ps.updated_at,
    p.id AS production_id,
    pi.id AS produced_item_id,
    pi.sku AS produced_item_sku,
    pi.description AS produced_item_description,
    pi.item_type_code AS produced_item_type_code,
    pq.id AS produced_quantity_id,
    pq.value AS produced_quantity_value,
    pu.id AS produced_unit_id,
    pu.abbreviation AS produced_unit_abbreviation,
    pu.unit_dimension_code AS produced_unit_type,
    p.created_at AS production_created_at,
    p.updated_at AS production_updated_at,
    ss.id AS scanning_station_id,
    ss.name AS scanning_station_name,
    lr.id AS labor_rate_id, lr.value AS labor_rate_value,
    lrnu.id AS labor_rate_num_unit_id, lrnu.abbreviation AS labor_rate_num_unit_abbr, lrnu.unit_dimension_code AS labor_rate_num_unit_type,
    lrdu.id AS labor_rate_den_unit_id, lrdu.abbreviation AS labor_rate_den_unit_abbr, lrdu.unit_dimension_code AS labor_rate_den_unit_type,
    lt.id AS labor_time_id, lt.value AS labor_time_value,
    ltnu.id AS labor_time_num_unit_id, ltnu.abbreviation AS labor_time_num_unit_abbr, ltnu.unit_dimension_code AS labor_time_num_unit_type,
    ltdu.id AS labor_time_den_unit_id, ltdu.abbreviation AS labor_time_den_unit_abbr, ltdu.unit_dimension_code AS labor_time_den_unit_type,
    ohr.id AS overhead_rate_id, ohr.value AS overhead_rate_value,
    ohrnu.id AS overhead_rate_num_unit_id, ohrnu.abbreviation AS overhead_rate_num_unit_abbr, ohrnu.unit_dimension_code AS overhead_rate_num_unit_type,
    ohrdu.id AS overhead_rate_den_unit_id, ohrdu.abbreviation AS overhead_rate_den_unit_abbr, ohrdu.unit_dimension_code AS overhead_rate_den_unit_type
FROM production_step ps
JOIN production p ON p.production_step_id = ps.id
JOIN item pi ON p.item_id = pi.id
JOIN quantity pq ON p.quantity_id = pq.id
JOIN unit pu ON pq.unit_id = pu.id
LEFT JOIN scanning_station ss ON ps.scanning_station_id = ss.id
LEFT JOIN rate lr ON ps.labor_rate_id = lr.id
LEFT JOIN unit lrnu ON lr.numerator_unit_id = lrnu.id
LEFT JOIN unit lrdu ON lr.denominator_unit_id = lrdu.id
LEFT JOIN rate lt ON ps.labor_time_id = lt.id
LEFT JOIN unit ltnu ON lt.numerator_unit_id = ltnu.id
LEFT JOIN unit ltdu ON lt.denominator_unit_id = ltdu.id
LEFT JOIN rate ohr ON ps.overhead_rate_id = ohr.id
LEFT JOIN unit ohrnu ON ohr.numerator_unit_id = ohrnu.id
LEFT JOIN unit ohrdu ON ohr.denominator_unit_id = ohrdu.id
WHERE ps.account_id = sqlc.arg('account_id')
AND (
    sqlc.narg('search_query') IS NULL
    OR MATCH(ps.name) AGAINST(sqlc.narg('search_query') IN BOOLEAN MODE)
)
AND (
    sqlc.arg('include_item_filter') = false
    OR p.item_id IN (sqlc.slice('item_ids'))
    OR EXISTS (
        SELECT 1 FROM consumption c
        WHERE c.production_step_id = ps.id
        AND c.item_id IN (sqlc.slice('item_ids'))
    )
)
AND (
    sqlc.arg('include_machine_filter') = false
    OR EXISTS (
        SELECT 1 FROM machine m
        WHERE m.production_step_id = ps.id
        AND m.id IN (sqlc.slice('machine_ids'))
    )
)
AND (
    sqlc.arg('include_scanning_station_filter') = false
    OR ps.scanning_station_id IN (sqlc.slice('scanning_station_ids'))
)
AND (
    sqlc.arg('include_input_step_filter') = false
    OR EXISTS (
        SELECT 1 FROM _parent_child_production_steps pcps
        WHERE pcps.A = ps.id
        AND pcps.B IN (sqlc.slice('input_step_ids'))
    )
)
AND (
    sqlc.arg('include_output_step_filter') = false
    OR EXISTS (
        SELECT 1 FROM _parent_child_production_steps pcps
        WHERE pcps.B = ps.id
        AND pcps.A IN (sqlc.slice('output_step_ids'))
    )
)
AND (sqlc.narg('start_date') IS NULL OR ps.created_at >= sqlc.narg('start_date'))
AND (sqlc.narg('end_date') IS NULL OR ps.created_at <= sqlc.narg('end_date'))
AND (
    ps.created_at > sqlc.arg('cursor_created_at')
    OR (ps.created_at = sqlc.arg('cursor_created_at') AND ps.id > sqlc.arg('cursor_id'))
)
ORDER BY ps.created_at ASC, ps.id ASC
LIMIT ?;

-- name: GetProductionStepFull :one
SELECT
    ps.id,
    ps.name,
    ps.notes,
    ps.leveling_factor,
    ps.allowances,
    ps.department_id,
    ps.created_at,
    ps.updated_at,
    p.id AS production_id,
    pi.id AS produced_item_id,
    pi.sku AS produced_item_sku,
    pi.description AS produced_item_description,
    pi.item_type_code AS produced_item_type_code,
    pq.id AS produced_quantity_id,
    pq.value AS produced_quantity_value,
    pu.id AS produced_unit_id,
    pu.abbreviation AS produced_unit_abbreviation,
    pu.unit_dimension_code AS produced_unit_type,
    p.created_at AS production_created_at,
    p.updated_at AS production_updated_at,
    ss.id AS scanning_station_id,
    ss.name AS scanning_station_name,
    lr.id AS labor_rate_id, lr.value AS labor_rate_value,
    lrnu.id AS labor_rate_num_unit_id, lrnu.abbreviation AS labor_rate_num_unit_abbr, lrnu.unit_dimension_code AS labor_rate_num_unit_type,
    lrdu.id AS labor_rate_den_unit_id, lrdu.abbreviation AS labor_rate_den_unit_abbr, lrdu.unit_dimension_code AS labor_rate_den_unit_type,
    lt.id AS labor_time_id, lt.value AS labor_time_value,
    ltnu.id AS labor_time_num_unit_id, ltnu.abbreviation AS labor_time_num_unit_abbr, ltnu.unit_dimension_code AS labor_time_num_unit_type,
    ltdu.id AS labor_time_den_unit_id, ltdu.abbreviation AS labor_time_den_unit_abbr, ltdu.unit_dimension_code AS labor_time_den_unit_type,
    ohr.id AS overhead_rate_id, ohr.value AS overhead_rate_value,
    ohrnu.id AS overhead_rate_num_unit_id, ohrnu.abbreviation AS overhead_rate_num_unit_abbr, ohrnu.unit_dimension_code AS overhead_rate_num_unit_type,
    ohrdu.id AS overhead_rate_den_unit_id, ohrdu.abbreviation AS overhead_rate_den_unit_abbr, ohrdu.unit_dimension_code AS overhead_rate_den_unit_type
FROM production_step ps
JOIN production p ON p.production_step_id = ps.id
JOIN item pi ON p.item_id = pi.id
JOIN quantity pq ON p.quantity_id = pq.id
JOIN unit pu ON pq.unit_id = pu.id
LEFT JOIN scanning_station ss ON ps.scanning_station_id = ss.id
LEFT JOIN rate lr ON ps.labor_rate_id = lr.id
LEFT JOIN unit lrnu ON lr.numerator_unit_id = lrnu.id
LEFT JOIN unit lrdu ON lr.denominator_unit_id = lrdu.id
LEFT JOIN rate lt ON ps.labor_time_id = lt.id
LEFT JOIN unit ltnu ON lt.numerator_unit_id = ltnu.id
LEFT JOIN unit ltdu ON lt.denominator_unit_id = ltdu.id
LEFT JOIN rate ohr ON ps.overhead_rate_id = ohr.id
LEFT JOIN unit ohrnu ON ohr.numerator_unit_id = ohrnu.id
LEFT JOIN unit ohrdu ON ohr.denominator_unit_id = ohrdu.id
WHERE ps.id = sqlc.arg('id')
AND ps.account_id = sqlc.arg('account_id');

-- name: GetProductionStepInputSteps :many
-- Upstream parents of step are B where this step is downstream A.
SELECT pcps.B AS id, ps.name
FROM _parent_child_production_steps pcps
JOIN production_step ps ON ps.id = pcps.B
WHERE pcps.A = sqlc.arg('step_id');

-- name: GetProductionStepOutputSteps :many
-- Downstream children are A where this step is upstream B.
SELECT pcps.A AS id, ps.name
FROM _parent_child_production_steps pcps
JOIN production_step ps ON ps.id = pcps.A
WHERE pcps.B = sqlc.arg('step_id');

-- name: GetProductionStepMachines :many
SELECT m.id, m.name FROM machine m
WHERE m.production_step_id = sqlc.arg('production_step_id');

-- name: InsertProductionStep :exec
INSERT INTO production_step (
    id, name, notes, leveling_factor, allowances,
    labor_rate_id, labor_time_id, overhead_rate_id,
    scanning_station_id, department_id, account_id,
    created_at, updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('name'),
    sqlc.narg('notes'),
    sqlc.arg('leveling_factor'),
    sqlc.arg('allowances'),
    sqlc.arg('labor_rate_id'),
    sqlc.arg('labor_time_id'),
    sqlc.arg('overhead_rate_id'),
    sqlc.narg('scanning_station_id'),
    sqlc.narg('department_id'),
    sqlc.arg('account_id'),
    NOW(3),
    NOW(3)
);

-- name: InsertRateForProductionStep :exec
INSERT INTO rate (
    id, value, numerator_unit_id, denominator_unit_id,
    created_at, updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('value'),
    sqlc.arg('numerator_unit_id'),
    sqlc.arg('denominator_unit_id'),
    NOW(3),
    NOW(3)
);

-- name: InsertProductionQuantity :exec
INSERT INTO quantity (id, value, unit_id) VALUES (
    sqlc.arg('id'),
    sqlc.arg('value'),
    sqlc.arg('unit_id')
);

-- name: InsertProduction :exec
INSERT INTO production (
    id, item_id, quantity_id, production_step_id,
    created_at, updated_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('item_id'),
    sqlc.arg('quantity_id'),
    sqlc.arg('production_step_id'),
    NOW(3),
    NOW(3)
);

-- name: UpdateProductionStepFields :execresult
UPDATE production_step SET
    name = COALESCE(sqlc.narg('name'), name),
    leveling_factor = COALESCE(sqlc.narg('leveling_factor'), leveling_factor),
    allowances = COALESCE(sqlc.narg('allowances'), allowances),
    scanning_station_id = CASE
        WHEN sqlc.arg('update_scanning_station') = true THEN sqlc.narg('scanning_station_id')
        ELSE scanning_station_id
    END,
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: DeleteProductionStepRow :execresult
DELETE FROM production_step
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: DeleteProductionStepParentChildLinks :exec
DELETE FROM _parent_child_production_steps
WHERE A = sqlc.arg('step_id') OR B = sqlc.arg('step_id');

-- name: ExistsProductionStepByName :one
SELECT COUNT(*) FROM production_step
WHERE name = sqlc.arg('name') AND account_id = sqlc.arg('account_id')
AND (sqlc.narg('exclude_id') IS NULL OR id != sqlc.narg('exclude_id'));

-- name: FindProductionStepIDByName :one
SELECT id FROM production_step
WHERE name = sqlc.arg('name') AND account_id = sqlc.arg('account_id')
LIMIT 1;

-- name: DeleteConsumptionQuantitiesByStepID :exec
DELETE q FROM quantity q
JOIN consumption c ON q.id = c.quantity_id OR q.id = c.waste_quantity_id
WHERE c.production_step_id = sqlc.arg('step_id');

-- name: DeleteConsumptionsByStepID :exec
DELETE FROM consumption WHERE production_step_id = sqlc.arg('step_id');

-- name: DeleteProductionQuantitiesByStepID :exec
DELETE q FROM quantity q
JOIN production p ON q.id = p.quantity_id
WHERE p.production_step_id = sqlc.arg('step_id');

-- name: DeleteProductionsByStepID :exec
DELETE FROM production WHERE production_step_id = sqlc.arg('step_id');

-- name: UpdateProductionStepFull :exec
UPDATE production_step SET
    leveling_factor = sqlc.arg('leveling_factor'),
    allowances = sqlc.arg('allowances'),
    scanning_station_id = sqlc.narg('scanning_station_id'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
AND account_id = sqlc.arg('account_id');

-- name: GetProductionFlowStep :one
-- Fetches a production step with all fields needed for flow display.
SELECT
    ps.id, ps.name, ps.scanning_station_id, ps.department_id,
    ps.allowances, ps.leveling_factor,
    p.id AS production_id, pi.id AS produced_item_id, pi.sku AS produced_item_sku,
    pq.id AS produced_quantity_id, pq.value AS produced_quantity_value,
    pu.id AS produced_unit_id, pu.abbreviation AS produced_unit_abbreviation,
    pu.unit_dimension_code AS produced_unit_type,
    lr.id AS labor_rate_id, lr.value AS labor_rate_value,
    lrnu.id AS labor_rate_num_unit_id, lrdu.id AS labor_rate_den_unit_id,
    lt.id AS labor_time_id, lt.value AS labor_time_value,
    ltnu.id AS labor_time_num_unit_id, ltdu.id AS labor_time_den_unit_id,
    ohr.id AS overhead_rate_id, ohr.value AS overhead_rate_value,
    ohrnu.id AS overhead_rate_num_unit_id, ohrdu.id AS overhead_rate_den_unit_id
FROM production_step ps
JOIN production p ON p.production_step_id = ps.id
JOIN item pi ON p.item_id = pi.id
JOIN quantity pq ON p.quantity_id = pq.id
JOIN unit pu ON pq.unit_id = pu.id
LEFT JOIN rate lr ON ps.labor_rate_id = lr.id
LEFT JOIN unit lrnu ON lr.numerator_unit_id = lrnu.id
LEFT JOIN unit lrdu ON lr.denominator_unit_id = lrdu.id
LEFT JOIN rate lt ON ps.labor_time_id = lt.id
LEFT JOIN unit ltnu ON lt.numerator_unit_id = ltnu.id
LEFT JOIN unit ltdu ON lt.denominator_unit_id = ltdu.id
LEFT JOIN rate ohr ON ps.overhead_rate_id = ohr.id
LEFT JOIN unit ohrnu ON ohr.numerator_unit_id = ohrnu.id
LEFT JOIN unit ohrdu ON ohr.denominator_unit_id = ohrdu.id
WHERE ps.id = sqlc.arg('id')
AND ps.account_id = sqlc.arg('account_id');
