-- Lines carry copies of their quantity and rate rows (migration 00081), which the API writes with every
-- change and core-service's backfill fills once. Seeds write lines in plain SQL, so a database re-seeded
-- after that backfill finished copies its new lines here. Runs last, after every file that writes lines.
UPDATE sales_order_line sol
LEFT JOIN quantity cq ON cq.id = sol.quantity_id
LEFT JOIN rate cp ON cp.id = sol.unit_price_id
LEFT JOIN rate cc ON cc.id = sol.unit_cost_id
SET sol.quantity_value = cq.value, sol.quantity_unit_id = cq.unit_id,
    sol.unit_price_value = cp.value, sol.unit_price_numerator_unit_id = cp.numerator_unit_id,
    sol.unit_price_denominator_unit_id = cp.denominator_unit_id,
    sol.unit_cost_value = cc.value, sol.unit_cost_numerator_unit_id = cc.numerator_unit_id,
    sol.unit_cost_denominator_unit_id = cc.denominator_unit_id
WHERE sol.quantity_unit_id IS NULL AND sol.unit_price_numerator_unit_id IS NULL AND sol.unit_cost_numerator_unit_id IS NULL;

UPDATE invoice_line il
LEFT JOIN quantity cq ON cq.id = il.quantity_id
SET il.quantity_value = cq.value, il.quantity_unit_id = cq.unit_id
WHERE il.quantity_unit_id IS NULL;

UPDATE pick_line pl
LEFT JOIN quantity cq ON cq.id = pl.quantity_id
SET pl.quantity_value = cq.value, pl.quantity_unit_id = cq.unit_id
WHERE pl.quantity_unit_id IS NULL;
