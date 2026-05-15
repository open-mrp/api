-- 0009_production.sql
-- Seeds production steps with labor rates, overhead rates, productions, and consumptions.
-- Only seeding a representative subset of the 20+ production steps for manageability.

-- ============================================================
-- RATES for production steps (labor_rate, overhead_rate)
-- QUANTITIES for production steps (labor_time)
-- ============================================================

-- Knit Large Sock
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedknitlg_labor0', 10, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedknitlg_overh0', 5, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedknitlg_ltime0', 10, 'minute', 'each', NOW(), NOW());

-- Sew Large Sock
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedsewlg_labor00', 10, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedsewlg_overh00', 5, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedsewlg_ltime00', 5, 'minute', 'each', NOW(), NOW());

-- Knit Small Sock
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedknitsm_labor0', 10, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedknitsm_overh0', 5, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedknitsm_ltime0', 8, 'minute', 'each', NOW(), NOW());

-- Wash Large Sock
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedwashlg_labor0', 10, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedwashlg_overh0', 5, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedwashlg_ltime0', 15, 'minute', 'each', NOW(), NOW());

-- Wash Small Sock
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedwashsm_labor0', 10, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedwashsm_overh0', 5, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedwashsm_ltime0', 15, 'minute', 'each', NOW(), NOW());

-- Dye Large Sock Beige
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seeddyelgbg_labor', 10, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seeddyelgbg_overh', 5, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seeddyelgbg_ltime', 20, 'minute', 'each', NOW(), NOW());

-- Dye Large Sock Black
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seeddyelgbk_labor', 10, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seeddyelgbk_overh', 5, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seeddyelgbk_ltime', 20, 'minute', 'each', NOW(), NOW());

-- Dye Small Sock Beige
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seeddyesmbg_labor', 10, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seeddyesmbg_overh', 5, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seeddyesmbg_ltime', 20, 'minute', 'each', NOW(), NOW());

-- Dye Small Sock Black
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seeddyesmbk_labor', 10, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seeddyesmbk_overh', 5, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seeddyesmbk_ltime', 20, 'minute', 'each', NOW(), NOW());

-- Board Large White Sock
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedbrdlgwh_labor', 10, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedbrdlgwh_overh', 5, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedbrdlgwh_ltime', 3, 'minute', 'each', NOW(), NOW());

-- Board Small White Sock
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedbrdsmwh_labor', 10, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedbrdsmwh_overh', 5, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedbrdsmwh_ltime', 3, 'minute', 'each', NOW(), NOW());

-- Board Large Beige Sock
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedbrdlgbg_labor', 10, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedbrdlgbg_overh', 5, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedbrdlgbg_ltime', 3, 'minute', 'each', NOW(), NOW());

-- Board Large Black Sock
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedbrdlgbk_labor', 10, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedbrdlgbk_overh', 5, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedbrdlgbk_ltime', 3, 'minute', 'each', NOW(), NOW());

-- Board Small Beige Sock
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedbrdsmbg_labor', 10, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedbrdsmbg_overh', 5, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedbrdsmbg_ltime', 3, 'minute', 'each', NOW(), NOW());

-- Board Small Black Sock
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedbrdsmbk_labor', 10, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedbrdsmbk_overh', 5, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedbrdsmbk_ltime', 3, 'minute', 'each', NOW(), NOW());

-- Pack Large White Sock
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedpcklgwh_labor', 10, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedpcklgwh_overh', 5, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedpcklgwh_ltime', 2, 'minute', 'each', NOW(), NOW());

-- Pack Large Beige Sock
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedpcklgbg_labor', 10, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedpcklgbg_overh', 5, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedpcklgbg_ltime', 2, 'minute', 'each', NOW(), NOW());

-- Pack Large Black Sock
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedpcklgbk_labor', 10, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedpcklgbk_overh', 5, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedpcklgbk_ltime', 2, 'minute', 'each', NOW(), NOW());

-- Pack Small White Sock
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedpcksmwh_labor', 10, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedpcksmwh_overh', 5, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedpcksmwh_ltime', 2, 'minute', 'each', NOW(), NOW());

-- Pack Small Beige Sock
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedpcksmbg_labor', 10, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedpcksmbg_overh', 5, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedpcksmbg_ltime', 2, 'minute', 'each', NOW(), NOW());

-- Pack Small Black Sock
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_01seedpcksmbk_labor', 10, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedpcksmbk_overh', 5, 'dollar', 'hour', NOW(), NOW()),
    ('rt_01seedpcksmbk_ltime', 2, 'minute', 'each', NOW(), NOW());

-- ============================================================
-- PRODUCTION STEPS
-- ============================================================

