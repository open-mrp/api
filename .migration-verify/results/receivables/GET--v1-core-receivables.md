# Verification: GET /v1/core/receivables

**Status: PARITY CONFIRMED** — No issues found.

## What was compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Auth: internal actor check | `checkIsInternalActor` | `CheckIsInternalActor()` | Yes |
| Permission | `invoices:read` | `PermissionDomainInvoices, ActionRead` | Yes |
| Target account required | `identity.targetAccountID` | `CheckTargetAccountSet()` | Yes |
| Cutoff date filter | Optional `cutoffDate` query param | Optional `cutoff_date` query param | Yes |
| Filter: `isPaidInFull = false` | Prisma `isPaidInFull: false` | SQL `inv.is_paid_in_full = false` | Yes |
| Filter: `createdAt < cutoffDate` | Prisma `createdAt: { lt: cutoffDate }` | SQL `inv.created_at < cutoff_date` | Yes |
| Order | `createdAt DESC` | `created_at DESC, id DESC` | Yes (tiebreaker added) |
| Balance calculation | `sum(qty * unitPrice) - sum(allocations)` rounded to 2 decimals | Same SQL subqueries, `ROUND(..., 2)` | Yes |
| Allocation cutoff | Filters allocations by cutoff date | `ta.created_at < allocation_cutoff_date` | Yes |
| Customer number source | `counterpartyAccountRelations[0].externalNumber` | `account_relation` JOIN on `account_relation_role_code = 'customer'` | Yes |
| Response fields | `invoiceID, invoiceNumber, poNumber, invoicedAt, customerID, customerNumber, customerName, remainingBalance, isPaidInFull` | Same fields, restructured with `invoice` and `customer` sub-resources per API conventions | Yes |
| Idempotency | N/A (GET endpoint) | N/A (GET endpoint) | Yes |
| Side effects | None | None | Yes |

## Differences (acceptable)

1. **Pagination**: Dashboard returns all results as a flat array. Go adds cursor-based pagination — this is an enhancement, not a regression. Clients can paginate through large result sets.
2. **Response structure**: Dashboard returns flat fields (`invoiceID`, `customerID`, etc.). Go uses sub-resources (`invoice: { id, object, number }`, `customer: { id, object, name, number }`) per API resource conventions. All the same data is present.
3. **Remaining balance type**: Dashboard returns a JS number. Go returns a string formatted to 2 decimal places (`fmt.Sprintf("%.2f", ...)`), which is appropriate for monetary values.

## No issues found or fixed
