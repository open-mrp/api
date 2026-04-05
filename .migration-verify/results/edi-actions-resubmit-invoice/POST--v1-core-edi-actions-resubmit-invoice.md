# POST /v1/core/edi/actions/resubmit-invoice

## Status: Parity confirmed (scaffolded — FTP/EDI integration pending)

## What was compared

| Aspect | Dashboard | Go | Match? |
|--------|-----------|-----|--------|
| Actor type check | `checkIsInternalActor()` | `identity.CheckIsInternalActor()` | Yes |
| Permission check | `PermissionDomains.invoices` / `update` | `PermissionDomainInvoices` / `ActionUpdate` | Yes |
| Target account required | Yes (implicit via `ownerAccountID`) | `identity.CheckTargetAccountSet()` | Yes |
| Invoice ID required | Yes (request body) | `validate:"required"` + service-level check | Yes |
| Idempotency | N/A in Dashboard | `WithIdempotencyTracking` in gRPC handler | Yes (Go adds this for all POST endpoints per convention) |
| Response shape | N/A (void/200) | `MessageResource { message }` | Acceptable |

## Findings

### Transport layer — correct
- Endpoint definition: `POST /v1/core/edi/actions/resubmit-invoice`, request takes `invoice_id` (required), returns `MessageResource`.
- gRPC handler correctly uses `WithIdempotencyTracking` for the POST endpoint.
- API gateway service correctly marshals request/response through gRPC.

### Auth/permission checks — match
- Both require internal actor with `update` permission on the `invoices` domain.
- Go also checks `TargetAccountSet`, which is consistent with all other Go service methods.

### Business logic — intentionally scaffolded
The Go service (`edi_service.go:386-411`) has a `TODO` comment and returns `nil` after auth checks. The Dashboard implementation performs:

1. Reset invoice `isEdiSent` to `false` via `invoiceRepo.updateEdiStatus()`
2. Fetch invoice + shipment from DB
3. Validate: invoice exists, customer is EDI-enabled, shipment exists
4. Transform invoice to customer-specific CSV format (Cardinal, Owens & Minor, Byram, Medline)
5. Upload CSV to FTP server
6. Set `isEdiSent` to `true` on success
7. On error: send email notification to admin (silent failure — does not return error to caller)

This requires FTP client infrastructure and EDI transformation logic that does not yet exist in the Go codebase. The `PullOrders` endpoint is similarly scaffolded. This is expected — the EDI action endpoints are infrastructure-dependent and were scaffolded with the intent to wire later.

## Issues found and fixed

None — no code changes needed. The scaffolding correctly implements the transport and auth layers. The missing business logic is behind a TODO and requires FTP/EDI infrastructure buildout.

## Remaining concerns

- The full FTP/EDI processing logic needs to be implemented when the infrastructure is available. This includes:
  - Invoice repository method to update `isEdiSent` status
  - Invoice + shipment lookup and validation
  - Customer-specific CSV transformation (4 different formats)
  - FTP client for file upload
  - Error notification via email on failure