INSERT IGNORE INTO production_step (id, name, account_id, department_id, scanning_station_id, labor_rate_id, labor_time_id, overhead_rate_id, created_at, updated_at) VALUES
    ('prs_01k0a51qxceydax5036pegvzzy', 'Knit Large Sock', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'dp_01k0a5r01yfx3sj1vy9qgv3dc0', 'sgsn_01k0a8201zegarjfsjaw5n7yfv', 'rt_01seedknitlg_labor0', 'rt_01seedknitlg_ltime0', 'rt_01seedknitlg_overh0', NOW(), NOW()),
    ('prs_01k0a56yc1e8wag6wexn4pp8t9', 'Sew Large Sock', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'dp_01k0a5r01yek6v7xnt0mxzzz8m', 'sgsn_01k0a8201zev8vyp148804tqa4', 'rt_01seedsewlg_labor00', 'rt_01seedsewlg_ltime00', 'rt_01seedsewlg_overh00', NOW(), NOW()),
    ('prs_01k0a575j3fqr97khk36v114nj', 'Knit Small Sock', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'dp_01k0a5r01yfx3sj1vy9qgv3dc0', 'sgsn_01k0a8201zegarjfsjaw5n7yfv', 'rt_01seedknitsm_labor0', 'rt_01seedknitsm_ltime0', 'rt_01seedknitsm_overh0', NOW(), NOW()),
    ('prs_01k0a57f3dfsmtzc8txbq43eth', 'Wash Small Sock', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'dp_01k0a5r01yf5csvz0jqfznf13d', 'sgsn_01k0a8201zfter9hb618v43j9p', 'rt_01seedwashsm_labor0', 'rt_01seedwashsm_ltime0', 'rt_01seedwashsm_overh0', NOW(), NOW()),
    ('prs_01k0a57qbefecte8erp0mp6vqb', 'Wash Large Sock', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'dp_01k0a5r01yf5csvz0jqfznf13d', 'sgsn_01k0a8201zfter9hb618v43j9p', 'rt_01seedwashlg_labor0', 'rt_01seedwashlg_ltime0', 'rt_01seedwashlg_overh0', NOW(), NOW()),
    ('prs_01k0a5k18seysr468ykrd8fpnj', 'Board Large White Sock', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'dp_01k0a5r01yfy9bg55aaqccjf9v', 'sgsn_01k0a8201zf1bb2rhmmmxcdqzn', 'rt_01seedbrdlgwh_labor', 'rt_01seedbrdlgwh_ltime', 'rt_01seedbrdlgwh_overh', NOW(), NOW()),
    ('prs_01k0a5kfpnf0gs570fjamctsca', 'Board Small White Sock', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'dp_01k0a5r01yfy9bg55aaqccjf9v', 'sgsn_01k0a8201zf1bb2rhmmmxcdqzn', 'rt_01seedbrdsmwh_labor', 'rt_01seedbrdsmwh_ltime', 'rt_01seedbrdsmwh_overh', NOW(), NOW()),
    ('prs_01k0a587pdene9ysk0xktc7zc7', 'Board Large Beige Sock', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'dp_01k0a5r01yfy9bg55aaqccjf9v', 'sgsn_01k0a8201zf1bb2rhmmmxcdqzn', 'rt_01seedbrdlgbg_labor', 'rt_01seedbrdlgbg_ltime', 'rt_01seedbrdlgbg_overh', NOW(), NOW()),
    ('prs_01k0a5a2ezen9rbvh3aa97m64f', 'Dye Large Sock Beige', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'dp_01k0a5r01yehsa8v1vkbdzs7rm', 'sgsn_01k0a8201zfcf8nj1x2mntjmwr', 'rt_01seeddyelgbg_labor', 'rt_01seeddyelgbg_ltime', 'rt_01seeddyelgbg_overh', NOW(), NOW()),
    ('prs_01k0a5a92wf7qrrgq893dq79pp', 'Dye Large Sock Black', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'dp_01k0a5r01yehsa8v1vkbdzs7rm', 'sgsn_01k0a8201zfcf8nj1x2mntjmwr', 'rt_01seeddyelgbk_labor', 'rt_01seeddyelgbk_ltime', 'rt_01seeddyelgbk_overh', NOW(), NOW()),
    ('prs_01k0a5kr3jf9w83bqnt3y70vjy', 'Dye Small Sock Beige', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'dp_01k0a5r01yehsa8v1vkbdzs7rm', 'sgsn_01k0a8201zfcf8nj1x2mntjmwr', 'rt_01seeddyesmbg_labor', 'rt_01seeddyesmbg_ltime', 'rt_01seeddyesmbg_overh', NOW(), NOW()),
    ('prs_01k0a5m0yjfk19kf3n52bkbve6', 'Dye Small Sock Black', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'dp_01k0a5r01yehsa8v1vkbdzs7rm', 'sgsn_01k0a8201zfcf8nj1x2mntjmwr', 'rt_01seeddyesmbk_labor', 'rt_01seeddyesmbk_ltime', 'rt_01seeddyesmbk_overh', NOW(), NOW()),
    ('prs_01k0a5m985fhzbasqkt6sx22a0', 'Board Large Black Sock', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'dp_01k0a5r01yfy9bg55aaqccjf9v', 'sgsn_01k0a8201zf1bb2rhmmmxcdqzn', 'rt_01seedbrdlgbk_labor', 'rt_01seedbrdlgbk_ltime', 'rt_01seedbrdlgbk_overh', NOW(), NOW()),
    ('prs_01k0a5mgq1fq5a9cvgev5zsf57', 'Board Small Beige Sock', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'dp_01k0a5r01yfy9bg55aaqccjf9v', 'sgsn_01k0a8201zf1bb2rhmmmxcdqzn', 'rt_01seedbrdsmbg_labor', 'rt_01seedbrdsmbg_ltime', 'rt_01seedbrdsmbg_overh', NOW(), NOW()),
    ('prs_01k0a5ncadf1tbcb91kae06tvq', 'Board Small Black Sock', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'dp_01k0a5r01yfy9bg55aaqccjf9v', 'sgsn_01k0a8201zf1bb2rhmmmxcdqzn', 'rt_01seedbrdsmbk_labor', 'rt_01seedbrdsmbk_ltime', 'rt_01seedbrdsmbk_overh', NOW(), NOW()),
    ('prs_01k0a5nzd2f3a9cffpw38qken6', 'Pack Large White Sock', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'dp_01k0a5r01yfwctjj0n7ev7q65y', 'sgsn_01k0a8201yf3cagc783svyhk0x', 'rt_01seedpcklgwh_labor', 'rt_01seedpcklgwh_ltime', 'rt_01seedpcklgwh_overh', NOW(), NOW()),
    ('prs_01k0a5nzd2fxnv34tm431kr7vv', 'Pack Large Beige Sock', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'dp_01k0a5r01yfwctjj0n7ev7q65y', 'sgsn_01k0a8201yf3cagc783svyhk0x', 'rt_01seedpcklgbg_labor', 'rt_01seedpcklgbg_ltime', 'rt_01seedpcklgbg_overh', NOW(), NOW()),
    ('prs_01k0a5nzd2e55rw1bwmt8sdwye', 'Pack Large Black Sock', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'dp_01k0a5r01yfwctjj0n7ev7q65y', 'sgsn_01k0a8201yf3cagc783svyhk0x', 'rt_01seedpcklgbk_labor', 'rt_01seedpcklgbk_ltime', 'rt_01seedpcklgbk_overh', NOW(), NOW()),
    ('prs_01k0a5nzd2e5fs4d3yvf8ehk41', 'Pack Small White Sock', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'dp_01k0a5r01yfwctjj0n7ev7q65y', 'sgsn_01k0a8201yf3cagc783svyhk0x', 'rt_01seedpcksmwh_labor', 'rt_01seedpcksmwh_ltime', 'rt_01seedpcksmwh_overh', NOW(), NOW()),
    ('prs_01k0a5nzd2ek69mct9w40w3h6c', 'Pack Small Beige Sock', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'dp_01k0a5r01yfwctjj0n7ev7q65y', 'sgsn_01k0a8201yf3cagc783svyhk0x', 'rt_01seedpcksmbg_labor', 'rt_01seedpcksmbg_ltime', 'rt_01seedpcksmbg_overh', NOW(), NOW()),
    ('prs_01k0a5nzd2fdw8hvff2sh4bvb3', 'Pack Small Black Sock', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'dp_01k0a5r01yfwctjj0n7ev7q65y', 'sgsn_01k0a8201yf3cagc783svyhk0x', 'rt_01seedpcksmbk_labor', 'rt_01seedpcksmbk_ltime', 'rt_01seedpcksmbk_overh', NOW(), NOW());

