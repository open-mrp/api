# POST /v1/core/scanning-stations — Verification Result

**Status: Issues found and fixed**

## What was compared

- **Permission checks**: Both require internal actor + `scanningStations` / `create` permission. Go also validates target account is set. **Parity confirmed.**
- **Request fields**: Dashboard accepts `name` (required), `type` (required), `departmentID` (required), `materialCheckRequired` (default false), `notes` (default null), `id` (auto-generated), `batchLabelTagSize` (default null), `batchLabelType` (default null). Go accepts the same required fields. `batchLabelTagSize`/`batchLabelType` are not accepted on create in Go, matching the DB defaults (null). **Parity confirmed.**
- **Validation**: Dashboard validates type via Prisma FK constraint on `scanning_station_type.code`. Go validates type explicitly with `constants.ScanningStationType.IsValid()`. **Equivalent behavior.**
- **ID generation**: Dashboard generates ID client-side via `genTypeID(TypeIDPrefixes.scanningStation)`. Go generates server-side via `id.GenID(id.ScanningStationIDPrefix, nil)`. **Equivalent behavior.**
- **DB insert**: Dashboard uses Prisma `create()`. Go uses raw INSERT via sqlc. Both insert the same columns. **Parity confirmed.**
- **Response shape**: Dashboard returns flat object with `departmentID` field. Go returns structured response with `department` as a sub-resource object `{ id, object, name }` and `production_steps` as expandable list, plus `label_size_code`, `label_type_code`. This follows Go API conventions. **Acceptable divergence per API resource conventions.**
- **Idempotency**: Go correctly implements idempotency keys with recovery points for POST. Dashboard does not have idempotency. **Enhancement, as expected.**
- **Side effects**: Neither implementation has side effects (no emails, webhooks, messages). **Parity confirmed.**
- **Error handling**: Both return appropriate errors for auth failures. Go adds name uniqueness check (conflict error) which Dashboard does not have. **Enhancement — reasonable guard.**

## Issues found and fixed

1. **Repository `Create` method was stubbed** — Implemented the `InsertScanningStation` SQL query and the repository `Create` method. The method inserts the record and then calls `Get` to return the full domain object with department name and production steps.

2. **Repository `ExistsByName` method was stubbed** — Implemented the `CountScanningStationsByName` SQL query and the repository `ExistsByName` method with `excludeID` support for update scenarios.

## Notes

- The Go API adds a **name uniqueness check** (`ExistsByName`) that the Dashboard does not have. This is a reasonable enhancement that prevents duplicate scanning station names within an account.
- The `scanning_station_type_code` column has a FK to `scanning_station_type.code` in the DB, so invalid types will also be caught at the DB level if they pass validation.
- The `Update`, `Delete`, and `ConnectProductionStepsByName` repository methods remain stubbed but are out of scope for this endpoint verification.
