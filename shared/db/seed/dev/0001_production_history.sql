-- Production history for the scheduling solver.
--
-- The solver measures everything from what the constraint department has actually
-- made: run rates, machine affinity, lead times, and the finished goods each greige
-- becomes. Without batch history it correctly plans nothing, which made the feature
-- impossible to evaluate on seed data.
--
-- Twelve months of knit batches per greige, each chained through the batch genealogy
-- to the packed product it becomes, so the demand pooling and the two inventory
-- stages both have something real to work with.

INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qty_sdlknk0000000000', 480, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdlknp0000000000', 480, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdlknk0100000000', 480, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdlknp0100000000', 480, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdlknk0200000000', 480, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdlknp0200000000', 480, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdlknk0300000000', 480, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdlknp0300000000', 480, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdlknk0400000000', 480, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdlknp0400000000', 480, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdlknk0500000000', 480, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdlknp0500000000', 480, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdlknk0600000000', 480, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdlknp0600000000', 480, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdlknk0700000000', 480, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdlknp0700000000', 480, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdlknk0800000000', 480, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdlknp0800000000', 480, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdlknk0900000000', 480, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdlknp0900000000', 480, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdlknk1000000000', 480, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdlknp1000000000', 480, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdlknk1100000000', 480, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdlknp1100000000', 480, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdsknk0000000000', 360, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdsknp0000000000', 360, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdsknk0100000000', 360, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdsknp0100000000', 360, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdsknk0200000000', 360, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdsknp0200000000', 360, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdsknk0300000000', 360, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdsknp0300000000', 360, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdsknk0400000000', 360, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdsknp0400000000', 360, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdsknk0500000000', 360, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdsknp0500000000', 360, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdsknk0600000000', 360, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdsknp0600000000', 360, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdsknk0700000000', 360, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdsknp0700000000', 360, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdsknk0800000000', 360, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdsknp0800000000', 360, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdsknk0900000000', 360, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdsknp0900000000', 360, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdsknk1000000000', 360, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdsknp1000000000', 360, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdsknk1100000000', 360, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdsknp1100000000', 360, 'un_01seedpair000000000', NOW(3), NOW(3));

