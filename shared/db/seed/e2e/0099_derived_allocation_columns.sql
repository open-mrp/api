-- Runs after every e2e seed that inserts allocations. transaction_allocation.account_id and
-- transaction_type_code are copied from the transaction (InsertTransactionAllocation does it in the
-- API), so seeded allocations need them derived the same way.
UPDATE transaction_allocation ta JOIN transaction t ON t.id = ta.transaction_id
SET ta.account_id = t.account_id, ta.transaction_type_code = t.transaction_type_code
WHERE ta.account_id IS NULL;
