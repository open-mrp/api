-- name: GetPriceListsByIDs :many
SELECT * FROM price_list
WHERE account_id = sqlc.arg('account_id') AND id IN (sqlc.slice('ids'));

-- name: GetPriceListBySource :one
SELECT * FROM price_list
WHERE account_id = sqlc.arg('account_id') AND source_type = sqlc.arg('source_type') AND source_id = sqlc.arg('source_id');

-- name: ListPriceListsBySourceType :many
SELECT * FROM price_list
WHERE account_id = sqlc.arg('account_id') AND source_type = sqlc.arg('source_type');

-- name: InsertPriceList :exec
INSERT INTO price_list (id, account_id, name, description, is_active, currency, source_type, source_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('account_id'), sqlc.arg('name'), sqlc.narg('description'), sqlc.arg('is_active'),
    sqlc.arg('currency'), sqlc.narg('source_type'), sqlc.narg('source_id'), NOW(3), NOW(3));

-- name: UpdatePriceList :exec
UPDATE price_list
SET name = sqlc.arg('name'), description = sqlc.narg('description'), is_active = sqlc.arg('is_active'), updated_at = NOW(3)
WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: DeletePriceList :exec
DELETE FROM price_list WHERE id = sqlc.arg('id') AND account_id = sqlc.arg('account_id');

-- name: DeletePriceListRulesByList :exec
DELETE FROM price_list_rule WHERE price_list_id = sqlc.arg('price_list_id');

-- name: GetPriceListRulesByIDs :many
SELECT * FROM price_list_rule
WHERE account_id = sqlc.arg('account_id') AND id IN (sqlc.slice('ids'));

-- name: GetPriceListRulesByListIDs :many
SELECT * FROM price_list_rule
WHERE price_list_id IN (sqlc.slice('price_list_ids'))
ORDER BY created_at DESC, id DESC;

-- name: GetPriceListRuleIDsByListIDs :many
SELECT id, price_list_id FROM price_list_rule
WHERE price_list_id IN (sqlc.slice('price_list_ids'));

-- name: GetPriceListRuleBySource :one
SELECT * FROM price_list_rule
WHERE price_list_id = sqlc.arg('price_list_id') AND source_id = sqlc.arg('source_id');

-- name: GetPriceListRulesBySource :many
SELECT * FROM price_list_rule
WHERE account_id = sqlc.arg('account_id') AND source_id = sqlc.arg('source_id');

-- name: InsertPriceListRule :exec
INSERT INTO price_list_rule (id, price_list_id, account_id, price_type, unit_price_amount, unit_price_currency,
    unit_price_per_unit_id, percent_of_base, tier_unit_id, source_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('price_list_id'), sqlc.arg('account_id'), sqlc.arg('price_type'),
    sqlc.narg('unit_price_amount'), sqlc.narg('unit_price_currency'), sqlc.narg('unit_price_per_unit_id'),
    sqlc.narg('percent_of_base'), sqlc.narg('tier_unit_id'), sqlc.narg('source_id'), NOW(3), NOW(3));

-- name: UpdatePriceListRulePrice :exec
UPDATE price_list_rule
SET price_type = sqlc.arg('price_type'), unit_price_amount = sqlc.narg('unit_price_amount'),
    unit_price_currency = sqlc.narg('unit_price_currency'), unit_price_per_unit_id = sqlc.narg('unit_price_per_unit_id'),
    percent_of_base = sqlc.narg('percent_of_base'), tier_unit_id = sqlc.narg('tier_unit_id'), updated_at = NOW(3)
WHERE id = sqlc.arg('id');

-- name: DeletePriceListRule :exec
DELETE FROM price_list_rule WHERE id = sqlc.arg('id');

-- name: GetPriceListRuleTiers :many
SELECT * FROM price_list_rule_tier
WHERE price_list_rule_id IN (sqlc.slice('rule_ids'))
ORDER BY price_list_rule_id, position;