INSERT IGNORE INTO batch (id, account_id, item_id, quantity_id, production_step_id, scanned_at, created_at, updated_at) VALUES
    ('bt_seedlknknit0000000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedlknitem000000', 'qty_sdlknk0000000000', 'prs_01k0a51qxceydax5036pegvzzy', DATE_SUB(NOW(3), INTERVAL 12 MONTH), DATE_SUB(NOW(3), INTERVAL 12 MONTH), NOW(3)),
    ('bt_seedlknpack0000000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01k0a7100aeysrs9vxpeq14yxj', 'qty_sdlknp0000000000', 'prs_01k0a5nzd2f3a9cffpw38qken6', DATE_SUB(NOW(3), INTERVAL 12 MONTH), DATE_SUB(NOW(3), INTERVAL 12 MONTH), NOW(3)),
    ('bt_seedlknknit0100000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedlknitem000000', 'qty_sdlknk0100000000', 'prs_01k0a51qxceydax5036pegvzzy', DATE_SUB(NOW(3), INTERVAL 11 MONTH), DATE_SUB(NOW(3), INTERVAL 11 MONTH), NOW(3)),
    ('bt_seedlknpack0100000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01k0a7100aeysrs9vxpeq14yxj', 'qty_sdlknp0100000000', 'prs_01k0a5nzd2f3a9cffpw38qken6', DATE_SUB(NOW(3), INTERVAL 11 MONTH), DATE_SUB(NOW(3), INTERVAL 11 MONTH), NOW(3)),
    ('bt_seedlknknit0200000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedlknitem000000', 'qty_sdlknk0200000000', 'prs_01k0a51qxceydax5036pegvzzy', DATE_SUB(NOW(3), INTERVAL 10 MONTH), DATE_SUB(NOW(3), INTERVAL 10 MONTH), NOW(3)),
    ('bt_seedlknpack0200000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01k0a7100aeysrs9vxpeq14yxj', 'qty_sdlknp0200000000', 'prs_01k0a5nzd2f3a9cffpw38qken6', DATE_SUB(NOW(3), INTERVAL 10 MONTH), DATE_SUB(NOW(3), INTERVAL 10 MONTH), NOW(3)),
    ('bt_seedlknknit0300000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedlknitem000000', 'qty_sdlknk0300000000', 'prs_01k0a51qxceydax5036pegvzzy', DATE_SUB(NOW(3), INTERVAL 9 MONTH), DATE_SUB(NOW(3), INTERVAL 9 MONTH), NOW(3)),
    ('bt_seedlknpack0300000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01k0a7100aeysrs9vxpeq14yxj', 'qty_sdlknp0300000000', 'prs_01k0a5nzd2f3a9cffpw38qken6', DATE_SUB(NOW(3), INTERVAL 9 MONTH), DATE_SUB(NOW(3), INTERVAL 9 MONTH), NOW(3)),
    ('bt_seedlknknit0400000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedlknitem000000', 'qty_sdlknk0400000000', 'prs_01k0a51qxceydax5036pegvzzy', DATE_SUB(NOW(3), INTERVAL 8 MONTH), DATE_SUB(NOW(3), INTERVAL 8 MONTH), NOW(3)),
    ('bt_seedlknpack0400000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01k0a7100aeysrs9vxpeq14yxj', 'qty_sdlknp0400000000', 'prs_01k0a5nzd2f3a9cffpw38qken6', DATE_SUB(NOW(3), INTERVAL 8 MONTH), DATE_SUB(NOW(3), INTERVAL 8 MONTH), NOW(3)),
    ('bt_seedlknknit0500000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedlknitem000000', 'qty_sdlknk0500000000', 'prs_01k0a51qxceydax5036pegvzzy', DATE_SUB(NOW(3), INTERVAL 7 MONTH), DATE_SUB(NOW(3), INTERVAL 7 MONTH), NOW(3)),
    ('bt_seedlknpack0500000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01k0a7100aeysrs9vxpeq14yxj', 'qty_sdlknp0500000000', 'prs_01k0a5nzd2f3a9cffpw38qken6', DATE_SUB(NOW(3), INTERVAL 7 MONTH), DATE_SUB(NOW(3), INTERVAL 7 MONTH), NOW(3)),
    ('bt_seedlknknit0600000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedlknitem000000', 'qty_sdlknk0600000000', 'prs_01k0a51qxceydax5036pegvzzy', DATE_SUB(NOW(3), INTERVAL 6 MONTH), DATE_SUB(NOW(3), INTERVAL 6 MONTH), NOW(3)),
    ('bt_seedlknpack0600000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01k0a7100aeysrs9vxpeq14yxj', 'qty_sdlknp0600000000', 'prs_01k0a5nzd2f3a9cffpw38qken6', DATE_SUB(NOW(3), INTERVAL 6 MONTH), DATE_SUB(NOW(3), INTERVAL 6 MONTH), NOW(3)),
    ('bt_seedlknknit0700000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedlknitem000000', 'qty_sdlknk0700000000', 'prs_01k0a51qxceydax5036pegvzzy', DATE_SUB(NOW(3), INTERVAL 5 MONTH), DATE_SUB(NOW(3), INTERVAL 5 MONTH), NOW(3)),
    ('bt_seedlknpack0700000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01k0a7100aeysrs9vxpeq14yxj', 'qty_sdlknp0700000000', 'prs_01k0a5nzd2f3a9cffpw38qken6', DATE_SUB(NOW(3), INTERVAL 5 MONTH), DATE_SUB(NOW(3), INTERVAL 5 MONTH), NOW(3)),
    ('bt_seedlknknit0800000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedlknitem000000', 'qty_sdlknk0800000000', 'prs_01k0a51qxceydax5036pegvzzy', DATE_SUB(NOW(3), INTERVAL 4 MONTH), DATE_SUB(NOW(3), INTERVAL 4 MONTH), NOW(3)),
    ('bt_seedlknpack0800000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01k0a7100aeysrs9vxpeq14yxj', 'qty_sdlknp0800000000', 'prs_01k0a5nzd2f3a9cffpw38qken6', DATE_SUB(NOW(3), INTERVAL 4 MONTH), DATE_SUB(NOW(3), INTERVAL 4 MONTH), NOW(3)),
    ('bt_seedlknknit0900000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedlknitem000000', 'qty_sdlknk0900000000', 'prs_01k0a51qxceydax5036pegvzzy', DATE_SUB(NOW(3), INTERVAL 3 MONTH), DATE_SUB(NOW(3), INTERVAL 3 MONTH), NOW(3)),
    ('bt_seedlknpack0900000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01k0a7100aeysrs9vxpeq14yxj', 'qty_sdlknp0900000000', 'prs_01k0a5nzd2f3a9cffpw38qken6', DATE_SUB(NOW(3), INTERVAL 3 MONTH), DATE_SUB(NOW(3), INTERVAL 3 MONTH), NOW(3)),
    ('bt_seedlknknit1000000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedlknitem000000', 'qty_sdlknk1000000000', 'prs_01k0a51qxceydax5036pegvzzy', DATE_SUB(NOW(3), INTERVAL 2 MONTH), DATE_SUB(NOW(3), INTERVAL 2 MONTH), NOW(3)),
    ('bt_seedlknpack1000000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01k0a7100aeysrs9vxpeq14yxj', 'qty_sdlknp1000000000', 'prs_01k0a5nzd2f3a9cffpw38qken6', DATE_SUB(NOW(3), INTERVAL 2 MONTH), DATE_SUB(NOW(3), INTERVAL 2 MONTH), NOW(3)),
    ('bt_seedlknknit1100000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedlknitem000000', 'qty_sdlknk1100000000', 'prs_01k0a51qxceydax5036pegvzzy', DATE_SUB(NOW(3), INTERVAL 1 MONTH), DATE_SUB(NOW(3), INTERVAL 1 MONTH), NOW(3)),
    ('bt_seedlknpack1100000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01k0a7100aeysrs9vxpeq14yxj', 'qty_sdlknp1100000000', 'prs_01k0a5nzd2f3a9cffpw38qken6', DATE_SUB(NOW(3), INTERVAL 1 MONTH), DATE_SUB(NOW(3), INTERVAL 1 MONTH), NOW(3)),
    ('bt_seedsknknit0000000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedsknitem000000', 'qty_sdsknk0000000000', 'prs_01k0a575j3fqr97khk36v114nj', DATE_SUB(NOW(3), INTERVAL 12 MONTH), DATE_SUB(NOW(3), INTERVAL 12 MONTH), NOW(3)),
    ('bt_seedsknpack0000000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01k0a7100afdnr1b41917qs27k', 'qty_sdsknp0000000000', 'prs_01k0a5nzd2fdw8hvff2sh4bvb3', DATE_SUB(NOW(3), INTERVAL 12 MONTH), DATE_SUB(NOW(3), INTERVAL 12 MONTH), NOW(3)),
    ('bt_seedsknknit0100000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedsknitem000000', 'qty_sdsknk0100000000', 'prs_01k0a575j3fqr97khk36v114nj', DATE_SUB(NOW(3), INTERVAL 11 MONTH), DATE_SUB(NOW(3), INTERVAL 11 MONTH), NOW(3)),
    ('bt_seedsknpack0100000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01k0a7100afdnr1b41917qs27k', 'qty_sdsknp0100000000', 'prs_01k0a5nzd2fdw8hvff2sh4bvb3', DATE_SUB(NOW(3), INTERVAL 11 MONTH), DATE_SUB(NOW(3), INTERVAL 11 MONTH), NOW(3)),
    ('bt_seedsknknit0200000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedsknitem000000', 'qty_sdsknk0200000000', 'prs_01k0a575j3fqr97khk36v114nj', DATE_SUB(NOW(3), INTERVAL 10 MONTH), DATE_SUB(NOW(3), INTERVAL 10 MONTH), NOW(3)),
    ('bt_seedsknpack0200000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01k0a7100afdnr1b41917qs27k', 'qty_sdsknp0200000000', 'prs_01k0a5nzd2fdw8hvff2sh4bvb3', DATE_SUB(NOW(3), INTERVAL 10 MONTH), DATE_SUB(NOW(3), INTERVAL 10 MONTH), NOW(3)),
    ('bt_seedsknknit0300000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedsknitem000000', 'qty_sdsknk0300000000', 'prs_01k0a575j3fqr97khk36v114nj', DATE_SUB(NOW(3), INTERVAL 9 MONTH), DATE_SUB(NOW(3), INTERVAL 9 MONTH), NOW(3)),
    ('bt_seedsknpack0300000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01k0a7100afdnr1b41917qs27k', 'qty_sdsknp0300000000', 'prs_01k0a5nzd2fdw8hvff2sh4bvb3', DATE_SUB(NOW(3), INTERVAL 9 MONTH), DATE_SUB(NOW(3), INTERVAL 9 MONTH), NOW(3)),
    ('bt_seedsknknit0400000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedsknitem000000', 'qty_sdsknk0400000000', 'prs_01k0a575j3fqr97khk36v114nj', DATE_SUB(NOW(3), INTERVAL 8 MONTH), DATE_SUB(NOW(3), INTERVAL 8 MONTH), NOW(3)),
    ('bt_seedsknpack0400000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01k0a7100afdnr1b41917qs27k', 'qty_sdsknp0400000000', 'prs_01k0a5nzd2fdw8hvff2sh4bvb3', DATE_SUB(NOW(3), INTERVAL 8 MONTH), DATE_SUB(NOW(3), INTERVAL 8 MONTH), NOW(3)),
    ('bt_seedsknknit0500000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedsknitem000000', 'qty_sdsknk0500000000', 'prs_01k0a575j3fqr97khk36v114nj', DATE_SUB(NOW(3), INTERVAL 7 MONTH), DATE_SUB(NOW(3), INTERVAL 7 MONTH), NOW(3)),
    ('bt_seedsknpack0500000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01k0a7100afdnr1b41917qs27k', 'qty_sdsknp0500000000', 'prs_01k0a5nzd2fdw8hvff2sh4bvb3', DATE_SUB(NOW(3), INTERVAL 7 MONTH), DATE_SUB(NOW(3), INTERVAL 7 MONTH), NOW(3)),
    ('bt_seedsknknit0600000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedsknitem000000', 'qty_sdsknk0600000000', 'prs_01k0a575j3fqr97khk36v114nj', DATE_SUB(NOW(3), INTERVAL 6 MONTH), DATE_SUB(NOW(3), INTERVAL 6 MONTH), NOW(3)),
    ('bt_seedsknpack0600000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01k0a7100afdnr1b41917qs27k', 'qty_sdsknp0600000000', 'prs_01k0a5nzd2fdw8hvff2sh4bvb3', DATE_SUB(NOW(3), INTERVAL 6 MONTH), DATE_SUB(NOW(3), INTERVAL 6 MONTH), NOW(3)),
    ('bt_seedsknknit0700000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedsknitem000000', 'qty_sdsknk0700000000', 'prs_01k0a575j3fqr97khk36v114nj', DATE_SUB(NOW(3), INTERVAL 5 MONTH), DATE_SUB(NOW(3), INTERVAL 5 MONTH), NOW(3)),
    ('bt_seedsknpack0700000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01k0a7100afdnr1b41917qs27k', 'qty_sdsknp0700000000', 'prs_01k0a5nzd2fdw8hvff2sh4bvb3', DATE_SUB(NOW(3), INTERVAL 5 MONTH), DATE_SUB(NOW(3), INTERVAL 5 MONTH), NOW(3)),
    ('bt_seedsknknit0800000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedsknitem000000', 'qty_sdsknk0800000000', 'prs_01k0a575j3fqr97khk36v114nj', DATE_SUB(NOW(3), INTERVAL 4 MONTH), DATE_SUB(NOW(3), INTERVAL 4 MONTH), NOW(3)),
    ('bt_seedsknpack0800000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01k0a7100afdnr1b41917qs27k', 'qty_sdsknp0800000000', 'prs_01k0a5nzd2fdw8hvff2sh4bvb3', DATE_SUB(NOW(3), INTERVAL 4 MONTH), DATE_SUB(NOW(3), INTERVAL 4 MONTH), NOW(3)),
    ('bt_seedsknknit0900000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedsknitem000000', 'qty_sdsknk0900000000', 'prs_01k0a575j3fqr97khk36v114nj', DATE_SUB(NOW(3), INTERVAL 3 MONTH), DATE_SUB(NOW(3), INTERVAL 3 MONTH), NOW(3)),
    ('bt_seedsknpack0900000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01k0a7100afdnr1b41917qs27k', 'qty_sdsknp0900000000', 'prs_01k0a5nzd2fdw8hvff2sh4bvb3', DATE_SUB(NOW(3), INTERVAL 3 MONTH), DATE_SUB(NOW(3), INTERVAL 3 MONTH), NOW(3)),
    ('bt_seedsknknit1000000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedsknitem000000', 'qty_sdsknk1000000000', 'prs_01k0a575j3fqr97khk36v114nj', DATE_SUB(NOW(3), INTERVAL 2 MONTH), DATE_SUB(NOW(3), INTERVAL 2 MONTH), NOW(3)),
    ('bt_seedsknpack1000000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01k0a7100afdnr1b41917qs27k', 'qty_sdsknp1000000000', 'prs_01k0a5nzd2fdw8hvff2sh4bvb3', DATE_SUB(NOW(3), INTERVAL 2 MONTH), DATE_SUB(NOW(3), INTERVAL 2 MONTH), NOW(3)),
    ('bt_seedsknknit1100000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedsknitem000000', 'qty_sdsknk1100000000', 'prs_01k0a575j3fqr97khk36v114nj', DATE_SUB(NOW(3), INTERVAL 1 MONTH), DATE_SUB(NOW(3), INTERVAL 1 MONTH), NOW(3)),
    ('bt_seedsknpack1100000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01k0a7100afdnr1b41917qs27k', 'qty_sdsknp1100000000', 'prs_01k0a5nzd2fdw8hvff2sh4bvb3', DATE_SUB(NOW(3), INTERVAL 1 MONTH), DATE_SUB(NOW(3), INTERVAL 1 MONTH), NOW(3));

