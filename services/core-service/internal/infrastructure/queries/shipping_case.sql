-- name: GetShippingCase :one
SELECT
    sc.id,
    sc.number,
    sc.sscc,
    sc.tracking_number,
    sc.shippo_transaction_id,
    sc.shipping_label_url,
    sc.shipped_at,
    sc.shipment_id,
    s.number AS shipment_number,
    ss.code AS shipment_status_code,
    ss.name AS shipment_status_name,
    s.created_at AS shipment_created_at,
    s.updated_at AS shipment_updated_at,
    sc.carrier_id,
    c.name AS carrier_name,
    c.is_portal_enabled AS carrier_is_portal_enabled,
    c.created_at AS carrier_created_at,
    c.updated_at AS carrier_updated_at,
    sc.account_id,
    sc.created_at,
    sc.updated_at,
    -- Freight amount
    fa.id AS freight_amount_id,
    fa.value AS freight_amount_value,
    fa.unit_id AS freight_amount_unit_id,
    fau.name AS freight_amount_unit_name,
    fau.abbreviation AS freight_amount_unit_abbreviation,
    fau.unit_dimension_code AS freight_amount_unit_type,
    fau.ratio_numerator AS freight_amount_unit_ratio_numerator,
    fau.ratio_denominator AS freight_amount_unit_ratio_denominator,
    fau.offset_numerator AS freight_amount_unit_offset_numerator,
    fau.offset_denominator AS freight_amount_unit_offset_denominator,
    fau.created_at AS freight_amount_unit_created_at,
    fau.updated_at AS freight_amount_unit_updated_at,
    -- Freight weight
    fw.id AS freight_weight_id,
    fw.value AS freight_weight_value,
    fw.unit_id AS freight_weight_unit_id,
    fwu.name AS freight_weight_unit_name,
    fwu.abbreviation AS freight_weight_unit_abbreviation,
    fwu.unit_dimension_code AS freight_weight_unit_type,
    fwu.ratio_numerator AS freight_weight_unit_ratio_numerator,
    fwu.ratio_denominator AS freight_weight_unit_ratio_denominator,
    fwu.offset_numerator AS freight_weight_unit_offset_numerator,
    fwu.offset_denominator AS freight_weight_unit_offset_denominator,
    fwu.created_at AS freight_weight_unit_created_at,
    fwu.updated_at AS freight_weight_unit_updated_at
FROM shipping_case sc
JOIN quantity fa ON sc.freight_amount_id = fa.id
JOIN unit fau ON fa.unit_id = fau.id
JOIN quantity fw ON sc.freight_weight_id = fw.id
JOIN unit fwu ON fw.unit_id = fwu.id
JOIN carrier c ON sc.carrier_id = c.id
JOIN shipment s ON s.id = sc.shipment_id
JOIN shipment_status ss ON ss.code = s.shipment_status_code
WHERE sc.id = sqlc.arg('id')
  AND sc.account_id = sqlc.arg('account_id');

-- name: UpdateShippingCaseTrackingNumber :execresult
UPDATE shipping_case SET
    tracking_number = COALESCE(sqlc.narg('tracking_number'), tracking_number),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
  AND account_id = sqlc.arg('account_id');

-- name: RepointShippingCasesToCarrier :exec
UPDATE shipping_case SET
    carrier_id = sqlc.arg('carrier_id'),
    updated_at = NOW(3)
WHERE shipment_id = sqlc.arg('shipment_id')
  AND account_id = sqlc.arg('account_id');

-- Follows the order-wide shipment re-point: a case's tracking link is built from its own carrier,
-- so a case left on the old carrier deep-links to the wrong one.
-- name: RepointShippingCasesToCarrierByOrder :exec
UPDATE shipping_case sc
JOIN shipment s ON s.id = sc.shipment_id
SET sc.carrier_id = sqlc.arg('carrier_id'),
    sc.updated_at = NOW(3)
WHERE s.sales_order_id = sqlc.arg('sales_order_id')
  AND sc.account_id = sqlc.arg('account_id');

-- name: DeleteShippingCase :exec
DELETE FROM shipping_case
WHERE id = sqlc.arg('id')
  AND account_id = sqlc.arg('account_id');

-- name: CheckShippingCaseInAccount :one
SELECT EXISTS(
    SELECT 1 FROM shipping_case
    WHERE id = sqlc.arg('id')
      AND account_id = sqlc.arg('account_id')
) AS `exists`;

-- name: GetShippingCaseNumber :one
SELECT number FROM shipping_case
WHERE id = sqlc.arg('id')
  AND account_id = sqlc.arg('account_id');