-- Fix scanning_station_id in case INSERT IGNORE kept stale values from older seeds.
UPDATE production_step SET scanning_station_id = 'sgsn_01k0a8201zegarjfsjaw5n7yfv' WHERE id IN ('prs_01k0a51qxceydax5036pegvzzy', 'prs_01k0a575j3fqr97khk36v114nj') AND scanning_station_id != 'sgsn_01k0a8201zegarjfsjaw5n7yfv';
UPDATE production_step SET scanning_station_id = 'sgsn_01k0a8201zev8vyp148804tqa4' WHERE id = 'prs_01k0a56yc1e8wag6wexn4pp8t9' AND scanning_station_id != 'sgsn_01k0a8201zev8vyp148804tqa4';
UPDATE production_step SET scanning_station_id = 'sgsn_01k0a8201zfter9hb618v43j9p' WHERE id IN ('prs_01k0a57f3dfsmtzc8txbq43eth', 'prs_01k0a57qbefecte8erp0mp6vqb') AND scanning_station_id != 'sgsn_01k0a8201zfter9hb618v43j9p';

UPDATE machine SET production_step_id = 'prs_01k0a51qxceydax5036pegvzzy' WHERE id = 'mc_01k0a52fb6eqhtbx9hdxj3vvnh' AND (production_step_id IS NULL OR production_step_id = '');

-- Sew Large Sock: machine on the sewing department so GET /production-steps/{id} includes
-- (machines + in_steps) can be exercised against the same seeded step as graph mid-nodes.
INSERT IGNORE INTO machine (id, name, serial_number, department_id, production_step_id, created_at, updated_at) VALUES
    ('mc_01seedsewlgmachine0', 'Sewing Machine 1', 'JUKI-001', 'dp_01k0a5r01yek6v7xnt0mxzzz8m', 'prs_01k0a56yc1e8wag6wexn4pp8t9', NOW(3), NOW(3));

