# DELETE /v1/core/transactions/{id} — Verification Result

**Status: PARITY CONFIRMED** — No issues found.

## What was compared

| Aspect | Dashboard | Go | Match? |
|--------|-----------|-----|--------|
| **Permission: internal actor** | `checkIsInternalActor` | `identity.CheckIsInternalActor()` | Yes |
| **Permission: domain/action** | `transactions` / `delete` | `PermissionDomainTransactions` / `ActionDelete` | Yes |
| **Account scoping** | `ownerAccountID` passed to repo, used in WHERE | `params.AccountID` from identity, used in `FindTransactionByID` WHERE | Yes |
| **Fetch before delete** | `this.find({ id, ownerAccountID })` — 404 if not found | `repo.Get(ctx, accountID, transactionID)` — 404 if not found | Yes |
| **Delete allocations** | `transactionAllocation.deleteMany` by transaction ID + account | `DeleteTransactionAllocationsByTransactionID` by transaction ID | Yes |
| **Delete transaction** | `transaction.delete` by ID + account | `DeleteTransaction` by ID | Yes |
| **Atomicity** | Both operations in sequence (optional context) | Wrapped in `withTx` DB transaction | Yes (Go is stricter) |
| **Return value** | Returns the fetched transaction object | Returns the fetched transaction object | Yes |
| **Response status** | 200 OK | 200 OK | Yes |
| **Idempotency keys** | N/A (DELETE) | N/A (DELETE) | Yes |
| **Side effects** | None | None | Yes |

## Notes

- **Quantity cleanup (improvement):** The Go code additionally deletes the associated `quantity` record (`DeleteQuantity`) during the transaction. The Dashboard does not do this, likely leaving orphaned quantity rows. This is a data hygiene improvement in Go, not a parity issue.
- **Account scoping approach:** The Go SQL `DELETE FROM transaction WHERE id = ?` does not include `account_id` in the WHERE clause, but the preceding `Get` call validates ownership via `AND t.account_id = ?`. Functionally equivalent to the Dashboard's `WHERE id AND accountID` approach.
- **Allocation quantity orphans:** Neither implementation cleans up quantity records associated with deleted allocations. This is consistent behavior across both codebases.
