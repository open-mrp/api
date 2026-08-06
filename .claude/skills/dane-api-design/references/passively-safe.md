# Designing a Passively Safe API

> Source: https://www.danealbaugh.com/articles/passively-safe-apis

**Passively safe**: architected so that crashes, timeouts, retries, and
partial outages *cannot* produce duplicate work, unexpected side effects, or
unrecoverable state. After any failure the system either completes the
workflow exactly once or lands in a terminal, explicitly visible state that
won't double-bill or duplicate work. (Inspiration: crumple zones, seismic
design, gravity-driven reactor cooling — resilience by construction, not by
operator vigilance.)

## The five problems

1. **External API calls can't be in transactions** — if the server dies
   after the foreign call but before commit, the foreign mutation is
   orphaned and unrecoverable.
2. **Requests aren't retry-safe** — after an error the client can't know
   whether retrying double-bills or duplicates records.
3. **External outages become your downtime** — synchronous foreign
   dependencies cascade.
4. **Synchronous processing is slow** — end-to-end sync workflows run
   2–30s+, inviting timeouts and disconnects.
5. **Message delivery is unreliable** — a broker adds two new failure
   modes: messages never enqueued (error after commit) and messages
   delivered twice (duplicate side effects).

## Technique 1 — Message outbox (transactionally staged jobs)

Insert messages into a `message_outbox` table **inside the business
transaction**. A background *enqueuer* drains the table: publish to broker,
delete the row only after broker ack.

- Rollback ⇒ no orphaned message; commit ⇒ message guaranteed.
- Gives **at-least-once publish** (not at-most-once processing).
- User gets a fast response; downstream work happens async.

## Technique 2 — Message inbox (consumer de-dup)

Consumers insert `(message_id, status='received')` under a **unique
constraint** before doing the work.

Schema: `message_id` (unique), `status` (`received`|`processed`),
`failed_at`, plus lease fields (`processing_started_at`,
`processing_owner`).

Workflow: insert → on conflict it's a duplicate, check status → else do the
work → set `status='processed'` (or `failed_at`) **before** acking the
broker.

Duplicate handling:
- `processed` → ack and drop.
- `failed_at` set → retry if safe, else dead-letter.
- still `received` (crash mid-work) → treat as in-flight until the lease
  expires, then retry.
- concurrent consumers → the lease/lock prevents double side effects.

Result: **at-most-once processing**. Outbox + inbox together ≈ effectively
exactly-once.

## Technique 3 — Idempotency keys

Let clients safely retry POST/PATCH. Client sends a UUID in the
`Idempotency-Key` header; server keeps an `idempotency_key` table:

- unique per **(route, method, user)**
- `status` (`received` / `in_progress` / `completed`)
- `recovery_point` — how far the request got
- `response` — cached response to replay
- `is_transient` — whether the stored error may be retried

Replay behavior:

| Situation | First attempt | Retry |
|---|---|---|
| POST success | `201 Created` | `200 OK`, same resource |
| Still processing | — | `409 Conflict` |
| Transient error | not cached | retry runs again |
| Deterministic error | cached | same error replayed |
| DELETE success | `204 No Content` | `410 Gone` |

Details:
- **Hash the request body** — same key + different body is rejected ("retry
  by mutation" is a bug factory).
- UUID-format keys keep the index efficient; validate the format (no
  smuggled data).
- Optionally send `Idempotent-Replay: true` on replayed responses.
- Stripe V2 retries failed requests when it cannot cause side effects
  instead of blindly replaying the old error — closer to user expectations.

## Atomic phases and recovery points

Definitions:
- **Foreign state mutation** — any mutation outside the local DB
  transaction (external API, another service).
- **Atomic phase** — the local mutations grouped in one transaction between
  foreign mutations.
- **Recovery point** — checkpoint committed with the phase, enabling safe
  resumption.
- **Final failure** — an error that can never succeed on retry (e.g.
  invalid address).

Grouping rules:
1. The idempotency-key upsert is its own atomic phase.
2. **Every foreign state mutation is its own phase.**
3. Everything between groups into additional phases.

Cardinal rule: **no network request inside a DB transaction — even a
read-only one** (e.g. address validation is isolated in its own phase).

Example — POST /shipments:
1. Create idempotency key → `started`
2. Validate address (foreign) → `address_validated`
3. Generate labels/tracking (foreign) → `tracking_generated`
4. Update shipment + invoice + order, enqueue update event (one tx) →
   `update_event_sent`
5. Enqueue notifications to outbox → `completed`
6. Store final response on the idempotency key

Every phase commits fully or rolls back; retries resume from the last
recovery point instead of redoing work. Notification-service downtime can't
fail the request — the outbox delivers late but correctly.

## Error classification: `is_transient`

Put an explicit boolean `is_transient` in the error envelope. The
idempotency layer caches and replays non-transient errors; transient errors
aren't cached so the retry re-executes. Decouples retry semantics from HTTP
status codes; the server classifies per error.

## Retry scheduling

- Naive retries → thundering herd.
- Decide **whether** via server signals (e.g. `Stripe-Should-Retry`);
  decide **when** via `Retry-After` when present, else **capped exponential
  backoff with randomized jitter** (AWS guidance).
- `RateLimit-Reset`: seconds, never a timestamp.

## Background processes

- **Completer** — re-drives abandoned non-terminal keys (stale
  `updated_at`, under max attempts) through the same idempotency machinery;
  rate-limited with backoff; stops on deterministic errors; quarantines
  keys stuck past a time/attempt threshold.
- **Reaper** — deletes only *terminal* keys older than the retention
  window (~30 days, Stripe's window); non-terminal ones get one last
  completion attempt or quarantine first. Delete in small batches (or drop
  time partitions). Keep a compact request ledger (key hash, route,
  timestamps, outcome) after discarding the heavyweight replay payload.

## The complete pattern

Outbox (guaranteed publish) + inbox (de-duped processing) + idempotency
keys (client retry safety) + atomic phases with recovery points (resumable
lifecycle) + completer + reaper = failures end in exactly-once completion
or an explicit terminal state.