-- ============================================================
-- PRODUCTION RUN (seeded for e2e / OpenAPI list path resolution)
-- ============================================================
INSERT IGNORE INTO production_run (id, responsible_user_id, number, account_id, created_at, updated_at) VALUES
    ('pnrn_01seedprod_run0000', 'acus_s83fjhyfmqen', 'E2E-SEED-PR-001', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- ============================================================
-- PRODUCTIONS (what each step produces)
-- ============================================================

INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedprod_knitlg00', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedprod_sewlg000', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedprod_knitsm00', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedprod_washsm00', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedprod_washlg00', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedprod_brdlgwh0', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedprod_brdsmwh0', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedprod_brdlgbg0', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedprod_dyelgbg0', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedprod_dyelgbk0', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedprod_dyesmbg0', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedprod_dyesmbk0', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedprod_brdlgbk0', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedprod_brdsmbg0', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedprod_brdsmbk0', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedprod_pcklgwh0', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedprod_pcklgbg0', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedprod_pcklgbk0', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedprod_pcksmwh0', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedprod_pcksmbg0', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedprod_pcksmbk0', 1, 'un_01seedpair000000000', NOW(), NOW());

INSERT IGNORE INTO production (id, item_id, quantity_id, production_step_id, created_at, updated_at) VALUES
    ('pn_01seedprod_knitlg00', 'it_01seedlknitem000000', 'qu_01seedprod_knitlg00', 'prs_01k0a51qxceydax5036pegvzzy', NOW(), NOW()),
    ('pn_01seedprod_sewlg000', 'it_01seedlsnitem000000', 'qu_01seedprod_sewlg000', 'prs_01k0a56yc1e8wag6wexn4pp8t9', NOW(), NOW()),
    ('pn_01seedprod_knitsm00', 'it_01seedsknitem000000', 'qu_01seedprod_knitsm00', 'prs_01k0a575j3fqr97khk36v114nj', NOW(), NOW()),
    ('pn_01seedprod_washsm00', 'it_01seedswsitem000000', 'qu_01seedprod_washsm00', 'prs_01k0a57f3dfsmtzc8txbq43eth', NOW(), NOW()),
    ('pn_01seedprod_washlg00', 'it_01seedlwsitem000000', 'qu_01seedprod_washlg00', 'prs_01k0a57qbefecte8erp0mp6vqb', NOW(), NOW()),
    ('pn_01seedprod_brdlgwh0', 'it_01seedlwbitem000000', 'qu_01seedprod_brdlgwh0', 'prs_01k0a5k18seysr468ykrd8fpnj', NOW(), NOW()),
    ('pn_01seedprod_brdsmwh0', 'it_01seedswbitem000000', 'qu_01seedprod_brdsmwh0', 'prs_01k0a5kfpnf0gs570fjamctsca', NOW(), NOW()),
    ('pn_01seedprod_brdlgbg0', 'it_01seedlbgbitem00000', 'qu_01seedprod_brdlgbg0', 'prs_01k0a587pdene9ysk0xktc7zc7', NOW(), NOW()),
    ('pn_01seedprod_dyelgbg0', 'it_01seedlbgitem000000', 'qu_01seedprod_dyelgbg0', 'prs_01k0a5a2ezen9rbvh3aa97m64f', NOW(), NOW()),
    ('pn_01seedprod_dyelgbk0', 'it_01seedlbkitem000000', 'qu_01seedprod_dyelgbk0', 'prs_01k0a5a92wf7qrrgq893dq79pp', NOW(), NOW()),
    ('pn_01seedprod_dyesmbg0', 'it_01seedsbgitem000000', 'qu_01seedprod_dyesmbg0', 'prs_01k0a5kr3jf9w83bqnt3y70vjy', NOW(), NOW()),
    ('pn_01seedprod_dyesmbk0', 'it_01seedsbkitem000000', 'qu_01seedprod_dyesmbk0', 'prs_01k0a5m0yjfk19kf3n52bkbve6', NOW(), NOW()),
    ('pn_01seedprod_brdlgbk0', 'it_01seedlbkbitem00000', 'qu_01seedprod_brdlgbk0', 'prs_01k0a5m985fhzbasqkt6sx22a0', NOW(), NOW()),
    ('pn_01seedprod_brdsmbg0', 'it_01seedsbgbitem00000', 'qu_01seedprod_brdsmbg0', 'prs_01k0a5mgq1fq5a9cvgev5zsf57', NOW(), NOW()),
    ('pn_01seedprod_brdsmbk0', 'it_01seedsbkbitem00000', 'qu_01seedprod_brdsmbk0', 'prs_01k0a5ncadf1tbcb91kae06tvq', NOW(), NOW()),
    ('pn_01seedprod_pcklgwh0', 'it_01k0a7100aeysrs9vxpeq14yxj', 'qu_01seedprod_pcklgwh0', 'prs_01k0a5nzd2f3a9cffpw38qken6', NOW(), NOW()),
    ('pn_01seedprod_pcklgbg0', 'it_01k0a7100ae85v16mmxx5gx2w3', 'qu_01seedprod_pcklgbg0', 'prs_01k0a5nzd2fxnv34tm431kr7vv', NOW(), NOW()),
    ('pn_01seedprod_pcklgbk0', 'it_01k0a7100af709nn7sgg8tbxte', 'qu_01seedprod_pcklgbk0', 'prs_01k0a5nzd2e55rw1bwmt8sdwye', NOW(), NOW()),
    ('pn_01seedprod_pcksmwh0', 'it_01k0a7100aeysrs9vxpeq14yxj', 'qu_01seedprod_pcksmwh0', 'prs_01k0a5nzd2e5fs4d3yvf8ehk41', NOW(), NOW()),
    ('pn_01seedprod_pcksmbg0', 'it_01k0a7100aef2997gw0t7nxd9d', 'qu_01seedprod_pcksmbg0', 'prs_01k0a5nzd2ek69mct9w40w3h6c', NOW(), NOW()),
    ('pn_01seedprod_pcksmbk0', 'it_01k0a7100afdnr1b41917qs27k', 'qu_01seedprod_pcksmbk0', 'prs_01k0a5nzd2fdw8hvff2sh4bvb3', NOW(), NOW());

-- ============================================================
-- CONSUMPTIONS
-- Each consumption needs a quantity + waste_quantity in the quantity table.
-- The consumed item must match the produced item of the parent step
-- in the production flow graph for findCommonConsumptionAndProduction to work.
--
-- Step IDs reference:
--   knitLarge  = prs_01k0a51qxceydax5036pegvzzy   sewLarge   = prs_01k0a56yc1e8wag6wexn4pp8t9
--   knitSmall  = prs_01k0a575j3fqr97khk36v114nj   washSmall  = prs_01k0a57f3dfsmtzc8txbq43eth
--   washLarge  = prs_01k0a57qbefecte8erp0mp6vqb   dyeLgBeige = prs_01k0a5a2ezen9rbvh3aa97m64f
--   dyeLgBlack = prs_01k0a5a92wf7qrrgq893dq79pp   dyeSmBeige = prs_01k0a5kr3jf9w83bqnt3y70vjy
--   dyeSmBlack = prs_01k0a5m0yjfk19kf3n52bkbve6   brdLgWhite = prs_01k0a5k18seysr468ykrd8fpnj
--   brdSmWhite = prs_01k0a5kfpnf0gs570fjamctsca   brdLgBeige = prs_01k0a587pdene9ysk0xktc7zc7
--   brdLgBlack = prs_01k0a5m985fhzbasqkt6sx22a0   brdSmBeige = prs_01k0a5mgq1fq5a9cvgev5zsf57
--   brdSmBlack = prs_01k0a5ncadf1tbcb91kae06tvq   pckLgWhite = prs_01k0a5nzd2f3a9cffpw38qken6
--   pckLgBeige = prs_01k0a5nzd2fxnv34tm431kr7vv   pckLgBlack = prs_01k0a5nzd2e55rw1bwmt8sdwye
--   pckSmWhite = prs_01k0a5nzd2e5fs4d3yvf8ehk41   pckSmBeige = prs_01k0a5nzd2ek69mct9w40w3h6c
--   pckSmBlack = prs_01k0a5nzd2fdw8hvff2sh4bvb3
-- ============================================================

-- Knit Large Sock consumes yarn1 and yarn2
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedcons_kl_y1qty', 0.06, 'un_01seedpound00000000', NOW(), NOW()),
    ('qu_01seedcons_kl_y1wst', 0, 'un_01seedpound00000000', NOW(), NOW()),
    ('qu_01seedcons_kl_y2qty', 0.06, 'un_01seedpound00000000', NOW(), NOW()),
    ('qu_01seedcons_kl_y2wst', 0, 'un_01seedpound00000000', NOW(), NOW());
INSERT IGNORE INTO consumption (id, item_id, quantity_id, waste_quantity_id, production_step_id, created_at, updated_at) VALUES
    ('cp_01seedcons_kl_yarn1', 'it_01seedyrn1item00000', 'qu_01seedcons_kl_y1qty', 'qu_01seedcons_kl_y1wst', 'prs_01k0a51qxceydax5036pegvzzy', NOW(), NOW()),
    ('cp_01seedcons_kl_yarn2', 'it_01seedyrn2item00000', 'qu_01seedcons_kl_y2qty', 'qu_01seedcons_kl_y2wst', 'prs_01k0a51qxceydax5036pegvzzy', NOW(), NOW());

-- Sew Large Sock consumes large knitted sock (LKN) + sewing yarn
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedcons_sl_lkqty', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_sl_lkwst', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_sl_y3qty', 0.01, 'un_01seedpound00000000', NOW(), NOW()),
    ('qu_01seedcons_sl_y3wst', 0, 'un_01seedpound00000000', NOW(), NOW());
INSERT IGNORE INTO consumption (id, item_id, quantity_id, waste_quantity_id, production_step_id, created_at, updated_at) VALUES
    ('cp_01seedcons_sl_lknit', 'it_01seedlknitem000000', 'qu_01seedcons_sl_lkqty', 'qu_01seedcons_sl_lkwst', 'prs_01k0a56yc1e8wag6wexn4pp8t9', NOW(), NOW()),
    ('cp_01seedcons_sl_yarn3', 'it_01seedyrn3item00000', 'qu_01seedcons_sl_y3qty', 'qu_01seedcons_sl_y3wst', 'prs_01k0a56yc1e8wag6wexn4pp8t9', NOW(), NOW());

-- Knit Small Sock consumes yarn1 and yarn2
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedcons_ks_y1qty', 0.04, 'un_01seedpound00000000', NOW(), NOW()),
    ('qu_01seedcons_ks_y1wst', 0, 'un_01seedpound00000000', NOW(), NOW()),
    ('qu_01seedcons_ks_y2qty', 0.04, 'un_01seedpound00000000', NOW(), NOW()),
    ('qu_01seedcons_ks_y2wst', 0, 'un_01seedpound00000000', NOW(), NOW());
INSERT IGNORE INTO consumption (id, item_id, quantity_id, waste_quantity_id, production_step_id, created_at, updated_at) VALUES
    ('cp_01seedcons_ks_yarn1', 'it_01seedyrn1item00000', 'qu_01seedcons_ks_y1qty', 'qu_01seedcons_ks_y1wst', 'prs_01k0a575j3fqr97khk36v114nj', NOW(), NOW()),
    ('cp_01seedcons_ks_yarn2', 'it_01seedyrn2item00000', 'qu_01seedcons_ks_y2qty', 'qu_01seedcons_ks_y2wst', 'prs_01k0a575j3fqr97khk36v114nj', NOW(), NOW());

-- Wash Small Sock consumes small knitted sock (SKN) + fabric softener
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedcons_ws_skqty', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_ws_skwst', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_ws_fsqty', 3, 'gram', NOW(), NOW()),
    ('qu_01seedcons_ws_fswst', 0, 'gram', NOW(), NOW());
INSERT IGNORE INTO consumption (id, item_id, quantity_id, waste_quantity_id, production_step_id, created_at, updated_at) VALUES
    ('cp_01seedcons_ws_sknit', 'it_01seedsknitem000000', 'qu_01seedcons_ws_skqty', 'qu_01seedcons_ws_skwst', 'prs_01k0a57f3dfsmtzc8txbq43eth', NOW(), NOW()),
    ('cp_01seedcons_ws_fabsf', 'it_01seedchm1item00000', 'qu_01seedcons_ws_fsqty', 'qu_01seedcons_ws_fswst', 'prs_01k0a57f3dfsmtzc8txbq43eth', NOW(), NOW());

-- Wash Large Sock consumes large sewn sock (LSN) + fabric softener
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedcons_wl_lsqty', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_wl_lswst', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_wl_fsqty', 3, 'gram', NOW(), NOW()),
    ('qu_01seedcons_wl_fswst', 0, 'gram', NOW(), NOW());
INSERT IGNORE INTO consumption (id, item_id, quantity_id, waste_quantity_id, production_step_id, created_at, updated_at) VALUES
    ('cp_01seedcons_wl_lsewn', 'it_01seedlsnitem000000', 'qu_01seedcons_wl_lsqty', 'qu_01seedcons_wl_lswst', 'prs_01k0a57qbefecte8erp0mp6vqb', NOW(), NOW()),
    ('cp_01seedcons_wl_fabsf', 'it_01seedchm1item00000', 'qu_01seedcons_wl_fsqty', 'qu_01seedcons_wl_fswst', 'prs_01k0a57qbefecte8erp0mp6vqb', NOW(), NOW());

-- Dye Large Sock Beige consumes large sewn sock (LSN) + beige dye + fabric softener
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedcons_dlb_lsqt', 50, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_dlb_lswt', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_dlb_dyqt', 1.5, 'gram', NOW(), NOW()),
    ('qu_01seedcons_dlb_dywt', 0.15, 'gram', NOW(), NOW()),
    ('qu_01seedcons_dlb_fsqt', 200, 'gram', NOW(), NOW()),
    ('qu_01seedcons_dlb_fswt', 7, 'gram', NOW(), NOW());
INSERT IGNORE INTO consumption (id, item_id, quantity_id, waste_quantity_id, production_step_id, created_at, updated_at) VALUES
    ('cp_01seedcons_dlb_sewn', 'it_01seedlsnitem000000', 'qu_01seedcons_dlb_lsqt', 'qu_01seedcons_dlb_lswt', 'prs_01k0a5a2ezen9rbvh3aa97m64f', NOW(), NOW()),
    ('cp_01seedcons_dlb_dye0', 'it_01seeddye1item00000', 'qu_01seedcons_dlb_dyqt', 'qu_01seedcons_dlb_dywt', 'prs_01k0a5a2ezen9rbvh3aa97m64f', NOW(), NOW()),
    ('cp_01seedcons_dlb_fsof', 'it_01seedchm1item00000', 'qu_01seedcons_dlb_fsqt', 'qu_01seedcons_dlb_fswt', 'prs_01k0a5a2ezen9rbvh3aa97m64f', NOW(), NOW());

-- Dye Large Sock Black consumes large sewn sock (LSN) + black dye + fabric softener
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedcons_dlk_lsqt', 50, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_dlk_lswt', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_dlk_dyqt', 2, 'gram', NOW(), NOW()),
    ('qu_01seedcons_dlk_dywt', 0.2, 'gram', NOW(), NOW()),
    ('qu_01seedcons_dlk_fsqt', 200, 'gram', NOW(), NOW()),
    ('qu_01seedcons_dlk_fswt', 7, 'gram', NOW(), NOW());
INSERT IGNORE INTO consumption (id, item_id, quantity_id, waste_quantity_id, production_step_id, created_at, updated_at) VALUES
    ('cp_01seedcons_dlk_sewn', 'it_01seedlsnitem000000', 'qu_01seedcons_dlk_lsqt', 'qu_01seedcons_dlk_lswt', 'prs_01k0a5a92wf7qrrgq893dq79pp', NOW(), NOW()),
    ('cp_01seedcons_dlk_dye0', 'it_01seeddye2item00000', 'qu_01seedcons_dlk_dyqt', 'qu_01seedcons_dlk_dywt', 'prs_01k0a5a92wf7qrrgq893dq79pp', NOW(), NOW()),
    ('cp_01seedcons_dlk_fsof', 'it_01seedchm1item00000', 'qu_01seedcons_dlk_fsqt', 'qu_01seedcons_dlk_fswt', 'prs_01k0a5a92wf7qrrgq893dq79pp', NOW(), NOW());

-- Dye Small Sock Beige consumes small knitted sock (SKN) + beige dye + fabric softener
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedcons_dsb_skqt', 50, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_dsb_skwt', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_dsb_dyqt', 1.5, 'gram', NOW(), NOW()),
    ('qu_01seedcons_dsb_dywt', 0.15, 'gram', NOW(), NOW()),
    ('qu_01seedcons_dsb_fsqt', 200, 'gram', NOW(), NOW()),
    ('qu_01seedcons_dsb_fswt', 7, 'gram', NOW(), NOW());
INSERT IGNORE INTO consumption (id, item_id, quantity_id, waste_quantity_id, production_step_id, created_at, updated_at) VALUES
    ('cp_01seedcons_dsb_knit', 'it_01seedsknitem000000', 'qu_01seedcons_dsb_skqt', 'qu_01seedcons_dsb_skwt', 'prs_01k0a5kr3jf9w83bqnt3y70vjy', NOW(), NOW()),
    ('cp_01seedcons_dsb_dye0', 'it_01seeddye1item00000', 'qu_01seedcons_dsb_dyqt', 'qu_01seedcons_dsb_dywt', 'prs_01k0a5kr3jf9w83bqnt3y70vjy', NOW(), NOW()),
    ('cp_01seedcons_dsb_fsof', 'it_01seedchm1item00000', 'qu_01seedcons_dsb_fsqt', 'qu_01seedcons_dsb_fswt', 'prs_01k0a5kr3jf9w83bqnt3y70vjy', NOW(), NOW());

-- Dye Small Sock Black consumes small knitted sock (SKN) + black dye + fabric softener
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedcons_dsk_skqt', 50, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_dsk_skwt', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_dsk_dyqt', 2, 'gram', NOW(), NOW()),
    ('qu_01seedcons_dsk_dywt', 0.2, 'gram', NOW(), NOW()),
    ('qu_01seedcons_dsk_fsqt', 200, 'gram', NOW(), NOW()),
    ('qu_01seedcons_dsk_fswt', 7, 'gram', NOW(), NOW());
INSERT IGNORE INTO consumption (id, item_id, quantity_id, waste_quantity_id, production_step_id, created_at, updated_at) VALUES
    ('cp_01seedcons_dsk_knit', 'it_01seedsknitem000000', 'qu_01seedcons_dsk_skqt', 'qu_01seedcons_dsk_skwt', 'prs_01k0a5m0yjfk19kf3n52bkbve6', NOW(), NOW()),
    ('cp_01seedcons_dsk_dye0', 'it_01seeddye2item00000', 'qu_01seedcons_dsk_dyqt', 'qu_01seedcons_dsk_dywt', 'prs_01k0a5m0yjfk19kf3n52bkbve6', NOW(), NOW()),
    ('cp_01seedcons_dsk_fsof', 'it_01seedchm1item00000', 'qu_01seedcons_dsk_fsqt', 'qu_01seedcons_dsk_fswt', 'prs_01k0a5m0yjfk19kf3n52bkbve6', NOW(), NOW());

-- Board Large White Sock consumes large washed sock (LWS)
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedcons_blw_qty0', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_blw_wst0', 0, 'un_01seedpair000000000', NOW(), NOW());
INSERT IGNORE INTO consumption (id, item_id, quantity_id, waste_quantity_id, production_step_id, created_at, updated_at) VALUES
    ('cp_01seedcons_blw_wash', 'it_01seedlwsitem000000', 'qu_01seedcons_blw_qty0', 'qu_01seedcons_blw_wst0', 'prs_01k0a5k18seysr468ykrd8fpnj', NOW(), NOW());

