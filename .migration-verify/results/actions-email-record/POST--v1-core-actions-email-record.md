# POST /v1/core/actions/email-record — Migration Verification

## Result: PARITY CONFIRMED

No code changes required. The Go implementation faithfully reproduces the Dashboard business logic.

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Request body | `{ id: string, type: enum }` | `{ id: string, type: string }` with validation | ✅ |
| Record types | invoice, sales_order, purchase_order | Same three types | ✅ |
| Actor check | Internal actor only | `CheckIsInternalActor()` | ✅ |
| Invoice permission | `PermissionDomains.invoices` / `read` | `PermissionDomainInvoices` / `ActionRead` | ✅ |
| Sales order permission | `PermissionDomains.salesOrders` / `read` | `PermissionDomainSalesOrders` / `ActionRead` | ✅ |
| Purchase order permission | `PermissionDomains.purchaseOrders` / `read` | `PermissionDomainPurchaseOrders` / `ActionRead` | ✅ |
| Target account required | Yes (via `this.identity.targetAccountID`) | `CheckTargetAccountSet()` | ✅ |
| Invoice: fetch record | `invoiceRepo.find({ id, ownerAccountID })` | `invoiceRepo.Get(ctx, GetInvoiceParams{...})` | ✅ |
| Invoice: get recipients | `accountRepo` email recipients by invoice ID | `invoiceRepo.GetEmailRecipients(ctx, invoiceID)` | ✅ |
| Invoice: no recipients | Marks as sent, returns `{}` | Marks as sent, returns empty | ✅ |
| Invoice: with recipients | Sends email, marks as sent | Publishes email, marks as sent in tx | ✅ |
| Sales order: fetch record | `orderRepo.find({ id, ownerAccountID })` | `salesOrderRepo.Get(ctx, accountID, salesOrderID)` | ✅ |
| Sales order: get recipients | Acknowledgement recipients | `GetAcknowledgementRecipients` | ✅ |
| Sales order: no recipients | Returns early (no mark as sent) | Returns early (no mark as sent) | ✅ |
| Sales order: with recipients | Sends email, marks acknowledgement sent | Publishes email, marks acknowledgement sent in tx | ✅ |
| Purchase order: fetch record | `purchaseOrderRepo.find({ id, ownerAccountID })` | `purchaseOrderRepo.Get(ctx, accountID, purchaseOrderID)` | ✅ |
| Purchase order: get recipients | Submission recipients | `GetSubmissionRecipients` | ✅ |
| Purchase order: no recipients | Returns early (no mark as sent) | Returns early (no mark as sent) | ✅ |
| Purchase order: with recipients | Sends email, marks submission sent | Publishes email, marks submission sent in tx | ✅ |
| Invalid type error | 400 Bad Request | Validation error with param "type" | ✅ |
| Record not found | 404 Not Found | Repository returns not-found error | ✅ |
| Idempotency | Not present | Full idempotency key support (improvement) | ✅ |
| Response body | `{}` | `{}` (EmptyResource) | ✅ |

## Architectural Differences (Acceptable)

1. **HTTP status code**: Dashboard returns `200 OK`, Go returns `202 Accepted`. Since this triggers an async email send via the notification service, 202 is more semantically correct. The route prefix also changed (`/v1/actions/` → `/v1/core/actions/`).

2. **Email delivery mechanism**: Dashboard sends emails directly via AWS SES (`AwsUtils.sendEmails()`) with inline HTML generation and PDF attachment. Go publishes email send requests via the notification service message bus (`notificationPublisher.PublishSendEmail`). This follows the Go API's standard messaging pattern.

3. **Purchase order sender details**: Dashboard uses the current user's name/email as the "from" address for PO emails (falling back to account). Go passes `account_name` as a template parameter and delegates sender details to the notification service's template system.

4. **Idempotency**: Go adds full idempotency key support with recovery points, which the Dashboard lacks. This is an improvement, not a regression.

## No Issues Found

The core business logic — permission checks, record lookups, recipient resolution, email triggering, and mark-as-sent behavior — all match between Dashboard and Go implementations.
