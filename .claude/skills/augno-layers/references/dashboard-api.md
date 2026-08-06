# Dashboard API layer map (`dashboard/apps/api`)

Express 4 + TypeScript (ESM) on Bun; Prisma over MySQL/PlanetScale; Zod
validation. Path alias `#/*` → `./src/*`. Shared workspace packages:
`@augno/db` (prisma singleton), `@augno/dtos` (endpoint specs + Zod
schemas), `@augno/adapters`, `@augno/errors` (HttpError, Logger),
`@augno/objects`. Stated rules: `dashboard/AGENTS.md:59-82`
("Controllers → Services → Repositories → Prisma").

**Status**: legacy V1 front door during the V1→V2 cutover. It shares the
same MySQL database as the Go API but never calls it — no HTTP/gRPC client
exists between them. Endpoints migrate to the Go api-gateway and are removed
here (`src/index.ts` is full of "migrated to the Go api-gateway" markers;
AGENTS.md:110-112 states the plan to eliminate this app). Don't add new
capability here if the Go API can own it.

## Layers in `src/`

| layer | location | notes |
|---|---|---|
| composition root | `src/index.ts` | builds app, middleware order, registers every endpoint via `regEP(app, api.spec, Ctrl.method)` (`index.ts:107-544`); boot checks in `start()` |
| route wrapper | `src/utils/register-endpoint.ts` | maps HTTP method, picks `authWrapper` vs plain by the spec's `AuthOptions` (`:8-40`) |
| middleware | `src/middleware/` | `request-log-handler`, `auth-handler` + `auth-wrapper`, `subscription-guard`, `async-wrapper`, `error-handler` |
| controllers | `src/controllers/*.ctrl.ts` (~47) | static classes; validate shape, instantiate service with `req.identity`, shape response |
| services | `src/services/*.svc.ts` (~50) | extend `BaseSvc` (holds `identity`); business logic, permissions, transactions |
| mediators | `src/mediators/*.med.ts` (rare) | cross-repo steps shared by services (`receiving-order-line.med.ts`) |
| repositories | `src/repositories/*.repo.ts` (~100) | extend `BaseRepo`; ALL Prisma access; adapters map rows → domain objects |
| repo interfaces | `*.repo.interface.ts` (~30) | input DTO types + contracts — typing aids, not real substitution points (services import concrete classes) |
| integrations | `src/integrations/` | S3/SES, Stripe, Shippo, Stedi |
| DTOs/schemas | `@augno/dtos` | imported as `api`; each endpoint spec carries requestSchema/responseSchema |

`BaseRepo.getDb(context?)` returns `context?.tx || prisma`
(`base.repo.ts:17-19`) — the single seam for transaction-vs-connection.

## Concern placement, with evidence

- **Authentication** — `src/middleware/auth-handler.ts`: Bearer token or
  `access` cookie → `AuthTokenUtils.decode` (`:52-54`) → load user (`:65`)
  → build `req.identity`; 401 on failure. Applied per-endpoint by spec
  (`AuthOptions.NormalAuthentication` → `authWrapper`,
  `register-endpoint.ts:33-34`). No global auth middleware.
- **Identity & tenant resolution** — also `auth-handler.ts`: flattens role
  permissions into a `Set<string>` like `salesOrders:update` (`:252-261`),
  resolves customer/supplier access via `accountRelation` (`:170-235`),
  403s locked/removed accounts (`:149-167`), reads
  `account_billing.subscriptionStatus` (written by the Go billing-service)
  to gate requests (`:122-147`).
- **Authorization** — enforced in the SERVICE layer:
  `checkIsInternalActor` / `checkHasPermission(domain, action)` from
  `src/utils/permission-check.ts` (throws `HttpError.forbidden`), called at
  the top of service methods (`order.svc.ts:47-48, 62-63, 93-94`). Tenant
  scoping: services thread `ownerAccountID: this.identity.targetAccountID`
  into every repo call (`order.svc.ts:53, 68-69`).
- **Shape validation** — controllers, first thing:
  `RequestValidator.validate(req, {body, params, query})` with `@augno/dtos`
  Zod schemas (`order.ctrl.ts:13-16`); ZodError →
  `HttpError.badRequest(flatten)` (`validators.ts:178-211`).
- **Business validation** — services (state transitions, existence,
  payment checks — `order.svc.ts:135, 276-288`); multi-repo rules in
  mediators (`receiving-order-line.med.ts:36,44`).
- **Idempotency** — NOT implemented; plumbing dormant
  (`request-log-handler.ts:153,181` — header read commented out).
- **Transactions** — services open `prisma.$transaction(async tx => …)` and
  pass `context: {tx}` to repos (`order.svc.ts:75-83, 102-137`). Repos
  never open transactions. Known exception to flag in review: an inline
  `tx.order.update` inside a service (`order.svc.ts:236`) bypasses the repo.
- **Mutation/ORM** — repositories only, via `getDb()` + adapters
  (`order.repo.ts:17`); raw SQL confined to 4 repo files
  (`batch`, `sys-property`, `customer`, `analytics`).
- **Pagination** — repo-layer contracts (`base.repo.interface.ts:6-15`):
  offset (`take`/`skip`) for most lists; keyset cursors for high-volume
  lists via `src/utils/cursor.ts` (opaque base64url, `(createdAt,id)`
  keyset, `limit+1`) — deliberately mirrors the Go
  `shared/pagination/cursor.go` (`cursor.ts:8-11`).
- **Errors** — throw `HttpError` (`@augno/errors`) anywhere; wrappers
  funnel to `next(error)`; terminal `src/middleware/error-handler.ts` maps
  `HttpError` → status+message, everything else → 500, logs, and alerts on
  500s (`error-handler.ts:40-51`).
- **Logging/request IDs** — `request-log-handler.ts` registered first:
  typed `requestID`, response-header echo, monkey-patched
  `res.send/json/end` to capture status/latency/bodies (size-capped),
  batch-persisted via `RequestLogRepo` (5s/100 records). Process-level
  uncaught handlers log with active-request context.

## Dependency direction

`index.ts` → controllers → services → (mediators) → repositories →
`@augno/db`. Controllers never import repos or `@augno/db`. Constructor
injection of `Identity` only; collaborators are directly `new`'d (no IoC).
**No mechanical enforcement** — no boundary lint rules, no architecture
test; discipline is convention (AGENTS.md:59-66) + review.

## End-to-end: `PUT` sales order

1. `index.ts:514` — `regEP(app, api.updateSalesOrder, OrderCtrl.updateOrder)`.
2. `register-endpoint.ts:14,33-34` — `app.put(...)` wrapped in `authWrapper`.
3. Per request: `requestLogHandler` → `authHandler` (builds `req.identity`)
   → `checkSubscriptionStatus` → handler.
4. `OrderCtrl.updateOrder` (`order.ctrl.ts:9`) — Zod validate →
   `new OrderSvc(req.identity).update({data, id})`.
5. `OrderSvc.update` (`order.svc.ts:40`) — `checkIsInternalActor` +
   `checkHasPermission(salesOrders,'update')` → repo call with
   `ownerAccountID`.
6. `OrderRepo.update` → `getDb()` → prisma + adapter mapping.
7. Controller responds 200 JSON; errors short-circuit to `error-handler.ts`.