-- Board Small White Sock consumes small washed sock (SWS)
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedcons_bsw_qty0', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_bsw_wst0', 0, 'un_01seedpair000000000', NOW(), NOW());
INSERT IGNORE INTO consumption (id, item_id, quantity_id, waste_quantity_id, production_step_id, created_at, updated_at) VALUES
    ('cp_01seedcons_bsw_wash', 'it_01seedswsitem000000', 'qu_01seedcons_bsw_qty0', 'qu_01seedcons_bsw_wst0', 'prs_01k0a5kfpnf0gs570fjamctsca', NOW(), NOW());

-- Board Large Beige Sock consumes large beige sock (LBG)
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedcons_blb_qty0', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_blb_wst0', 0, 'un_01seedpair000000000', NOW(), NOW());
INSERT IGNORE INTO consumption (id, item_id, quantity_id, waste_quantity_id, production_step_id, created_at, updated_at) VALUES
    ('cp_01seedcons_blb_dyed', 'it_01seedlbgitem000000', 'qu_01seedcons_blb_qty0', 'qu_01seedcons_blb_wst0', 'prs_01k0a587pdene9ysk0xktc7zc7', NOW(), NOW());

-- Board Large Black Sock consumes large black sock (LBK)
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedcons_blk_qty0', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_blk_wst0', 0, 'un_01seedpair000000000', NOW(), NOW());
INSERT IGNORE INTO consumption (id, item_id, quantity_id, waste_quantity_id, production_step_id, created_at, updated_at) VALUES
    ('cp_01seedcons_blk_dyed', 'it_01seedlbkitem000000', 'qu_01seedcons_blk_qty0', 'qu_01seedcons_blk_wst0', 'prs_01k0a5m985fhzbasqkt6sx22a0', NOW(), NOW());

