# WIP: Custom Portal Domains — Remaining TODOs

Feature state: fully implemented and verified offline (unit tests, sqlc prepare smoke against real schema, apidocs/OpenAPI tests, frontend typecheck/lint/build all green) across `api/` and `dashboard/`. What remains is live verification, rollout, and a few deferred items.

## Blocked on externals

- [x] **Swap dashboard to typed SDK** — `dashboard/apps/frontend/src/app/_lib/api/client/portal-domain.api.ts` now calls `augnoClient.settings.portalDomains.{list,create,delete}` and `.actions.verify`, and re-exports `PortalDomain`/`DNSRecord` from the SDK. Local types deleted. `dns_records` is nullable in the SDK type, so `PortalDomainSection.tsx` guards it. Frontend `check-types` green.



## Live verification (before shipping)

- [x] **Deployed-API smoke (prod)** — after the API deploy: `GET https://api.augno.com/healthz` → 200; `GET /v1/settings/portal-hosts/{unknown}` with `Augno-Version` header → 404 `resource_not_found` (route live). **Bug found + fixed:** the endpoint 400s (`api_version_required`) without an `Augno-Version` header, and `proxy.ts:resolvePortalSlug` was fetching it headerless — so every custom host would resolve to null and rewrite to `/domain-not-configured`. Fixed by sending `Augno-Version: API_VERSION` on that fetch (`dashboard/apps/frontend/src/proxy.ts`). Frontend `check-types` green. **This must ship in the frontend deploy or custom domains stay dead at the proxy.**
- [x] **Stub-provider smoke → e2e tests** — the create+idempotency, double-verify (pending→verified), and unauthenticated host-resolution smoke checks are now permanent e2e coverage in `tests/e2e/api/cov_settings_portal-domains_test.go` (15 tests, all passing against the e2e stack; PLATFORM=test → stub provider). Covers: create shape (podn id, CNAME dns_record), idempotency replay, one-per-account 409, list, delete→404, verify lifecycle, resolve (unverified→404, verified→public_account slug), unknown-host 404, **missing-version→400 regression guard**, validation (no-dot/`@`/augno.com), get/verify not-found, no-auth. Cardinality-1 handled by serializing slot-consuming tests (non-parallel + account-clear + cleanup).
- [ ] **Frontend host smoke (manual)** — visit `shop.lvh.me:3000` (lvh.me resolves to 127.0.0.1 and is NOT a primary host) → proxy resolves the host, portal renders, login sets host-only cookies, `/v1/*` proxies same-origin.
- [ ] **Staging end-to-end with a real domain** — add domain in Account Settings → publish CNAME to `cname.vercel-dns.com` → verify action → Vercel issues TLS → portal renders on the domain → customer login works → password-reset email links to the custom domain → live notifications connect via WS ticket.
- [ ] **Confirm Vercel API versions/payloads** against current docs at first real use: `POST /v10/projects/{id}/domains`, `GET /v9/projects/{id}/domains/{domain}` + `POST .../verify`, `GET /v6/domains/{domain}/config` (implemented in `services/core-service/internal/infrastructure/vercel/vercel_domains_client.go`).



## Rollout (ordered)

1. [x] **core-service env (BEFORE deploying this code)**: `VERCEL_API_TOKEN`, `VERCEL_PROJECT_ID` (+ `VERCEL_TEAM_ID` if team-scoped) — config validation hard-requires the first two in production; startup fails without them. Wired via `secrets.tf` (`vercel_credentials`) → `core-service.yaml` env; values live in the `prod/api` Secrets Manager blob.
2. [x] **api-gateway env**: `WS_TICKET_SECRET` (strong random string, shared across replicas). Optional — without it, custom-domain portals work but get no live WS notifications (`/v1/ws/ticket` returns 501 and clients back off). Wired via `secrets.tf` (`ws_ticket_secret`) → `api-gateway.yaml` env (`optional: true`); value in the `prod/api` blob.
3. [ ] **Vercel project env**: `API_ORIGIN` (e.g. `https://api.augno.com`) and optionally `NEXT_PUBLIC_PRIMARY_HOSTS` (comma-separated extra first-party hosts).
4. [ ] Ship backend + settings UI first (inert until DNS points anywhere), frontend proxy/cookie changes second (backward-compatible for `*.augno.com`).



## Deferred / phase 2