-- name: InsertPriceListRuleTier :exec
INSERT INTO price_list_rule_tier (price_list_rule_id, position, min_quantity, percent_of_base, unit_price_amount,
    unit_price_currency, unit_price_per_unit_id)
VALUES (sqlc.arg('price_list_rule_id'), sqlc.arg('position'), sqlc.arg('min_quantity'), sqlc.narg('percent_of_base'),
    sqlc.narg('unit_price_amount'), sqlc.narg('unit_price_currency'), sqlc.narg('unit_price_per_unit_id'));

-- name: DeletePriceListRuleTiers :exec
DELETE FROM price_list_rule_tier WHERE price_list_rule_id IN (sqlc.slice('rule_ids'));

-- name: GetPriceListRuleItems :many
SELECT price_list_rule_id, item_id FROM price_list_rule_item
WHERE price_list_rule_id IN (sqlc.slice('rule_ids')) ORDER BY price_list_rule_id, item_id;

-- name: GetPriceListRuleProductLines :many
SELECT price_list_rule_id, product_line_id FROM price_list_rule_product_line
WHERE price_list_rule_id IN (sqlc.slice('rule_ids')) ORDER BY price_list_rule_id, product_line_id;

-- name: GetPriceListRuleCategories :many
SELECT price_list_rule_id, item_category_id FROM price_list_rule_item_category
WHERE price_list_rule_id IN (sqlc.slice('rule_ids')) ORDER BY price_list_rule_id, item_category_id;

-- name: GetPriceListRuleAttributes :many
SELECT price_list_rule_id, attribute_id FROM price_list_rule_attribute
WHERE price_list_rule_id IN (sqlc.slice('rule_ids')) ORDER BY price_list_rule_id, attribute_id;

-- name: InsertPriceListRuleItem :exec
INSERT INTO price_list_rule_item (price_list_rule_id, item_id) VALUES (sqlc.arg('price_list_rule_id'), sqlc.arg('item_id'));

-- name: InsertPriceListRuleProductLine :exec
INSERT INTO price_list_rule_product_line (price_list_rule_id, product_line_id) VALUES (sqlc.arg('price_list_rule_id'), sqlc.arg('product_line_id'));

-- name: InsertPriceListRuleCategory :exec
INSERT INTO price_list_rule_item_category (price_list_rule_id, item_category_id) VALUES (sqlc.arg('price_list_rule_id'), sqlc.arg('item_category_id'));

-- name: InsertPriceListRuleAttribute :exec
INSERT INTO price_list_rule_attribute (price_list_rule_id, attribute_id) VALUES (sqlc.arg('price_list_rule_id'), sqlc.arg('attribute_id'));

-- name: DeletePriceListRuleItems :exec
DELETE FROM price_list_rule_item WHERE price_list_rule_id IN (sqlc.slice('rule_ids'));

-- name: DeletePriceListRuleProductLines :exec
DELETE FROM price_list_rule_product_line WHERE price_list_rule_id IN (sqlc.slice('rule_ids'));

-- name: DeletePriceListRuleCategories :exec
DELETE FROM price_list_rule_item_category WHERE price_list_rule_id IN (sqlc.slice('rule_ids'));

-- name: DeletePriceListRuleAttributes :exec
DELETE FROM price_list_rule_attribute WHERE price_list_rule_id IN (sqlc.slice('rule_ids'));

-- name: GetPriceListCustomerAssignments :many
SELECT account_relation_id, price_list_id FROM price_list_customer
WHERE account_relation_id IN (sqlc.slice('relation_ids'))
ORDER BY account_relation_id, created_at, price_list_id;

-- name: GetPriceListGroupAssignments :many
SELECT account_group_id, price_list_id FROM price_list_account_group
WHERE account_group_id IN (sqlc.slice('group_ids'))
ORDER BY account_group_id, created_at, price_list_id;