-- A = batch, B = machine. The solver joins through this to measure run rates, so a
-- batch with no link is invisible to it.
INSERT IGNORE INTO _batches_machines (A, B) VALUES
    ('bt_seedlknknit0000000', 'mc_01k0a52fb6eqhtbx9hdxj3vvnh'),
    ('bt_seedlknknit0100000', 'mc_01k0a52fb6eqhtbx9hdxj3vvnh'),
    ('bt_seedlknknit0200000', 'mc_01k0a52fb6eqhtbx9hdxj3vvnh'),
    ('bt_seedlknknit0300000', 'mc_01k0a52fb6eqhtbx9hdxj3vvnh'),
    ('bt_seedlknknit0400000', 'mc_01k0a52fb6eqhtbx9hdxj3vvnh'),
    ('bt_seedlknknit0500000', 'mc_01k0a52fb6eqhtbx9hdxj3vvnh'),
    ('bt_seedlknknit0600000', 'mc_01k0a52fb6eqhtbx9hdxj3vvnh'),
    ('bt_seedlknknit0700000', 'mc_01k0a52fb6eqhtbx9hdxj3vvnh'),
    ('bt_seedlknknit0800000', 'mc_01k0a52fb6eqhtbx9hdxj3vvnh'),
    ('bt_seedlknknit0900000', 'mc_01k0a52fb6eqhtbx9hdxj3vvnh'),
    ('bt_seedlknknit1000000', 'mc_01k0a52fb6eqhtbx9hdxj3vvnh'),
    ('bt_seedlknknit1100000', 'mc_01k0a52fb6eqhtbx9hdxj3vvnh'),
    ('bt_seedsknknit0000000', 'mc_01k0a52r3vf9p9tn962fkszst5'),
    ('bt_seedsknknit0100000', 'mc_01k0a52r3vf9p9tn962fkszst5'),
    ('bt_seedsknknit0200000', 'mc_01k0a52r3vf9p9tn962fkszst5'),
    ('bt_seedsknknit0300000', 'mc_01k0a52r3vf9p9tn962fkszst5'),
    ('bt_seedsknknit0400000', 'mc_01k0a52r3vf9p9tn962fkszst5'),
    ('bt_seedsknknit0500000', 'mc_01k0a52r3vf9p9tn962fkszst5'),
    ('bt_seedsknknit0600000', 'mc_01k0a52r3vf9p9tn962fkszst5'),
    ('bt_seedsknknit0700000', 'mc_01k0a52r3vf9p9tn962fkszst5'),
    ('bt_seedsknknit0800000', 'mc_01k0a52r3vf9p9tn962fkszst5'),
    ('bt_seedsknknit0900000', 'mc_01k0a52r3vf9p9tn962fkszst5'),
    ('bt_seedsknknit1000000', 'mc_01k0a52r3vf9p9tn962fkszst5'),
    ('bt_seedsknknit1100000', 'mc_01k0a52r3vf9p9tn962fkszst5');

