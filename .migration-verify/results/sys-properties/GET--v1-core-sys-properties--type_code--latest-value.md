# Verification: GET /v1/core/sys-properties/{type_code}/latest-value

## Result: PARITY CONFIRMED — No issues found

## What Was Compared

### Permission Checks
- **Dashboard**: `checkIsInternalActor` + `checkHasPermission(systemProperties, 'update')`
- **Go**: `CheckIsInternalActor()` + `CheckHasPermission(SystemProperties, Update)` + `CheckTargetAccountSet()`
- **Verdict**: ✅ Match. Go additionally validates target account is set (consistent pattern across Go services).

### Business Logic Flow
Both implementations follow the same flow:
1. Look up sys property by type code and account ID
2. If not found → create with initial value of 1 → return "1"
3. If found → check if current value is a duplicate in the related entity table
4. If duplicate → increment the value → return incremented value as string
5. If not duplicate → return current value as string

- **SSCC Count handling**: Dashboard's `isDuplicate` returns `false` for SSCC, so value is returned without increment. Go has an explicit early-return for SSCC that skips the duplicate check. Same net behavior. ✅

### Duplicate Check SQL Queries
| Type Code | Dashboard Table/Field | Go SQL | Match |
|---|---|---|---|
| `transaction_number` | `transaction.number` WHERE `accountID` | `transaction.number` WHERE `account_id` | ✅ |
| `settlement_number` | `settlement.number` WHERE `accountID` | `settlement.number` WHERE `account_id` | ✅ |
| `sales_order_number` | `order.number` WHERE `ownerAccountID`, `sellerAccountID=owner`, `typeCode=salesOrder` | `sales_order.number` WHERE `owner_account_id`, `seller_account_id=account_id`, `sales_order_type_code='sales_order'` | ✅ |
| `purchase_order_number` | `order.number` WHERE `ownerAccountID`, `buyerAccountID=owner`, `typeCode=purchaseOrder` | `sales_order.number` WHERE `owner_account_id`, `buyer_account_id=account_id`, `sales_order_type_code='purchase_order'` | ✅ |
| `supplier_number` | `accountRelation.externalNumber` WHERE `roleCode=supplier`, `ownerAccountID` | `account_relation.external_number` WHERE `role_code='supplier'`, `owner_account_id` | ✅ |
| `customer_number` | `accountRelation.externalNumber` WHERE `roleCode=customer`, `ownerAccountID` | `account_relation.external_number` WHERE `role_code='customer'`, `owner_account_id` | ✅ |
| `production_run_number` | `productionRun.number` WHERE `accountID` | `production_run.number` WHERE `account_id` | ✅ |
| `sscc_count` | Returns `false` (no check) | Returns `false` (no check) | ✅ |

### Auto-Creation Behavior
- **Dashboard**: `createProperty` inserts with `value: 1`, returns `property.value.toString()` → "1"
- **Go**: `repo.Create` inserts with `value: 1`, returns `strconv.Itoa(int(created.Value))` → "1"
- **Verdict**: ✅ Match

### Response Shape
- **Dashboard**: Returns raw string (`z.string()` response schema) — e.g., `"42"`
- **Go**: Returns `{"object":"sys_property_value","value":"42"}` — structured resource with Object field
- **Verdict**: ✅ Intentional convention change. Go API conventions require all responses to have an `object` field. The value content is identical.

### Validation
- **Dashboard**: Validates `type` param via `z.nativeEnum(SysPropertyTypes)` at the DTO level
- **Go**: Type code passed as string; invalid codes will fail at DB lookup (not-found) then at `Create` (FK constraint), or at `IsDuplicate` (validation error for unknown type code)
- **Verdict**: ✅ Acceptable — invalid type codes are rejected, just at a different layer.

### Error Handling
- Both return not-found when property doesn't exist (then auto-create)
- Both handle DB errors appropriately
- Go has additional tracing spans for observability
- **Verdict**: ✅ Match

### Side Effects
- Both create a new sys_property row if one doesn't exist for the type/account
- Both increment the value if a duplicate is detected
- No emails, webhooks, or messages in either implementation
- **Verdict**: ✅ Match

### Idempotency
- This is a GET endpoint — no idempotency keys required
- Note: This endpoint has side effects (auto-create, increment) but the Dashboard also treats it as GET without idempotency
- **Verdict**: ✅ Match (both treat as GET)

## Notes
- The Dashboard `isDuplicate` methods accept an optional `id` parameter to exclude from the check (for update uniqueness). When called from `getSysPropertyValue`, no `id` is passed, so this filter is inactive. The Go duplicate check queries don't have this exclusion, which is correct for this use case.
- Dashboard trims values with `.trim()` before duplicate checks, but since values originate from integer-to-string conversion, whitespace is never present. No behavioral difference.