-- name: InsertPriceListCustomer :exec
INSERT IGNORE INTO price_list_customer (account_relation_id, price_list_id, created_at)
VALUES (sqlc.arg('account_relation_id'), sqlc.arg('price_list_id'), NOW(3));

-- name: InsertPriceListAccountGroup :exec
INSERT IGNORE INTO price_list_account_group (account_group_id, price_list_id, created_at)
VALUES (sqlc.arg('account_group_id'), sqlc.arg('price_list_id'), NOW(3));

-- name: DeletePriceListCustomer :exec
DELETE FROM price_list_customer WHERE account_relation_id = sqlc.arg('account_relation_id') AND price_list_id = sqlc.arg('price_list_id');

-- name: DeletePriceListCustomersByRelations :exec
DELETE FROM price_list_customer WHERE account_relation_id IN (sqlc.slice('relation_ids'));

-- name: DeletePriceListAccountGroupsByGroup :exec
DELETE FROM price_list_account_group WHERE account_group_id = sqlc.arg('account_group_id');

-- name: DeletePriceListAccountGroup :exec
DELETE FROM price_list_account_group WHERE account_group_id = sqlc.arg('account_group_id') AND price_list_id = sqlc.arg('price_list_id');

-- name: DeletePriceListAssignmentsByList :exec
DELETE FROM price_list_customer WHERE price_list_id = sqlc.arg('price_list_id');

-- name: DeletePriceListGroupAssignmentsByList :exec
DELETE FROM price_list_account_group WHERE price_list_id = sqlc.arg('price_list_id');

-- name: ListPriceListCustomerRelations :many
SELECT account_relation_id FROM price_list_customer WHERE price_list_id = sqlc.arg('price_list_id');

-- name: GetApplicablePriceLists :many
-- The active lists that apply to each customer account, with where each applies from: 0 the customer's own list,
-- 1 assigned to the customer, 2 assigned to its parent customer, 3 assigned to its account group, 4 every customer.
SELECT ar.counterparty_account_id AS customer_account_id, pl.id AS price_list_id,
    CAST(IF(pl.source_type = 'customer' AND pl.source_id = ar.id, 0, 1) AS SIGNED) AS source_rank
FROM account_relation ar
JOIN price_list_customer plc ON plc.account_relation_id = ar.id
JOIN price_list pl ON pl.id = plc.price_list_id AND pl.is_active = 1
WHERE ar.owner_account_id = sqlc.arg('owner_account_id')
AND ar.counterparty_account_id IN (sqlc.slice('customer_account_ids'))
AND ar.account_relation_role_code = 'customer'
UNION ALL
SELECT ar.counterparty_account_id, pl.id, CAST(2 AS SIGNED)
FROM account_relation ar
JOIN price_list_customer plc ON plc.account_relation_id = ar.parent_account_relation_id
JOIN price_list pl ON pl.id = plc.price_list_id AND pl.is_active = 1
WHERE ar.owner_account_id = sqlc.arg('owner_account_id')
AND ar.counterparty_account_id IN (sqlc.slice('customer_account_ids'))
AND ar.account_relation_role_code = 'customer'
UNION ALL
SELECT ar.counterparty_account_id, pl.id, CAST(3 AS SIGNED)
FROM account_relation ar
JOIN price_list_account_group plg ON plg.account_group_id = ar.account_group_id
JOIN price_list pl ON pl.id = plg.price_list_id AND pl.is_active = 1
WHERE ar.owner_account_id = sqlc.arg('owner_account_id')
AND ar.counterparty_account_id IN (sqlc.slice('customer_account_ids'))
AND ar.account_relation_role_code = 'customer'
UNION ALL
SELECT ar.counterparty_account_id, pl.id, CAST(4 AS SIGNED)
FROM account_relation ar
JOIN price_list pl ON pl.account_id = ar.owner_account_id AND pl.source_type = 'account' AND pl.is_active = 1
WHERE ar.owner_account_id = sqlc.arg('owner_account_id')
AND ar.counterparty_account_id IN (sqlc.slice('customer_account_ids'))
AND ar.account_relation_role_code = 'customer';

