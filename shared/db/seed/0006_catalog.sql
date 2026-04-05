-- 0006_catalog.sql
-- Seeds properties, attributes, categories, and product lines.

-- Properties
INSERT IGNORE INTO property (id, name, account_id, created_at, updated_at) VALUES
    ('pp_01k0a7ntn1ez6aw8x850femxeh', 'Color', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('pp_01k0a7ntn1egx9jjek42zsstrz', 'Size', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('pp_01k0a7ntn1e5g90cp12w4b007v', 'Twist', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('pp_1gf7a8200f5e9x1xtfj7x5ra1', 'Denier', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('pp_01gf7a8200fkgvjzchrnmet5fy', 'Material', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- Attributes
INSERT IGNORE INTO attribute (id, text, `order`, property_id, color_code, account_id, created_at, updated_at) VALUES
    ('at_01seedbeige00000000', 'Beige', 1, 'pp_01k0a7ntn1ez6aw8x850femxeh', 'brown', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('at_01seedblack00000000', 'Black', 2, 'pp_01k0a7ntn1ez6aw8x850femxeh', 'gray', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('at_01seedsmall00000000', 'Small', 1, 'pp_01k0a7ntn1egx9jjek42zsstrz', 'blue', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('at_01seedmedium0000000', 'Medium', 2, 'pp_01k0a7ntn1egx9jjek42zsstrz', 'green', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('at_01seedlarge00000000', 'Large', 3, 'pp_01k0a7ntn1egx9jjek42zsstrz', 'red', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('at_01seedztwist0000000', 'Z-Twist', 1, 'pp_01k0a7ntn1e5g90cp12w4b007v', 'red', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('at_01seedstwist0000000', 'S-Twist', 2, 'pp_01k0a7ntn1e5g90cp12w4b007v', 'orange', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('at_01seeddenier70000000', 'Denier 70', 1, 'pp_1gf7a8200f5e9x1xtfj7x5ra1', 'red', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- Item categories
INSERT IGNORE INTO item_category (id, name, item_category_type_code, unit_group_id, account_id, created_at, updated_at) VALUES
    ('itcg_01seedsocks000000', 'Socks', 'product_category', 'ungp_01k0a5ecy9edg9za40dnccw53n', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('itcg_01seedyarn0000000', 'Yarn', 'material_category', 'ungp_01k0a51qxceydax5036pegvzzy', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('itcg_01seeddye00000000', 'Dye', 'material_category', 'ungp_1gf7a8200e2gar6402qnvjem9', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('itcg_01seedchemicals00', 'Chemicals', 'material_category', 'ungp_1gf7a8200e2gar6402qnvjem9', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('itcg_01seedpackaging00', 'Packaging', 'material_category', 'each_group', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('itcg_01seedshipping000', 'Shipping', 'product_category', 'each_group', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('itcg_01seedcredit00000', 'Credit', 'product_category', 'each_group', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('itcg_01seedebad0000000', 'eBad', 'product_category', 'each_group', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('itcg_01seedlabel000000', 'Label', 'material_category', 'each_group', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- Category-property associations (M2M join table)
-- _item_categories_properties: A = item_category id (alphabetically first: Category), B = property id (Property)
INSERT IGNORE INTO _item_categories_properties (A, B) VALUES
    ('itcg_01seedsocks000000', 'pp_01k0a7ntn1ez6aw8x850femxeh'),
    ('itcg_01seedsocks000000', 'pp_01k0a7ntn1egx9jjek42zsstrz');

-- Product lines
INSERT IGNORE INTO product_line (id, name, unit_group_id, account_id, created_at, updated_at) VALUES
    ('pdln_01k0a735ype5e8nrhv1n5dhq1q', 'Socks', 'ungp_1gf7a8200f8x8jjpq5a9kdrhd', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('shipping', 'Shipping', 'each_group', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('credit', 'Credit', 'each_group', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('pdln_01k0a735ypfjva933tg57wfx0t', 'eBad', 'each_group', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('pdln_01gf7a8200ef99y3gj77z4q25z', 'Pace', 'ungp_1gf7a8200f8x8jjpq5a9kdrhd', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());
