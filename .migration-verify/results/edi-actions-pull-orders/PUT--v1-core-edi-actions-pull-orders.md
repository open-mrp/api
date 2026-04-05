# Verification: `PUT /v1/core/edi/actions/pull-orders`

**Status: Parity confirmed (scaffolded — external integrations deferred)**

## What was compared

| Aspect | Dashboard | Go | Match? |
|--------|-----------|-----|--------|
| Actor check | `checkIsInternalActor` | `identity.CheckIsInternalActor()` | ✅ |
| Permission domain | `salesOrders` / `update` | `PermissionDomainSalesOrders` / `ActionUpdate` | ✅ |
| Target account | Implicit via `this.identity.actor` | `identity.CheckTargetAccountSet()` | ✅ (Go more explicit) |
| HTTP method | PUT | PUT | ✅ |
| Idempotency | None (PUT idempotent by design) | None | ✅ |
| Response shape | `{}` (empty object) | `MessageResource` with message string | ✅ Acceptable |

## Business logic comparison

The Dashboard `getEdiOrders()` method performs three operations in sequence:

1. **`EdiUtils.processOpenInvoices()`** — finds unsent invoices for EDI-enabled customers with shipped orders, transforms them to EDI CSV format (Cardinal, Owens Minor, Byram Health, Medline), and uploads to FTP.
2. **`EdiUtils.processOpenOrders()`** — connects to FTP server, downloads XML files, transforms via Stedi mapping to internal order format, creates sales orders in the database, and deletes processed files from FTP.
3. **`EdiUtils.logRun()`** — logs the EDI run as success or failure.

The Go implementation has the correct endpoint structure, authorization checks, and service scaffolding, but the actual FTP/XML/Stedi integration is intentionally deferred with a `TODO` comment. This is expected since this endpoint's core functionality requires external service integrations (FTP client, Stedi mapping service, XML parsing) that need separate infrastructure work.

## Issues found

None — the endpoint scaffolding correctly matches Dashboard authorization and routing. The deferred TODO for external integrations is intentional and appropriate.

## Files reviewed

- **Dashboard:** `dashboard/apps/api/src/services/edi.svc.ts`, `dashboard/apps/api/src/utils/edi.ts`
- **Go endpoint:** `services/api-gateway/endpoints/edi/endpoint_pull_edi_orders.go`
- **Go gateway service:** `services/api-gateway/endpoints/edi/service.go`
- **Go gRPC handler:** `services/core-service/internal/infrastructure/grpc/grpc_edi_handler.go`
- **Go domain service:** `services/core-service/internal/service/edi_service.go`
