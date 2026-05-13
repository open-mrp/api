-- 0007_items.sql
-- Seeds rates, quantities, items, products, materials, parts, and item-attribute associations.

-- ============================================================
-- RATES for products (unitValue, unitCost, burnRate)
-- ============================================================

-- White Small Sock rates
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedwssunitval000', 10, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedwssunitcost00', 7, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedwssburnrate00', 1, 'un_01seedpair000000000', 'day', NOW(), NOW());

-- White Large Sock rates
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedwlsunitval000', 10, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedwlsunitcost00', 7, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedwlsburnrate00', 1, 'un_01seedpair000000000', 'day', NOW(), NOW());

-- Black Small Sock rates
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedbssunitval000', 10, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedbssunitcost00', 7, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedbssburnrate00', 1, 'un_01seedpair000000000', 'day', NOW(), NOW());

-- Black Large Sock rates
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedblsunitval000', 10, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedblsunitcost00', 7, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedblsburnrate00', 1, 'un_01seedpair000000000', 'day', NOW(), NOW());

-- Beige Small Sock rates
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedbgsunitval000', 10, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedbgsunitcost00', 7, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedbgsburnrate00', 1, 'un_01seedpair000000000', 'day', NOW(), NOW());

-- Beige Large Sock rates
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedbglunitval000', 10, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedbglunitcost00', 7, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedbglburnrate00', 1, 'un_01seedpair000000000', 'day', NOW(), NOW());

-- Shipping product rates
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedshpunitval000', 0, 'dollar', 'each', NOW(), NOW()),
    ('rt_01seedshpunitcost00', 0, 'dollar', 'each', NOW(), NOW()),
    ('rt_01seedshpburnrate00', 0, 'each', 'day', NOW(), NOW());

-- Credit product rates
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedcrdunitval000', 0, 'dollar', 'each', NOW(), NOW()),
    ('rt_01seedcrdunitcost00', 0, 'dollar', 'each', NOW(), NOW()),
    ('rt_01seedcrdburnrate00', 0, 'each', 'day', NOW(), NOW());

-- ============================================================
-- RATES for materials (unitValue, unitCost, burnRate)
-- ============================================================

-- Yarn 1 (YRN-001) rates: $6/lb cost
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedyrn1unitval00', 0, 'dollar', 'un_01seedpound00000000', NOW(), NOW()),
    ('rt_01seedyrn1unitcost0', 6, 'dollar', 'un_01seedpound00000000', NOW(), NOW()),
    ('rt_01seedyrn1burnrate0', 0, 'un_01seedpound00000000', 'day', NOW(), NOW());

-- Yarn 2 (YRN-002) rates
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedyrn2unitval00', 0, 'dollar', 'un_01seedpound00000000', NOW(), NOW()),
    ('rt_01seedyrn2unitcost0', 6, 'dollar', 'un_01seedpound00000000', NOW(), NOW()),
    ('rt_01seedyrn2burnrate0', 0, 'un_01seedpound00000000', 'day', NOW(), NOW());

-- Sewing Yarn (YRN-003) rates: $4/lb cost
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedyrn3unitval00', 0, 'dollar', 'un_01seedpound00000000', NOW(), NOW()),
    ('rt_01seedyrn3unitcost0', 4, 'dollar', 'un_01seedpound00000000', NOW(), NOW()),
    ('rt_01seedyrn3burnrate0', 0, 'un_01seedpound00000000', 'day', NOW(), NOW());

-- Beige Dye (DYE-001) rates: $4/g cost
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seeddye1unitval00', 0, 'dollar', 'gram', NOW(), NOW()),
    ('rt_01seeddye1unitcost0', 4, 'dollar', 'gram', NOW(), NOW()),
    ('rt_01seeddye1burnrate0', 0, 'gram', 'day', NOW(), NOW());

-- Black Dye (DYE-002) rates: $3/g cost
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seeddye2unitval00', 0, 'dollar', 'gram', NOW(), NOW()),
    ('rt_01seeddye2unitcost0', 3, 'dollar', 'gram', NOW(), NOW()),
    ('rt_01seeddye2burnrate0', 0, 'gram', 'day', NOW(), NOW());

-- Fabric Softener (CHEM-001) rates: $1/g cost
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedchm1unitval00', 0, 'dollar', 'gram', NOW(), NOW()),
    ('rt_01seedchm1unitcost0', 1, 'dollar', 'gram', NOW(), NOW()),
    ('rt_01seedchm1burnrate0', 0, 'gram', 'day', NOW(), NOW());

-- Detergent (CHEM-002) rates: $1/g cost
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedchm2unitval00', 0, 'dollar', 'gram', NOW(), NOW()),
    ('rt_01seedchm2unitcost0', 1, 'dollar', 'gram', NOW(), NOW()),
    ('rt_01seedchm2burnrate0', 0, 'gram', 'day', NOW(), NOW());

-- Box (BOX-001) rates: $0.5/ea cost
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedbox1unitval00', 0, 'dollar', 'each', NOW(), NOW()),
    ('rt_01seedbox1unitcost0', 0.5, 'dollar', 'each', NOW(), NOW()),
    ('rt_01seedbox1burnrate0', 10, 'each', 'day', NOW(), NOW());

