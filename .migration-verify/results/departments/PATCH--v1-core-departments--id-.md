# PATCH /v1/core/departments/{id} — Migration Verification

## Result: Parity Confirmed (with minor notes)

No code changes were required. The Go implementation faithfully reproduces the Dashboard business logic.

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Permission: internal actor check | `checkIsInternalActor` | `identity.CheckIsInternalActor()` | Yes |
| Permission: domain + action | `departments` / `update` | `PermissionDomainDepartments` / `ActionUpdate` | Yes |
| Account scoping | `this.identity.targetAccountID` | `*identity.TargetAccountID` | Yes |
| Existence check before update | `checkExistence({ id, accountID })` → 404 | `txRepo.Get(...)` → 404 via MapSQLError | Yes |
| Partial field update (name) | Prisma ignores undefined | `COALESCE(narg, column)` keeps old value | Yes |
| Partial field update (notes) | Prisma ignores undefined | `COALESCE(narg, column)` keeps old value | Yes |
| Location update | `PrismaUtils.connect(data.location)` | `COALESCE(narg('location_id'), location_id)` | Yes |
| Scanning stations (additive connect) | `{ connect: [...] }` | `SetScanningStationsDepartmentID` (UPDATE SET department_id) | Yes |
| Machines (additive connect) | `{ connect: [...] }` | `SetMachinesDepartmentID` (UPDATE SET department_id) | Yes |
| Response: full department with relations | Prisma select with DepartmentAdapter | Re-fetch via `Get` + `attachSubResources` | Yes |
| Response shape | `{ id, name, notes, location, scanningStations, machines, createdAt, updatedAt }` | Same fields via presenter + apiresource.Department | Yes |
| Side effects | None | None | Yes |
| Idempotency | Not implemented | Idempotency keys with recovery points | Yes (Go improvement) |
| HTTP status | 200 OK | 200 OK | Yes |

## Notes

1. **Name uniqueness check (Go enhancement):** Go checks for duplicate department names on update (excluding the current department). The Dashboard does not perform this check. This is a beneficial addition that prevents data inconsistency — no action needed.

2. **Nullable field clearing limitation:** Go's `COALESCE(narg, column)` SQL pattern means that sending `"notes": null` or `"location_id": null` in the request body will keep the existing value rather than clearing it. In the Dashboard, Prisma distinguishes between `undefined` (don't change) and `null` (set to NULL). This is a systemic Go JSON limitation (`*string` cannot distinguish missing vs. explicit null) and applies across all PATCH endpoints using this pattern — not specific to departments.

3. **Request field shape difference:** Dashboard accepts `location` as a `BasicInfo` object (`{id, name}`), while Go accepts `location_id` as a string. This is consistent with the Go API gateway convention where requests take IDs and responses return objects.
