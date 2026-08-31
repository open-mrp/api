-- 0005_measures.sql
-- Seeds units, unit groups, and unit group assignments.

-- Built-in units (no account_id — global)
INSERT IGNORE INTO unit (id, name, abbreviation, unit_dimension_code, ratio_numerator, ratio_denominator, offset_numerator, offset_denominator, is_base_unit, created_at, updated_at) VALUES
    ('each',   'Each',   'ea',  'quantity', 1, 1, 0, 1, 0, NOW(), NOW()),
    ('dollar', 'Dollar', '$',   'currency', 1, 1, 0, 1, 0, NOW(), NOW()),
    ('hour',   'Hour',   'hr',  'time',     1, 1, 0, 1, 0, NOW(), NOW()),
    ('gram',   'Grams',  'g',   'mass',     1, 1, 0, 1, 0, NOW(), NOW()),
    ('day',    'Day',    'd',   'time',     24, 1, 0, 1, 0, NOW(), NOW()),
    ('minute', 'Minute', 'min', 'time',     1, 60, 0, 1, 0, NOW(), NOW()),
    ('second', 'Second', 's',   'time',     1, 3600, 0, 1, 0, NOW(), NOW());

-- Account-scoped units
INSERT IGNORE INTO unit (id, name, abbreviation, unit_dimension_code, account_id, ratio_numerator, ratio_denominator, offset_numerator, offset_denominator, is_base_unit, created_at, updated_at) VALUES
    ('un_01seeddozen00000000', 'Dozen',  'dz',   'quantity', 'ac_01k0a5smf9ekb8rqg12555zjqa', 12, 1, 0, 1, 0, NOW(), NOW()),
    ('un_01seedpair000000000', 'Pair',   'pr',   'quantity', 'ac_01k0a5smf9ekb8rqg12555zjqa', 2, 1, 0, 1, 0, NOW(), NOW()),
    ('un_01seedpound00000000', 'Pound',  'lbs',  'mass',     'ac_01k0a5smf9ekb8rqg12555zjqa', 45359237, 100000, 0, 1, 0, NOW(), NOW()),
    ('un_01seedgrain00000000', 'Grain',  'gr',   'mass',     'ac_01k0a5smf9ekb8rqg12555zjqa', 6479891, 100000000, 0, 1, 0, NOW(), NOW());

-- Unit groups
-- each_group (base: each, type: quantity) — shared, like time_group and currency_group: the
-- system product lines in 0006_catalog point at it, so it must resolve for every tenant.
INSERT IGNORE INTO unit_group (id, name, base_unit_id, account_id, unit_type_code, created_at, updated_at) VALUES
    ('each_group', 'Each', 'each', NULL, 'quantity', NOW(), NOW());

INSERT IGNORE INTO unit_group_unit (id, unit_group_id, unit_id, created_at, updated_at) VALUES
    ('ungpun_01seedeachgrpeach', 'each_group', 'each', NOW(), NOW());

-- time_group (base: hour, type: time)
INSERT IGNORE INTO unit_group (id, name, base_unit_id, account_id, unit_type_code, created_at, updated_at) VALUES
    ('time_group', 'Time', 'hour', NULL, 'time', NOW(), NOW());

INSERT IGNORE INTO unit_group_unit (id, unit_group_id, unit_id, created_at, updated_at) VALUES
    ('ungpun_01seedtimegrpday', 'time_group', 'day', NOW(), NOW()),
    ('ungpun_01seedtimegrphr0', 'time_group', 'hour', NOW(), NOW()),
    ('ungpun_01seedtimegrpmin', 'time_group', 'minute', NOW(), NOW()),
    ('ungpun_01seedtimegrpsec', 'time_group', 'second', NOW(), NOW());

-- currency_group (base: dollar, type: currency)
INSERT IGNORE INTO unit_group (id, name, base_unit_id, account_id, unit_type_code, created_at, updated_at) VALUES
    ('currency_group', 'Currency', 'dollar', NULL, 'currency', NOW(), NOW());

INSERT IGNORE INTO unit_group_unit (id, unit_group_id, unit_id, created_at, updated_at) VALUES
    ('ungpun_01seedcurrgrpdlr', 'currency_group', 'dollar', NOW(), NOW());

-- socks unit group (base: pair, type: quantity)
INSERT IGNORE INTO unit_group (id, name, base_unit_id, account_id, unit_type_code, created_at, updated_at) VALUES
    ('ungp_01k0a5ecy9edg9za40dnccw53n', 'Socks', 'un_01seedpair000000000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'quantity', NOW(), NOW());

INSERT IGNORE INTO unit_group_unit (id, unit_group_id, unit_id, created_at, updated_at) VALUES
    ('ungpun_01seedsocksea000', 'ungp_01k0a5ecy9edg9za40dnccw53n', 'each', NOW(), NOW()),
    ('ungpun_01seedsockspr000', 'ungp_01k0a5ecy9edg9za40dnccw53n', 'un_01seedpair000000000', NOW(), NOW()),
    ('ungpun_01seedsocksdz000', 'ungp_01k0a5ecy9edg9za40dnccw53n', 'un_01seeddozen00000000', NOW(), NOW());

-- sellable socks unit group (base: pair, type: quantity)
INSERT IGNORE INTO unit_group (id, name, base_unit_id, account_id, unit_type_code, created_at, updated_at) VALUES
    ('ungp_1gf7a8200f8x8jjpq5a9kdrhd', 'Sellable Socks', 'un_01seedpair000000000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'quantity', NOW(), NOW());

INSERT IGNORE INTO unit_group_unit (id, unit_group_id, unit_id, created_at, updated_at) VALUES
    ('ungpun_01seedsellsockpr', 'ungp_1gf7a8200f8x8jjpq5a9kdrhd', 'un_01seedpair000000000', NOW(), NOW()),
    ('ungpun_01seedsellsockdz', 'ungp_1gf7a8200f8x8jjpq5a9kdrhd', 'un_01seeddozen00000000', NOW(), NOW());

-- yarn unit group (base: pound, type: mass)
INSERT IGNORE INTO unit_group (id, name, base_unit_id, account_id, unit_type_code, created_at, updated_at) VALUES
    ('ungp_01k0a51qxceydax5036pegvzzy', 'Yarn', 'un_01seedpound00000000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'mass', NOW(), NOW());

INSERT IGNORE INTO unit_group_unit (id, unit_group_id, unit_id, created_at, updated_at) VALUES
    ('ungpun_01seedyarnlbs000', 'ungp_01k0a51qxceydax5036pegvzzy', 'un_01seedpound00000000', NOW(), NOW()),
    ('ungpun_01seedyarngr0000', 'ungp_01k0a51qxceydax5036pegvzzy', 'un_01seedgrain00000000', NOW(), NOW());

-- chemicals unit group (base: grams, type: mass)
INSERT IGNORE INTO unit_group (id, name, base_unit_id, account_id, unit_type_code, created_at, updated_at) VALUES
    ('ungp_1gf7a8200e2gar6402qnvjem9', 'Chemicals', 'gram', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'mass', NOW(), NOW());

INSERT IGNORE INTO unit_group_unit (id, unit_group_id, unit_id, created_at, updated_at) VALUES
    ('ungpun_01seedchemg00000', 'ungp_1gf7a8200e2gar6402qnvjem9', 'gram', NOW(), NOW());