-- name: IsPriceListLegacySynced :one
SELECT EXISTS (SELECT 1 FROM price_list_legacy_sync WHERE account_id = sqlc.arg('account_id')) AS synced;

-- name: MarkPriceListLegacySynced :exec
INSERT IGNORE INTO price_list_legacy_sync (account_id, synced_at) VALUES (sqlc.arg('account_id'), NOW(3));

-- name: GetLegacyAccountPrices :many
-- Account prices with the customer relation they belong to. ids, when given, narrows to those prices.
SELECT ap.id, ap.recipient_account_id, ar.id AS relation_id, a.name AS recipient_name, ap.product_line_id,
    r.value, r.denominator_unit_id, ap.created_at
FROM account_price ap
JOIN rate r ON r.id = ap.unit_value_id
JOIN account a ON a.id = ap.recipient_account_id
JOIN account_relation ar ON ar.owner_account_id = ap.owner_account_id
    AND ar.counterparty_account_id = ap.recipient_account_id AND ar.account_relation_role_code = 'customer'
WHERE ap.owner_account_id = sqlc.arg('owner_account_id')
ORDER BY ap.created_at, ap.id;

-- name: GetLegacyAccountPriceCategoryIDs :many
SELECT account_price_id, item_category_id FROM account_price_item_category
WHERE account_price_id IN (sqlc.slice('ids'));

-- name: GetLegacyAccountPriceAttributeIDs :many
SELECT account_price_id, attribute_id FROM account_price_attribute
WHERE account_price_id IN (sqlc.slice('ids'));

-- name: GetLegacyVolumeDiscounts :many
SELECT id, name, created_at FROM quantity_discount
WHERE account_id = sqlc.arg('account_id')
ORDER BY created_at, id;

-- name: GetLegacyVolumeDiscountGroupIDs :many
SELECT quantity_discount_id, account_group_id FROM account_group_quantity_discount
WHERE quantity_discount_id IN (sqlc.slice('ids'));

-- name: GetLegacyPricingGroups :many
-- Account groups that carry volume discounts, or that a customer lists as a price group.
SELECT ag.id, ag.name, ag.description FROM account_group ag
WHERE ag.owner_account_id = sqlc.arg('owner_account_id')
AND (
    EXISTS (SELECT 1 FROM account_group_quantity_discount agqd WHERE agqd.account_group_id = ag.id)
    OR EXISTS (SELECT 1 FROM account_relation_price_group arpg WHERE arpg.account_group_id = ag.id)
    OR ag.account_group_type_code = 'pricing_group'
)
ORDER BY ag.created_at, ag.id;

-- name: GetLegacyPriceGroupMembers :many
SELECT account_group_id, account_relation_id FROM account_relation_price_group
WHERE account_group_id IN (sqlc.slice('group_ids'));

-- name: GetLegacyRelationPriceGroupIDs :many
SELECT account_group_id FROM account_relation_price_group WHERE account_relation_id = sqlc.arg('account_relation_id');

-- name: ListAccountIDsAfter :many
SELECT id FROM account WHERE id > sqlc.arg('cursor') ORDER BY id LIMIT ?;

-- name: GetPriceListIDsByCustomerAccounts :many
SELECT ar.counterparty_account_id AS customer_account_id, plc.price_list_id
FROM account_relation ar
JOIN price_list_customer plc ON plc.account_relation_id = ar.id
WHERE ar.owner_account_id = sqlc.arg('owner_account_id')
AND ar.counterparty_account_id IN (sqlc.slice('customer_account_ids'))
AND ar.account_relation_role_code = 'customer'
ORDER BY ar.counterparty_account_id, plc.created_at, plc.price_list_id;
