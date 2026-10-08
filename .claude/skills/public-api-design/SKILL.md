---
name: public-api-design
description: >-
  Stable public API shape rules (forge.1+): expose less by default, status active|archived
  settable on create/update, delete blocked while referenced (409 resource_in_use) and
  returning {id, object, deleted}, limit default 25 max 100, documented per-endpoint q
  search columns and match mode, docstring style. Use when designing or reviewing an
  endpoint's public shape, a list's search/filter/pagination, a delete, or API docstrings.
---

# Public API design

Human spec: `docs/patterns/public-api-design-conventions.md` (wins on conflict). Per-resource decisions: `docs/forge1-api-review.md`. Build mechanics: `api-resources`. Indexing: `performant-lists`.

Goal: a contract supported unbroken for years; every endpoint <25 ms, worst case <50 ms.

## Expose less by default

Adding later is non-breaking; removing is breaking forever. Related objects are expandable (`null` unless included) even when cheap. No speculative fields. Names, enum values, error codes, and route namespaces are permanent; settle them in review.

- **Request mirrors response**: same names and nesting both ways; related objects go in as `<field>_id`; PATCH merges section fields. Own-data sections are always populated, no `object`.
- **Inherited defaults** live in `defaults` at every level with the same names; `null` = inherit.
- **Fixed platform enums are fields**, not resources (no list endpoint/ID/owner); reference by code. A later resource keeps `code` as its identifier.
- **Enums, not booleans** in responses (`service_level_scope: all | selected`, not `all_service_levels: bool`). An enum can grow; a boolean needs a new field. Only exceptions: `has_next_page`/`has_previous_page`, delete stub `deleted: true`, error `is_transient`.
- **Dates**: instants are `_at` (RFC 3339 UTC); business days are `_on` (`YYYY-MM-DD`, account timezone). Lead times are time Quantities in business days (no `_days` ints). Per-run effort (`setup_time`) is a Quantity; per-unit effort is a Rate.
- **Collisions**: `lot` = traceability lot only; sizes are `lot_size`. `job` = work for one item (with `demand[]`); `work_order` = a release of jobs; `batch` = WIP scan unit (merges/splits). Pegging is the plan; traceability is lot/batch genealogy. Movement `type` = business event (`receipt | shipment | consumption | output | transfer | adjustment | reconciliation | return`), never the channel.
- **Contract limits**: totals 2 dp, unit prices/quantities ≤ 6 dp; prices tax-exclusive; one currency per document (`currency` field); metadata ≤ 50 keys / 40-char keys / 500-char values on documented resources only; Idempotency-Key on POST (≤ 255 chars, 24 h, mismatch 422 `idempotency_key_reused`, in flight 409); IDs opaque ≤ 64 chars.
- **Errors**: `{type, code, message, param, is_transient, errors: [{param, code, message}], hint, doc_url}`; every public code has a hint and a docs page; bulk rows and jobs use the same object.
- **Agents are callers**: public endpoints become MCP tools and docs pages automatically, so docstrings must let an agent act unaided. API keys carry `client` (shown in `actor`) and optional `expires_at`.
- **Lifecycles**: documents start `draft` and can end `canceled` via `actions/cancel` (closes the remainder, keeps history); never mark fulfilled/completed to stop. Statuses that follow child documents (fulfilled, completed) are derived, never actions.
- **Contacts** are account users of the customer/supplier account; adding one grants no access and sends no email (explicit `actions/invite`, explicit recipient lists).
- **Money is `{amount: "decimal string", currency: "usd"}`** (ISO 4217, lowercase), never a currency-unit `Quantity`. Prices are `{amount, currency, per_unit}`.
- **Defaults are a field on the parent** (`default_service_level_id`), never `is_default` on children.
- **Discount** = `{type: percent_off, percent_off: "5"}` | `{type: amount_off, amount_off: Money}`; percents are whole numbers, never fractions.
- **Two reference patterns only**: single-resource field = expandable, `null` unless included; document links = `related` stubs; a resource *about* one record of any type (note, subscription, tool call) holds that stub in a single `record` field. No `{id,name}` stubs, no screen-specific shapes.
- **Related records** = lightweight `record` stubs in `related`, not full sub-objects.
- **Workflow status** is read-only; transitions are `POST …/actions/{verb}`. US English names.
- **Who vs what**: `actor` (user | api_key | agent | device | system; plus `operator` at devices) says who; `type` says what happened, never who.
- **Value shapes** (`Money {amount, currency}`, `Quantity {value, unit}`, `Rate {value, unit, per_unit}`, `UnitPrice {amount, currency, per_unit}`) have no `id`, `object`, timestamps or `display_value`. `unit`/`per_unit` are always populated. Requests use `unit_id`/`per_unit_id`. Written only through the parent. Store money as columns on the owning row.

