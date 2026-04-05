# GET /v1/core/invoices/{id} — Migration Verification

## Result: PARITY CONFIRMED

No issues found. No code changes needed.

## What Was Compared

- **Permission checks**: Both require internal actor + `invoices:read` permission
- **Account isolation**: Both filter by account ID from identity context
- **DB query logic**: Dashboard uses Prisma `findUnique` by id + accountID; Go uses SQL with JOINs on invoice → sales_order → address → geolocation, LEFT JOIN shipment, filtered by id + account_id
- **404 handling**: Both return not-found error when invoice doesn't exist
- **Computed field `acceptsInvoiceEmails`**: Dashboard computes from order email contacts in adapter; Go computes via SQL EXISTS subquery on `order_email_contact` where `notification_type_code = 'invoice'` — equivalent logic
- **Core fields**: id, number, note, isPaidInFull, isEdiSent, hasBeenSent, acceptsInvoiceEmails — all present in both
- **Nested resources**: order, billing address, shipment — all present (naming follows Go API conventions)
- **Lines and allocations**: Dashboard eagerly loads; Go uses expandable includes (`?include=lines&include=allocations`) per Go API convention
- **Side effects**: None in either implementation
- **Idempotency**: N/A (GET endpoint)
- **Error handling**: Consistent error types

## Notes

- Go adds `is_over_paid` field not present in Dashboard response — additive, not a parity gap (column exists in DB)
- Lines/allocations use Go API's include/expand pattern rather than eager loading — this is by design per codebase conventions
- Field naming differences (`billTo` → `billing_address`) follow Go API resource conventions
