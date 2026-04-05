# Verification: GET /v1/core/receivables/accounts/{account_id}

**Status: PARITY CONFIRMED — No issues found**

## What was compared

### Permission checks
- **Dashboard:** `checkIsInternalActor` + `checkHasPermission(customers, read)`
- **Go:** `CheckIsInternalActor()` + `CheckHasPermission(PermissionDomainCustomers, ActionRead)` + `CheckTargetAccountSet()`
- **Result:** Match. Go adds explicit target account check (standard pattern).

### Query parameters
- **Dashboard:** `accountID` (path), `cutoffDate` (optional query)
- **Go:** `account_id` (path), `cutoff_date` (optional query), plus `cursor`/`limit` for pagination
- **Result:** Match. Go adds cursor-based pagination (enhancement over Dashboard's unbounded array).

### DB query logic
- **Dashboard (Prisma):** Finds invoices where `accountID = ownerAccountID`, `isPaidInFull = false`, `order.buyerAccountID = customerID`, `order.ownerAccountID = ownerAccountID`, `order.sellerAccountID = ownerAccountID`, optional `createdAt < cutoffDate`. Orders by `createdAt DESC`.
- **Go (SQL):** Filters `inv.account_id = account_id`, `so.buyer_account_id = customer_account_id`, `inv.is_paid_in_full = false`, optional `inv.created_at < cutoff_date`. Joins through `account_relation` with role `customer` to get customer number/name. Orders by `created_at DESC, id DESC`.
- **Result:** Functionally equivalent. The Dashboard's extra `ownerAccountID`/`sellerAccountID` checks on the sales order are redundant since `inv.account_id` already scopes to the owner account. Go adds secondary sort by `id` for cursor pagination stability.

### Balance calculation
- **Dashboard:** `sum(quantity * rate) - sum(allocation amounts)`, rounded to 2 decimal places with `Math.round(... * 100) / 100`
- **Go SQL:** `ROUND(COALESCE(SUM(q.value * r.value), 0) - COALESCE(SUM(aq.value), 0), 2)`
- **Result:** Match. Both calculate remaining balance the same way.

### Allocation cutoff date
- **Dashboard:** Filters `accountTransactionAllocations` by `createdAt < cutoffDate` when cutoff provided
- **Go SQL:** Filters `ta.created_at < allocation_cutoff_date` when provided
- **Result:** Match.

### Response shape
- **Dashboard:** Flat array of `UnpaidInvoiceReportEntry` with camelCase (`invoiceID`, `invoiceNumber`, `customerID`, `customerName`, `customerNumber`, `poNumber`, `invoicedAt`, `remainingBalance` (number), `isPaidInFull`)
- **Go:** Paginated `List[ReceivableEntry]` with sub-resources: `object`, `invoice: { id, object, number }`, `customer: { id, object, name, number }`, `po_number`, `invoiced_at`, `remaining_balance` (string), `is_paid_in_full`
- **Result:** Go follows API resource conventions with proper sub-resources and pagination. All Dashboard data fields are represented. `remaining_balance` is string-formatted ("%.2f") per monetary value conventions.

### Side effects
- None in either implementation (GET endpoint).

### Error handling
- Both use standard patterns for their respective frameworks.

### Idempotency
- N/A — GET endpoint, inherently idempotent.

## Conclusion

The Go implementation faithfully reproduces all Dashboard business logic for this endpoint. Differences are intentional improvements following Go API conventions: cursor-based pagination, proper sub-resource nesting, and string-formatted monetary values.
