# Verification: GET /v1/core/open-credits

## Status: Issues found and fixed

## What was compared

- **Permission checks**: Both require internal actor + `settlements:read` permission. Go also checks `TargetAccountID` is set. **Parity confirmed.**
- **Query parameters**: `start_date`, `end_date`, `customer_ids` — all present in both implementations. **Parity confirmed.**
- **Date filtering**: `created_at >= start_date` (inclusive), `created_at < end_date` (exclusive) — matches Dashboard. **Parity confirmed.**
- **Customer filtering**: Optional `IN (customer_ids)` filter — matches Dashboard. **Parity confirmed.**
- **Sort order**: `created_at DESC` — matches Dashboard. **Parity confirmed.**
- **DB query logic**: Filters `is_fully_allocated = false`, joins to quantity, customer account, account_relation (for external number), transaction_method, adjustment_type, account_user/user, and computes allocated amounts via subquery. **Parity confirmed.**
- **Leftover calculation**: `leftoverAmount = originalAmount - allocatedAmount` — matches Dashboard. **Parity confirmed.**
- **Invoice allocations**: Fetched via separate query grouped by transaction ID — matches Dashboard behavior. **Parity confirmed.**
- **Response fields**: id, number, original_amount, allocated_amount, leftover_amount, customer (name + number), transaction_type, transaction_method, adjustment_type, responsible_user_name, note, stripe_payment_id, invoice_allocations, created_at — all present. **Parity confirmed.**
- **Side effects**: None in either implementation. **Parity confirmed.**
- **Idempotency**: GET endpoint — not required. **Parity confirmed.**

## Issues found and fixed

### 1. Transaction type returned code instead of name
- **Problem**: The SQL query used `t.transaction_type_code AS transaction_type` which returns the raw code (e.g., `credit_memo`), but the Dashboard returns the human-readable name from the `transaction_type` table (e.g., `Credit Memo`).
- **Fix**: Added `JOIN transaction_type tt ON tt.code = t.transaction_type_code` and changed the select to `tt.name AS transaction_type`.
- **Files changed**: `services/core-service/internal/infrastructure/queries/transaction_allocation.sql`, regenerated `sqlc/transaction_allocation.sql.go`.

## Notes

- **Number formatting**: The Dashboard applies `formatRecordNumber()` to pad the number field with leading zeros (e.g., `123` becomes `000123`). The Go API does not apply this formatting, but this is consistent across all Go endpoints (including the transaction list endpoint). This appears to be a deliberate convention difference rather than a per-endpoint bug.
- **Response structure**: The Go API wraps results in the standard `List` resource with `data` array, adds `object` fields, and nests customer as `{ name, number }` sub-object rather than flat fields. These are expected Go API convention differences, not parity issues.
- **`allocated_amount` field**: The Go API additionally returns `allocated_amount` which the Dashboard does not. This is additive and does not break parity.