-- A = downstream (child), B = upstream (parent), per the Prisma orientation of _batch_flow. This is the genealogy the descendant walk follows to discover which finished goods a greige becomes, which is what pools demand back onto the greige.
INSERT IGNORE INTO _batch_flow (A, B) VALUES
    ('bt_seedlknpack0000000', 'bt_seedlknknit0000000'),
    ('bt_seedlknpack0100000', 'bt_seedlknknit0100000'),
    ('bt_seedlknpack0200000', 'bt_seedlknknit0200000'),
    ('bt_seedlknpack0300000', 'bt_seedlknknit0300000'),
    ('bt_seedlknpack0400000', 'bt_seedlknknit0400000'),
    ('bt_seedlknpack0500000', 'bt_seedlknknit0500000'),
    ('bt_seedlknpack0600000', 'bt_seedlknknit0600000'),
    ('bt_seedlknpack0700000', 'bt_seedlknknit0700000'),
    ('bt_seedlknpack0800000', 'bt_seedlknknit0800000'),
    ('bt_seedlknpack0900000', 'bt_seedlknknit0900000'),
    ('bt_seedlknpack1000000', 'bt_seedlknknit1000000'),
    ('bt_seedlknpack1100000', 'bt_seedlknknit1100000'),
    ('bt_seedsknpack0000000', 'bt_seedsknknit0000000'),
    ('bt_seedsknpack0100000', 'bt_seedsknknit0100000'),
    ('bt_seedsknpack0200000', 'bt_seedsknknit0200000'),
    ('bt_seedsknpack0300000', 'bt_seedsknknit0300000'),
    ('bt_seedsknpack0400000', 'bt_seedsknknit0400000'),
    ('bt_seedsknpack0500000', 'bt_seedsknknit0500000'),
    ('bt_seedsknpack0600000', 'bt_seedsknknit0600000'),
    ('bt_seedsknpack0700000', 'bt_seedsknknit0700000'),
    ('bt_seedsknpack0800000', 'bt_seedsknknit0800000'),
    ('bt_seedsknpack0900000', 'bt_seedsknknit0900000'),
    ('bt_seedsknpack1000000', 'bt_seedsknknit1000000'),
    ('bt_seedsknpack1100000', 'bt_seedsknknit1100000');

