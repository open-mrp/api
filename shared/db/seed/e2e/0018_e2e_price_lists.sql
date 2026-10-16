-- Price lists for the generic list, include and path-param tests. Neither list is assigned to anyone, so they price
-- no order.
INSERT IGNORE INTO price_list (id, account_id, name, description, is_active, currency, created_at, updated_at) VALUES
    ('prls_01seedpricelist0', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'Seed distributor pricing', 'Unassigned seed list.', 1, 'usd', NOW() - INTERVAL 1 MINUTE, NOW()),
    ('prls_01seedpricelist1', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'Seed retail pricing', NULL, 1, 'usd', NOW() - INTERVAL 2 MINUTE, NOW());

INSERT IGNORE INTO price_list_rule (id, price_list_id, account_id, price_type, percent_of_base, created_at, updated_at) VALUES
    ('prlsrw_01seedpricerule0', 'prls_01seedpricelist0', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'percent_of_base', 90, NOW() - INTERVAL 1 MINUTE, NOW()),
    ('prlsrw_01seedpricerule1', 'prls_01seedpricelist0', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'percent_of_base', 95, NOW() - INTERVAL 2 MINUTE, NOW());

INSERT IGNORE INTO price_list_rule_item (price_list_rule_id, item_id) VALUES
    ('prlsrw_01seedpricerule0', 'it_01k0a7100aeysrs9vxpeq14yxj');
INSERT IGNORE INTO price_list_rule_product_line (price_list_rule_id, product_line_id) VALUES
    ('prlsrw_01seedpricerule0', 'pdln_01k0a735ype5e8nrhv1n5dhq1q');
INSERT IGNORE INTO price_list_rule_item_category (price_list_rule_id, item_category_id) VALUES
    ('prlsrw_01seedpricerule0', 'itcg_01seedsocks000000');
INSERT IGNORE INTO price_list_rule_attribute (price_list_rule_id, attribute_id) VALUES
    ('prlsrw_01seedpricerule0', 'at_01seedbeige00000000');
