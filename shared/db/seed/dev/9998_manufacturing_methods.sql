-- Runs after every seed that inserts production steps. Each step is the single operation of an active manufacturing
-- method for the item it produces, as the API and the adoption backfill make it, with its consumptions as materials
-- and its machines on the operation.
INSERT INTO manufacturing_method (id, account_id, item_id, version, status_code, output_quantity_value, output_quantity_unit_id, activated_at, created_at, updated_at)
SELECT CONCAT('prmt_', LEFT(MD5(CONCAT('production_step:', s.id)), 24)), s.account_id, s.item_id,
    COALESCE((SELECT MAX(m.version) FROM manufacturing_method m WHERE m.item_id = s.item_id), 0) + s.rn,
    'active', s.value, s.unit_id, s.created_at, s.created_at, NOW(3)
FROM (
    SELECT ps.id, ps.account_id, ps.created_at, p.item_id, q.value, q.unit_id,
        ROW_NUMBER() OVER (PARTITION BY p.item_id ORDER BY ps.created_at, ps.id) AS rn
    FROM production_step ps
    JOIN production p ON p.id = (SELECT p2.id FROM production p2 WHERE p2.production_step_id = ps.id ORDER BY p2.created_at, p2.id LIMIT 1)
    JOIN quantity q ON q.id = p.quantity_id
    WHERE ps.manufacturing_method_id IS NULL
) s;

UPDATE production_step ps
JOIN manufacturing_method m ON m.id = CONCAT('prmt_', LEFT(MD5(CONCAT('production_step:', ps.id)), 24))
SET ps.manufacturing_method_id = m.id, ps.position = 1
WHERE ps.manufacturing_method_id IS NULL;

UPDATE consumption c
JOIN production_step ps ON ps.id = c.production_step_id
SET c.manufacturing_method_id = ps.manufacturing_method_id,
    c.supply_code = IF(EXISTS (SELECT 1 FROM production p WHERE p.item_id = c.item_id), 'make', 'stock')
WHERE c.manufacturing_method_id IS NULL
  AND ps.manufacturing_method_id IS NOT NULL;

INSERT IGNORE INTO method_operation_machine (account_id, operation_id, machine_id, created_at)
SELECT m.account_id, m.production_step_id, m.id, NOW(3)
FROM machine m
WHERE m.production_step_id IS NOT NULL;
