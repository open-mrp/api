# Verification: GET /v1/core/transactions/{id}

## Result: Issues found and fixed

## What was compared

- **Permission checks**: Both check internal actor + read permission on `transactions` domain + account isolation via `targetAccountID`. Go additionally explicitly validates target account is set. **Parity confirmed.**
- **DB query**: Both query the same data via equivalent JOINs: transaction + quantity/unit + transaction_type + account + account_relation + LEFT JOIN transaction_method, adjustment_type, user. Both filter by `(id, account_id)`. **Parity confirmed.**
- **404 handling**: Dashboard returns null from Prisma `findUnique` and throws `HttpError.notFound`. Go maps `sql.ErrNoRows` to a not-found API error. **Parity confirmed.**
- **Error handling**: Both return appropriate error types (forbidden for auth, not found for missing records). **Parity confirmed.**
- **Response shape**: Both return the same core fields (id, number, amount, customer, responsible_user, note, transaction_type, transaction_method, adjustment_type, is_fully_allocated, stripe_payment_id, allocation_count, timestamps). **Parity confirmed with one fix (see below).**
- **Allocations**: Dashboard always includes allocations inline. Go makes them expandable via `?include=allocations`. This is an intentional Go API design pattern (include system), not a bug. Data is fully available when requested.
- **Idempotency**: GET endpoint — no idempotency keys needed. **Correct.**
- **Side effects**: None in either implementation. **Parity confirmed.**

## Issues found and fixed

1. **ResponsibleUser name not populated in presenter** — The `TransactionDetailPresenter` was creating the `AccountUser` sub-resource with only `ID` and `Object`, but not setting the `Name` field. The proto has `ResponsibleUserName` available. The Dashboard returns the user's name in the `responsibleUser` field. Fixed by setting `Name: d.ResponsibleUserName` in the presenter.

## Minor accepted differences (by design)

- **Allocations are expandable** in Go (`?include=allocations`) vs always included in Dashboard. This follows the Go API's include pattern consistently.
- **Transaction sub-resource in allocations** only has `{id, object}` in Go vs Dashboard's `lightTransaction` which includes number, type, method, etc. Since allocations are nested within the parent transaction, this extra data is redundant. The Go API consistently uses minimal sub-resource references.
- **Customer sub-resource**: Dashboard's `BaseCustomer` includes `billToAddress`. Go's `LightCustomer` has `{id, object, name, number}`. The Go API uses lightweight customer references consistently; full customer data can be expanded via includes if needed.
