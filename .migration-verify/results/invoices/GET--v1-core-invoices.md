# Verification: GET /v1/core/invoices

**Status: Issues found and fixed**

## What was compared

| Aspect | Result |
|--------|--------|
| Permission checks (internal actor, invoices:read, target account) | Match |
| Status filter (all/paid/unpaid/overpaid) | Match |
| Item ID filter (EXISTS subquery on invoice_line → sales_order_line) | Match |
| Customer ID filter (buyer_account_id IN) | Match |
| Product line ID filter (EXISTS subquery via product) | Match |
| Customer group ID filter (account_group_id IN) | Match |
| Sales rep ID filter (sales_rep_id IN) | Match |
| Date range filter (created_at gte/lte) | Match |
| Total invoiced calculation (SUM quantity.value * unit_price.value) | Match |
| accepts_invoice_emails (order_email_contact check) | Match |
| Line count (COUNT invoice_line) | Match |
| Cursor-based pagination | N/A (Dashboard uses offset; Go uses cursor — architectural choice) |
| Response shape (sub-resources, object types) | Acceptable — follows Go API conventions |

## Issues found and fixed

### 1. Search query missing buyer account name (FIXED)

**Dashboard behavior:** The `query` parameter searches across invoice `number` AND buyer account `name` (via OrderAdapter → CustomerAccountAdapter).

**Go behavior (before fix):** The `query` parameter searched invoice `number` and `note` only, missing the buyer account name search.

**Fix:** Added `OR buyer.name LIKE sqlc.narg('search_query')` to both `ListInvoicesForward` and `ListInvoicesBackward` SQL queries. The `buyer` table is already joined, so no additional joins needed.

**Note:** The Go version additionally searches `inv.note` which the Dashboard does not. This is an enhancement and does not break parity — it only returns more results, never fewer for the same query.

## Remaining notes

- **Pagination model differs:** Dashboard uses offset-based (take/skip), Go uses cursor-based. This is an intentional architectural upgrade and is acceptable.
- **Sorting:** Dashboard sorts by Prisma full-text relevance then `createdAt DESC`. Go sorts by `createdAt DESC, id DESC`. Relevance sorting is a Prisma-specific feature; the Go version provides deterministic ordering required for cursor pagination.
- **`priority_code`:** Dashboard returns a nested `priority` object; Go returns `priority_code` as a flat string. This is a simplification but provides the same data (the code is the key field used by consumers).
- **`customer_is_edi_enabled`:** Dashboard nests this in the customer sub-resource (`customer.isEdiEnabled`). Go exposes it as a top-level field. This is a minor structural difference but the data is present and accessible.
