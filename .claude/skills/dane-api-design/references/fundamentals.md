# HTTP Fundamentals

> Source: https://www.danealbaugh.com/articles/api-fundamentals

HTTP semantics, not implementation. HTTP is a stateless protocol: a client
sends a request targeting a **resource**; the server returns a
**representation** of that resource's state (the client never sees the
underlying state directly). Statelessness — every message understandable on
its own — is what enables proxies, caching, and load balancing.

## REST

REST builds predictable, discoverable APIs on HTTP by:
- Treating everything as resources manipulated through a uniform interface.
- Modeling **resources (nouns), not actions (verbs)**.
- Consistent patterns across all endpoints (`/users`, `/products`), and
  consistent response formats, error handling, and pagination.

## Methods

| Method | Idempotent | Safe | Action | Primary use |
|--------|-----------|------|--------|-------------|
| GET | Yes | Yes | Read | Retrieve resources |
| HEAD | Yes | Yes | Read | Headers only |
| POST | **No** | No | Create | Create resources / non-idempotent actions |
| PUT | Yes | No | Replace | Replace ENTIRE resource state; create-or-update |
| PATCH | **No** | No | Update | Partial modification |
| DELETE | Yes | No | Delete | Remove resources |
| OPTIONS | — | Yes | Query | Capabilities / CORS preflight |

- **Safe** = read-only semantics; client expects no state change beyond
  logging/analytics (GET, HEAD, OPTIONS, TRACE).
- **Idempotent** = same request N times has the same intended effect as
  once (PUT, DELETE, and all safe methods).
- **Cacheable**: typically only GET and HEAD; POST/PATCH only with explicit
  freshness info + Content-Location.

Method notes:
- **GET** — no request body; params in URI/query/headers. Never put
  sensitive data in a URL. 200 / 400 / 404. Supports `Range`.
- **POST** — processes per the resource's own semantics. On create: **201
  with a `Location` header**; also 200, 202 (async), 400, 409. Not
  idempotent — support `Idempotency-Key` for safe retries.
- **PUT** — create or replace the WHOLE representation, never partial.
  201 (created) / 200 / 409 / 415. Invalidates caches for the URI.
- **PATCH** — partial updates. Not idempotent by definition — use
  conditional requests (`If-Match` + ETag) to prevent races. 200 / 400 /
  409 / 412 / 415 / 422. Advertise formats via `Accept-Patch` in OPTIONS.
- **DELETE** — idempotent. 200 (with representation) / 202 / 204.
- **TRACE/CONNECT** — diagnostics/tunneling; restrict or disable in
  production.

## Status codes

| Code | Meaning | Notes |
|------|---------|-------|
| 200 | OK | Successful synchronous request; PUT updating existing |
| 201 | Created | Include `Location` header |
| 202 | Accepted | Accepted for ASYNC processing |
| 204 | No Content | Success, empty body (typical DELETE) |
| 206 | Partial Content | Range / paginated responses |
| 400 | Bad Request | Malformed request |
| 401 | Unauthorized | Not authenticated |
| 403 | Forbidden | Authenticated but not allowed |
| 404 | Not Found | |
| 409 | Conflict | e.g. duplicate create, in-flight idempotent replay |
| 410 | Gone | No longer available (e.g. replayed DELETE) |
| 412 | Precondition Failed | `If-Match` ETag mismatch |
| 422 | Unprocessable Entity | Understood but semantically invalid params |
| 429 | Too Many Requests | Rate limited — send `Retry-After` |
| 500 | Internal Server Error | |

## URIs

`protocol://host:port/path?query-parameters`

- **Consistency** across the whole API; **lowercase** everywhere (URIs
  normalize to lowercase; avoid confusion and redirects).
- URIs are public — they land in logs and history. **Never** API keys,
  passwords, or personal data.
- Hierarchical, self-documenting paths: `/users/123/posts` = posts of user
  123. Collections are **plural nouns**; identifiers are ids, UUIDs, or
  slugs.
- Query params for filtering/sorting/paging: `?page=2&limit=10`,
  `?status=active`.

## Headers

- **Content-Type / Accept** — media types; APIs default to
  `application/json`.
- **Location** — URI of the created resource (201) or redirect target (3xx).
- **Idempotency-Key** — makes POST retry-safe (UUID).
- **Cache-Control** — `no-store` (safest for dynamic/sensitive), `no-cache`
  (revalidate), `public`/`private`, `max-age=<s>`.
- **Authorization** — Bearer tokens etc.; only ever over HTTPS.
- **ETag** — version identifier for a representation; strong (`"abc123"`) or
  weak (`W/"abc123"`). Include in all responses; clients must treat it as
  opaque — make its format distinct from resource ids.
- **If-None-Match** — conditional GET; 304 Not Modified when unchanged.
- **If-Match** — conditional PUT/PATCH; 412 when the resource changed since
  the client last read it (optimistic concurrency).
- **Retry-After** — with 429/503; prefer seconds over a date.
- **RateLimit-Limit / RateLimit-Remaining / RateLimit-Reset** —
  standardized rate-limit reporting; `Reset` in seconds, not a timestamp.
- **Request-Id** — unique per request, in every response, for tracing.
- **Range / Content-Range / Next-Range** — partial content AND list
  pagination (`Range: id ..; max=10;` → 206 + `Next-Range`).
- **User-Agent** — identify the client; useful for debugging/analytics.
- The `X-` prefix convention is **deprecated** — don't mint `X-` headers.

## Body

The body carries the representation. GET has none; POST/PUT/PATCH commonly
do. Presence/format indicated by `Content-Type`/`Content-Length`. Clients
may negotiate different media types of the same resource.
