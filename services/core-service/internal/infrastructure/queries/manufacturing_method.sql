-- Manufacturing methods. Operations are production_step rows and materials are consumption rows (see 00109). Only an
-- active method's operations carry production rows and _parent_child_production_steps links (A = downstream,
-- B = upstream), which execution, costing and planning read.

-- name: InsertManufacturingMethod :exec
INSERT INTO manufacturing_method (
    id, account_id, item_id, version, status_code, output_quantity_value, output_quantity_unit_id,
    sales_order_line_id, activated_at, created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('account_id'), sqlc.arg('item_id'), sqlc.arg('version'), sqlc.arg('status_code'),
    sqlc.arg('output_quantity_value'), sqlc.arg('output_quantity_unit_id'), NULL, sqlc.narg('activated_at'), NOW(3), NOW(3)
);

-- name: GetManufacturingMethodsByIDs :many
SELECT m.id, m.account_id, m.item_id, m.version, m.status_code, m.output_quantity_value, m.output_quantity_unit_id,
    u.unit_dimension_code AS output_quantity_unit_type, m.sales_order_line_id, m.activated_at, m.created_at, m.updated_at
FROM manufacturing_method m
JOIN unit u ON u.id = m.output_quantity_unit_id
WHERE m.account_id = sqlc.arg('account_id')
  AND m.id IN (sqlc.slice('ids'));

-- name: LockManufacturingMethod :one
-- Status changes and draft edits serialize on the method row.
SELECT status_code, item_id FROM manufacturing_method
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id')
FOR UPDATE;

-- name: MaxManufacturingMethodVersion :one
-- Served by manufacturing_method_item_id_version_key.
SELECT CAST(COALESCE(MAX(version), 0) AS SIGNED) AS version FROM manufacturing_method
WHERE item_id = sqlc.arg('item_id');