-- Board Small Beige Sock consumes small beige sock (SBG)
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedcons_bsb_qty0', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_bsb_wst0', 0, 'un_01seedpair000000000', NOW(), NOW());
INSERT IGNORE INTO consumption (id, item_id, quantity_id, waste_quantity_id, production_step_id, created_at, updated_at) VALUES
    ('cp_01seedcons_bsb_dyed', 'it_01seedsbgitem000000', 'qu_01seedcons_bsb_qty0', 'qu_01seedcons_bsb_wst0', 'prs_01k0a5mgq1fq5a9cvgev5zsf57', NOW(), NOW());

-- Board Small Black Sock consumes small black sock (SBK)
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedcons_bsk_qty0', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_bsk_wst0', 0, 'un_01seedpair000000000', NOW(), NOW());
INSERT IGNORE INTO consumption (id, item_id, quantity_id, waste_quantity_id, production_step_id, created_at, updated_at) VALUES
    ('cp_01seedcons_bsk_dyed', 'it_01seedsbkitem000000', 'qu_01seedcons_bsk_qty0', 'qu_01seedcons_bsk_wst0', 'prs_01k0a5ncadf1tbcb91kae06tvq', NOW(), NOW());

-- Pack Large White Sock consumes large white boarded sock (LWB) + box + packing paper
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedcons_plw_bqty', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_plw_bwst', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_plw_xqty', 1, 'each', NOW(), NOW()),
    ('qu_01seedcons_plw_xwst', 0, 'each', NOW(), NOW()),
    ('qu_01seedcons_plw_pqty', 1, 'each', NOW(), NOW()),
    ('qu_01seedcons_plw_pwst', 0, 'each', NOW(), NOW());
