-- 0008_infrastructure.sql
-- Seeds departments, storage locations, scanning stations, and machines.

-- Departments
INSERT IGNORE INTO department (id, name, account_id, created_at, updated_at) VALUES
    ('dp_01k0a5r01yfx3sj1vy9qgv3dc0', 'Knitting', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('dp_01k0a5r01yf5csvz0jqfznf13d', 'Washing', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('dp_01k0a5r01yehsa8v1vkbdzs7rm', 'Dyeing', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('dp_01k0a5r01yek6v7xnt0mxzzz8m', 'Sewing', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('dp_01k0a5r01yfy9bg55aaqccjf9v', 'Boarding', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('dp_01k0a5r01yfwctjj0n7ev7q65y', 'Packing', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('dp_01gf7a8200e57vj5reeb7y4fhn', 'Inspection', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- Machines (all in knitting department)
INSERT IGNORE INTO machine (id, account_id, name, serial_number, department_id, created_at, updated_at) VALUES
    ('mc_01k0a52fb6eqhtbx9hdxj3vvnh', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'Knitting Machine 1', 'J24-001', 'dp_01k0a5r01yfx3sj1vy9qgv3dc0', NOW(), NOW()),
    ('mc_01k0a52r3vf9p9tn962fkszst5', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'Knitting Machine 2', 'J24-002', 'dp_01k0a5r01yfx3sj1vy9qgv3dc0', NOW(), NOW()),
    ('mc_01k0a52zcjfbzaxy9xtdeym16p', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'Knitting Machine 3', 'J24-003', 'dp_01k0a5r01yfx3sj1vy9qgv3dc0', NOW(), NOW());

-- Scanning stations (with label_type_code and label_size_code)
INSERT IGNORE INTO scanning_station (id, name, scanning_station_type_code, material_check_required, label_type_code, label_size_code, department_id, account_id, created_at, updated_at) VALUES
    ('sgsn_01k0a8201zegarjfsjaw5n7yfv', 'Knitting Station', 'init_batch', 0, 'tag', '1x4', 'dp_01k0a5r01yfx3sj1vy9qgv3dc0', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('sgsn_01k0a8201zfter9hb618v43j9p', 'Washing Station', 'move_batch', 1, 'tag', '1x4', 'dp_01k0a5r01yf5csvz0jqfznf13d', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('sgsn_01k0a8201zfcf8nj1x2mntjmwr', 'Dyeing Station', 'move_batch', 1, 'tag', '1x4', 'dp_01k0a5r01yehsa8v1vkbdzs7rm', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('sgsn_01k0a8201zev8vyp148804tqa4', 'Sewing Station', 'move_batch', 0, 'tag', '1x4', 'dp_01k0a5r01yek6v7xnt0mxzzz8m', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('sgsn_01k0a8201zf1bb2rhmmmxcdqzn', 'Boarding Station', 'split_batch', 0, 'tag', '1x4', 'dp_01k0a5r01yfy9bg55aaqccjf9v', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('sgsn_01k0a8201yf3cagc783svyhk0x', 'Packing Station', 'move_batch', 0, 'tag', '1x4', 'dp_01k0a5r01yfwctjj0n7ev7q65y', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('sgsn_01gf7a8200e57vj5reeb7y4fhn', 'Inspection Station', 'split_batch', 0, 'tag', '1x4', 'dp_01gf7a8200e57vj5reeb7y4fhn', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- Storage locations (1 building + 6 department sections + Internal 1 + Customer Held 1)
INSERT IGNORE INTO storage_location (id, account_id, storage_location_type_code, name, created_at, updated_at) VALUES
    ('sglc_01seedbuilding0000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'building', 'Main Building', NOW(), NOW()),
    ('sglc_01seedknitting000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'section', 'Knitting Section', NOW(), NOW()),
    ('sglc_01seedwashing0000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'section', 'Washing Section', NOW(), NOW()),
    ('sglc_01seeddyeing00000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'section', 'Dyeing Section', NOW(), NOW()),
    ('sglc_01seedsewing00000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'section', 'Sewing Section', NOW(), NOW()),
    ('sglc_01seedboarding000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'section', 'Boarding Section', NOW(), NOW()),
    ('sglc_01seedpacking0000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'section', 'Packing Section', NOW(), NOW()),
    ('sglc_01seedinternal000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'section', 'Internal 1', NOW(), NOW()),
    ('sglc_01seedcustheld000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'section', 'Customer Held 1', NOW(), NOW());

-- Set parent for all sections
UPDATE storage_location SET parent_id = 'sglc_01seedbuilding0000'
    WHERE id IN (
        'sglc_01seedknitting000', 'sglc_01seedwashing0000', 'sglc_01seeddyeing00000',
        'sglc_01seedsewing00000', 'sglc_01seedboarding000', 'sglc_01seedpacking0000',
        'sglc_01seedinternal000', 'sglc_01seedcustheld000'
    )
    AND parent_id IS NULL;

-- Set department location_id to corresponding storage location section
UPDATE department SET location_id = 'sglc_01seedknitting000' WHERE id = 'dp_01k0a5r01yfx3sj1vy9qgv3dc0' AND location_id IS NULL;
UPDATE department SET location_id = 'sglc_01seedwashing0000' WHERE id = 'dp_01k0a5r01yf5csvz0jqfznf13d' AND location_id IS NULL;
UPDATE department SET location_id = 'sglc_01seeddyeing00000' WHERE id = 'dp_01k0a5r01yehsa8v1vkbdzs7rm' AND location_id IS NULL;
UPDATE department SET location_id = 'sglc_01seedsewing00000' WHERE id = 'dp_01k0a5r01yek6v7xnt0mxzzz8m' AND location_id IS NULL;
UPDATE department SET location_id = 'sglc_01seedboarding000' WHERE id = 'dp_01k0a5r01yfy9bg55aaqccjf9v' AND location_id IS NULL;
UPDATE department SET location_id = 'sglc_01seedpacking0000' WHERE id = 'dp_01k0a5r01yfwctjj0n7ev7q65y' AND location_id IS NULL;
