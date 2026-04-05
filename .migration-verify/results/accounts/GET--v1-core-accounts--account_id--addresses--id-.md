# Verification: GET /v1/core/accounts/{account_id}/addresses/{id}

**Status: PARITY CONFIRMED** (no code changes needed)

## What was compared

- **Validation**: Both require account_id and address ID as path parameters
- **Permission checks**: Both check `isAssignedActor`, skip permission check for customer/supplier actors, and check read access for external targets
- **DB query**: Both query the address by ID filtered through the account_address join table to ensure the address belongs to the target account
- **Error handling**: Both return 404 when address not found
- **Customer actor access**: Both allow customer actors (permission check only applies to internal actors)
- **Idempotency**: N/A — GET endpoint, idempotent by design
- **Side effects**: None in either implementation

## Intentional differences

1. **Permission domain**: Dashboard uses `customers` domain with `read` action; Go uses dedicated `addresses` domain with `read` action. This is an intentional improvement — the Go API has a finer-grained permission model with a dedicated `addresses` permission domain (defined in `auth-service/pkg/types/permissions.go`).

2. **Response shape**: Dashboard returns flat address fields (`line1`, `line2`, `city`, `state`, `postalCode`, `country`). Go nests these under a `Geolocation` sub-resource with updated field names (`street_line_1`, `street_line_2`, `locality`, `state`, `postal_code`, `country`) plus additional fields (`google_place_id`, `latitude`, `longitude`). This follows the Go API's sub-resource conventions documented in CLAUDE.md.

## Files reviewed

### Dashboard (Express.js)
- `dashboard/apps/api/src/services/address.svc.ts` — service logic with `find()` method
- `dashboard/apps/api/src/repositories/address.repo.ts` — Prisma query with account_address join
- `dashboard/apps/api/src/controllers/address.ctrl.ts` — request validation and routing

### Go API
- `services/api-gateway/endpoints/addresses/endpoint_get_address.go` — endpoint definition
- `services/api-gateway/endpoints/addresses/service.go` — gateway service calling gRPC
- `services/api-gateway/endpoints/addresses/presenter.go` — proto-to-resource conversion
- `services/core-service/internal/service/address_service.go` — business logic (lines 105-134)
- `services/core-service/internal/infrastructure/grpc/grpc_handler.go` — gRPC handler
- `services/core-service/internal/infrastructure/repository/address_repository.go` — DB access
- `services/core-service/internal/infrastructure/queries/address.sql` — SQL query with JOIN
- `services/core-service/internal/domain/address_models.go` — domain types

## Remaining concerns

None. The Go implementation faithfully preserves the Dashboard business logic with two intentional structural improvements (finer-grained permissions and sub-resource nesting).