-- Packing Paper (PP-001) rates: $0.5/ea cost
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedpp01unitval00', 0, 'dollar', 'each', NOW(), NOW()),
    ('rt_01seedpp01unitcost0', 0.5, 'dollar', 'each', NOW(), NOW()),
    ('rt_01seedpp01burnrate0', 20, 'each', 'day', NOW(), NOW());

-- ============================================================
-- QUANTITIES for materials (orderPoint, leadTime)
-- ============================================================

INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    -- Yarn 1
    ('qu_01seedyrn1orderpt00', 100, 'un_01seedpound00000000', NOW(), NOW()),
    ('qu_01seedyrn1leadtime0', 14, 'day', NOW(), NOW()),
    -- Yarn 2
    ('qu_01seedyrn2orderpt00', 100, 'un_01seedpound00000000', NOW(), NOW()),
    ('qu_01seedyrn2leadtime0', 14, 'day', NOW(), NOW()),
    -- Sewing Yarn
    ('qu_01seedyrn3orderpt00', 50, 'un_01seedpound00000000', NOW(), NOW()),
    ('qu_01seedyrn3leadtime0', 14, 'day', NOW(), NOW()),
    -- Beige Dye
    ('qu_01seeddye1orderpt00', 500, 'gram', NOW(), NOW()),
    ('qu_01seeddye1leadtime0', 7, 'day', NOW(), NOW()),
    -- Black Dye
    ('qu_01seeddye2orderpt00', 500, 'gram', NOW(), NOW()),
    ('qu_01seeddye2leadtime0', 7, 'day', NOW(), NOW()),
    -- Fabric Softener
    ('qu_01seedchm1orderpt00', 1000, 'gram', NOW(), NOW()),
    ('qu_01seedchm1leadtime0', 10, 'day', NOW(), NOW()),
    -- Detergent
    ('qu_01seedchm2orderpt00', 1000, 'gram', NOW(), NOW()),
    ('qu_01seedchm2leadtime0', 10, 'day', NOW(), NOW()),
    -- Box
    ('qu_01seedbox1orderpt00', 10, 'each', NOW(), NOW()),
    ('qu_01seedbox1leadtime0', 10, 'day', NOW(), NOW()),
    -- Packing Paper
    ('qu_01seedpp01orderpt00', 20, 'each', NOW(), NOW()),
    ('qu_01seedpp01leadtime0', 15, 'day', NOW(), NOW());

-- ============================================================
-- RATES for parts (unitValue, unitCost, burnRate)
-- ============================================================

-- Large Knitted Sock (LKN)
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedlknunitval000', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedlknunitcost00', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedlknburnrate00', 0, 'un_01seedpair000000000', 'day', NOW(), NOW());

-- Large Sewn Sock (LSN)
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedlsnunitval000', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedlsnunitcost00', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedlsnburnrate00', 0, 'un_01seedpair000000000', 'day', NOW(), NOW());

-- Large Washed Sock (LWS)
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedlwsunitval000', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedlwsunitcost00', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedlwsburnrate00', 0, 'un_01seedpair000000000', 'day', NOW(), NOW());

-- Large White Boarded Sock (LWB)
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedlwbunitval000', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedlwbunitcost00', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedlwbburnrate00', 0, 'un_01seedpair000000000', 'day', NOW(), NOW());

-- Large Beige Sock (LBG)
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedlbgunitval000', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedlbgunitcost00', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedlbgburnrate00', 0, 'un_01seedpair000000000', 'day', NOW(), NOW());

-- Large Beige Boarded Sock (LBGB)
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedlbgbunitval00', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedlbgbunitcost0', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedlbgbburnrate0', 0, 'un_01seedpair000000000', 'day', NOW(), NOW());

-- Large Black Sock (LBK)
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedlbkunitval000', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedlbkunitcost00', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedlbkburnrate00', 0, 'un_01seedpair000000000', 'day', NOW(), NOW());

-- Large Black Boarded Sock (LBKB)
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedlbkbunitval00', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedlbkbunitcost0', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedlbkbburnrate0', 0, 'un_01seedpair000000000', 'day', NOW(), NOW());

-- Small Knitted Sock (SKN)
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedsknunitval000', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedsknunitcost00', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedsknburnrate00', 0, 'un_01seedpair000000000', 'day', NOW(), NOW());

-- Small Washed Sock (SWS)
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedswsunitval000', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedswsunitcost00', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedswsburnrate00', 0, 'un_01seedpair000000000', 'day', NOW(), NOW());

-- Small White Boarded Sock (SWB)
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedswbunitval000', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedswbunitcost00', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedswbburnrate00', 0, 'un_01seedpair000000000', 'day', NOW(), NOW());

-- Small Black Sock (SBK)
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedsbkunitval000', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedsbkunitcost00', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedsbkburnrate00', 0, 'un_01seedpair000000000', 'day', NOW(), NOW());

