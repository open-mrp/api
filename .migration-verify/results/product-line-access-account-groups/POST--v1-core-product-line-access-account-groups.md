# Verification: POST /v1/core/product-line-access/account-groups

## Result: PARITY CONFIRMED — No issues found

## What was compared

### Validation Rules
- **Dashboard:** Validates request body via Zod schema (`api.createAccountGroupProductLineAccess.requestSchema.body`), requires `accountGroup` object with `id`/`name` and `productLines` array.
- **Go:** Validates `account_group_id` (required) and `product_line_ids` (required) via struct tags.
- **Verdict:** ✅ Equivalent — both require the account group identifier and product line identifiers.

### Permission Checks
- **Dashboard:** `checkIsInternalActor()` + `checkHasPermission(PermissionDomains.productLineAccess, 'create')`
- **Go:** `identity.CheckIsInternalActor()` + `identity.CheckHasPermission(types.PermissionDomainProductLineAccess, types.ActionCreate)` + `identity.CheckTargetAccountSet()`
- **Verdict:** ✅ Match. Go additionally explicitly checks target account is set (the Dashboard relies on this being present implicitly via `this.identity.targetAccountID`).

### DB Queries and Logic
- **Dashboard repo:**
  1. Check if `accountGroupProductLine` record exists for account group → 409 Conflict
  2. Validate account group exists and belongs to account (`AccountGroupRepo.checkExistence()`)
  3. Validate each product line exists and belongs to account (`ProductLineRepo.checkExistence()`)
  4. Insert records via `createMany()` with generated IDs
- **Go repo:**
  1. Validate account group exists and belongs to account (`GetAccountGroupByIDAndAccount` query)
  2. Check existing records (`CountAccountGroupProductLinesByAccountGroupID`) → conflict if > 0
  3. Validate each product line exists and belongs to account (`ProductLineExistsByIDAndAccount`)
  4. Insert records one-by-one with generated IDs (`InsertAccountGroupProductLine`)
  5. Re-fetch and return the created access
- **Verdict:** ✅ Same validations in slightly different order. Both check existence, ownership, and uniqueness.

### Error Handling
- **Dashboard:** 409 Conflict with message including group name: "A relevant product for the customer group {name} already exists..."
- **Go service:** 409 Conflict: "Product line access for this account group already exists." (service-level check)
- **Go repo:** 409 Conflict: "Product line access already exists for this account group. Use update instead." (repo-level safety check)
- **Verdict:** ✅ Functionally equivalent. Go has a redundant double-check (service + repo) which is a safety pattern. Error messages differ in wording but convey the same meaning.

### Side Effects
- **Dashboard:** None beyond DB writes.
- **Go:** None beyond DB writes.
- **Verdict:** ✅ Match.

### Response Shape
- **Dashboard:** Returns the original input `data` object (AccountGroupProductLines with accountGroup and productLines).
- **Go:** Re-fetches from DB and returns `AccountGroupProductLineAccess` resource with `account_group` sub-resource (id, object, name), `object` field, `product_lines` list, `created_at`, `updated_at`.
- **Verdict:** ✅ Go provides a richer, more standard response with proper object types and timestamps. This is an improvement consistent with API resource conventions.

### Idempotency
- **Dashboard:** No idempotency key support.
- **Go:** Full idempotency key support with recovery points (`RecoveryPointStarted`, `RecoveryPointFinished`), cached responses, and transactional execution.
- **Verdict:** ✅ Go adds idempotency as required by the migration patterns.

## Issues Found
None. The Go implementation faithfully reproduces all Dashboard business logic with expected improvements (idempotency, richer response shape, explicit target account validation, transactional execution).
