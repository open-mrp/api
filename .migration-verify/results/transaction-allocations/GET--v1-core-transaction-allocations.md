# GET /v1/core/transaction-allocations

## Result: Issues found and fixed

## What was compared

- **Permission checks**: Both require internal actor + `settlements:read` permission + target account ID
- **Query parameters/filters**: transaction_type, start_date, end_date, search query
- **DB query logic**: JOINs, filters, ordering, pagination
- **Response shape**: Field names, types, nested sub-resources
- **Error handling**: Standard error propagation through layers
- **Side effects**: None (read-only endpoint)

## Issues found and fixed

### 1. Note field source (FIXED)
- **Dashboard**: Returns `data.transaction.note` (the transaction's note)
- **Go (before)**: Returned `ta.note` (the allocation's own note)
- **Fix**: Changed SQL from `ta.note` to `t.note` in both ListAllocationEntriesForward and ListAllocationEntriesBackward queries

### 2. Transaction method/adjustment type values (FIXED)
- **Dashboard**: Returns the code (`transactionMethodCode`, `adjustmentTypeCode`) — e.g., `"check"`, `"credit_note"`
- **Go (before)**: Returned the display name from joined tables (`tm.name`, `adjt.name`) — e.g., `"Check"`, `"Credit Note"`
- **Fix**: Changed SQL from `tm.name`/`adjt.name` to `t.transaction_method_code`/`t.adjustment_type_code` and removed now-unnecessary LEFT JOINs on `transaction_method` and `adjustment_type` tables from both list queries

## Remaining notes (acceptable differences)

### Pagination model
- Dashboard uses offset-based (take/skip), Go uses cursor-based. This is intentional per Go API conventions.

### Response structure
- Dashboard returns flat fields (`transactionID`, `invoiceID`), Go wraps in sub-objects (`transaction: {id, object, type, method}`, `invoice: {id, object, number}`). This is intentional per Go API resource conventions.

### Search scope
- Dashboard searches via Prisma `contains` on: invoice number, transaction number, transaction method name, customer name, responsible user name, and nested allocations
- Go searches via MySQL FULLTEXT on: invoice number and transaction number only
- The Go search covers the primary use case (searching by invoice/transaction number). Adding customer name search would require additional FULLTEXT indexes or LIKE queries.

### Customer sub-resource
- Dashboard includes `customer.id` and `customer.billToAddress` in the response
- Go only includes `customer.name` and `customer.number`
- The Go response provides sufficient customer identification for the list view

## Files modified
- `services/core-service/internal/infrastructure/queries/transaction_allocation.sql` — Changed note source, method/adjustment type to codes, removed unused JOINs
- `services/core-service/internal/infrastructure/sqlc/transaction_allocation.sql.go` — Regenerated via `make sqlc core`
