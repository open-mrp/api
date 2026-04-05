# DELETE /v1/core/production-steps/{production_step_id}/consumptions/{id}

## Result: PARITY CONFIRMED

No issues found. The Go implementation correctly mirrors the Dashboard business logic.

## What was compared

- **Permission checks**: Both require internal actor + `production_steps:delete` permission. Go additionally validates target account ID is set (standard Go pattern).
- **Validation**: Both verify the production step belongs to the account before proceeding, returning 404 if not found.
- **Consumption lookup**: Both fetch the consumption scoped to the account. Dashboard returns null and checks; Go's `Get` returns an API error on not-found. Same 404 outcome.
- **Source step discovery**: Both find source production steps connected via the consumption to disconnect before deletion. Dashboard uses `findStepsToRemoveByConsumption`; Go uses `FindSourceStepsByConsumption` mediator — equivalent queries.
- **Transaction operations**: Both execute the same operations atomically:
  1. Disconnect source steps from the production step
  2. Delete the consumption record
  3. Re-link the production flow
- **Quantity cleanup**: Go explicitly deletes the associated quantity and waste quantity records after deleting the consumption row. Dashboard relies on Prisma/DB cascade behavior. This is correct — Go uses raw SQL and must handle cleanup manually.
- **Response**: Both return HTTP 200 with the deleted consumption object.
- **Side effects**: Neither implementation triggers emails, webhooks, or messaging — consistent.
- **Error handling**: Error types and messages match (404 for missing production step, 404 for missing consumption).

## Notes

- Go fetches consumption and source steps sequentially rather than in parallel (Dashboard uses `Promise.all`). This is a minor performance difference, not a logic difference.
- The Go `DeleteConsumptionRow` SQL (`DELETE FROM consumption WHERE id = ?`) does not re-check account ownership, but this is safe because account ownership is already validated upstream via the `Get` call which joins on the production step's account_id.
- DELETE is idempotent by design — no idempotency key handling needed, consistent with project conventions.
