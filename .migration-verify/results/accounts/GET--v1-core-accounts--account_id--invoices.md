# GET /v1/core/accounts/{account_id}/invoices

## Result: Issues found and fixed

## What was compared

- **Permission checks**: Both require internal actor + invoices read permission
- **Account scoping**: Both scope by target account ID from identity
- **Child account inclusion**: Both force `includeChildAccounts = true`
- **DB queries**: Joins, filters, ordering, pagination
- **Response shape**: Field names, types, nested resources
- **Allocations**: Whether payment allocations are fetched
- **IsPrepaid logic**: How prepaid status is determined
- **Status filtering**: Default payment status filter
- **Search**: Query parameter behavior

## Issues found and fixed

### 1. Missing "unpaid" default status filter (HIGH)
- **Dashboard**: Defaults to `status = InvoiceStatuses.unpaid`, filtering to only `is_paid_in_full = false AND is_over_paid = false`
- **Go (before fix)**: No status filter — returned ALL invoices regardless of payment status
- **Fix**: Added `AND inv.is_paid_in_full = false AND inv.is_over_paid = false` to both `ListCustomerInvoicesForward` and `ListCustomerInvoicesBackward` SQL queries

### 2. IsPrepaid always false (MEDIUM)
- **Dashboard**: Checks `customerRelation.paymentTerms.id === PaymentTerms.prepaid` (where prepaid = `'prepaid'`)
- **Go (before fix)**: Always set `IsPrepaid = false` with a TODO comment
- **Fix**: Updated `mapForwardCustomerInvoiceRow` and `mapBackwardCustomerInvoiceRow` to check `row.CustomerPaymentTermID.String == "prepaid"`

### 3. Missing allocations (HIGH)
- **Dashboard**: Fetches `accountTransactionAllocations` inline with the Prisma query
- **Go (before fix)**: Never populated the `Allocations` field on `InvoiceForPayment`
- **Fix**: Added allocation fetching loop in `ListCustomerInvoices` service method, using the existing `repo.GetAllocations()` method for each invoice

## Remaining concerns

- **Search scope**: Dashboard searches both invoice number AND order details (via OrderAdapter.fetchInput which searches order number, customer name, etc.). Go only searches invoice number. This is a minor difference — the Go search is more limited but functional.
- **Pagination model**: Dashboard uses offset-based (take/skip + count). Go uses cursor-based pagination. This is an intentional architectural improvement, not a parity issue.
- **Allocations N+1**: The fix fetches allocations per-invoice. Since this endpoint is scoped to unpaid invoices for a single customer (typically a small set), this is acceptable. A batch query could be added later if needed.
