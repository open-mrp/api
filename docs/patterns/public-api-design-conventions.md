# Public API Design Conventions

The shape rules for the stable public API (forge.1 onward). These decide **what** an endpoint exposes. `api-resource-conventions.md` covers how to build it in Go, and `performant-list-endpoint-patterns.md` covers how to index it.

The public API is a contract we intend to support without breaking for many years. Every endpoint should answer in under 25 ms, and in under 50 ms in the worst case. When a design choice trades durability or latency against convenience, durability and latency win.

Decisions are recorded with the resource review that settled them. The per-resource change list is in `docs/forge1-api-review.md`.

## Principles

### Expose less by default

Adding a field, param, include, or enum value later is non-breaking. Removing or narrowing one is breaking forever. When in doubt, leave it out, or put it behind an include.

- Related objects are expandable (`null` unless `include[]=...`), even when cheap to compute. Example: `owner` on payment terms stays expandable although `owner.type` comes from the row itself.
- Don't add a field "in case". Add it when a caller needs it.
- Deferring a feature is fine when its real shape is not known yet. A half-modeled field (e.g. `days_until_due` on payment terms) is worse than no field.

### Request mirrors response

A field has one name and one place in both directions. The request writes a related object as `<field>_id` where the response returns the object (`defaults.carrier_id` ↔ `defaults.carrier`). Sections nest the same way in both, and PATCH merges a section's fields individually. Sections that are the resource's own data (`contact`, `defaults`) are always populated and carry no `object`; related objects inside them stay expandable.

### Inherited defaults

When a value cascades (customer → account group → account), each level holds it in a `defaults` section under the same name (`defaults.lead_time_days`). `null` at a level means "inherit from the next one". Don't keep inheritance steps nobody can set through the API.

### Fixed enums are fields, not resources

A value set the platform defines, and no account can extend, is an enum field. Its values and meanings go in the field docstring. There's no list endpoint, ID or `owner`, and references use the code (`priority: "high"`). Display labels are a client concern. If such a set later becomes a real resource, the resource keeps `code` as its stable identifier, so existing fields go on holding the string and nothing breaks. An expandable object can be added beside the field.

### Enums, not booleans

Response fields use an enum for any state or mode, never a boolean. A boolean can only ever mean two things. Adding a third case later means another field, and every combination of the two has to be documented forever. An enum just gains a value.

Example: shipping terms use `free_shipping.service_level_scope: all | selected`, not `all_service_levels: true | false`. A later `all_except` is one more value.

The only exceptions are protocol fields that can never gain a third value: `page_info.has_next_page` / `has_previous_page`, the delete stub's `deleted: true`, and the error object's `is_transient`.

### Dates, instants and durations