-- Small Black Boarded Sock (SBKB)
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedsbkbunitval00', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedsbkbunitcost0', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedsbkbburnrate0', 0, 'un_01seedpair000000000', 'day', NOW(), NOW());

-- Small Beige Sock (SBG)
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedsbgunitval000', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedsbgunitcost00', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedsbgburnrate00', 0, 'un_01seedpair000000000', 'day', NOW(), NOW());

-- Small Beige Boarded Sock (SBGB)
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedsbgbunitval00', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedsbgbunitcost0', 0, 'dollar', 'un_01seedpair000000000', NOW(), NOW()),
    ('rt_01seedsbgbburnrate0', 0, 'un_01seedpair000000000', 'day', NOW(), NOW());

-- ============================================================
-- ITEMS (products)
-- ============================================================

INSERT IGNORE INTO item (id, sku, description, unit_value_id, unit_cost_id, burn_rate_id, account_id, item_type_code, item_category_id, created_at, updated_at) VALUES
    ('it_01k0a7100aeysrs9vxpeq14yxj', 'SCK-001', 'Small white sock', 'rt_01seedwssunitval000', 'rt_01seedwssunitcost00', 'rt_01seedwssburnrate00', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'product', 'itcg_01seedsocks000000', NOW(), NOW()),
    ('it_01k0a7100aedgv8416p4p2v9ks', 'SCK-002', 'Large white sock', 'rt_01seedwlsunitval000', 'rt_01seedwlsunitcost00', 'rt_01seedwlsburnrate00', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'product', 'itcg_01seedsocks000000', NOW(), NOW()),
    ('it_01k0a7100afdnr1b41917qs27k', 'SCK-003', 'Small black sock', 'rt_01seedbssunitval000', 'rt_01seedbssunitcost00', 'rt_01seedbssburnrate00', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'product', 'itcg_01seedsocks000000', NOW(), NOW()),
    ('it_01k0a7100af709nn7sgg8tbxte', 'SCK-004', 'Large black sock', 'rt_01seedblsunitval000', 'rt_01seedblsunitcost00', 'rt_01seedblsburnrate00', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'product', 'itcg_01seedsocks000000', NOW(), NOW()),
    ('it_01k0a7100aef2997gw0t7nxd9d', 'SCK-005', 'Small beige sock', 'rt_01seedbgsunitval000', 'rt_01seedbgsunitcost00', 'rt_01seedbgsburnrate00', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'product', 'itcg_01seedsocks000000', NOW(), NOW()),
    ('it_01k0a7100ae85v16mmxx5gx2w3', 'SCK-006', 'Large beige sock', 'rt_01seedbglunitval000', 'rt_01seedbglunitcost00', 'rt_01seedbglburnrate00', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'product', 'itcg_01seedsocks000000', NOW(), NOW()),
    ('it_01k0a71009fc5szdjy8mn2nzq5', 'Freight', 'Charges for shipping', 'rt_01seedshpunitval000', 'rt_01seedshpunitcost00', 'rt_01seedshpburnrate00', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'product', 'itcg_01seedshipping000', NOW(), NOW()),
    ('it_01gf7a8200fxat3nef6sh54wpp', 'Credit', 'Charges for credit', 'rt_01seedcrdunitval000', 'rt_01seedcrdunitcost00', 'rt_01seedcrdburnrate00', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'product', 'itcg_01seedcredit00000', NOW(), NOW());

-- Product records
INSERT IGNORE INTO product (id, item_id, product_type_code, product_line_id, created_at, updated_at) VALUES
    ('pd_01k0a65nx2e2crfxrvryyxnmdh', 'it_01k0a7100aeysrs9vxpeq14yxj', 'sale', 'pdln_01k0a735ype5e8nrhv1n5dhq1q', NOW(), NOW()),
    ('pd_01k0a65nx5e3haz2fgfm34hmcz', 'it_01k0a7100aedgv8416p4p2v9ks', 'sale', 'pdln_01k0a735ype5e8nrhv1n5dhq1q', NOW(), NOW()),
    ('pd_01k0a65nx5fjz8m1s3ytayfdby', 'it_01k0a7100afdnr1b41917qs27k', 'sale', 'pdln_01k0a735ype5e8nrhv1n5dhq1q', NOW(), NOW()),
    ('pd_01k0a65nx5eeavcs322b06pgr8', 'it_01k0a7100af709nn7sgg8tbxte', 'sale', 'pdln_01k0a735ype5e8nrhv1n5dhq1q', NOW(), NOW()),
    ('pd_01k0a65nx5fwmt17sqp317ekyr', 'it_01k0a7100aef2997gw0t7nxd9d', 'sale', 'pdln_01k0a735ype5e8nrhv1n5dhq1q', NOW(), NOW()),
    ('pd_01k0a65nx5e67rd1rahv4tdnrp', 'it_01k0a7100ae85v16mmxx5gx2w3', 'sale', 'pdln_01k0a735ype5e8nrhv1n5dhq1q', NOW(), NOW()),
    ('pd_01k0a65nx5fj1bxedew2jvjpwz', 'it_01k0a71009fc5szdjy8mn2nzq5', 'shipping', 'shipping', NOW(), NOW()),
    ('pd_01gf7a8200ffb9wfj69jj6pwj0', 'it_01gf7a8200fxat3nef6sh54wpp', 'credit', 'credit', NOW(), NOW());