- [ ] **Canonical redirect**: when an account has a verified custom domain, optionally 308 portal paths on `augno.com/[slug]/...` to the custom domain (needs a slug→domain hint in the branding lookup — `PublicAccountInfo.portal_domain` already carries it).
- [ ] **DNS drift reconcile job**: DB stays `verified` if a customer later breaks their DNS (Vercel stops routing regardless). A periodic reconcile that re-polls `GetDomainState` and flips status back to `pending` was explicitly scoped out (repo has no background-poller precedent for domains).
- [ ] **Server-rendered portal auth pages** (`(customer)/[accountSlug]/auth/`*) still emit slug-prefixed hrefs on custom domains; the proxy's 308 canonicalization covers the clicks. Converting them to use the base-path helper would remove the extra redirect hop.
- [ ] **Multiple domains per account**: schema supports relaxing (drop `portal_domain_account_uq`, keep an index; add `is_primary`); Create's cardinality check and the settings UI would need updating.



## Customer-portal → Go API migration (custom-domain support)

Custom portal domains proxy all same-origin `/v1|/v2` to the Go API, so every legacy Express `callApi` used in the customer portal 404s there. Migrating the whole portal data surface off `callApi` onto the Go SDK. Direction: the portal consumes **lightweight portal-specific types**, not the heavy shared domain objects — fetch only what the portal renders.

### Backend changes
- **Rate-shop origin → server-side.** `POST /v1/operations/shipments/actions/rate-shop`: `from_address` made optional (`field.Optional[AddressInput]`); when omitted, core resolves the seller's ship-from origin via `GetAccountOriginAddress`. Portal no longer sends the seller address. (`endpoint_rate_shop.go` + `service.go` + core `shipment_service.go`.)
- **NEW authenticated endpoint** `GET /v1/settings/portal-profiles/{slug}` → `PortalProfile {id,name,slug,logo_url,support_email,address}`. Serves the seller's letterhead address to logged-in portal pages; resolves the seller's own address server-side (works unauthenticated-caller-safe, no cross-account loader). New `core_portal.proto` + core RPC `GetPortalProfileBySlug` + gateway endpoint/resource/presenter + `portal_profile` object type.

### Frontend call migrations (legacy `callApi` → Go SDK)
| # | Frontend fn | Go route | SDK method | Notes |
|---|---|---|---|---|
| 1 | `fetchAccountBySlug` | `GET /v1/settings/branding/{slug}` | `settings.retrieveBranding` | public; fixes reported `/v1/accounts/slug/*` 404 |
| 2 | `fetchSellerProfile` | `GET /v1/settings/portal-profiles/{slug}` | `settings.retrievePortalProfiles` | NEW; authenticated; letterhead address |
| 3 | logo `<img>` (5 sites) | — | uses `logo_url` | presigned; dropped `/v1/accounts/{id}/logo` |
| 4 | `fetchRegistrationFlowBySlug` | `GET /v1/sales/registration-flows/by-slug/{slug}` | `sales.registrationFlows.retrieveBySlug` | |
| 5 | `fetchPortalProfile` | `GET /v1/sales/customers/{id}` | `sales.customers.retrieve` | lean includes → `PortalCustomer` |
| 6 | `fetchPortalFrequentlyOrderedProducts` | `GET /v1/sales/customers/{id}/frequently-ordered-products` | `sales.customers.retrieveFrequentlyOrderedProducts` | lightweight `PortalFrequentlyOrderedProduct` (itemID/sku/description); full product hydrated on add. Legacy `fetchFrequentlyOrderedProducts` kept for internal `(user)` dashboard |
| 7 | `fetchCustomerUsers` | `GET /v1/identity/account-users` | `identity.accountUsers.list` | notif-pref gap flagged |
| 8 | `createCheckoutSession` | `POST /v1/sales/checkout-sessions` | `sales.checkoutSessions` | dollars→cents ×100 verified |
| 9 | `fetchCustomerInventory` | `PUT /v1/core/analytics/inventory-receipts` | `core.analytics.updateInventoryReceipts` | lightweight refactor (was stubbed) |
| 10 | `fetchCatalogProducts` | `GET /v1/catalog/catalog/product-lines/{id}/products` | `catalog.catalog.productLines.retrieveProducts` | |
| 11 | `findDiscountByCode` | `POST /v1/sales/order-discounts/actions/find-by-code` | `sales.orderDiscounts.actions.findByCode` | |
| 12 | `validateUnits` | `PUT /v1/catalog/units/actions/validate` | `catalog.units.actions.validate` | |
| 13 | `rateShop` | `POST /v1/operations/shipments/actions/rate-shop` | `operations.shipments.actions.rateShop` | origin dropped (server-side) |
| 14 | `fetchCustomerOrder`/`fetchOrder` | `GET /v1/sales/sales-orders/{id}` | `sales.salesOrders.retrieve` | order detail via `mapSalesOrderToOrder` adapter (heavy Order kept so OrderUtils/checkout UI unchanged); letterhead account switched to `useSellerProfile` (portal-profiles). Parity note: per-line invoiced/picked *quantities* not in Go response → progress bars read 0% |
| 15 | `registerCustomer` | `POST /v1/sales/customers/registration` | `sales.customers.registration` | migrated to Go one-shot (custom domains now work). Session-based flow (below) built server-side |

