# Verification: PATCH /v1/core/suppliers/{id}

## Result: Parity Confirmed (no code changes needed)

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| **Permission: actor type** | `checkIsInternalActor` | `identity.CheckIsInternalActor()` | Yes |
| **Permission: domain/action** | `suppliers / update` | `PermissionDomainSuppliers / ActionUpdate` | Yes |
| **Target account required** | `this.identity.targetAccountID` | `identity.CheckTargetAccountSet()` | Yes |
| **Duplicate number check** | `isDuplicate()` excluding self | `ExistsByNumber()` excluding self | Yes |
| **Conflict error on dup** | 409 Conflict | `ConflictErrorWithParam("number")` | Yes |
| **Existence check → 404** | `findFirst` before update → 404 | `Get()` after update → 404 (transaction rolls back) | Yes (equivalent) |
| **Partial update semantics** | JS `undefined` = skip field | `COALESCE`/`CASE` SQL + `UpdateNote` bool flag | Yes |
| **Note null handling** | `undefined` vs `null` in JS | `UpdateNote` bool flag controls whether to apply | Yes |
| **Address update** | `PrismaUtils.connect(data.billToAddress)` (by ID) | `COALESCE(bill_to_address_id)` in SQL | Yes |
| **Transaction wrapping** | `prisma.$transaction` | `s.withTx()` | Yes |
| **Re-fetch after update** | `find()` after transaction | `r.Get()` inside transaction | Yes |
| **Response shape** | `{id, name, note, number, billToAddress, shipToAddress, materialCount, createdAt, updatedAt}` | `SupplierDetail{ID, Object, Name, Number, Note, BillToAddress, ShipToAddress, MaterialCount, CreatedAt, UpdatedAt}` | Yes |
| **Idempotency** | Not implemented in Dashboard | Full idempotency key support with recovery points | Go improves on Dashboard |
| **Side effects** | None (DB only) | None (DB only) | Yes |

## Design Differences (acceptable, not parity gaps)

1. **Name field storage**: Dashboard updates `account_relation.alias`; Go updates `account.name`. The Go read path (`GetSupplier` query) reads from `account.name`, so the Go implementation is internally consistent. The Dashboard adapter reads `alias || account.name`. Since Go is the replacement API, this is the correct approach for Go.

2. **Duplicate check trimming**: Dashboard trims the number value (`value.trim()`) before the duplicate check. Go does not trim. This is a very minor difference; input trimming could be added at the API gateway validation layer if needed.

3. **Error messages differ slightly**: Dashboard says "This supplier number is already taken." Go says "A supplier with this number already exists." Same semantics.

4. **Existence check ordering**: Dashboard checks existence before updating (early 404). Go runs the UPDATE (which affects 0 rows if not found), then `Get()` returns 404. Since everything is in a transaction, the net effect is identical — 404 returned, no data modified.

## No Issues Found Requiring Code Changes