-- ============================================================
-- ITEMS (materials)
-- ============================================================

INSERT IGNORE INTO item (id, sku, description, unit_value_id, unit_cost_id, burn_rate_id, account_id, item_type_code, item_category_id, created_at, updated_at) VALUES
    ('it_01seedyrn1item00000', 'YRN-001', 'Yarn Type 1', 'rt_01seedyrn1unitval00', 'rt_01seedyrn1unitcost0', 'rt_01seedyrn1burnrate0', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'material', 'itcg_01seedyarn0000000', NOW(), NOW()),
    ('it_01seedyrn2item00000', 'YRN-002', 'Yarn Type 2', 'rt_01seedyrn2unitval00', 'rt_01seedyrn2unitcost0', 'rt_01seedyrn2burnrate0', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'material', 'itcg_01seedyarn0000000', NOW(), NOW()),
    ('it_01seedyrn3item00000', 'YRN-003', 'Sewing Yarn', 'rt_01seedyrn3unitval00', 'rt_01seedyrn3unitcost0', 'rt_01seedyrn3burnrate0', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'material', 'itcg_01seedyarn0000000', NOW(), NOW()),
    ('it_01seeddye1item00000', 'DYE-001', 'Beige Dye', 'rt_01seeddye1unitval00', 'rt_01seeddye1unitcost0', 'rt_01seeddye1burnrate0', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'material', 'itcg_01seeddye00000000', NOW(), NOW()),
    ('it_01seeddye2item00000', 'DYE-002', 'Black Dye', 'rt_01seeddye2unitval00', 'rt_01seeddye2unitcost0', 'rt_01seeddye2burnrate0', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'material', 'itcg_01seeddye00000000', NOW(), NOW()),
    ('it_01seedchm1item00000', 'CHEM-001', 'Fabric Softener', 'rt_01seedchm1unitval00', 'rt_01seedchm1unitcost0', 'rt_01seedchm1burnrate0', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'material', 'itcg_01seedchemicals00', NOW(), NOW()),
    ('it_01seedchm2item00000', 'CHEM-002', 'Detergent', 'rt_01seedchm2unitval00', 'rt_01seedchm2unitcost0', 'rt_01seedchm2burnrate0', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'material', 'itcg_01seedchemicals00', NOW(), NOW()),
    ('it_01seedbox1item00000', 'BOX-001', 'Box', 'rt_01seedbox1unitval00', 'rt_01seedbox1unitcost0', 'rt_01seedbox1burnrate0', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'material', 'itcg_01seedpackaging00', NOW(), NOW()),
    ('it_01seedpp01item00000', 'PP-001', 'Packing Paper', 'rt_01seedpp01unitval00', 'rt_01seedpp01unitcost0', 'rt_01seedpp01burnrate0', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'material', 'itcg_01seedpackaging00', NOW(), NOW());

-- Material records
INSERT IGNORE INTO material (id, item_id, order_point_id, lead_time_id, created_at, updated_at) VALUES
    ('ml_01seedyrn1mat000000', 'it_01seedyrn1item00000', 'qu_01seedyrn1orderpt00', 'qu_01seedyrn1leadtime0', NOW(), NOW()),
    ('ml_01seedyrn2mat000000', 'it_01seedyrn2item00000', 'qu_01seedyrn2orderpt00', 'qu_01seedyrn2leadtime0', NOW(), NOW()),
    ('ml_01seedyrn3mat000000', 'it_01seedyrn3item00000', 'qu_01seedyrn3orderpt00', 'qu_01seedyrn3leadtime0', NOW(), NOW()),
    ('ml_01seeddye1mat000000', 'it_01seeddye1item00000', 'qu_01seeddye1orderpt00', 'qu_01seeddye1leadtime0', NOW(), NOW()),
    ('ml_01seeddye2mat000000', 'it_01seeddye2item00000', 'qu_01seeddye2orderpt00', 'qu_01seeddye2leadtime0', NOW(), NOW()),
    ('ml_01seedchm1mat000000', 'it_01seedchm1item00000', 'qu_01seedchm1orderpt00', 'qu_01seedchm1leadtime0', NOW(), NOW()),
    ('ml_01seedchm2mat000000', 'it_01seedchm2item00000', 'qu_01seedchm2orderpt00', 'qu_01seedchm2leadtime0', NOW(), NOW()),
    ('ml_01seedbox1mat000000', 'it_01seedbox1item00000', 'qu_01seedbox1orderpt00', 'qu_01seedbox1leadtime0', NOW(), NOW()),
    ('ml_01seedpp01mat000000', 'it_01seedpp01item00000', 'qu_01seedpp01orderpt00', 'qu_01seedpp01leadtime0', NOW(), NOW());

-- ============================================================
-- ITEMS (parts) — intermediate production stages
-- ============================================================

