# POST /v1/core/settlements — Migration Verification

## Result: Issues found and partially fixed

## What was compared

| Aspect | Dashboard | Go | Match? |
|--------|-----------|-----|--------|
| Permission checks (internal actor + settlements.create) | Yes | Yes | Yes |
| Target account required | Yes | Yes | Yes |
| Idempotency keys (POST) | N/A (Express) | Yes (recovery points) | Yes (improved) |
| Responsible user validation | Validates user exists in account | **Was missing** | **Fixed** |
| Responsible user ID resolution | Resolves user ID → account_user ID | **Was missing** | **Fixed** |
| Settlement number generation | Via sys_property increment | Via sys_property increment | Yes |
| Allocation creation (quantity + allocation records) | Yes | Yes | Yes |
| Dollar unit lookup | Yes | Yes | Yes |
| Response shape (settlement + allocations) | Yes | Yes | Yes |
| Error handling | 403/404 | 403/404 + invariant violations | Yes |

## Issues found and fixed

### 1. Missing responsible user validation and ID resolution (FIXED)

**Dashboard behavior:** Accepts a `responsibleUserID` (user ID), looks up the account_user record via `AccountUserRepo.find({ userID, accountID })`, returns 404 if not found, and stores the **account_user ID** in the settlement.

**Go behavior (before fix):** Accepted `responsible_user_id` and stored it directly without validation. If a non-existent or invalid ID was passed, it would either fail on a foreign key constraint or store incorrect data.

**Fix:** Added validation in `settlement_service.go` CreateSettlement method. Now resolves user ID to account_user ID via `AccountUserRepo.FindByAccountAndUserID()` and returns 404 if the user is not found in the account.

## Remaining parity gaps (not fixed — require broader changes)

### 2. createMissingTransactions not implemented (SIGNIFICANT)

**Dashboard behavior:** The allocation request includes a `lightTransaction` object with transaction metadata (type, method, adjustmentType, customerID, createdAt). Before creating the settlement, the Dashboard calls `TransactionRepo.createMissingTransactions()` which:
- Groups allocations by transaction ID
- For each transaction that doesn't exist in the DB, creates it with an auto-incremented number and the provided metadata
- This allows the settlement creation to simultaneously create new transactions

**Go behavior:** The allocation request only includes `transaction_id`, `invoice_id`, `amount`, and `note`. All referenced transactions must already exist. There is no mechanism to create transactions as part of settlement creation.

**Impact:** If the Dashboard frontend relies on this behavior to create adjustment or other transactions alongside settlements, those flows will break with the Go API.

**Required changes:** Proto field additions (optional transaction metadata on `CreateSettlementAllocationParam`), proto regeneration, API gateway request expansion, domain model updates, and service logic to check transaction existence and create missing ones using the existing `TransactionRepo.Create()` and `FetchAndIncrementTransactionNumber()` methods.

### 3. Payment status evaluation is incomplete

**Dashboard behavior (post-creation side effect):**
- For each unique **invoice** in the settlement: calculates the actual balance (total invoiced minus total allocated across all settlements), then sets `isPaidInFull` (balance == 0) and `isOverPaid` (balance < 0)
- For each unique **transaction**: calculates the actual balance (transaction amount minus total allocated), then sets `isFullyAllocated` (balance <= 0)

**Go behavior:** The `updatePaymentStatuses` method:
- Blindly marks all affected transactions as `isFullyAllocated = true` without calculating the actual balance
- Skips invoice status updates entirely (comment says "For now, leave the invoice status as-is")

**Impact:** Transaction and invoice payment status flags may be incorrect, leading to data inconsistencies in payment tracking.

**Required changes:** Add SQL queries to calculate transaction balance (transaction amount minus sum of all allocations) and invoice balance (total invoiced minus sum of all allocations). Add repo methods for these balance queries. Update `updatePaymentStatuses` to evaluate actual balances before setting flags.

## Files reviewed

### Dashboard
- `dashboard/apps/api/src/services/settlement.svc.ts` — Service layer with permission checks, user validation, and create/update/delete
- `dashboard/apps/api/src/repositories/settlement.repo.ts` — Repository with `create`, `createMissingTransactions` call, `updateSettlementPaymentStatus`
- `dashboard/apps/api/src/controllers/settlement.ctrl.ts` — Controller with post-creation payment status update
- `dashboard/packages/dtos/src/sections/settlements.ts` — Request DTO schema
- `dashboard/packages/objects/src/classes/payments/BaseTransactionAllocation.ts` — Allocation object with lightTransaction

### Go
- `services/api-gateway/endpoints/settlements/endpoint_create_settlement.go` — Endpoint definition
- `services/api-gateway/endpoints/settlements/service.go` — API gateway service layer
- `services/api-gateway/endpoints/settlements/presenter.go` — Response presenter
- `services/core-service/internal/service/settlement_service.go` — **Modified** — Business logic
- `services/core-service/internal/infrastructure/repository/settlement_repository.go` — Repository
- `services/core-service/internal/infrastructure/queries/settlement.sql` — SQL queries
- `services/core-service/internal/domain/settlement_models.go` — Domain models
- `services/core-service/internal/infrastructure/grpc/grpc_settlement_handler.go` — gRPC handler
