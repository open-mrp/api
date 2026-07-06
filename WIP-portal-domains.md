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



## Key references

- Plan: `~/.claude/plans/i-want-to-allow-sprightly-tiger.md`
- Provider seam: `services/core-service/internal/domain/clients.go` (`PortalDomainProvider`) — swap point if serving ever moves off Vercel.
- WS tickets: `services/api-gateway/internal/ws/ticket.go` (+ `ticket_handler.go`), frontend `dashboard/apps/frontend/src/app/_lib/ws/ws-auth.ts`.
- Host routing: `dashboard/apps/frontend/src/proxy.ts` (Next 16 — middleware lives here, not middleware.ts).