INSERT IGNORE INTO item (id, sku, description, unit_value_id, unit_cost_id, burn_rate_id, account_id, item_type_code, item_category_id, created_at, updated_at) VALUES
    ('it_01seedlknitem000000', 'LKN', 'Large Knitted Sock', 'rt_01seedlknunitval000', 'rt_01seedlknunitcost00', 'rt_01seedlknburnrate00', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'part', 'itcg_01seedsocks000000', NOW(), NOW()),
    ('it_01seedlsnitem000000', 'LSN', 'Large Sewn Sock', 'rt_01seedlsnunitval000', 'rt_01seedlsnunitcost00', 'rt_01seedlsnburnrate00', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'part', 'itcg_01seedsocks000000', NOW(), NOW()),
    ('it_01seedlwsitem000000', 'LWS', 'Large Washed Sock', 'rt_01seedlwsunitval000', 'rt_01seedlwsunitcost00', 'rt_01seedlwsburnrate00', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'part', 'itcg_01seedsocks000000', NOW(), NOW()),
    ('it_01seedlwbitem000000', 'LWB', 'Large White Boarded Sock', 'rt_01seedlwbunitval000', 'rt_01seedlwbunitcost00', 'rt_01seedlwbburnrate00', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'part', 'itcg_01seedsocks000000', NOW(), NOW()),
    ('it_01seedlbgitem000000', 'LBG', 'Large Beige Sock', 'rt_01seedlbgunitval000', 'rt_01seedlbgunitcost00', 'rt_01seedlbgburnrate00', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'part', 'itcg_01seedsocks000000', NOW(), NOW()),
    ('it_01seedlbgbitem00000', 'LBGB', 'Large Beige Boarded Sock', 'rt_01seedlbgbunitval00', 'rt_01seedlbgbunitcost0', 'rt_01seedlbgbburnrate0', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'part', 'itcg_01seedsocks000000', NOW(), NOW()),
    ('it_01seedlbkitem000000', 'LBK', 'Large Black Sock', 'rt_01seedlbkunitval000', 'rt_01seedlbkunitcost00', 'rt_01seedlbkburnrate00', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'part', 'itcg_01seedsocks000000', NOW(), NOW()),
    ('it_01seedlbkbitem00000', 'LBKB', 'Large Black Boarded Sock', 'rt_01seedlbkbunitval00', 'rt_01seedlbkbunitcost0', 'rt_01seedlbkbburnrate0', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'part', 'itcg_01seedsocks000000', NOW(), NOW()),
    ('it_01seedsknitem000000', 'SKN', 'Small Knitted Sock', 'rt_01seedsknunitval000', 'rt_01seedsknunitcost00', 'rt_01seedsknburnrate00', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'part', 'itcg_01seedsocks000000', NOW(), NOW()),
    ('it_01seedswsitem000000', 'SWS', 'Small Washed Sock', 'rt_01seedswsunitval000', 'rt_01seedswsunitcost00', 'rt_01seedswsburnrate00', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'part', 'itcg_01seedsocks000000', NOW(), NOW()),
    ('it_01seedswbitem000000', 'SWB', 'Small White Boarded Sock', 'rt_01seedswbunitval000', 'rt_01seedswbunitcost00', 'rt_01seedswbburnrate00', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'part', 'itcg_01seedsocks000000', NOW(), NOW()),
    ('it_01seedsbkitem000000', 'SBK', 'Small Black Sock', 'rt_01seedsbkunitval000', 'rt_01seedsbkunitcost00', 'rt_01seedsbkburnrate00', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'part', 'itcg_01seedsocks000000', NOW(), NOW()),
    ('it_01seedsbkbitem00000', 'SBKB', 'Small Black Boarded Sock', 'rt_01seedsbkbunitval00', 'rt_01seedsbkbunitcost0', 'rt_01seedsbkbburnrate0', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'part', 'itcg_01seedsocks000000', NOW(), NOW()),
    ('it_01seedsbgitem000000', 'SBG', 'Small Beige Sock', 'rt_01seedsbgunitval000', 'rt_01seedsbgunitcost00', 'rt_01seedsbgburnrate00', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'part', 'itcg_01seedsocks000000', NOW(), NOW()),
    ('it_01seedsbgbitem00000', 'SBGB', 'Small Beige Boarded Sock', 'rt_01seedsbgbunitval00', 'rt_01seedsbgbunitcost0', 'rt_01seedsbgbburnrate0', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'part', 'itcg_01seedsocks000000', NOW(), NOW());

