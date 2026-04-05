# POST /v1/core/account-prices — Verification Result

**Status: PARITY CONFIRMED**

## What Was Compared

| Aspect | Dashboard | Go | Match |
|--------|-----------|-----|-------|
| Permission domain | `discounts` | `PermissionDomainDiscounts` | Yes |
| Permission action | `create` | `ActionCreate` | Yes |
| Actor check | `checkIsInternalActor` | `CheckIsInternalActor` | Yes |
| Target account required | Yes (via `identity.targetAccountID`) | Yes (`CheckTargetAccountSet`) | Yes |
| DB: insert rate | Via `RateAdapter.createInput` | Direct SQL insert into `rate` | Yes |
| DB: insert account_price | Prisma create with owner/recipient/productLine/unitValue | SQL insert with same columns | Yes |
| DB: category associations | `createMany` with generated IDs | Loop insert with generated IDs | Yes |
| DB: attribute associations | `createMany` with generated IDs | Loop insert with generated IDs | Yes |
| Response status | 201 Created | 201 Created | Yes |
| Side effects | None | None | Yes |
| Uniqueness checks | None (DB constraints only) | None (DB constraints only) | Yes |

## Deliberate Go Improvements (Not Discrepancies)

- **Server-side ID generation**: Dashboard accepts client-supplied `id`; Go generates IDs server-side (more secure)
- **Flat request shape**: Dashboard accepts nested objects; Go accepts flat IDs (consistent with Go API conventions)
- **Idempotency**: Go adds idempotency key support for POST (required by architecture patterns)
- **Tracing**: Go adds OpenTelemetry span tracing
- **Transaction wrapping**: Go wraps all inserts in an explicit transaction

## Issues Found

None. The Go implementation faithfully reproduces all Dashboard business logic.
