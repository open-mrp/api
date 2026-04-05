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
    sc.carrier_id,
    c.name AS carrier_name,
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
    -- Freight weight
    fw.id AS freight_weight_id,
    fw.value AS freight_weight_value,
    fw.unit_id AS freight_weight_unit_id,
    fwu.name AS freight_weight_unit_name,
    fwu.abbreviation AS freight_weight_unit_abbreviation,
    fwu.unit_dimension_code AS freight_weight_unit_type
FROM shipping_case sc
JOIN quantity fa ON sc.freight_amount_id = fa.id
JOIN unit fau ON fa.unit_id = fau.id
JOIN quantity fw ON sc.freight_weight_id = fw.id
JOIN unit fwu ON fw.unit_id = fwu.id
JOIN carrier c ON sc.carrier_id = c.id
WHERE sc.id = sqlc.arg('id')
  AND sc.account_id = sqlc.arg('account_id');

-- name: UpdateShippingCaseTrackingNumber :execresult
UPDATE shipping_case SET
    tracking_number = COALESCE(sqlc.narg('tracking_number'), tracking_number),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
  AND account_id = sqlc.arg('account_id');

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
    sc.account_id,
    sc.created_at,
    sc.updated_at,
    fa.id AS freight_amount_id,
    fa.value AS freight_amount_value,
    fa.unit_id AS freight_amount_unit_id,
    fau.name AS freight_amount_unit_name,
    fau.abbreviation AS freight_amount_unit_abbreviation,
    fau.unit_dimension_code AS freight_amount_unit_type,
    fw.id AS freight_weight_id,
    fw.value AS freight_weight_value,
    fw.unit_id AS freight_weight_unit_id,
    fwu.name AS freight_weight_unit_name,
    fwu.abbreviation AS freight_weight_unit_abbreviation,
    fwu.unit_dimension_code AS freight_weight_unit_type
FROM shipping_case sc
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