-- name: VoidShippingCasesByShipment :exec
UPDATE shipping_case sc
JOIN quantity fa ON sc.freight_amount_id = fa.id
SET
    sc.shipped_at = NULL,
    sc.tracking_number = NULL,
    sc.shippo_transaction_id = NULL,
    sc.shipping_label_url = NULL,
    fa.value = '0',
    sc.updated_at = NOW(3)
WHERE sc.shipment_id = sqlc.arg('shipment_id');

-- name: MarkShippingCasesShippedByShipment :exec
UPDATE shipping_case SET
    shipped_at = NOW(3),
    updated_at = NOW(3)
WHERE shipment_id = sqlc.arg('shipment_id');

-- name: ListShippingCasesByShipment :many
SELECT
    sc.id,
    sc.number,
    sc.sscc,
    sc.tracking_number,
    sc.shippo_transaction_id,
    sc.shipping_label_url,
    sc.shipped_at,
    sc.shipment_id,
    sc.carrier_id,
    c.name AS carrier_name,
    c.is_portal_enabled AS carrier_is_portal_enabled,
    c.created_at AS carrier_created_at,
    c.updated_at AS carrier_updated_at,
    sc.account_id,
    sc.created_at,
    sc.updated_at,
    fa.id AS freight_amount_id,
    fa.value AS freight_amount_value,
    fa.unit_id AS freight_amount_unit_id,
    fau.name AS freight_amount_unit_name,
    fau.abbreviation AS freight_amount_unit_abbreviation,
    fau.unit_dimension_code AS freight_amount_unit_type,
    fau.ratio_numerator AS freight_amount_unit_ratio_numerator,
    fau.ratio_denominator AS freight_amount_unit_ratio_denominator,
    fau.offset_numerator AS freight_amount_unit_offset_numerator,
    fau.offset_denominator AS freight_amount_unit_offset_denominator,
    fau.created_at AS freight_amount_unit_created_at,
    fau.updated_at AS freight_amount_unit_updated_at,
    fw.id AS freight_weight_id,
    fw.value AS freight_weight_value,
    fw.unit_id AS freight_weight_unit_id,
    fwu.name AS freight_weight_unit_name,
    fwu.abbreviation AS freight_weight_unit_abbreviation,
    fwu.unit_dimension_code AS freight_weight_unit_type,
    fwu.ratio_numerator AS freight_weight_unit_ratio_numerator,
    fwu.ratio_denominator AS freight_weight_unit_ratio_denominator,
    fwu.offset_numerator AS freight_weight_unit_offset_numerator,
    fwu.offset_denominator AS freight_weight_unit_offset_denominator,
    fwu.created_at AS freight_weight_unit_created_at,
    fwu.updated_at AS freight_weight_unit_updated_at
FROM shipping_case sc
-- Inner joins: every column here is a NOT NULL reference on shipping_case, so a case whose
-- freight or carrier does not resolve is a broken row rather than an ordinary one. Note the
-- consequence if one ever does dangle — the case disappears from its shipment entirely rather
-- than listing with empty freight, and a case that cannot be listed cannot be labeled,
-- shipped, or deleted either. Packing once wrote a unit abbreviation where a unit ID belongs,
-- and every packed case was invisible until that was fixed.
JOIN quantity fa ON sc.freight_amount_id = fa.id
JOIN unit fau ON fa.unit_id = fau.id
JOIN quantity fw ON sc.freight_weight_id = fw.id
JOIN unit fwu ON fw.unit_id = fwu.id
JOIN carrier c ON sc.carrier_id = c.id
WHERE sc.shipment_id = sqlc.arg('shipment_id');

-- name: AddSsccToShippingCase :exec
UPDATE shipping_case SET
    sscc = sqlc.arg('sscc'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id');

-- name: UpdateShippingCaseWithShipmentInfo :exec
UPDATE shipping_case SET
    tracking_number = sqlc.arg('tracking_number'),
    shippo_transaction_id = sqlc.arg('shippo_transaction_id'),
    shipping_label_url = sqlc.arg('shipping_label_url'),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id');

-- name: DeleteShippingCasesByShipment :exec
DELETE FROM shipping_case
WHERE shipment_id = sqlc.arg('shipment_id');

-- name: GetSalesOrderIDByShippingCase :one
-- Walks the case to the order it ultimately belongs to. Audit events stamp that order as their
-- root so a case edit shows up in the order's history, and the route is only ever case → shipment.
SELECT s.sales_order_id
FROM shipping_case sc
JOIN shipment s ON s.id = sc.shipment_id
WHERE sc.id = sqlc.arg('shipping_case_id')
AND sc.account_id = sqlc.arg('account_id');
