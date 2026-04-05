# PATCH /v1/core/transaction-allocations/{id}

## Result: PARITY CONFIRMED — No issues found

## What was compared

### Permission checks
- **Dashboard:** `checkIsInternalActor` + `checkHasPermission(settlements, update)`
- **Go:** `CheckIsInternalActor` + `CheckHasPermission(PermissionDomainSettlements, ActionUpdate)` + `CheckTargetAccountSet`
- **Verdict:** Match. Go additionally validates target account is set (improvement).

### Updatable fields
- **Dashboard:** Accepts `TransactionAllocationUtils.schema.partial()` (all fields), but repo only persists `amount` (via QuantityAdapter) and `createdAt`.
- **Go:** Request accepts `Amount *string` and `CreatedAt *string` only.
- **Verdict:** Match. Go is actually cleaner — it doesn't accept fields it won't use.

### Account ownership / data access
- **Dashboard:** `WHERE transaction.accountID = ownerAccountID AND id = :id`
- **Go:** `GetByID` query joins `transaction` table with `WHERE ta.id = :id AND t.account_id = :account_id`
- **Verdict:** Match. Both enforce account-scoped access via the transaction relationship.

### Update logic
- **Dashboard:** Prisma `update()` on the allocation record, updating `amount` (via QuantityAdapter.updateInput) and `createdAt`.
- **Go:** Separate `UpdateAmount` (updates `quantity.value`) and `UpdateCreatedAt` (updates `transaction_allocation.created_at`), both within a transaction.
- **Verdict:** Match. Go splits into two targeted SQL updates within an atomic transaction.

### SQL queries
- `UpdateAllocationAmount`: `UPDATE quantity SET value = ?, updated_at = NOW(3) WHERE id = ?`
- `UpdateAllocationCreatedAt`: `UPDATE transaction_allocation SET created_at = ?, updated_at = NOW(3) WHERE id = ?`
- Both correctly update the `updated_at` timestamp.

### Error handling
- Both return not-found if allocation doesn't exist or doesn't belong to the account.
- Go uses standard `db.MapSQLError` for SQL errors.

### Response shape
- **Dashboard:** Returns `TransactionAllocation` with `id`, `amount` (Quantity), `note`, `createdAt`, `updatedAt`, `invoice` (LightInvoice), `transaction` (BaseTransaction).
- **Go:** Returns `TransactionAllocation` with `id`, `object`, `amount` (Quantity sub-resource), `note`, `transaction` (sub-resource), `invoice` (InvoiceSummary sub-resource), `created_at`, `updated_at`.
- **Verdict:** Match. Go follows the API resource conventions with `object` field and sub-resources.

### Idempotency
- **Dashboard:** No idempotency key support.
- **Go:** Full idempotency key support with recovery points (RecoveryPointStarted/RecoveryPointFinished).
- **Verdict:** Go improvement — required by the architecture patterns for PATCH endpoints.

### Side effects
- Neither implementation has side effects (no emails, webhooks, or messages).

## No issues found — no changes made.
