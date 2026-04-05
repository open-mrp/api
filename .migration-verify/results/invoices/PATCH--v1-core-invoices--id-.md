# PATCH /v1/core/invoices/{id} — Migration Verification

## Result: PARITY CONFIRMED

No issues found. No code changes needed.

## What Was Compared

### Updatable Fields
Both implementations update exactly the same 4 fields:
- `note` (string, nullable)
- `has_been_sent` / `hasBeenSent` (boolean)
- `is_edi_sent` / `isEdiSent` (boolean)
- `is_paid_in_full` / `isPaidInFull` (boolean)

### Permission Checks
- **Actor type:** Both require internal actor
- **Permission domain:** Both check `invoices` domain with `update` action
- **Account scoping:** Both require target account ID and scope the UPDATE + re-fetch queries to `account_id`

### DB Queries and Logic
- Dashboard: Prisma partial update on the 4 fields, scoped to `id` + `accountID`
- Go: `UPDATE invoice SET ... WHERE id = ? AND account_id = ?` using `COALESCE(sqlc.narg(...), column)` for partial updates
- Go re-fetches the full invoice summary via `GetInvoiceSummaryByID` with same joins as list query (customer, order, shipment, address, payment term, line count, total invoiced, accepts_invoice_emails)

### Response Shape
Both return an InvoiceSummary with:
- id, number, note
- customer (id, name, number) as sub-resource
- order (id, number) as sub-resource
- shipment (id) as optional sub-resource
- billing_address with geolocation
- payment_term (id, name) as optional sub-resource
- priority_code, line_count, total_invoiced
- is_paid_in_full, is_edi_sent, has_been_sent
- accepts_invoice_emails, customer_is_edi_enabled
- created_at, updated_at

### Idempotency
- Dashboard: No idempotency key support
- Go: Full idempotency key support with recovery points (improvement over Dashboard)

### Side Effects
Neither implementation triggers any side effects (no emails, webhooks, or messages).

### Error Handling
Both return appropriate errors for missing permissions and invalid requests.

## Files Reviewed
- Dashboard: `dashboard/apps/api/src/services/invoice.svc.ts`, `dashboard/apps/api/src/repositories/invoice.repo.ts`, `dashboard/apps/api/src/controllers/invoice.ctrl.ts`
- Go gateway: `services/api-gateway/endpoints/invoices/endpoint_update_invoice.go`, `services/api-gateway/endpoints/invoices/service.go`, `services/api-gateway/endpoints/invoices/presenter.go`, `services/api-gateway/pkg/resource/invoice_resource.go`
- Go core: `services/core-service/internal/infrastructure/grpc/grpc_invoice_handler.go`, `services/core-service/internal/service/invoice_service.go`, `services/core-service/internal/infrastructure/repository/invoice_repo.go`, `services/core-service/internal/infrastructure/queries/invoice.sql`