INSERT IGNORE INTO consumption (id, item_id, quantity_id, waste_quantity_id, production_step_id, created_at, updated_at) VALUES
    ('cp_01seedcons_plw_brd0', 'it_01seedlwbitem000000', 'qu_01seedcons_plw_bqty', 'qu_01seedcons_plw_bwst', 'prs_01k0a5nzd2f3a9cffpw38qken6', NOW(), NOW()),
    ('cp_01seedcons_plw_box0', 'it_01seedbox1item00000', 'qu_01seedcons_plw_xqty', 'qu_01seedcons_plw_xwst', 'prs_01k0a5nzd2f3a9cffpw38qken6', NOW(), NOW()),
    ('cp_01seedcons_plw_papr', 'it_01seedpp01item00000', 'qu_01seedcons_plw_pqty', 'qu_01seedcons_plw_pwst', 'prs_01k0a5nzd2f3a9cffpw38qken6', NOW(), NOW());

-- Pack Large Beige Sock consumes large beige boarded sock (LBGB) + box + packing paper
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedcons_plb_bqty', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_plb_bwst', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_plb_xqty', 1, 'each', NOW(), NOW()),
    ('qu_01seedcons_plb_xwst', 0, 'each', NOW(), NOW()),
    ('qu_01seedcons_plb_pqty', 1, 'each', NOW(), NOW()),
    ('qu_01seedcons_plb_pwst', 0, 'each', NOW(), NOW());
INSERT IGNORE INTO consumption (id, item_id, quantity_id, waste_quantity_id, production_step_id, created_at, updated_at) VALUES
    ('cp_01seedcons_plb_brd0', 'it_01seedlbgbitem00000', 'qu_01seedcons_plb_bqty', 'qu_01seedcons_plb_bwst', 'prs_01k0a5nzd2fxnv34tm431kr7vv', NOW(), NOW()),
    ('cp_01seedcons_plb_box0', 'it_01seedbox1item00000', 'qu_01seedcons_plb_xqty', 'qu_01seedcons_plb_xwst', 'prs_01k0a5nzd2fxnv34tm431kr7vv', NOW(), NOW()),
    ('cp_01seedcons_plb_papr', 'it_01seedpp01item00000', 'qu_01seedcons_plb_pqty', 'qu_01seedcons_plb_pwst', 'prs_01k0a5nzd2fxnv34tm431kr7vv', NOW(), NOW());

-- Pack Large Black Sock consumes large black boarded sock (LBKB) + box + packing paper
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedcons_plk_bqty', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_plk_bwst', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_plk_xqty', 1, 'each', NOW(), NOW()),
    ('qu_01seedcons_plk_xwst', 0, 'each', NOW(), NOW()),
    ('qu_01seedcons_plk_pqty', 1, 'each', NOW(), NOW()),
    ('qu_01seedcons_plk_pwst', 0, 'each', NOW(), NOW());
INSERT IGNORE INTO consumption (id, item_id, quantity_id, waste_quantity_id, production_step_id, created_at, updated_at) VALUES
    ('cp_01seedcons_plk_brd0', 'it_01seedlbkbitem00000', 'qu_01seedcons_plk_bqty', 'qu_01seedcons_plk_bwst', 'prs_01k0a5nzd2e55rw1bwmt8sdwye', NOW(), NOW()),
    ('cp_01seedcons_plk_box0', 'it_01seedbox1item00000', 'qu_01seedcons_plk_xqty', 'qu_01seedcons_plk_xwst', 'prs_01k0a5nzd2e55rw1bwmt8sdwye', NOW(), NOW()),
    ('cp_01seedcons_plk_papr', 'it_01seedpp01item00000', 'qu_01seedcons_plk_pqty', 'qu_01seedcons_plk_pwst', 'prs_01k0a5nzd2e55rw1bwmt8sdwye', NOW(), NOW());

-- Pack Small White Sock consumes small white boarded sock (SWB) + box + packing paper
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedcons_psw_bqty', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_psw_bwst', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_psw_xqty', 1, 'each', NOW(), NOW()),
    ('qu_01seedcons_psw_xwst', 0, 'each', NOW(), NOW()),
    ('qu_01seedcons_psw_pqty', 1, 'each', NOW(), NOW()),
    ('qu_01seedcons_psw_pwst', 0, 'each', NOW(), NOW());
INSERT IGNORE INTO consumption (id, item_id, quantity_id, waste_quantity_id, production_step_id, created_at, updated_at) VALUES
    ('cp_01seedcons_psw_brd0', 'it_01seedswbitem000000', 'qu_01seedcons_psw_bqty', 'qu_01seedcons_psw_bwst', 'prs_01k0a5nzd2e5fs4d3yvf8ehk41', NOW(), NOW()),
    ('cp_01seedcons_psw_box0', 'it_01seedbox1item00000', 'qu_01seedcons_psw_xqty', 'qu_01seedcons_psw_xwst', 'prs_01k0a5nzd2e5fs4d3yvf8ehk41', NOW(), NOW()),
    ('cp_01seedcons_psw_papr', 'it_01seedpp01item00000', 'qu_01seedcons_psw_pqty', 'qu_01seedcons_psw_pwst', 'prs_01k0a5nzd2e5fs4d3yvf8ehk41', NOW(), NOW());

-- Pack Small Beige Sock consumes small beige boarded sock (SBGB) + box + packing paper
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedcons_psb_bqty', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_psb_bwst', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_psb_xqty', 1, 'each', NOW(), NOW()),
    ('qu_01seedcons_psb_xwst', 0, 'each', NOW(), NOW()),
    ('qu_01seedcons_psb_pqty', 1, 'each', NOW(), NOW()),
    ('qu_01seedcons_psb_pwst', 0, 'each', NOW(), NOW());
