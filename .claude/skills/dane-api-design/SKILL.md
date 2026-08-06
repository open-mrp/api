---
name: dane-api-design
description: Dane Albaugh's HTTP API Design doctrine — the house style for all API work. Use when designing, implementing, or reviewing API endpoints, resources, layers (controller/service/repository), error handling, idempotency, or anything in api/. Distilled from danealbaugh.com's API series.
---

# Dane Albaugh's HTTP API Design

The authority for how APIs are built here — this repo is where the doctrine
comes from. This file holds the rules to apply; the `references/` files hold
the full distilled articles — read the relevant one before making a
non-obvious design decision.

Source series: https://www.danealbaugh.com/articles/api-series-intro

## The four principles (references/principles.md)

1. **Make no assumptions** — networks fail, "can't-fail" operations fail,
   users do unexpected things, and every observable behavior becomes someone's
   dependency. Design for the failure case first.
2. **As simple as possible** — fewest concepts that solve the problem. Done
   when there is nothing left to take away.
3. **Separate concerns** — single-responsibility layers behind clean
   interfaces; consumers depend on interfaces, never implementations.
4. **APIs are for humans** — first designs are rarely optimal; watch real
   usage, hunt friction, iterate.

## The four layers (references/anatomy.md)

Data flows strictly downward; each layer imports only what is beneath it.

| layer | owns | never |
|---|---|---|
| routing | map method+path → controller | any processing or logic |
| controller | parse request, validate **shape** (DTO), call service, build HTTP response | business rules, DB access |
| service | business rules + **business validation**, workflows, domain objects | HTTP types, SQL/driver types |
| repository | all data access behind an interface | leaking driver types upward |

- Request-shape validation lives in the **controller**; business-rule
  validation lives in the **service**. Never blur these.
- When a resource-oriented service bloats, split into use-case ("mediator")
  services (CreateOrder, CancelOrder) rather than growing a god service.
- Here the routing+controller layers are largely absorbed by the
  declarative endpoint framework (`services/api-gateway/endpoints/`); layer
  adherence is enforced by `services/structure_adherence_test.go`.

## HTTP semantics (references/fundamentals.md)

- Model **resources (nouns), not actions (verbs)**; collections are plural
  (`/users`, `/posts`); consistent patterns everywhere. Lowercase URIs; never
  put secrets or personal data in a URI.
- Respect method semantics: GET/HEAD safe+idempotent; PUT full-replace,
  idempotent; PATCH partial, NOT idempotent (guard with `If-Match`/ETag);
  DELETE idempotent; POST neither (offer `Idempotency-Key`).
- Status codes: 201 + `Location` on create; 202 for async accept; 204 for
  bodiless success; 409 conflict; 412 precondition failed; 422 understood but
  invalid; 429 + `Retry-After`; 410 for gone (incl. replayed DELETE).
- Headers worth reaching for: `ETag`/`If-None-Match`/`If-Match`,
  `Request-Id` on every response, `RateLimit-*`, `Retry-After` (seconds, not
  a timestamp), `Cache-Control: no-store` for anything sensitive. Never mint
  `X-` headers.

## Passive safety (references/passively-safe.md)

A passively safe endpoint either completes its workflow exactly once or lands
in an explicit, visible terminal state — no duplicates, no double-billing, no
orphaned foreign state — regardless of crashes, retries, or outages.

- **Never make a network call inside a DB transaction** — even a read-only
  one.
- **Outbox**: stage messages in a `message_outbox` row inside the business
  transaction; a background enqueuer drains it → at-least-once publish.
- **Inbox**: consumers insert `message_id` under a unique constraint before
  doing work → at-most-once processing.
- **Idempotency keys** for POST/PATCH: unique per (route, method, user);
  cache the response; replay it on retry (first POST → 201, replay → 200;
  in-flight → 409; DELETE replay → 410). Hash the body — same key with a
  different body is rejected.
- **Atomic phases**: split a request at every foreign state mutation; each
  phase commits a `recovery_point` so retries resume, never redo.
- Mark errors with an explicit `is_transient` flag: cache and replay
  deterministic failures; let transient ones retry.
- Retries: capped exponential backoff **with jitter**; honor `Retry-After`.
- Background hygiene: a **completer** re-drives abandoned keys, a **reaper**
  deletes terminal keys past the retention window (~30 days).

## Declarative endpoints (references/declarative-endpoints.md)

This repo's own registry pattern — endpoints as struct literals, one generic
`Execute` handling bind/validate/respond for every endpoint
(`services/api-gateway/endpoints/`, `shared/field` for `field.Optional` /
`field.Clearable`). New endpoints MUST go through the framework: declare a
`Materialize()` endpoint + request/response types with tags; never hand-roll
a handler. Cross-cutting concerns belong in `Execute`, once.

## Review checklist

Before calling API work done:
- [ ] Layer imports point strictly downward; no HTTP in services, no SQL
      outside repositories.
- [ ] Shape validation in controller, business validation in service.
- [ ] Method + status codes match the semantics tables above.
- [ ] 201s carry `Location`; errors distinguish transient vs deterministic.
- [ ] No network call inside any transaction.
- [ ] Non-idempotent writes are retry-safe (idempotency key, outbox, or
      documented why not needed).
- [ ] The simple version — is there anything left to take away?