-- Part records
INSERT IGNORE INTO part (id, item_id, created_at, updated_at) VALUES
    ('pt_01seedlknpart000000', 'it_01seedlknitem000000', NOW(), NOW()),
    ('pt_01seedlsnpart000000', 'it_01seedlsnitem000000', NOW(), NOW()),
    ('pt_01seedlwspart000000', 'it_01seedlwsitem000000', NOW(), NOW()),
    ('pt_01seedlwbpart000000', 'it_01seedlwbitem000000', NOW(), NOW()),
    ('pt_01seedlbgpart000000', 'it_01seedlbgitem000000', NOW(), NOW()),
    ('pt_01seedlbgbpart00000', 'it_01seedlbgbitem00000', NOW(), NOW()),
    ('pt_01seedlbkpart000000', 'it_01seedlbkitem000000', NOW(), NOW()),
    ('pt_01seedlbkbpart00000', 'it_01seedlbkbitem00000', NOW(), NOW()),
    ('pt_01seedsknpart000000', 'it_01seedsknitem000000', NOW(), NOW()),
    ('pt_01seedswspart000000', 'it_01seedswsitem000000', NOW(), NOW()),
    ('pt_01seedswbpart000000', 'it_01seedswbitem000000', NOW(), NOW()),
    ('pt_01seedsbkpart000000', 'it_01seedsbkitem000000', NOW(), NOW()),
    ('pt_01seedsbkbpart00000', 'it_01seedsbkbitem00000', NOW(), NOW()),
    ('pt_01seedsbgpart000000', 'it_01seedsbgitem000000', NOW(), NOW()),
    ('pt_01seedsbgbpart00000', 'it_01seedsbgbitem00000', NOW(), NOW());

-- ============================================================
-- INVENTORY LOGS — one per item (products, materials, parts)
-- Each gets a quantity (value=0) + inventory_log + inventory_change_log
-- ============================================================

-- Quantities for inventory logs (value=0, using category base unit)
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    -- Products (6 socks → pair, freight → each, credit → each)
    ('qu_01seedinvlog_wss0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvlog_wls0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvlog_bss0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvlog_bls0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvlog_bgs0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvlog_bgl0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvlog_shp0000', 0, 'each', NOW(), NOW()),
    ('qu_01seedinvlog_crd0000', 0, 'each', NOW(), NOW()),
    -- Materials (3 yarn → pound, 2 dye → gram, 2 chem → gram, 2 packaging → each)
    ('qu_01seedinvlog_yrn1000', 0, 'un_01seedpound00000000', NOW(), NOW()),
    ('qu_01seedinvlog_yrn2000', 0, 'un_01seedpound00000000', NOW(), NOW()),
    ('qu_01seedinvlog_yrn3000', 0, 'un_01seedpound00000000', NOW(), NOW()),
    ('qu_01seedinvlog_dye1000', 0, 'gram', NOW(), NOW()),
    ('qu_01seedinvlog_dye2000', 0, 'gram', NOW(), NOW()),
    ('qu_01seedinvlog_chm1000', 0, 'gram', NOW(), NOW()),
    ('qu_01seedinvlog_chm2000', 0, 'gram', NOW(), NOW()),
    ('qu_01seedinvlog_box1000', 0, 'each', NOW(), NOW()),
    ('qu_01seedinvlog_pp01000', 0, 'each', NOW(), NOW()),
    -- Parts (all socks → pair)
    ('qu_01seedinvlog_lkn0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvlog_lsn0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvlog_lws0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvlog_lwb0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvlog_lbg0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvlog_lbgb000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvlog_lbk0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvlog_lbkb000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvlog_skn0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvlog_sws0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvlog_swb0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvlog_sbk0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvlog_sbkb000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvlog_sbg0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvlog_sbgb000', 0, 'un_01seedpair000000000', NOW(), NOW());

