# DELETE /v1/core/product-line-access/customers/{customer_id}

## Result: PARITY CONFIRMED

No issues found. The Go implementation correctly mirrors the Dashboard business logic.

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Permission: internal actor | `checkIsInternalActor` | `CheckIsInternalActor()` | Yes |
| Permission: domain/action | `productLineAccess` / `delete` | `PermissionDomainProductLineAccess` / `ActionDelete` | Yes |
| Account scoping | `this.identity.targetAccountID` | `*identity.TargetAccountID` | Yes |
| Existence check | `find({ customerID, accountID })` → 404 if null | `Get(accountID, customerID)` → 404 if not found | Yes |
| Account relation lookup | `accountRelation.findFirst(owner, counterparty, role=customer)` → 404 | `GetAccountRelationForCustomer` SQL query → mapped 404 | Yes |
| Deletion | `accountRelationProductLine.deleteMany({ accountRelationID })` | `DeleteAccountRelationProductLinesByRelationID` SQL | Yes |
| Response status | 200 OK with deleted data | 204 No Content | Acceptable (Go convention) |
| Idempotency keys | N/A (DELETE) | N/A (DELETE) | Yes |
| Side effects | None | None | Yes |

## Notes

- The Go endpoint returns 204 No Content (empty body) instead of 200 OK with the deleted resource data. This is consistent with the Go API's convention for all DELETE endpoints and is an acceptable deviation.
- The Go implementation adds an explicit database transaction around the delete, which is stricter than the Dashboard's implementation — this is a safe improvement.
- The Go repository has a more explicit existence check: it counts product line records and returns a specific 404 message ("No product line access found for this customer") if none exist, in addition to checking the account relation exists. The Dashboard handles this via the `find()` call in the service layer. Both paths produce equivalent 404 behavior.