## Status, archive, soft delete

- Archive ≠ soft delete. Archived records stay visible and valid; only new assignments are blocked.
- No soft delete (`deleted_at`) by default. Delete = hard delete guarded by 409 `resource_in_use`.
- Exception: a deleted item that must keep its place for others (a chat message) stays visible as `status: deleted` with content `null` and `deleted_at`. Visible state, not hidden rows.
- Add `status` only when long-lived records reference the resource and users routinely retire it. Otherwise just block delete.

- `status: active | archived`. Never `inactive` or `is_active`. Exception: resources paused and resumed routinely (integrations, agents, account users) use `active | disabled`.
- Settable on create (`field.Optional`, default `active`) and update.
- Archived: existing references stay; newly assigning it fails with 422.
- List filter `statuses[]`; omitted = all.

## Notes

- No `notes` fields. Workflow guidance on documents is `instructions` (only where a review decides). Commentary (and agent memory) is the `note` resource (`/v1/core/notes`, attributed via `actor`, filtered by `record_ids[]`, never embedded).

## Delete

- Referenced elsewhere → 409 `resource_in_use`, message points to archiving. Reference checks are indexed `EXISTS` probes.
- 200 `{ "id", "object", "deleted": true }`. Never `{}` or 204.
- Already deleted → 410. System-owned → not updatable or deletable.

## Actions

- `POST .../actions/{verb}`: collection-level for bulk, resource-level for single-record operations.
- Work that can ever exceed the latency budget always returns 202 + job, never sometimes.
- Ambiguous intent → required `mode` enum (`convert | relabel`), never a guess.

## Routes and documents

- Every endpoint is `/v1/{namespace}/{resource}`; only `/healthz` is exempt. Cross-domain resources live in `core`.
- Documents (orders, invoices, shipments) snapshot printed master data (addresses) at creation; edits to the source never rewrite issued documents.

## Versioned definitions

- Recipe-like definitions (manufacturing methods) are versioned: draft → `actions/activate`; active/archived immutable; running work keeps a copy of its version.

## Accounting

- Two-party documents (orders, invoices, credit notes, payments) are ONE row with both parties + owner; each side reads it through its own collection (sales/purchase order, invoice/bill). Per-party data (metadata, instructions) in a party table. Never mirror rows.

- Invoices and credit notes are immutable once finalized; never deleted. Corrections = new documents or status changes. Payments/refunds are reversed, never deleted.

## Lists

- `limit` default 25, max 100. Keyset `(created_at, id)` newest first unless decided otherwise.
- `q` only when needed. Its docstring names the exact columns and match mode. It covers own-table columns only. Related-resource narrowing uses `*_ids[]`.
- Match mode: small per-tenant tables use `LIKE '%q%'` with tenant scope. Codes/SKUs use prefix `LIKE 'q%'` on an index. Large free text uses ngram FULLTEXT. Never use the default word FULLTEXT parser for name search; drop unused FULLTEXT indexes.
- Embedded expandable lists return the first 10; `page_info.next_page_url` points at the sub-resource list. Never cap silently.
- Created-date range: `created_after` / `created_before` (never `starts_at`/`ends_at`, which are for analytics periods). Natural-key lookups get an exact filter (`skus[]`), not `q`.
- Multi-value filters are plural arrays (`statuses[]`, `customer_ids[]`).

## Docstrings (become the API reference)

- `summary ¶ description`. Summary: one sentence, verb-first ("Returns a list of…", "Creates a…", "Retrieves a…", "Updates a…", "Deletes a…").
- List: what's included plus sort order. Update: only sent fields change. Delete: in-use rule plus archive pointer.
- Fields: caller-facing meaning, precise uniqueness (say "case-insensitive"). No table names or internal consumers.

## Release gate

Finalizing a resource in the forge.1 review clears `Preview: true` on its endpoints.