- An instant is a timestamp ending in `_at` (RFC 3339, UTC): `created_at`, `shipped_at`.
- A business day is a date ending in `_on` (`YYYY-MM-DD`, read in the account's timezone): `promised_on`, `ship_by_on`, `due_on`, `expires_on`. A day the customer was promised is never a timestamp.
- A lead time is a time `Quantity` in days, counted in business days on the account's working calendar (`defaults.lead_time`, `commitment.lead_time_override`, `material.lead_time`). There are no `_days` integers.
- Effort that doesn't scale with quantity (`setup_time`) is a time `Quantity`; effort per unit is a `Rate`.

### Names that collide

- `lot` is a traceability lot only. A quantity made or bought together is a `lot_size`.
- A `batch` is a production work-in-progress unit (batch scanning), not a lot. Batches merge and split; their genealogy is a graph.
- A `job` is the work to make one item (method copy, `demand[]` saying what it's for). A `work_order` releases many jobs for a period. Planning stages, cadences and queues are configuration and never change these shapes.
- Pegging (`demand[]`) is the plan. Traceability follows lots and batch genealogy, which record what physically happened.
- An inventory movement's `type` is the business event (`receipt`, `shipment`, `consumption`, `output`, `transfer`, `adjustment`, `reconciliation`, `return`), never the channel. The channel is the `actor` (`device` for a scan).

### Contract limits

- **Decimal scale:** document totals have 2 places, unit prices up to 6, Quantity values up to 6.
- **Tax:** `unit_price` and line `amount` are tax-exclusive. Line `amount = quantity × unit_price − discount`.
- **Currency:** one currency per document, in a document-level `currency`; each Money on it repeats that currency.
- **Metadata:** `map[string]string`, at most 50 keys, keys ≤ 40 characters, values ≤ 500. Only on customers, suppliers, items, sales orders, purchase orders, invoices, shipments and production runs. Typed fields, when they come, are `custom_fields`.
- **Idempotency:** any POST accepts `Idempotency-Key` (≤ 255 characters, kept 24 hours). Replays return the first response. The same key with a different body is 422 `idempotency_key_reused`; a request still in flight is 409.
- **IDs:** opaque strings ≤ 64 characters. Don't parse the prefix.
- **Rate limits:** the `RateLimit-Limit`, `RateLimit-Remaining`, `RateLimit-Reset` and `Retry-After` headers, and 429 `rate_limited`.
- **Deprecation:** a stable version is supported for at least 24 months after its successor ships. Responses on a deprecated version carry `Deprecation` and `Sunset` headers.

### Errors

`{ type, code, message, is_transient, errors: [{ param, code, message }], doc_url }`. A 422 lists every failing field in `errors`; other errors return it empty. There is no top-level `param`. `doc_url` is the stable docs page for `code`, and every public error code has one. Bulk rows and job results carry this same object, never a string.

### Agents are first-class callers

Every capability works through the public API, which also feeds the MCP server and the docs. A new endpoint is automatically a tool and a docs page, so its docstrings must be good enough for an agent to use it without help. API keys record which `client` uses them, and that shows in `actor`.

### Lifecycles end in `canceled`

A document that work happens against (sales order, purchase order, production run) can be `canceled` through `POST …/actions/cancel`. Canceling closes what's left and keeps history. Delete is only for untouched drafts. Never mark something `fulfilled` or `completed` to stop it. New documents start as `draft`.

A status that follows from child documents (`fulfilled` from shipments, `completed` from receipts) is derived, never set by an action. It moves back on its own when a child is undone.

### People are users; access is explicit

A customer's or supplier's contacts are account users of that account. Adding one never grants access or sends email. Access needs an explicit `actions/invite`, and email goes only to explicit recipient lists.

### Value shapes

Embedded values (`Money`, `Quantity`, `Rate`, `UnitPrice`) are part of their parent. They carry no `id`, `object`, timestamps or `display_value`, and they are written only through the parent's create or update. Exposing the storage row ID would lock in how values are stored. Nothing public references a quantity or rate by ID.

| Shape | Response | Request |
|---|---|---|
| Money | `{amount, currency}` | same |
| Quantity | `{value, unit}` | `{value, unit_id}` |
| Rate | `{value, unit, per_unit}` | `{value, unit_id, per_unit_id}` |
| UnitPrice | `{amount, currency, per_unit}` | `{amount, currency, per_unit_id}` |

- `unit` and `per_unit` are always populated, never expandable. A value without its unit is meaningless, and units are cached lookups.
- No `display_value`. Formatting is the client's job. A server-formatted string can't change locale or rounding without breaking whoever parses it.
- Name the field for its role (`unit_price`, `labor_rate`, `flat_rate`), not its shape.
- Store money as columns on the owning row (`*_amount`, `*_currency`), not as `quantity`/`rate` rows in a currency unit. Shape the tables to match the API, don't translate at the gateway.

### Money is its own type

Monetary amounts are `Money`, never a `Quantity` whose unit is a currency:

```json
{ "amount": "12.50", "currency": "usd" }
```

- `amount` is a decimal string, because unit prices go below a cent. It is never integer minor units.
- `currency` is a lowercase ISO 4217 code. It's inline, so the value can be read without an include, and integrations don't map unit IDs.
- Requests use the same shape. A non-money unit can't be expressed, so no "must be a currency unit" validation is needed.
- Price-type rates (currency per unit) are `{ amount, currency, per_unit }`.

### Defaults live on the parent

"Which child is the default" is a field on the parent (`carrier.default_service_level`, `item.default_order_unit`), set with `default_*_id`. Never an `is_default` boolean on each child: the parent field guarantees exactly one, with no side effect clearing another row.

### `object` is for resources

Every resource has an `object` field. Value shapes (money, quantities, rates) don't: they're never the subject of an audit event, request log or webhook, so filtering keys on the parent's `resource_type` plus a field path (`flat_rate.amount`). Any field that can hold more than one shape carries a discriminator, usually `type`.

### Discounts and percentages

Anything that takes money off uses one value shape: `{ "type": "percent_off", "percent_off": "5" }` or `{ "type": "amount_off", "amount_off": Money }`. Percentages are whole numbers as decimal strings (`"5"` = 5%), never fractions, so a client can't misread one as the other. A price set relative to a base is `percent_of_base` (`"80"` = 20% off, `"120"` = 20% markup), never a negative discount.

### Two reference patterns

A field naming one other resource is expandable and `null` unless the client includes it; including it returns the full resource. Links to other documents or records go in `related` as lightweight stubs. One variant of the stub: a resource that exists *about* one record of any type (a note, a subscription, a tool call) names it in a single `record` field holding the same stub, because a polymorphic field can't be expanded. Nothing else: no always-present `{ id, name }` stubs, and no response shapes built for one screen. Clients compose what they show from standard resources, includes and `related`; aggregates that can't be composed cheaply belong in analytics.

### Related records are stubs

Records produced from or linked to a resource (an order's pick, shipments, invoices) are returned as lightweight `record` stubs in a `related` section (`{ id, object: "record", type, number, status, metadata }`), not as full expanded objects. Clients fetch the full record by ID when they need it. That keeps responses small and avoids clients routinely pulling large sub-objects.

### Lifecycle actions

A status that moves through a workflow is read-only. Each transition is `POST …/actions/{verb}`, one verb per transition, named for what it produces (`issue`, `fulfill`, `reopen`). Never `PUT`, and never a writable `status` for workflow states. (Archive `status` is the exception: it's a plain field.)

### Spelling

US English in every name (`acknowledgment`, not `acknowledgement`).

### Who vs what

A record caused by someone carries an `actor` (`user`, `api_key`, `agent`, `device`, `system`); work done at a device by an identified person also carries `operator`. Its `type` describes what happened, never who did it, so new kinds of actor (agents, integrations) fit without new type values.

### Pick names once

Enum values, field names, error codes, and route segments are permanent. Settle them in review, not in a follow-up.

## Routes

- Every endpoint is namespaced by domain: `/v1/{namespace}/{resource}` (`/v1/finance/payment-terms`). The only exception is `/healthz`. The namespace is permanent once shipped. A resource shared across domains (addresses) lives in `core`.
- Collections are plural kebab-case nouns. Non-CRUD operations live under `/actions/{verb}`.

## Lifecycle status and archiving

Archive is not soft delete. An archived record is fully visible (retrievable, listed, filterable) and stays valid for everything that references it; archiving only blocks *new* assignments. Soft delete hides a row the database still holds, so every query, join and uniqueness rule must remember to skip it, and references point at something the API claims doesn't exist.

- **No soft delete by default.** Delete is a hard delete guarded by `resource_in_use`. Don't add `deleted_at` columns. Keep soft delete only where a resource review explicitly justifies it. The `deleted_record` tombstone table (for 410 replays) is not soft delete.
- **Visible deletion is a state, not soft delete.** When a deleted item must keep its place for others to see (a message in a conversation), it stays listed with `status: deleted`, its content fields `null`, and `deleted_at`. Nothing is hidden, so queries need no extra filter.
- **Add `status` only where it earns its cost.** It's warranted when long-lived records reference the resource *and* users routinely need to stop new use without rewriting that history (payment terms, carriers, attributes). Otherwise, just block delete while in use; `status` can be added later without breaking anything.
- Resources that can be retired use `status` with `active | archived`. Never `inactive` or a bare `is_active` boolean.
- Resources that are paused and resumed rather than retired (integrations, agents, account users) use `active | disabled`. Use this only when switching off is routine and temporary.
- `status` is settable on **create** (to set a record up before it goes live) and on **update**. It is not read-only.
- An archived resource stays attached to everything that already references it, but it **cannot be newly assigned**. Referencing an archived resource on create or update fails validation (422).
- Lists that expose `status` take a `statuses[]` filter. Omitting it returns all statuses.

## Documents snapshot what they print

Orders, invoices and shipments copy the master data they show (addresses today) as a value when the document is created, and keep a reference to the source only as provenance. Editing an address or customer later changes future documents, never ones already issued. A document never reads printed values live from a mutable row.

## Two-party documents are one row

Documents between two accounts (orders, invoices, credit notes, payments) are stored once, with both parties on the row (`seller` / `buyer`, `payer` / `payee`) and an `owner` who authored it. Each party reads the same row through its own collection and field names: a sales order for the seller is a purchase order for the buyer, and an invoice is a bill. Fields that depend on the viewer (payment `direction`, counterparty) are computed per viewer. Data that belongs to one party (`metadata`, internal instructions, approval state) lives in a per-party table, so neither side sees or overwrites the other's. Only the owner can edit a draft, finalize or void. Never mirror a row per party.

## Accounting documents are immutable

Invoices and credit notes are finalized and then never edited or deleted. Corrections are new documents (a credit note, a manual invoice) or status changes (`void`, `uncollectible`). Payments and refunds are reversed, never deleted or edited. Documents snapshot their lines, prices and totals at issue, so nothing upstream (an order edit, a price change) can alter them.

## Versioned definitions

Definitions that drive work (a manufacturing method) are versioned: `status: draft | active | archived` and a `version`. Edit a draft, then `POST …/actions/activate`; the previous active version becomes archived. Active and archived versions are immutable, and work in progress (a production run) keeps a copy of the version it started with.

## Private beta

An area still being designed ships internal: its routes are `Public: false`, and fields it adds to public resources are internal fields excluded from the public OpenAPI. Making it public later is additive; making a public shape private is not.

## Notes, instructions, comments

- **No free-form `notes` fields.** Commentary belongs in the `note` resource (`/v1/core/notes`), not in a field on the record. A note attaches to any record (or the whole account), records its `actor` (user, API key, agent, or system for migrated data), and is listed by filtering on `record_ids[]`, never embedded. Agent memory is notes too: don't add a separate memory or comment store for a new resource.
- **`instructions`** is internal staff guidance (customer service, fulfillment, billing), always `null` for portal users. It lives on customers and suppliers (standing defaults) and on orders. An order created without `instructions` copies its counterparty's once, as a snapshot; an explicit value (including `null`) is used exactly. Never concatenate silently.
- Child documents (shipments, invoices, picks, receiving orders) don't store a copy. They expose a read-only `order_instructions`, read from the order at query time through the join they already have.
- A message *from* the customer is a different audience and would be a separate field. Never mix it into `instructions`.
- Event records keep their own explanation when it is the event's content (why a machine went down), not commentary about it.

## Delete

- **In-use resources can't be deleted.** If other records still reference the resource, DELETE returns 409 with error code `resource_in_use` and a message pointing to archiving. Never hard-delete and leave dangling references. Each reference check must be an indexed `EXISTS` probe.
- **Response is a deleted stub**, 200:

  ```json
  { "id": "pt_01J9…", "object": "payment_term", "deleted": true }
  ```

  This gives SDKs a typed return value and keeps the ID in client logs. Never return `{}` or a 204.
- Replaying a delete of an already-deleted resource returns 410 `resource_gone`.
- System-owned (`owner.type = system`) resources can't be updated or deleted.

## Actions and jobs

- Non-CRUD operations are `POST .../actions/{verb}`: on the collection for bulk work (`/items/actions/set-order-units`), on the resource for single-record operations (`/items/{id}/actions/change-stocking-unit`).
- An action whose work can ever exceed the latency budget **always** returns 202 with a job, even when a given call would be fast. A response shape that depends on data size is two contracts.
- When one request could mean two different things, make the caller say which with a `mode` enum rather than guessing (e.g. `convert` vs `relabel`).

## Lists

### Pagination

- `limit` defaults to **25** and maxes at **100**. Large pages with includes are the most likely way a list misses the 50 ms worst case. Raising the max later is additive, lowering it is breaking.
- Keyset pagination on `(created_at, id)`, newest first, unless the resource review decides otherwise. The list docstring states the sort order.

### Search (`q`)

- `q` is optional. Only add it when callers need free-text lookup.
- Every list endpoint documents **exactly which columns** `q` matches and how (e.g. "Case-insensitive substring match on `name`."). Never "varies by endpoint".
- `q` only matches columns on the resource's own table. To narrow by a related resource, use an ID filter (`customer_ids[]`), never `q`.
- Match mode follows table size per tenant:

  | Rows per tenant | Use |
  |---|---|
  | Up to a few thousand (lookups: terms, units, carriers) | `LIKE '%q%'`, limited by an `account_id`-leading index |
  | Any size, code/SKU/number lookup | Prefix `LIKE 'q%'` on an indexed column |
  | Large, free text (items, products, customers) | ngram FULLTEXT with an exact-match-first path |

  Don't use the default (word) FULLTEXT parser for name search. It skips tokens under 3 characters ("30" in "Net 30") and stopwords. It also can't do substring matches, and it can't be scoped to a tenant before matching. Drop FULLTEXT indexes no query uses.

### Embedded lists

An expandable list inside a resource (`carrier.service_levels`, `item.order_units`) returns its first 10 items when expanded. If more exist, `page_info.has_next_page` is `true` and `page_info.next_page_url` points at the sub-resource list endpoint. Never cap silently.

### Filters

- Created-date ranges are `created_after` / `created_before` on every list. `starts_at` / `ends_at` are only for analytics and reporting periods.
- Lookups by a natural key get an exact filter on its unique index (`skus[]`), separate from relevance search `q`.
- Multi-value filters are plural arrays: `statuses[]`, `customer_ids[]`.

## Docstrings

Docstrings become the public API reference. They follow the `summary ¶ description` form.

- Summary is one sentence, present tense, starting with the verb: "Returns a list of…", "Creates a…", "Retrieves a…", "Updates a…", "Deletes a…".
- List docs say what is included (e.g. system defaults) and the sort order.
- Update docs say only sent fields change.
- Delete docs state the in-use rule and point to archiving.
- Field docs describe meaning for the caller, not storage. State uniqueness rules precisely, including case-insensitivity.
- Write for the reader: no internal table names, no "the dashboard".

## Release gate

Finalizing a resource in the forge.1 review clears `Preview: true` on all of its endpoints.
