# Comment Conventions (Internal / Microservice Code)

How we write comments in non-public-facing Go code: services, mediators, event consumers and publishers, repositories, domain types, config structs, and other internal microservice plumbing.

This is **not** about API resources or API endpoints. Public-facing request/response structs and endpoint handlers are documented by the OpenAPI spec, and we deliberately do **not** restate schema facts (optional/nullable/type/enum-lists/defaults/expandable) in their doc comments — see `api-resource-conventions.md` and the OpenAPI generator. The rules below cover everything else: the code that *does the work*.

## The two rules that matter most

### 1. One paragraph stays on one physical line — never hard-wrap prose

Do **not** manually break a paragraph across multiple comment lines to fit some column width. Write the whole paragraph as a single line and let the editor soft-wrap it. Hard-wrapped comments are painful to edit (every word added reflows the whole block by hand) and produce noisy diffs where one reworded sentence touches five lines.

```go
// Good — one paragraph, one line. The editor wraps it for display.
// Populate the derived payment status for the whole page in one batched query (no per-order N+1), defaulting any order without payment activity to unpaid.
```

```go
// Bad — a single paragraph hand-wrapped into multiple lines.
// Populate the derived payment status for the whole page in one batched
// query (no per-order N+1), defaulting any order without payment activity
// to unpaid.
```

Distinct paragraphs still get their own lines (separated by a blank `//` line), and genuinely list-shaped content (numbered steps, TODO checklists) may use one line per item. The rule targets *prose paragraphs*: don't chop one thought into several lines.

### 2. Comments above a service/method explain business logic and side effects — not the mechanics

A doc comment on a service, a method, or a meaningful block should answer *why this exists and what it does to the world*, not narrate what the next line of code literally says. Prioritize, in roughly this order:

- **Business intent** — what business rule or use case this serves.
- **Side effects** — writes, outbox/event publishes, external API calls (Stripe, Shippo, HubSpot), emails, notifications.
- **Ordering & transactionality** — what must happen atomically, what runs in the same DB transaction, what is deliberately out-of-band.
- **Idempotency & replay** — how the code behaves on retry or message redelivery.
- **Failure modes & fallbacks** — what happens when a dependency is nil/unavailable, what the default value is.
- **Cross-system parity** — when behavior must match the Dashboard or another service, say so.

## Examples from this codebase

Type-level comment that states intent *and* the threading/side-effect model:

```go
// SalesOrderCreatedConsumer processes sales-order-created events and runs the out-of-band side effects that should not block the create response — currently dispatching CRM sync for accounts with a connected integration (e.g. HubSpot).
type SalesOrderCreatedConsumer struct { ... }
```

Method comment that captures the side effect, idempotency, and the contract callers must uphold:

```go
// dispatchIntegrations runs each connected third-party integration's reaction to a new sales order. Each integration is independent and idempotent on msg replay (the inbox guarantees at-most-once delivery to this handler; integrations should additionally use data.SalesOrderID as their upstream idempotency key).
func (c *SalesOrderCreatedConsumer) dispatchIntegrations(...) error { ... }
```

Comment that explains the *transactional reason* a thing is built the way it is:

```go
// outboxSalesOrderEventPublisher writes sales-order domain events to the outbox table instead of publishing directly to RabbitMQ, so the event commits atomically with the order in the same transaction.
type outboxSalesOrderEventPublisher struct{}
```

Inline comment justifying authorization logic and flagging the parity requirement:

```go
// Authorization (matches Dashboard): internal users need the create permission; customer users may self-create only for their own account; other actor types cannot create orders.
switch {
case identity.IsInternalUser():
    ...
```

## Config & dependency fields: document optionality and failure behavior

Optional dependencies are the high-risk part of a service — a nil one usually means a code path silently no-ops or panics at runtime. Document that on the field, on one line each:

```go
type SalesOrderSvcConfig struct {
    // Repos (required) is the repository factory.
    Repos domain.RepoFactory

    // SalesOrderPublisher (optional; default: nil) publishes sales-order domain events to the outbox. When nil, the sales-order-created event is skipped. It is not validated at construction.
    SalesOrderPublisher domain.SalesOrderEventPublisher

    // ShippoFactory (optional; default: nil) builds Shippo clients for live shipping-rate estimation on create. When nil, the synthesized shipping line falls back to rate 0 (after honoring all freight-exemption / flat-rate / minimum-order rules).
    ShippoFactory domain.ShippoClientFactory
}
```

Convention: lead with `(required)` or `(optional; default: <value>)`, then state the consequence of the default. If it's optional-but-unvalidated and a missing value panics at runtime rather than at construction, say so.

## TODOs

A `TODO` should be actionable: state what's missing and the concrete steps to finish it, so the next person doesn't have to reverse-engineer the plan. This is the one place a numbered list (one item per line) is preferred over a single paragraph.

```go
// TODO: implement the actual HubSpot sync once a HubSpot API client exists. The wiring below establishes the trigger point and the enabled-check; the remaining work is to (1) build a HubSpot client from the account's encrypted credentials, (2) re-fetch the full order via the sales-order repo, and (3) upsert the corresponding HubSpot deal/line items keyed on data.SalesOrderID for idempotency.
```

## What not to comment

- **Don't narrate obvious mechanics.** `// start a span` above `ctx, span := tracer.Start(...)` adds nothing.
- **Don't restate the signature.** `// GetSalesOrder gets a sales order` is noise; comment it only when there's intent, a side effect, or a gotcha to convey.
- **Don't duplicate schema facts on internal types** that mirror API resources — same reasoning as the public-facing rule.
- **Don't leave stale comments.** A wrong comment is worse than none; update it in the same change as the code.