INSERT IGNORE INTO consumption (id, item_id, quantity_id, waste_quantity_id, production_step_id, created_at, updated_at) VALUES
    ('cp_01seedcons_psb_brd0', 'it_01seedsbgbitem00000', 'qu_01seedcons_psb_bqty', 'qu_01seedcons_psb_bwst', 'prs_01k0a5nzd2ek69mct9w40w3h6c', NOW(), NOW()),
    ('cp_01seedcons_psb_box0', 'it_01seedbox1item00000', 'qu_01seedcons_psb_xqty', 'qu_01seedcons_psb_xwst', 'prs_01k0a5nzd2ek69mct9w40w3h6c', NOW(), NOW()),
    ('cp_01seedcons_psb_papr', 'it_01seedpp01item00000', 'qu_01seedcons_psb_pqty', 'qu_01seedcons_psb_pwst', 'prs_01k0a5nzd2ek69mct9w40w3h6c', NOW(), NOW());

-- Pack Small Black Sock consumes small black boarded sock (SBKB) + box + packing paper
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedcons_psk_bqty', 1, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_psk_bwst', 0, 'un_01seedpair000000000', NOW(), NOW()),
    ('qu_01seedcons_psk_xqty', 1, 'each', NOW(), NOW()),
    ('qu_01seedcons_psk_xwst', 0, 'each', NOW(), NOW()),
    ('qu_01seedcons_psk_pqty', 1, 'each', NOW(), NOW()),
    ('qu_01seedcons_psk_pwst', 0, 'each', NOW(), NOW());
INSERT IGNORE INTO consumption (id, item_id, quantity_id, waste_quantity_id, production_step_id, created_at, updated_at) VALUES
    ('cp_01seedcons_psk_brd0', 'it_01seedsbkbitem00000', 'qu_01seedcons_psk_bqty', 'qu_01seedcons_psk_bwst', 'prs_01k0a5nzd2fdw8hvff2sh4bvb3', NOW(), NOW()),
    ('cp_01seedcons_psk_box0', 'it_01seedbox1item00000', 'qu_01seedcons_psk_xqty', 'qu_01seedcons_psk_xwst', 'prs_01k0a5nzd2fdw8hvff2sh4bvb3', NOW(), NOW()),
    ('cp_01seedcons_psk_papr', 'it_01seedpp01item00000', 'qu_01seedcons_psk_pqty', 'qu_01seedcons_psk_pwst', 'prs_01k0a5nzd2fdw8hvff2sh4bvb3', NOW(), NOW());

-- ============================================================
-- PRODUCTION STEP GRAPH (parent → child edges)
-- _parent_child_production_steps(A, B): A is parent, B is child
-- A appears in B's "in" list; B appears in A's "out" list
-- ============================================================

INSERT IGNORE INTO _parent_child_production_steps (A, B) VALUES
    -- Knit Large Sock → Sew Large Sock
    ('prs_01k0a51qxceydax5036pegvzzy', 'prs_01k0a56yc1e8wag6wexn4pp8t9'),
    -- Sew Large Sock → Wash Large Sock
    ('prs_01k0a56yc1e8wag6wexn4pp8t9', 'prs_01k0a57qbefecte8erp0mp6vqb'),
    -- Sew Large Sock → Dye Large Sock Beige
    ('prs_01k0a56yc1e8wag6wexn4pp8t9', 'prs_01k0a5a2ezen9rbvh3aa97m64f'),
    -- Sew Large Sock → Dye Large Sock Black
    ('prs_01k0a56yc1e8wag6wexn4pp8t9', 'prs_01k0a5a92wf7qrrgq893dq79pp'),
    -- Knit Small Sock → Wash Small Sock
    ('prs_01k0a575j3fqr97khk36v114nj', 'prs_01k0a57f3dfsmtzc8txbq43eth'),
    -- Knit Small Sock → Dye Small Sock Beige
    ('prs_01k0a575j3fqr97khk36v114nj', 'prs_01k0a5kr3jf9w83bqnt3y70vjy'),
    -- Knit Small Sock → Dye Small Sock Black
    ('prs_01k0a575j3fqr97khk36v114nj', 'prs_01k0a5m0yjfk19kf3n52bkbve6'),
    -- Wash Large Sock → Board Large White Sock
    ('prs_01k0a57qbefecte8erp0mp6vqb', 'prs_01k0a5k18seysr468ykrd8fpnj'),
    -- Wash Small Sock → Board Small White Sock
    ('prs_01k0a57f3dfsmtzc8txbq43eth', 'prs_01k0a5kfpnf0gs570fjamctsca'),
    -- Dye Large Sock Beige → Board Large Beige Sock
    ('prs_01k0a5a2ezen9rbvh3aa97m64f', 'prs_01k0a587pdene9ysk0xktc7zc7'),
    -- Dye Large Sock Black → Board Large Black Sock
    ('prs_01k0a5a92wf7qrrgq893dq79pp', 'prs_01k0a5m985fhzbasqkt6sx22a0'),
    -- Dye Small Sock Beige → Board Small Beige Sock
    ('prs_01k0a5kr3jf9w83bqnt3y70vjy', 'prs_01k0a5mgq1fq5a9cvgev5zsf57'),
    -- Dye Small Sock Black → Board Small Black Sock
    ('prs_01k0a5m0yjfk19kf3n52bkbve6', 'prs_01k0a5ncadf1tbcb91kae06tvq'),
    -- Board Large White Sock → Pack Large White Sock
    ('prs_01k0a5k18seysr468ykrd8fpnj', 'prs_01k0a5nzd2f3a9cffpw38qken6'),
    -- Board Large Beige Sock → Pack Large Beige Sock
    ('prs_01k0a587pdene9ysk0xktc7zc7', 'prs_01k0a5nzd2fxnv34tm431kr7vv'),
    -- Board Large Black Sock → Pack Large Black Sock
    ('prs_01k0a5m985fhzbasqkt6sx22a0', 'prs_01k0a5nzd2e55rw1bwmt8sdwye'),
    -- Board Small White Sock → Pack Small White Sock
    ('prs_01k0a5kfpnf0gs570fjamctsca', 'prs_01k0a5nzd2e5fs4d3yvf8ehk41'),
    -- Board Small Beige Sock → Pack Small Beige Sock
    ('prs_01k0a5mgq1fq5a9cvgev5zsf57', 'prs_01k0a5nzd2ek69mct9w40w3h6c'),
    -- Board Small Black Sock → Pack Small Black Sock
    ('prs_01k0a5ncadf1tbcb91kae06tvq', 'prs_01k0a5nzd2fdw8hvff2sh4bvb3');