-- name: UpdateManufacturingMethodOutput :exec
UPDATE manufacturing_method SET
    output_quantity_value = sqlc.arg('output_quantity_value'),
    output_quantity_unit_id = sqlc.arg('output_quantity_unit_id'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: UpdateManufacturingMethodStatus :exec
UPDATE manufacturing_method SET
    status_code = sqlc.arg('status_code'),
    activated_at = COALESCE(sqlc.narg('activated_at'), activated_at),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: DeleteManufacturingMethodRow :exec
DELETE FROM manufacturing_method WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: ListActiveStandardMethodIDsForItem :many
-- An item's active methods that aren't specific to an order. Served by manufacturing_method_item_id_created_at_id_idx.
SELECT id FROM manufacturing_method
WHERE item_id = sqlc.arg('item_id')
  AND account_id = sqlc.arg('account_id')
  AND status_code = 'active'
  AND sales_order_line_id IS NULL;

-- name: ListDefaultMethodIDsForItems :many
-- The method each item is made by when nothing more specific applies: its newest active standard method.
SELECT m.item_id, m.id
FROM manufacturing_method m
WHERE m.account_id = sqlc.arg('account_id')
  AND m.item_id IN (sqlc.slice('item_ids'))
  AND m.status_code = 'active'
  AND m.sales_order_line_id IS NULL
ORDER BY m.item_id, m.version DESC;

-- name: ListMethodOperationIDs :many
-- A method's operations in order. Served by production_step_manufacturing_method_id_position_idx.
SELECT id FROM production_step
WHERE manufacturing_method_id = sqlc.arg('manufacturing_method_id')
ORDER BY position, id;

-- name: ListMethodOperationsAfter :many
SELECT id, position FROM production_step
WHERE manufacturing_method_id = sqlc.arg('manufacturing_method_id')
  AND position > sqlc.arg('after_position')
ORDER BY position, id
LIMIT ?;

-- name: ListMethodOperationsBefore :many
SELECT id, position FROM production_step
WHERE manufacturing_method_id = sqlc.arg('manufacturing_method_id')
  AND position < sqlc.arg('before_position')
ORDER BY position DESC, id DESC
LIMIT ?;

-- name: GetMethodOperationsByIDs :many
SELECT
    ps.id,
    ps.manufacturing_method_id,
    ps.position,
    ps.sequence_code,
    ps.type_code,
    ps.department_id,
    ps.scanning_station_id,
    ps.leveling_factor,
    ps.allowances,
    ps.work_instructions,
    ps.labor_rate_id,
    ps.labor_time_id,
    ps.overhead_rate_id,
    ps.setup_time_quantity_id,
    ps.machine_time_id,
    st.value AS setup_time_value, st.unit_id AS setup_time_unit_id, stu.unit_dimension_code AS setup_time_unit_type,
    lt.value AS labor_time_value, lt.numerator_unit_id AS labor_time_unit_id, lt.denominator_unit_id AS labor_time_per_unit_id,
    mt.value AS machine_time_value, mt.numerator_unit_id AS machine_time_unit_id, mt.denominator_unit_id AS machine_time_per_unit_id,
    ps.labor_rate_amount, ps.labor_rate_currency, ps.labor_rate_per_unit_id,
    lr.value AS legacy_labor_rate_value, lr.denominator_unit_id AS legacy_labor_rate_per_unit_id,
    ps.overhead_rate_amount, ps.overhead_rate_currency, ps.overhead_rate_per_unit_id,
    ohr.value AS legacy_overhead_rate_value, ohr.denominator_unit_id AS legacy_overhead_rate_per_unit_id,
    ps.created_at,
    ps.updated_at
FROM production_step ps
LEFT JOIN quantity st ON st.id = ps.setup_time_quantity_id
LEFT JOIN unit stu ON stu.id = st.unit_id
LEFT JOIN rate lt ON lt.id = ps.labor_time_id
LEFT JOIN rate mt ON mt.id = ps.machine_time_id
-- An operation written before its money columns reads its legacy rate rows; a filled one never looks them up.
LEFT JOIN rate lr ON lr.id = IF(ps.labor_rate_amount IS NULL, ps.labor_rate_id, NULL)
LEFT JOIN rate ohr ON ohr.id = IF(ps.overhead_rate_amount IS NULL, ps.overhead_rate_id, NULL)
WHERE ps.account_id = sqlc.arg('account_id')
  AND ps.id IN (sqlc.slice('ids'))
  AND ps.manufacturing_method_id IS NOT NULL;

-- name: ListOperationMachineIDs :many
SELECT operation_id, machine_id FROM method_operation_machine
WHERE operation_id IN (sqlc.slice('operation_ids'))
ORDER BY operation_id, created_at, machine_id;

-- name: InsertOperationMachine :exec
INSERT INTO method_operation_machine (account_id, operation_id, machine_id, created_at)
VALUES (sqlc.arg('account_id'), sqlc.arg('operation_id'), sqlc.arg('machine_id'), NOW(3));

-- name: DeleteOperationMachines :exec
DELETE FROM method_operation_machine WHERE operation_id = sqlc.arg('operation_id');

-- name: InsertMethodOperation :exec
INSERT INTO production_step (
    id, name, leveling_factor, allowances,
    labor_rate_id, labor_time_id, overhead_rate_id,
    labor_rate_amount, labor_rate_currency, labor_rate_per_unit_id,
    overhead_rate_amount, overhead_rate_currency, overhead_rate_per_unit_id,
    scanning_station_id, department_id, account_id,
    manufacturing_method_id, position, sequence_code, type_code, setup_time_quantity_id, machine_time_id, work_instructions,
    created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('name'), sqlc.arg('leveling_factor'), sqlc.arg('allowances'),
    sqlc.arg('labor_rate_id'), sqlc.arg('labor_time_id'), sqlc.arg('overhead_rate_id'),
    sqlc.arg('labor_rate_amount'), sqlc.arg('labor_rate_currency'), sqlc.arg('labor_rate_per_unit_id'),
    sqlc.arg('overhead_rate_amount'), sqlc.arg('overhead_rate_currency'), sqlc.arg('overhead_rate_per_unit_id'),
    sqlc.narg('scanning_station_id'), sqlc.narg('department_id'), sqlc.arg('account_id'),
    sqlc.arg('manufacturing_method_id'), sqlc.arg('position'), sqlc.arg('sequence_code'), sqlc.arg('type_code'),
    sqlc.narg('setup_time_quantity_id'), sqlc.narg('machine_time_id'), sqlc.narg('work_instructions'),
    NOW(3), NOW(3)
);

-- name: UpdateMethodOperation :exec
-- The service merges the change into the stored operation and writes the whole row. Rate and quantity rows are never
-- mutated in place: a changed value points at a fresh row.
UPDATE production_step SET
    leveling_factor = sqlc.arg('leveling_factor'),
    allowances = sqlc.arg('allowances'),
    labor_rate_id = sqlc.arg('labor_rate_id'),
    labor_time_id = sqlc.arg('labor_time_id'),
    overhead_rate_id = sqlc.arg('overhead_rate_id'),
    labor_rate_amount = sqlc.arg('labor_rate_amount'),
    labor_rate_currency = sqlc.arg('labor_rate_currency'),
    labor_rate_per_unit_id = sqlc.arg('labor_rate_per_unit_id'),
    overhead_rate_amount = sqlc.arg('overhead_rate_amount'),
    overhead_rate_currency = sqlc.arg('overhead_rate_currency'),
    overhead_rate_per_unit_id = sqlc.arg('overhead_rate_per_unit_id'),
    scanning_station_id = sqlc.narg('scanning_station_id'),
    department_id = sqlc.narg('department_id'),
    sequence_code = sqlc.arg('sequence_code'),
    setup_time_quantity_id = sqlc.narg('setup_time_quantity_id'),
    machine_time_id = sqlc.narg('machine_time_id'),
    work_instructions = sqlc.narg('work_instructions'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: SetMethodOperationPosition :exec
UPDATE production_step SET position = sqlc.arg('position'), updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: DeleteMethodOperationRow :exec
DELETE FROM production_step WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: CountOperationMaterials :one
SELECT COUNT(*) FROM consumption WHERE production_step_id = sqlc.arg('operation_id');

-- name: InsertMethodRate :exec
INSERT INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('value'), sqlc.arg('numerator_unit_id'), sqlc.arg('denominator_unit_id'), NOW(3), NOW(3));

-- name: InsertMethodQuantity :exec
INSERT INTO quantity (id, value, unit_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('value'), sqlc.arg('unit_id'), NOW(3), NOW(3));

-- name: DeleteMethodRates :exec
DELETE FROM rate WHERE id IN (sqlc.slice('ids'));

-- name: DeleteMethodQuantities :exec
DELETE FROM quantity WHERE id IN (sqlc.slice('ids'));

-- name: ListMethodMaterialIDs :many
-- A method's materials in the order they were added. Served by consumption_manufacturing_method_id_created_at_id_idx.
SELECT id, created_at FROM consumption
WHERE manufacturing_method_id = sqlc.arg('manufacturing_method_id')
ORDER BY created_at, id
LIMIT ?;

-- name: ListMethodMaterialsAfter :many
SELECT id, created_at FROM consumption
WHERE manufacturing_method_id = sqlc.arg('manufacturing_method_id')
  AND (created_at > sqlc.arg('after_created_at') OR (created_at = sqlc.arg('after_created_at') AND id > sqlc.arg('after_id')))
ORDER BY created_at, id
LIMIT ?;

-- name: ListMethodMaterialsBefore :many
SELECT id, created_at FROM consumption
WHERE manufacturing_method_id = sqlc.arg('manufacturing_method_id')
  AND (created_at < sqlc.arg('before_created_at') OR (created_at = sqlc.arg('before_created_at') AND id < sqlc.arg('before_id')))
ORDER BY created_at DESC, id DESC
LIMIT ?;

-- name: GetMethodMaterialsByIDs :many
SELECT
    c.id,
    c.manufacturing_method_id,
    c.item_id,
    c.production_step_id,
    c.supply_code,
    c.instructions,
    c.quantity_id,
    q.value AS quantity_value, q.unit_id AS quantity_unit_id, qu.unit_dimension_code AS quantity_unit_type,
    c.waste_quantity_id,
    w.value AS scrap_quantity_value, w.unit_id AS scrap_quantity_unit_id, wu.unit_dimension_code AS scrap_quantity_unit_type,
    c.created_at,
    c.updated_at
FROM consumption c
JOIN manufacturing_method m ON m.id = c.manufacturing_method_id
JOIN quantity q ON q.id = c.quantity_id
JOIN unit qu ON qu.id = q.unit_id
JOIN quantity w ON w.id = c.waste_quantity_id
JOIN unit wu ON wu.id = w.unit_id
WHERE m.account_id = sqlc.arg('account_id')
  AND c.id IN (sqlc.slice('ids'));

-- name: InsertMethodMaterial :exec
INSERT INTO consumption (
    id, item_id, quantity_id, waste_quantity_id, production_step_id, instructions, manufacturing_method_id, supply_code,
    created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('item_id'), sqlc.arg('quantity_id'), sqlc.arg('waste_quantity_id'), sqlc.arg('production_step_id'),
    sqlc.narg('instructions'), sqlc.arg('manufacturing_method_id'), sqlc.arg('supply_code'), NOW(3), NOW(3)
);

-- name: UpdateMethodMaterial :exec
UPDATE consumption SET
    item_id = sqlc.arg('item_id'),
    quantity_id = sqlc.arg('quantity_id'),
    waste_quantity_id = sqlc.arg('waste_quantity_id'),
    production_step_id = sqlc.arg('production_step_id'),
    instructions = sqlc.narg('instructions'),
    supply_code = sqlc.arg('supply_code'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND manufacturing_method_id = sqlc.arg('manufacturing_method_id');

-- name: DeleteMethodMaterialRow :exec
DELETE FROM consumption WHERE id = sqlc.arg('id') AND manufacturing_method_id = sqlc.arg('manufacturing_method_id');

-- name: ItemsWithActiveMethods :many
-- Which of these items have an active standard method, for a material's default supply.
SELECT DISTINCT item_id FROM manufacturing_method
WHERE account_id = sqlc.arg('account_id')
  AND item_id IN (sqlc.slice('item_ids'))
  AND status_code = 'active'
  AND sales_order_line_id IS NULL;

-- name: InsertMethodProduction :exec
INSERT INTO production (id, item_id, quantity_id, production_step_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('item_id'), sqlc.arg('quantity_id'), sqlc.arg('production_step_id'), NOW(3), NOW(3));

-- name: ListProductionQuantityIDsByStep :many
SELECT quantity_id FROM production WHERE production_step_id = sqlc.arg('step_id');

-- name: DeleteProductionsOfStep :exec
DELETE FROM production WHERE production_step_id = sqlc.arg('step_id');

-- name: ClearStepLinksOfStep :exec
DELETE FROM _parent_child_production_steps WHERE A = sqlc.arg('step_id');

-- name: ClearStepLinksFromStep :exec
DELETE FROM _parent_child_production_steps WHERE B = sqlc.arg('step_id');

-- name: ConnectMethodSteps :exec
-- (A, B) = (downstream, upstream).
INSERT IGNORE INTO _parent_child_production_steps (A, B) VALUES (sqlc.arg('downstream_id'), sqlc.arg('upstream_id'));

-- name: ClearMachinesFromStep :exec
UPDATE machine SET production_step_id = NULL, updated_at = NOW(3)
WHERE production_step_id = sqlc.arg('step_id');

-- name: SetMachinesStep :exec
UPDATE machine SET production_step_id = sqlc.arg('step_id'), updated_at = NOW(3)
WHERE id IN (sqlc.slice('ids')) AND account_id = sqlc.arg('account_id');

-- name: ExistsOperationNamed :one
SELECT COUNT(*) FROM production_step WHERE account_id = sqlc.arg('account_id') AND name = sqlc.arg('name');

-- Production steps written through the preview.9 routes (and those from before methods) belong to a method too: each
-- step is the single operation of an active method for the item it produces. These keep that method in step.

-- name: GetLegacyStepOutput :one
-- The step's method, and the first output it records.
SELECT ps.account_id, ps.manufacturing_method_id, ps.created_at, p.item_id, q.value AS quantity_value, q.unit_id AS quantity_unit_id
FROM production_step ps
JOIN production p ON p.id = (SELECT p2.id FROM production p2 WHERE p2.production_step_id = ps.id ORDER BY p2.created_at, p2.id LIMIT 1)
JOIN quantity q ON q.id = p.quantity_id
WHERE ps.id = sqlc.arg('step_id');

-- name: InsertAdoptedManufacturingMethod :exec
INSERT INTO manufacturing_method (
    id, account_id, item_id, version, status_code, output_quantity_value, output_quantity_unit_id,
    sales_order_line_id, activated_at, created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('account_id'), sqlc.arg('item_id'), sqlc.arg('version'), 'active',
    sqlc.arg('output_quantity_value'), sqlc.arg('output_quantity_unit_id'), NULL, sqlc.arg('activated_at'), sqlc.arg('created_at'), NOW(3)
);

-- name: SetStepMethod :exec
UPDATE production_step SET manufacturing_method_id = sqlc.arg('manufacturing_method_id'), position = 1
WHERE id = sqlc.arg('step_id') AND manufacturing_method_id IS NULL;

-- name: SyncAdoptedMethodOutput :exec
UPDATE manufacturing_method SET
    item_id = sqlc.arg('item_id'),
    version = sqlc.arg('version'),
    output_quantity_value = sqlc.arg('output_quantity_value'),
    output_quantity_unit_id = sqlc.arg('output_quantity_unit_id'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id');

-- name: GetManufacturingMethodItemAndOutput :one
SELECT item_id, version, output_quantity_value, output_quantity_unit_id FROM manufacturing_method WHERE id = sqlc.arg('id');

-- name: AdoptStepConsumptions :exec
-- A consumed item something makes is supplied by production; anything else comes from stock. Served by
-- consumption_production_step_id_idx and production_item_id_idx.
UPDATE consumption SET
    manufacturing_method_id = sqlc.arg('manufacturing_method_id'),
    supply_code = IF(EXISTS (SELECT 1 FROM production p WHERE p.item_id = consumption.item_id), 'make', 'stock')
WHERE consumption.production_step_id = sqlc.arg('step_id')
  AND consumption.manufacturing_method_id IS NULL;

-- name: ListStepMachineIDs :many
SELECT id FROM machine WHERE production_step_id = sqlc.arg('step_id');

-- name: CountMethodOperations :one
SELECT COUNT(*) FROM production_step WHERE manufacturing_method_id = sqlc.arg('manufacturing_method_id');

-- name: ListUnadoptedStepIDsAfter :many
-- The backfill's walk over steps not yet in a method, by primary key.
SELECT id FROM production_step
WHERE id > sqlc.arg('after_id')
ORDER BY id
LIMIT ?;

-- name: DeleteMachineFromOperations :exec
-- A deleted machine leaves every operation it ran on, as it left its production step before methods. Served by
-- method_operation_machine_machine_id_idx.
DELETE FROM method_operation_machine WHERE machine_id = sqlc.arg('machine_id');