### Session-based registration (net-new feature)
Full-stack backend + API + SDK **done and green**; frontend wizard is the last piece.
- **Prisma** `portal_registration_session` (scoped to `(user, seller)`; you generated the Go migration) → sqlc.
- **Constants**: `PortalRegistrationSessionIDPrefix` (`porgse_`), object types `portal_registration_session` + `..._data`, `PortalRegistrationStep` enum.
- **Core-service**: domain + repo + `PortalRegistrationSessionSvc` (create-or-resume w/ `(user,seller)` dedupe + 7-day logical TTL, ownership-checked get/update, forward-only steps, complete → delegates to existing `RegisterCustomer` via a `CustomerRegistrar` seam, abandon) + gRPC (5 RPCs, `core_portal.proto`) + run.go wiring.
- **Gateway**: `PortalRegistrationSession` resource, service, 5 endpoints (`POST /v1/sales/portal-registration-sessions`, `GET`/`PATCH /{id}`, `POST /{id}/actions/{complete,abandon}`), group + registration. `openapi` + Stainless + SDK regenerated → `sales.portalRegistrationSessions.{create,retrieve,update,actions.complete,actions.abandon}`.
- **E2E (done)**: `tests/e2e/api/crud_portal_registration_sessions_test.go` — 13 tests, all green against the e2e stack. Each test registers a fresh email-verified buyer (`newPortalBuyerClient`, no account) so `(buyer, seller)` session spaces are isolated and parallel-safe. Covers: create shape (`porgse_` id, step `customer_details`), resume dedup (same id), update persists step+data (GET reflects), forward-only step guard (400), unknown id (404), **ownership (403 not-yours vs 404 absent — deliberate split)**, abandon (not resumed after), complete-after-abandon (400), create validation (missing→400 / unknown slug→404), full new-customer completion journey (+ idempotent replay + no-update-after-complete), existing-customer completion by number (`SeedCustomerNumber`). Required creating the `portal_registration_session` table + rebuilding core-service/api-gateway e2e images.
- **Frontend (done)**: `_lib/api/client/registration-session.api.ts` wraps create-or-resume/update/complete/abandon. `CreateCustomerPageContent` now: create-or-resume on mount (React Query), restores saved fields + jumps to the furthest step on resume, persists each step on Next (non-blocking), and on submit does final update → complete (backend registers from the session). `ExistingCustomerRegistrationForm` routes existing-customer linking through the same session (`is_existing_customer=true` + `customer_number`). One-shot `registerCustomer` kept only as a no-session fallback. `check-types` + `lint` green. **SDK yalc-linked → `bun run sdk:unlink` before commit.**
| 16 | `updateUser` (profile) | — | deferred | read-only; no self-service `/v1/identity/me` yet |

## Key references

- Plan: `~/.claude/plans/i-want-to-allow-sprightly-tiger.md`
- Provider seam: `services/core-service/internal/domain/clients.go` (`PortalDomainProvider`) — swap point if serving ever moves off Vercel.
- WS tickets: `services/api-gateway/internal/ws/ticket.go` (+ `ticket_handler.go`), frontend `dashboard/apps/frontend/src/app/_lib/ws/ws-auth.ts`.
- Host routing: `dashboard/apps/frontend/src/proxy.ts` (Next 16 — middleware lives here, not middleware.ts).

