-- +goose Up

-- Copy each allocation's account and transaction type from its transaction. New allocations get them
-- at insert (InsertTransactionAllocation), and neither ever changes on a transaction, so this is
-- history only. Idempotent: it touches only rows that differ from their transaction, so it can be
-- re-run to catch allocations written by an older image during the rollout.
-- updated_at is left alone: this is a derived value, not a change to the allocation.
UPDATE `transaction_allocation` ta
JOIN `transaction` t ON t.id = ta.transaction_id
SET ta.account_id = t.account_id, ta.transaction_type_code = t.transaction_type_code
WHERE ta.account_id IS NULL OR ta.account_id <> t.account_id
   OR ta.transaction_type_code IS NULL OR ta.transaction_type_code <> t.transaction_type_code;

-- +goose Down

-- Not reversible: the columns are dropped by their schema migration's Down.
SELECT 1;
