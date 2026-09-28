-- +goose Up

-- The scan consumer booked production as step × (scanned ÷ step) with the quotient rounded to 16
-- places, so two knit scans on 2026-08-27 received 7 and 8 pairs as 6.9999999999999996 and
-- 8.0000000000000004. They are the only receipts in prod it touched. Allocations and two reconciles
-- then carried the 4e-16 residue forward exactly; this sets every row in both chains to what the
-- scans meant, which leaves each receipt and issue fully and exactly allocated.
--
-- Every UPDATE matches the residue value it replaces, so a row that has since moved is left alone
-- and re-running is a no-op. Cost rows on the allocations are fractions of a nanocent and stay.

-- First item: the 7-pair scan, and the 7-pair issue it was drawn against.
-- receipt inrp_g8jk2nt3d5n7
UPDATE `quantity` SET value = 7 WHERE id = 'qu_hhb195bxki90' AND value = 6.9999999999999996;
-- change log inchlg_3vibytn7w8n0
UPDATE `quantity` SET value = 7 WHERE id = 'qu_8lyuywtuihru' AND value = 6.9999999999999996;
-- allocation inal_rkkyceegfqtk
UPDATE `quantity` SET value = 7 WHERE id = 'qu_xny2nt0zj6xy' AND value = 6.9999999999999996;
-- allocation inal_u5nuw4njof2x, the top-up of that issue
UPDATE `quantity` SET value = 0 WHERE id = 'qu_zy77hoc6558z' AND value = 0.0000000000000004;

-- First item: receipt inrp_c5srqv3vekos now gives its whole 60 to the 60-pair issue, which is
-- therefore covered and closes.
-- allocation inal_r9d52mg2f0kj
UPDATE `quantity` SET value = 60 WHERE id = 'qu_by9r0kdamdg2' AND value = 59.9999999999999996;
UPDATE `inventory_issue` SET status_code = 'closed', updated_at = CURRENT_TIMESTAMP(3)
WHERE id = 'inrs_ihogryt4uon1' AND status_code = 'open';

-- First item: the 2026-09-28 reconcile that added the missing 4e-16 back.
-- receipt inrp_sfyd4f9f42fp
UPDATE `quantity` SET value = 0 WHERE id = 'qu_8za5h4957m2r' AND value = 0.0000000000000004;
-- change log inchlg_fj5xg7qgr4eh
UPDATE `quantity` SET value = 0 WHERE id = 'qu_bvr2uh4nr06s' AND value = 0.0000000000000004;

-- Second item: the 8-pair scan, allocated 8.
-- receipt inrp_300ld73pkkm2
UPDATE `quantity` SET value = 8 WHERE id = 'qu_88fiu09qkmfe' AND value = 8.0000000000000004;
-- change log inchlg_fyrvc765whuq
UPDATE `quantity` SET value = 8 WHERE id = 'qu_l4ixdyz2qyit' AND value = 8.0000000000000004;

-- Second item: the 2026-09-22 reconcile that issued the stranded 4e-16 away as open demand.
-- issue inrs_9j2oes70t3ea
UPDATE `quantity` SET value = 0 WHERE id = 'qu_0cww3i3o5gjw' AND value = 0.0000000000000004;
-- change log inchlg_k5d2c23p4ppz
UPDATE `quantity` SET value = 0 WHERE id = 'qu_8qn5cmo493kx' AND value = -0.0000000000000004;
UPDATE `inventory_issue` SET status_code = 'closed', updated_at = CURRENT_TIMESTAMP(3)
WHERE id = 'inrs_9j2oes70t3ea' AND status_code = 'open';

-- +goose Down

SELECT 1;
