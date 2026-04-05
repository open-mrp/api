# PATCH /v1/core/production-steps/{id} — Migration Verification

## Result: PARITY CONFIRMED

No issues found. The Go implementation correctly mirrors the Dashboard business logic.

## What Was Compared

### Permission Checks
- **Dashboard:** `checkIsInternalActor` + `checkHasPermission(PermissionDomains.productionSteps, 'update')`
- **Go:** `CheckIsInternalActor()` + `CheckHasPermission(PermissionDomainProductionSteps, ActionUpdate)` + `CheckTargetAccountSet()`
- **Verdict:** Match. Go additionally validates target account is set (standard Go pattern).

### Updatable Fields
- **Dashboard:** `name`, `levelingFactor`, `allowances`, `scanningStation` (object with `.id`)
- **Go:** `name`, `leveling_factor`, `allowances`, `scanning_station_id` (flat string)
- **Verdict:** Match. All four fields are present. The Go API accepts `scanning_station_id` as a flat string rather than a nested object in the request, which is the standard Go API convention for request payloads.

### Scanning Station Update Logic
- **Dashboard:** `data.scanningStation ? PrismaUtils.connect(data.scanningStation.id) : undefined` — only updates if truthy object provided; cannot unset.
- **Go:** `UpdateScanningStation: params.ScanningStationID != nil` with SQL `CASE WHEN update_scanning_station = true THEN narg ELSE scanning_station_id END` — only updates if pointer is non-nil.
- **Verdict:** Match. Both systems skip the scanning station column when the field is absent/null, and neither supports unlinking a scanning station via this endpoint.

### Partial Update Behavior
- **Dashboard:** Prisma `update()` with explicit field mapping; unmapped fields ignored.
- **Go:** SQL `COALESCE(sqlc.narg('field'), field)` pattern preserves existing values for absent fields.
- **Verdict:** Match.

### Name Uniqueness Validation
- **Dashboard:** No uniqueness check on name.
- **Go:** Checks `ExistsByName` (excluding current step) before updating; returns `ConflictErrorWithParam` if duplicate.
- **Verdict:** Go is stricter. This is a defensive improvement, not a breaking change.

### 404 Handling
- **Dashboard:** Prisma throws if record not found (implicit 404).
- **Go:** Checks `RowsAffected() == 0` and returns `ResourceNotFoundError`.
- **Verdict:** Match — both return errors for missing records.

### Account Scoping
- **Dashboard:** `WHERE id AND accountID` in Prisma query.
- **Go:** `WHERE id = ? AND account_id = ?` in SQL.
- **Verdict:** Match.

### Idempotency
- **Dashboard:** None.
- **Go:** Full idempotency key support with recovery points (`Started` / `Finished`), cached responses, and transactional consistency.
- **Verdict:** Go adds required idempotency per project conventions for PATCH endpoints.

### Response Shape
- **Dashboard:** Returns `LightProductionStep` (lightweight adapter with subset of fields).
- **Go:** Returns full `ProductionStep` resource with nested rates, consumptions, machines, scanning station, in/out steps, and department.
- **Verdict:** Go returns a richer response. Not a breaking change — strictly additive.

### Side Effects
- **Dashboard:** Database update only. No events, messages, or external calls.
- **Go:** Database update only. No events, messages, or external calls.
- **Verdict:** Match.

## Issues Found and Fixed

None. No code changes were necessary.
