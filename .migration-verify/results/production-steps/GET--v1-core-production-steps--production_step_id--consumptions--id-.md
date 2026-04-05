# Verification: GET /v1/core/production-steps/{production_step_id}/consumptions/{id}

## Result: Parity Confirmed

No issues found. The Go implementation faithfully reproduces the Dashboard business logic.

## What Was Compared

### Validation & Permissions
- **Dashboard**: `checkIsInternalActor`, `checkHasPermission(productionSteps, 'read')`
- **Go**: `CheckIsInternalActor()`, `CheckHasPermission(PermissionDomainProductionSteps, ActionRead)`, `CheckTargetAccountSet()`
- **Verdict**: Match. Go adds an explicit target-account-set check, which is standard in the Go codebase.

### DB Query & Account Isolation
- **Dashboard**: `findUnique({ where: { id, productionStep: { accountID } } })` — filters by consumption ID and ensures the parent production step belongs to the account. Does NOT filter by productionStepID from the URL.
- **Go**: First checks `ProductionStepQueryRepo.IsInAccount(accountID, productionStepID)`, then queries `GetConsumption` SQL which filters by `consumption_id`, `step_id`, AND `account_id` (via JOIN on production_step).
- **Verdict**: Go is stricter — it validates the production step exists in the account AND that the consumption belongs to that specific production step. Dashboard only validates the consumption belongs to some production step in the account. This is more correct behavior and not a regression.

### Error Handling
- **Dashboard**: Returns `HttpError.notFound('Consumption not found.')` when the consumption doesn't exist.
- **Go**: `db.MapSQLError` maps `sql.ErrNoRows` → `NewResourceNotFoundError("Resource not found.")`. If the production step isn't in the account, returns `NewResourceNotFoundError("Production step not found.")`.
- **Verdict**: Match (both return 404). Error messages differ slightly, which is expected per Go API conventions.

### Response Shape
- **Dashboard** (`LightConsumption`): `id`, `quantity`, `wasteQuantity`, `consumedItem` (with `itemID`, `sku`, `description`, `itemTypeCode`), `instructions`
- **Go** (`apiresource.Consumption`): `id`, `object`, `quantity` (expandable), `waste_quantity` (expandable), `consumed_item` (expandable with `id`, `object`, `sku`, `description`, `item_type_code`), `instructions`, `created_at`, `updated_at`
- **Verdict**: Match. Go adds `object`, `created_at`, `updated_at` per standard API resource conventions. Sub-resource fields map correctly. `consumed_item` is expandable via the includes system.

### Side Effects
- **Dashboard**: None (read-only)
- **Go**: None (read-only)
- **Verdict**: Match.

### Idempotency
- Not applicable (GET endpoint).

## Notes
- The Go endpoint uses the `v1/core/` route prefix as expected for migrated endpoints.
- The `consumed_item` include expansion is properly configured.
- No customer actor access in either implementation (internal actors only).