-- Quantities for inventory change logs (value=0, same units)
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedinvcl_wss0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvcl_wls0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvcl_bss0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvcl_bls0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvcl_bgs0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvcl_bgl0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvcl_shp0000', 0, 'each', NOW(), NOW()),
    ('qu_01seedinvcl_crd0000', 0, 'each', NOW(), NOW()),
    ('qu_01seedinvcl_yrn1000', 0, 'un_01seedpound00000000', NOW(), NOW()),
    ('qu_01seedinvcl_yrn2000', 0, 'un_01seedpound00000000', NOW(), NOW()),
    ('qu_01seedinvcl_yrn3000', 0, 'un_01seedpound00000000', NOW(), NOW()),
    ('qu_01seedinvcl_dye1000', 0, 'gram', NOW(), NOW()),
    ('qu_01seedinvcl_dye2000', 0, 'gram', NOW(), NOW()),
    ('qu_01seedinvcl_chm1000', 0, 'gram', NOW(), NOW()),
    ('qu_01seedinvcl_chm2000', 0, 'gram', NOW(), NOW()),
    ('qu_01seedinvcl_box1000', 0, 'each', NOW(), NOW()),
    ('qu_01seedinvcl_pp01000', 0, 'each', NOW(), NOW()),
    ('qu_01seedinvcl_lkn0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvcl_lsn0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvcl_lws0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvcl_lwb0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvcl_lbg0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvcl_lbgb000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvcl_lbk0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvcl_lbkb000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvcl_skn0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvcl_sws0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvcl_swb0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvcl_sbk0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvcl_sbkb000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvcl_sbg0000', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedinvcl_sbgb000', 0, 'un_01seedpair000000000', NOW(), NOW());

-- Inventory log records
INSERT IGNORE INTO inventory_log (id, item_id, quantity_id, account_id, created_at, updated_at) VALUES
    -- Products
    ('ivlg_01seedwss000000000', 'it_01k0a7100aeysrs9vxpeq14yxj', 'qu_01seedinvlog_wss0000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivlg_01seedwls000000000', 'it_01k0a7100aedgv8416p4p2v9ks', 'qu_01seedinvlog_wls0000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivlg_01seedbss000000000', 'it_01k0a7100afdnr1b41917qs27k', 'qu_01seedinvlog_bss0000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivlg_01seedbls000000000', 'it_01k0a7100af709nn7sgg8tbxte', 'qu_01seedinvlog_bls0000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivlg_01seedbgs000000000', 'it_01k0a7100aef2997gw0t7nxd9d', 'qu_01seedinvlog_bgs0000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivlg_01seedbgl000000000', 'it_01k0a7100ae85v16mmxx5gx2w3', 'qu_01seedinvlog_bgl0000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivlg_01seedshp000000000', 'it_01k0a71009fc5szdjy8mn2nzq5', 'qu_01seedinvlog_shp0000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivlg_01seedcrd000000000', 'it_01gf7a8200fxat3nef6sh54wpp', 'qu_01seedinvlog_crd0000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    -- Materials
    ('ivlg_01seedyrn1mat00000', 'it_01seedyrn1item00000', 'qu_01seedinvlog_yrn1000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivlg_01seedyrn2mat00000', 'it_01seedyrn2item00000', 'qu_01seedinvlog_yrn2000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivlg_01seedyrn3mat00000', 'it_01seedyrn3item00000', 'qu_01seedinvlog_yrn3000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivlg_01seeddye1mat00000', 'it_01seeddye1item00000', 'qu_01seedinvlog_dye1000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivlg_01seeddye2mat00000', 'it_01seeddye2item00000', 'qu_01seedinvlog_dye2000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivlg_01seedchm1mat00000', 'it_01seedchm1item00000', 'qu_01seedinvlog_chm1000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivlg_01seedchm2mat00000', 'it_01seedchm2item00000', 'qu_01seedinvlog_chm2000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivlg_01seedbox1mat00000', 'it_01seedbox1item00000', 'qu_01seedinvlog_box1000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivlg_01seedpp01mat00000', 'it_01seedpp01item00000', 'qu_01seedinvlog_pp01000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    -- Parts
    ('ivlg_01seedlknpart0000', 'it_01seedlknitem000000', 'qu_01seedinvlog_lkn0000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivlg_01seedlsnpart0000', 'it_01seedlsnitem000000', 'qu_01seedinvlog_lsn0000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivlg_01seedlwspart0000', 'it_01seedlwsitem000000', 'qu_01seedinvlog_lws0000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivlg_01seedlwbpart0000', 'it_01seedlwbitem000000', 'qu_01seedinvlog_lwb0000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivlg_01seedlbgpart0000', 'it_01seedlbgitem000000', 'qu_01seedinvlog_lbg0000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivlg_01seedlbgbpart000', 'it_01seedlbgbitem00000', 'qu_01seedinvlog_lbgb000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivlg_01seedlbkpart0000', 'it_01seedlbkitem000000', 'qu_01seedinvlog_lbk0000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivlg_01seedlbkbpart000', 'it_01seedlbkbitem00000', 'qu_01seedinvlog_lbkb000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivlg_01seedsknpart0000', 'it_01seedsknitem000000', 'qu_01seedinvlog_skn0000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivlg_01seedswspart0000', 'it_01seedswsitem000000', 'qu_01seedinvlog_sws0000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivlg_01seedswbpart0000', 'it_01seedswbitem000000', 'qu_01seedinvlog_swb0000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivlg_01seedsbkpart0000', 'it_01seedsbkitem000000', 'qu_01seedinvlog_sbk0000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivlg_01seedsbkbpart000', 'it_01seedsbkbitem00000', 'qu_01seedinvlog_sbkb000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivlg_01seedsbgpart0000', 'it_01seedsbgitem000000', 'qu_01seedinvlog_sbg0000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivlg_01seedsbgbpart000', 'it_01seedsbgbitem00000', 'qu_01seedinvlog_sbgb000', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- Inventory change log records
INSERT IGNORE INTO inventory_change_log (id, item_id, quantity_id, action_type_code, account_id, created_at, updated_at) VALUES
    -- Products
    ('ivcl_01seedwss000000000', 'it_01k0a7100aeysrs9vxpeq14yxj', 'qu_01seedinvcl_wss0000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivcl_01seedwls000000000', 'it_01k0a7100aedgv8416p4p2v9ks', 'qu_01seedinvcl_wls0000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivcl_01seedbss000000000', 'it_01k0a7100afdnr1b41917qs27k', 'qu_01seedinvcl_bss0000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivcl_01seedbls000000000', 'it_01k0a7100af709nn7sgg8tbxte', 'qu_01seedinvcl_bls0000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivcl_01seedbgs000000000', 'it_01k0a7100aef2997gw0t7nxd9d', 'qu_01seedinvcl_bgs0000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivcl_01seedbgl000000000', 'it_01k0a7100ae85v16mmxx5gx2w3', 'qu_01seedinvcl_bgl0000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivcl_01seedshp000000000', 'it_01k0a71009fc5szdjy8mn2nzq5', 'qu_01seedinvcl_shp0000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivcl_01seedcrd000000000', 'it_01gf7a8200fxat3nef6sh54wpp', 'qu_01seedinvcl_crd0000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    -- Materials
    ('ivcl_01seedyrn1mat00000', 'it_01seedyrn1item00000', 'qu_01seedinvcl_yrn1000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivcl_01seedyrn2mat00000', 'it_01seedyrn2item00000', 'qu_01seedinvcl_yrn2000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivcl_01seedyrn3mat00000', 'it_01seedyrn3item00000', 'qu_01seedinvcl_yrn3000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivcl_01seeddye1mat00000', 'it_01seeddye1item00000', 'qu_01seedinvcl_dye1000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivcl_01seeddye2mat00000', 'it_01seeddye2item00000', 'qu_01seedinvcl_dye2000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivcl_01seedchm1mat00000', 'it_01seedchm1item00000', 'qu_01seedinvcl_chm1000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivcl_01seedchm2mat00000', 'it_01seedchm2item00000', 'qu_01seedinvcl_chm2000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivcl_01seedbox1mat00000', 'it_01seedbox1item00000', 'qu_01seedinvcl_box1000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivcl_01seedpp01mat00000', 'it_01seedpp01item00000', 'qu_01seedinvcl_pp01000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    -- Parts
    ('ivcl_01seedlknpart0000', 'it_01seedlknitem000000', 'qu_01seedinvcl_lkn0000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivcl_01seedlsnpart0000', 'it_01seedlsnitem000000', 'qu_01seedinvcl_lsn0000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivcl_01seedlwspart0000', 'it_01seedlwsitem000000', 'qu_01seedinvcl_lws0000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivcl_01seedlwbpart0000', 'it_01seedlwbitem000000', 'qu_01seedinvcl_lwb0000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivcl_01seedlbgpart0000', 'it_01seedlbgitem000000', 'qu_01seedinvcl_lbg0000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivcl_01seedlbgbpart000', 'it_01seedlbgbitem00000', 'qu_01seedinvcl_lbgb000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivcl_01seedlbkpart0000', 'it_01seedlbkitem000000', 'qu_01seedinvcl_lbk0000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivcl_01seedlbkbpart000', 'it_01seedlbkbitem00000', 'qu_01seedinvcl_lbkb000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivcl_01seedsknpart0000', 'it_01seedsknitem000000', 'qu_01seedinvcl_skn0000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivcl_01seedswspart0000', 'it_01seedswsitem000000', 'qu_01seedinvcl_sws0000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivcl_01seedswbpart0000', 'it_01seedswbitem000000', 'qu_01seedinvcl_swb0000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivcl_01seedsbkpart0000', 'it_01seedsbkitem000000', 'qu_01seedinvcl_sbk0000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivcl_01seedsbkbpart000', 'it_01seedsbkbitem00000', 'qu_01seedinvcl_sbkb000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivcl_01seedsbgpart0000', 'it_01seedsbgitem000000', 'qu_01seedinvcl_sbg0000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('ivcl_01seedsbgbpart000', 'it_01seedsbgbitem00000', 'qu_01seedinvcl_sbgb000', 'user_action', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- ============================================================
-- Item-Attribute associations (M2M join table)
-- _item_attributes: A = attribute id (alphabetically first), B = item id
-- ============================================================

INSERT IGNORE INTO _item_attributes (A, B) VALUES
    -- White Small Sock: small
    ('at_01seedsmall00000000', 'it_01k0a7100aeysrs9vxpeq14yxj'),
    -- White Large Sock: large
    ('at_01seedlarge00000000', 'it_01k0a7100aedgv8416p4p2v9ks'),
    -- Black Small Sock: small, black
    ('at_01seedsmall00000000', 'it_01k0a7100afdnr1b41917qs27k'),
    ('at_01seedblack00000000', 'it_01k0a7100afdnr1b41917qs27k'),
    -- Black Large Sock: large, black
    ('at_01seedlarge00000000', 'it_01k0a7100af709nn7sgg8tbxte'),
    ('at_01seedblack00000000', 'it_01k0a7100af709nn7sgg8tbxte'),
    -- Beige Small Sock: small, beige
    ('at_01seedsmall00000000', 'it_01k0a7100aef2997gw0t7nxd9d'),
    ('at_01seedbeige00000000', 'it_01k0a7100aef2997gw0t7nxd9d'),
    -- Beige Large Sock: large, beige
    ('at_01seedlarge00000000', 'it_01k0a7100ae85v16mmxx5gx2w3'),
    ('at_01seedbeige00000000', 'it_01k0a7100ae85v16mmxx5gx2w3'),
    -- Seeded material fixture (YRN-001): beige
    ('at_01seedbeige00000000', 'it_01seedyrn1item00000'),
    -- Seeded part fixture: large
    ('at_01seedlarge00000000', 'it_01seedlknitem000000');
