-- name: CreateUnit :exec
INSERT INTO unit (
    id, name, abbreviation, account_id, unit_dimension_code,
    ratio_numerator, ratio_denominator, offset_numerator, offset_denominator,
    is_base_unit, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, 0, 1, ?, NOW(3), NOW(3));

-- name: CreateUnitGroup :exec
INSERT INTO unit_group (
    id, name, notes, base_unit_id, account_id, unit_type_code,
    created_at, updated_at
) VALUES (?, ?, NULL, ?, ?, ?, NOW(3), NOW(3));

-- name: CreateUnitGroupUnit :exec
INSERT INTO unit_group_unit (
    id, unit_group_id, unit_id, discount_percentage, is_visible,
    created_at, updated_at
) VALUES (?, ?, ?, 0, true, NOW(3), NOW(3));

-- name: CreateItemCategory :exec
INSERT INTO item_category (
    id, name, notes, account_id, item_category_type_code, unit_group_id,
    created_at, updated_at
) VALUES (?, ?, NULL, ?, ?, ?, NOW(3), NOW(3));

-- name: CreateProductLine :exec
INSERT INTO product_line (
    id, name, description, notes, account_id, unit_group_id,
    is_commission_exempt, is_freight_exempt,
    created_at, updated_at
) VALUES (?, ?, ?, NULL, ?, ?, ?, ?, NOW(3), NOW(3));

-- name: CreateRate :exec
INSERT INTO rate (
    id, value, numerator_unit_id, denominator_unit_id,
    created_at, updated_at
) VALUES (?, ?, ?, ?, NOW(3), NOW(3));

-- name: CreateItem :exec
INSERT INTO item (
    id, sku, description, notes, unit_value_id, burn_rate_id,
    account_id, item_type_code, unit_cost_id, item_category_id, is_dirty,
    created_at, updated_at
) VALUES (?, ?, ?, NULL, ?, ?, ?, ?, ?, ?, false, NOW(3), NOW(3));

-- name: CreateProduct :exec
INSERT INTO product (
    id, item_id, product_type_code, product_line_id,
    created_at, updated_at
) VALUES (?, ?, ?, ?, NOW(3), NOW(3));