-- Stock at both stages. The build decision uses the echelon total; the greige figure
-- is what is actually in the greige store, and the finished rows are measured against
-- their own stock.
--
-- Deliberately below the reorder points these inputs produce, so a freshly generated
-- schedule plans campaigns in week one. Well-stocked is the more common real state, but
-- it leaves the seeded plan empty for the first nine weeks and there is nothing to look
-- at. Raise these to watch the (s,S) policy hold off instead.
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qty_sdlknoh0000000000', 300, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdsknoh0000000000', 200, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdsck1oh000000000', 500, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qty_sdsck3oh000000000', 350, 'un_01seedpair000000000', NOW(3), NOW(3));


-- unit_cost_id is unique per receipt, so each one carries its own cost rate.
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_sdohlkn0000000000', 4.25, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdohskn0000000000', 3.8, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdohsck1000000000', 9.5, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdohsck3000000000', 8.75, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3));

INSERT IGNORE INTO inventory_receipt (id, owner_account_id, holder_account_id, item_id, quantity_id, unit_cost_id, received_at, status_code, created_at, updated_at) VALUES
    ('ivrc_seedlknonhand000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedlknitem000000', 'qty_sdlknoh0000000000', 'rt_sdohlkn0000000000', NOW(3), 'available', NOW(3), NOW(3)),
    ('ivrc_seedsknonhand000', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01seedsknitem000000', 'qty_sdsknoh0000000000', 'rt_sdohskn0000000000', NOW(3), 'available', NOW(3), NOW(3)),
    ('ivrc_seedsck1onhand00', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01k0a7100aeysrs9vxpeq14yxj', 'qty_sdsck1oh000000000', 'rt_sdohsck1000000000', NOW(3), 'available', NOW(3), NOW(3)),
    ('ivrc_seedsck3onhand00', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'it_01k0a7100afdnr1b41917qs27k', 'qty_sdsck3oh000000000', 'rt_sdohsck3000000000', NOW(3), 'available', NOW(3), NOW(3));


-- Order history for the two finished goods the genealogy above resolves to.
--
-- Demand is pooled from finished goods back onto the greige they are made from, so
-- without orders the solver has nothing to plan against. Eighteen months of it, on a
-- repeating seasonal shape: the safety stocks and the pooled greige buffer are all
-- sized off demand variability, and perfectly flat demand would make every one of
-- them zero and the two-stage inventory picture look broken rather than empty.
INSERT IGNORE INTO sales_order (id, number, billing_address_id, shipping_address_id, is_acknowledgment_sent, priority_code, sales_order_status_code, sales_order_type_code, buyer_account_id, seller_account_id, owner_account_id, issued_at, created_at, updated_at) VALUES
    ('or_seeddemand0000000000', 'SEED-1000', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ad_01k09wnpvrea0awz7vem2j8j7g', 0, 'normal', 'issued', 'sales_order', 'ac_01k09wm2fgevdsc344gpbcj30f', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_SUB(NOW(3), INTERVAL 18 MONTH), DATE_SUB(NOW(3), INTERVAL 18 MONTH), NOW(3)),
    ('or_seeddemand0100000000', 'SEED-1001', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ad_01k09wnpvrea0awz7vem2j8j7g', 0, 'normal', 'issued', 'sales_order', 'ac_01k09wm2fgevdsc344gpbcj30f', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_SUB(NOW(3), INTERVAL 17 MONTH), DATE_SUB(NOW(3), INTERVAL 17 MONTH), NOW(3)),
    ('or_seeddemand0200000000', 'SEED-1002', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ad_01k09wnpvrea0awz7vem2j8j7g', 0, 'normal', 'issued', 'sales_order', 'ac_01k09wm2fgevdsc344gpbcj30f', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_SUB(NOW(3), INTERVAL 16 MONTH), DATE_SUB(NOW(3), INTERVAL 16 MONTH), NOW(3)),
    ('or_seeddemand0300000000', 'SEED-1003', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ad_01k09wnpvrea0awz7vem2j8j7g', 0, 'normal', 'issued', 'sales_order', 'ac_01k09wm2fgevdsc344gpbcj30f', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_SUB(NOW(3), INTERVAL 15 MONTH), DATE_SUB(NOW(3), INTERVAL 15 MONTH), NOW(3)),
    ('or_seeddemand0400000000', 'SEED-1004', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ad_01k09wnpvrea0awz7vem2j8j7g', 0, 'normal', 'issued', 'sales_order', 'ac_01k09wm2fgevdsc344gpbcj30f', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_SUB(NOW(3), INTERVAL 14 MONTH), DATE_SUB(NOW(3), INTERVAL 14 MONTH), NOW(3)),
    ('or_seeddemand0500000000', 'SEED-1005', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ad_01k09wnpvrea0awz7vem2j8j7g', 0, 'normal', 'issued', 'sales_order', 'ac_01k09wm2fgevdsc344gpbcj30f', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_SUB(NOW(3), INTERVAL 13 MONTH), DATE_SUB(NOW(3), INTERVAL 13 MONTH), NOW(3)),
    ('or_seeddemand0600000000', 'SEED-1006', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ad_01k09wnpvrea0awz7vem2j8j7g', 0, 'normal', 'issued', 'sales_order', 'ac_01k09wm2fgevdsc344gpbcj30f', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_SUB(NOW(3), INTERVAL 12 MONTH), DATE_SUB(NOW(3), INTERVAL 12 MONTH), NOW(3)),
    ('or_seeddemand0700000000', 'SEED-1007', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ad_01k09wnpvrea0awz7vem2j8j7g', 0, 'normal', 'issued', 'sales_order', 'ac_01k09wm2fgevdsc344gpbcj30f', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_SUB(NOW(3), INTERVAL 11 MONTH), DATE_SUB(NOW(3), INTERVAL 11 MONTH), NOW(3)),
    ('or_seeddemand0800000000', 'SEED-1008', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ad_01k09wnpvrea0awz7vem2j8j7g', 0, 'normal', 'issued', 'sales_order', 'ac_01k09wm2fgevdsc344gpbcj30f', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_SUB(NOW(3), INTERVAL 10 MONTH), DATE_SUB(NOW(3), INTERVAL 10 MONTH), NOW(3)),
    ('or_seeddemand0900000000', 'SEED-1009', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ad_01k09wnpvrea0awz7vem2j8j7g', 0, 'normal', 'issued', 'sales_order', 'ac_01k09wm2fgevdsc344gpbcj30f', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_SUB(NOW(3), INTERVAL 9 MONTH), DATE_SUB(NOW(3), INTERVAL 9 MONTH), NOW(3)),
    ('or_seeddemand1000000000', 'SEED-1010', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ad_01k09wnpvrea0awz7vem2j8j7g', 0, 'normal', 'issued', 'sales_order', 'ac_01k09wm2fgevdsc344gpbcj30f', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_SUB(NOW(3), INTERVAL 8 MONTH), DATE_SUB(NOW(3), INTERVAL 8 MONTH), NOW(3)),
    ('or_seeddemand1100000000', 'SEED-1011', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ad_01k09wnpvrea0awz7vem2j8j7g', 0, 'normal', 'issued', 'sales_order', 'ac_01k09wm2fgevdsc344gpbcj30f', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_SUB(NOW(3), INTERVAL 7 MONTH), DATE_SUB(NOW(3), INTERVAL 7 MONTH), NOW(3)),
    ('or_seeddemand1200000000', 'SEED-1012', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ad_01k09wnpvrea0awz7vem2j8j7g', 0, 'normal', 'issued', 'sales_order', 'ac_01k09wm2fgevdsc344gpbcj30f', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_SUB(NOW(3), INTERVAL 6 MONTH), DATE_SUB(NOW(3), INTERVAL 6 MONTH), NOW(3)),
    ('or_seeddemand1300000000', 'SEED-1013', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ad_01k09wnpvrea0awz7vem2j8j7g', 0, 'normal', 'issued', 'sales_order', 'ac_01k09wm2fgevdsc344gpbcj30f', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_SUB(NOW(3), INTERVAL 5 MONTH), DATE_SUB(NOW(3), INTERVAL 5 MONTH), NOW(3)),
    ('or_seeddemand1400000000', 'SEED-1014', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ad_01k09wnpvrea0awz7vem2j8j7g', 0, 'normal', 'issued', 'sales_order', 'ac_01k09wm2fgevdsc344gpbcj30f', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_SUB(NOW(3), INTERVAL 4 MONTH), DATE_SUB(NOW(3), INTERVAL 4 MONTH), NOW(3)),
    ('or_seeddemand1500000000', 'SEED-1015', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ad_01k09wnpvrea0awz7vem2j8j7g', 0, 'normal', 'issued', 'sales_order', 'ac_01k09wm2fgevdsc344gpbcj30f', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_SUB(NOW(3), INTERVAL 3 MONTH), DATE_SUB(NOW(3), INTERVAL 3 MONTH), NOW(3)),
    ('or_seeddemand1600000000', 'SEED-1016', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ad_01k09wnpvrea0awz7vem2j8j7g', 0, 'normal', 'issued', 'sales_order', 'ac_01k09wm2fgevdsc344gpbcj30f', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_SUB(NOW(3), INTERVAL 2 MONTH), DATE_SUB(NOW(3), INTERVAL 2 MONTH), NOW(3)),
    ('or_seeddemand1700000000', 'SEED-1017', 'ad_01k09wnac0e1ar211e0sy0ba4g', 'ad_01k09wnpvrea0awz7vem2j8j7g', 0, 'normal', 'issued', 'sales_order', 'ac_01k09wm2fgevdsc344gpbcj30f', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'ac_01k0a5smf9ekb8rqg12555zjqa', DATE_SUB(NOW(3), INTERVAL 1 MONTH), DATE_SUB(NOW(3), INTERVAL 1 MONTH), NOW(3));

INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_sdadem000000000', 426, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdbdem000000000', 312, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdadem010000000', 447, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdbdem010000000', 327, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdadem020000000', 494, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdbdem020000000', 361, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdadem030000000', 546, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdbdem030000000', 399, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdadem040000000', 582, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdbdem040000000', 426, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdadem050000000', 624, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdbdem050000000', 456, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdadem060000000', 614, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdbdem060000000', 448, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdadem070000000', 572, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdbdem070000000', 418, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdadem080000000', 530, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdbdem080000000', 388, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdadem090000000', 489, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdbdem090000000', 357, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdadem100000000', 458, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdbdem100000000', 334, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdadem110000000', 442, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdbdem110000000', 323, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdadem120000000', 426, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdbdem120000000', 312, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdadem130000000', 447, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdbdem130000000', 327, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdadem140000000', 494, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdbdem140000000', 361, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdadem150000000', 546, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdbdem150000000', 399, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdadem160000000', 582, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdbdem160000000', 426, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdadem170000000', 624, 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('qu_sdbdem170000000', 456, 'un_01seedpair000000000', NOW(3), NOW(3));


-- One price rate per line: unit_price_id carries a unique key, so lines cannot share
-- a rate row.
INSERT IGNORE INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES
    ('rt_sdadem000000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdadem010000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdadem020000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdadem030000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdadem040000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdadem050000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdadem060000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdadem070000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdadem080000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdadem090000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdadem100000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdadem110000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdadem120000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdadem130000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdadem140000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdadem150000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdadem160000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdadem170000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdbdem000000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdbdem010000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdbdem020000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdbdem030000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdbdem040000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdbdem050000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdbdem060000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdbdem070000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdbdem080000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdbdem090000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdbdem100000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdbdem110000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdbdem120000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdbdem130000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdbdem140000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdbdem150000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdbdem160000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3)),
    ('rt_sdbdem170000000', 12.500000000000000000000000000000, 'dollar', 'un_01seedpair000000000', NOW(3), NOW(3));

INSERT IGNORE INTO sales_order_line (id, product_sku, product_id, sales_order_id, quantity_id, unit_price_id, created_at, updated_at) VALUES
    ('orln_sdadem0000000', 'SCK-001', 'pd_01k0a65nx2e2crfxrvryyxnmdh', 'or_seeddemand0000000000', 'qu_sdadem000000000', 'rt_sdadem000000000', NOW(3), NOW(3)),
    ('orln_sdbdem0000000', 'SCK-003', 'pd_01k0a65nx5fjz8m1s3ytayfdby', 'or_seeddemand0000000000', 'qu_sdbdem000000000', 'rt_sdbdem000000000', NOW(3), NOW(3)),
    ('orln_sdadem0100000', 'SCK-001', 'pd_01k0a65nx2e2crfxrvryyxnmdh', 'or_seeddemand0100000000', 'qu_sdadem010000000', 'rt_sdadem010000000', NOW(3), NOW(3)),
    ('orln_sdbdem0100000', 'SCK-003', 'pd_01k0a65nx5fjz8m1s3ytayfdby', 'or_seeddemand0100000000', 'qu_sdbdem010000000', 'rt_sdbdem010000000', NOW(3), NOW(3)),
    ('orln_sdadem0200000', 'SCK-001', 'pd_01k0a65nx2e2crfxrvryyxnmdh', 'or_seeddemand0200000000', 'qu_sdadem020000000', 'rt_sdadem020000000', NOW(3), NOW(3)),
    ('orln_sdbdem0200000', 'SCK-003', 'pd_01k0a65nx5fjz8m1s3ytayfdby', 'or_seeddemand0200000000', 'qu_sdbdem020000000', 'rt_sdbdem020000000', NOW(3), NOW(3)),
    ('orln_sdadem0300000', 'SCK-001', 'pd_01k0a65nx2e2crfxrvryyxnmdh', 'or_seeddemand0300000000', 'qu_sdadem030000000', 'rt_sdadem030000000', NOW(3), NOW(3)),
    ('orln_sdbdem0300000', 'SCK-003', 'pd_01k0a65nx5fjz8m1s3ytayfdby', 'or_seeddemand0300000000', 'qu_sdbdem030000000', 'rt_sdbdem030000000', NOW(3), NOW(3)),
    ('orln_sdadem0400000', 'SCK-001', 'pd_01k0a65nx2e2crfxrvryyxnmdh', 'or_seeddemand0400000000', 'qu_sdadem040000000', 'rt_sdadem040000000', NOW(3), NOW(3)),
    ('orln_sdbdem0400000', 'SCK-003', 'pd_01k0a65nx5fjz8m1s3ytayfdby', 'or_seeddemand0400000000', 'qu_sdbdem040000000', 'rt_sdbdem040000000', NOW(3), NOW(3)),
    ('orln_sdadem0500000', 'SCK-001', 'pd_01k0a65nx2e2crfxrvryyxnmdh', 'or_seeddemand0500000000', 'qu_sdadem050000000', 'rt_sdadem050000000', NOW(3), NOW(3)),
    ('orln_sdbdem0500000', 'SCK-003', 'pd_01k0a65nx5fjz8m1s3ytayfdby', 'or_seeddemand0500000000', 'qu_sdbdem050000000', 'rt_sdbdem050000000', NOW(3), NOW(3)),
    ('orln_sdadem0600000', 'SCK-001', 'pd_01k0a65nx2e2crfxrvryyxnmdh', 'or_seeddemand0600000000', 'qu_sdadem060000000', 'rt_sdadem060000000', NOW(3), NOW(3)),
    ('orln_sdbdem0600000', 'SCK-003', 'pd_01k0a65nx5fjz8m1s3ytayfdby', 'or_seeddemand0600000000', 'qu_sdbdem060000000', 'rt_sdbdem060000000', NOW(3), NOW(3)),
    ('orln_sdadem0700000', 'SCK-001', 'pd_01k0a65nx2e2crfxrvryyxnmdh', 'or_seeddemand0700000000', 'qu_sdadem070000000', 'rt_sdadem070000000', NOW(3), NOW(3)),
    ('orln_sdbdem0700000', 'SCK-003', 'pd_01k0a65nx5fjz8m1s3ytayfdby', 'or_seeddemand0700000000', 'qu_sdbdem070000000', 'rt_sdbdem070000000', NOW(3), NOW(3)),
    ('orln_sdadem0800000', 'SCK-001', 'pd_01k0a65nx2e2crfxrvryyxnmdh', 'or_seeddemand0800000000', 'qu_sdadem080000000', 'rt_sdadem080000000', NOW(3), NOW(3)),
    ('orln_sdbdem0800000', 'SCK-003', 'pd_01k0a65nx5fjz8m1s3ytayfdby', 'or_seeddemand0800000000', 'qu_sdbdem080000000', 'rt_sdbdem080000000', NOW(3), NOW(3)),
    ('orln_sdadem0900000', 'SCK-001', 'pd_01k0a65nx2e2crfxrvryyxnmdh', 'or_seeddemand0900000000', 'qu_sdadem090000000', 'rt_sdadem090000000', NOW(3), NOW(3)),
    ('orln_sdbdem0900000', 'SCK-003', 'pd_01k0a65nx5fjz8m1s3ytayfdby', 'or_seeddemand0900000000', 'qu_sdbdem090000000', 'rt_sdbdem090000000', NOW(3), NOW(3)),
    ('orln_sdadem1000000', 'SCK-001', 'pd_01k0a65nx2e2crfxrvryyxnmdh', 'or_seeddemand1000000000', 'qu_sdadem100000000', 'rt_sdadem100000000', NOW(3), NOW(3)),
    ('orln_sdbdem1000000', 'SCK-003', 'pd_01k0a65nx5fjz8m1s3ytayfdby', 'or_seeddemand1000000000', 'qu_sdbdem100000000', 'rt_sdbdem100000000', NOW(3), NOW(3)),
    ('orln_sdadem1100000', 'SCK-001', 'pd_01k0a65nx2e2crfxrvryyxnmdh', 'or_seeddemand1100000000', 'qu_sdadem110000000', 'rt_sdadem110000000', NOW(3), NOW(3)),
    ('orln_sdbdem1100000', 'SCK-003', 'pd_01k0a65nx5fjz8m1s3ytayfdby', 'or_seeddemand1100000000', 'qu_sdbdem110000000', 'rt_sdbdem110000000', NOW(3), NOW(3)),
    ('orln_sdadem1200000', 'SCK-001', 'pd_01k0a65nx2e2crfxrvryyxnmdh', 'or_seeddemand1200000000', 'qu_sdadem120000000', 'rt_sdadem120000000', NOW(3), NOW(3)),
    ('orln_sdbdem1200000', 'SCK-003', 'pd_01k0a65nx5fjz8m1s3ytayfdby', 'or_seeddemand1200000000', 'qu_sdbdem120000000', 'rt_sdbdem120000000', NOW(3), NOW(3)),
    ('orln_sdadem1300000', 'SCK-001', 'pd_01k0a65nx2e2crfxrvryyxnmdh', 'or_seeddemand1300000000', 'qu_sdadem130000000', 'rt_sdadem130000000', NOW(3), NOW(3)),
    ('orln_sdbdem1300000', 'SCK-003', 'pd_01k0a65nx5fjz8m1s3ytayfdby', 'or_seeddemand1300000000', 'qu_sdbdem130000000', 'rt_sdbdem130000000', NOW(3), NOW(3)),
    ('orln_sdadem1400000', 'SCK-001', 'pd_01k0a65nx2e2crfxrvryyxnmdh', 'or_seeddemand1400000000', 'qu_sdadem140000000', 'rt_sdadem140000000', NOW(3), NOW(3)),
    ('orln_sdbdem1400000', 'SCK-003', 'pd_01k0a65nx5fjz8m1s3ytayfdby', 'or_seeddemand1400000000', 'qu_sdbdem140000000', 'rt_sdbdem140000000', NOW(3), NOW(3)),
    ('orln_sdadem1500000', 'SCK-001', 'pd_01k0a65nx2e2crfxrvryyxnmdh', 'or_seeddemand1500000000', 'qu_sdadem150000000', 'rt_sdadem150000000', NOW(3), NOW(3)),
    ('orln_sdbdem1500000', 'SCK-003', 'pd_01k0a65nx5fjz8m1s3ytayfdby', 'or_seeddemand1500000000', 'qu_sdbdem150000000', 'rt_sdbdem150000000', NOW(3), NOW(3)),
    ('orln_sdadem1600000', 'SCK-001', 'pd_01k0a65nx2e2crfxrvryyxnmdh', 'or_seeddemand1600000000', 'qu_sdadem160000000', 'rt_sdadem160000000', NOW(3), NOW(3)),
    ('orln_sdbdem1600000', 'SCK-003', 'pd_01k0a65nx5fjz8m1s3ytayfdby', 'or_seeddemand1600000000', 'qu_sdbdem160000000', 'rt_sdbdem160000000', NOW(3), NOW(3)),
    ('orln_sdadem1700000', 'SCK-001', 'pd_01k0a65nx2e2crfxrvryyxnmdh', 'or_seeddemand1700000000', 'qu_sdadem170000000', 'rt_sdadem170000000', NOW(3), NOW(3)),
    ('orln_sdbdem1700000', 'SCK-003', 'pd_01k0a65nx5fjz8m1s3ytayfdby', 'or_seeddemand1700000000', 'qu_sdbdem170000000', 'rt_sdbdem170000000', NOW(3), NOW(3));
