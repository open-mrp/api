# forge.1 API review — required changes

Resource-by-resource review of the public API ahead of the forge.1 stable release. This file records **what must change in code**; the conventions behind each change live in `docs/patterns/public-api-design-conventions.md`. Nothing here is implemented yet.

Finalizing a resource in this review also means clearing `Preview: true` on all of its endpoints.

## Review order

1. Small shared lookups: payment terms ✅, shipping terms ✅, units ✅, unit groups ✅ (retired → order units), carriers ✅
2. Catalog: item categories ✅, properties ✅, product lines ✅, items ✅ — catalog done
3. Sales: customers ✅, account groups ✅, account statuses ✅ (removed), addresses ✅, contacts ✅, priorities ✅ (removed; lead time), pricing ✅ (price lists), sales orders ✅; accounts receivable documents ✅
4. Purchasing ✅ (suppliers, supplier items, purchase orders, receiving); accounts payable ✅
5. Operations: fulfillment ✅ (picks, shipments, shipping cases, deliveries); inventory ✅; production structure ✅ (manufacturing methods); production execution ✅ (runs, batches, scanning, downtime); production planning ✅ (redesigned; internal beta)
6. Core ✅ (audit/request/email logs, jobs, sandboxes, search; analytics internal); identity/auth ✅ (passwordless, devices, role templates); settings/integrations ✅
7. Messaging ✅ (notifications, subscriptions, fresh conversations)
8. AI ✅ (agents + versions, tool calls, notes; runs internal)

## Cross-cutting changes (apply to every resource)

These came out of a single resource review but apply everywhere. Implement once in the framework where possible, then sweep. The framework is `apikit` (see Shared API kit below).

- [ ] **Page size: default 25, max 100.** `apiresource.PaginationRequest.Limit` is `default:"100" validate:"min=1,max=1000"` → `default:"25" validate:"min=1,max=100"`. Audit dashboard callers that pass `limit` > 100 (dropdowns, exports) before shipping; they must paginate. (Payment terms review.)
- [ ] **Drop `page_info` booleans.** Remove `has_next_page` and `has_prev_page`. Another page exists exactly when `next_page_url` or `previous_page_url` is present; the booleans repeat that.
- [ ] **Delete returns a deleted stub.** Replace `*apiresource.EmptyResource` on every DELETE with a `DeletedResource` `{ "id", "object", "deleted": true }` where `object` is the deleted resource's object type. 200 status. Replayed delete stays 410. (Payment terms review.)
- [ ] **Delete is blocked while referenced.** Deleting a resource that other records still reference returns 409 with a new error code `resource_in_use`; the message points to archiving. Each resource's review lists the references to probe. (Payment terms review.)
- [ ] **No soft delete by default.** Delete is a hard delete guarded by `resource_in_use`; no new `deleted_at` columns. Existing soft deletes (`carrier`, `item`, `message`, `message_attachment`, `operating_calendar`) move to hard delete unless that resource's review justifies keeping it. `deleted_record` (tombstone for 410 replays) is not soft delete and stays. (Product lines review.)
- [ ] **`status` only where it earns its cost.** Add archivable `status` only when long-lived records reference the resource *and* users routinely need to stop new use without rewriting history; otherwise just block delete while in use. (Product lines review.)
- [ ] **No `notes` fields.** Remove free-form `notes` from master data. Document-level guidance becomes `instructions` where it's part of the workflow. Attributed notes (by user, agent or API key) are the `note` resource, which ships with forge.1 (AI review, A6); existing note data migrates into it. (Product lines review; see "Notes → instructions".)
- [ ] **`inactive` → `archived`.** Every lifecycle enum value `inactive` becomes `archived`, except resources that are paused and resumed rather than retired (integrations, agents, account users), which use `disabled`. `status` is settable on create and update wherever a resource can be archived, with a `statuses[]` list filter. (Payment terms review.)
- [ ] **Archived resources can't be newly referenced.** Assigning an archived resource to a new or updated record fails validation (422); existing references remain. (Payment terms review.)
- [ ] **`q` is documented per endpoint.** Remove the generic "varies by endpoint" docstring from `PaginationRequest.Query`; each list endpoint documents the exact columns and match mode. Endpoints without search drop `q` entirely. (Payment terms review.)
- [ ] **Clear `Preview: true`** on each resource as it is finalized here.
- [ ] **Value shapes (units review).** Response: `Money {amount, currency}`, `Quantity {value, unit}`, `Rate {value, unit, per_unit}`, `UnitPrice {amount, currency, per_unit}`. Requests: `{amount, currency}`, `{value, unit_id}`, `{value, unit_id, per_unit_id}`, `{amount, currency, per_unit_id}`. `unit`/`per_unit` are always populated (not expandable). No `object`, `id`, timestamps or `display_value`. `Rate.numerator_unit`/`denominator_unit` → `unit`/`per_unit`. Fields named for their role (`unit_price`, `labor_rate`).
- [ ] **Quantities and rates are value objects with no ID.** Drop `id` (and `created_at`/`updated_at` on `Rate`) from the public `Quantity` and `Rate` shapes; they are written only through their parent resource. Fold `ComputedQuantity` into `Quantity` (units review: `unit` always populated). Retire the internal `PATCH /v1/operations/quantities/{id}` and `PATCH /v1/operations/rates/{id}` (no dashboard callers found; confirm before removal). Affects ~22 resources with `Quantity`, ~10 with `Rate`, ~11 with `ComputedQuantity`. Goal for the new version: nothing public references a quantity or rate by ID. (Shipping terms review.)
- [ ] **Money is its own type.** Monetary amounts are `Money` = `{ "amount": "decimal string", "currency": "usd" }` (ISO 4217, lowercase), never a `Quantity` in a currency unit. Price-type rates (currency per unit) become `{ amount, currency, per_unit }`. Requests take the same shape (`currency` code, not a unit ID). Only `usd` is accepted at forge.1 (other codes → 422 `currency_not_supported`); more currencies are additive later. **Refactor the tables, don't translate at the gateway**: money columns (`*_amount decimal`, `*_currency char(3)`) live on the owning row instead of `quantity`/`rate` rows in a currency unit. Prod (2026-10-07): the only currency units are 1 system + 29 per-account copies of "Dollar" (`$`, ratio 1); 1,272,946 `rate` rows have a currency numerator (none as denominator) and 473,911 `quantity` rows are in a currency unit. All map to `usd`. Data migration per owning table, then drop currency units and the `currency` dimension. `shared/pricing` and the `pricing_*_ratio_*` columns move to money columns. (Shipping terms + units reviews.)
- [ ] **Drop `display_value`** from every value shape; clients format. (Units review.)
- [ ] **`object` is for resources only.** Every resource has `object`; value shapes don't; any field that can hold more than one shape carries a discriminator (`type`). (Unit groups review.)
- [ ] **Embedded expandable lists page.** When expanded, an embedded list returns its first 10 items; if more exist, `page_info.next_page_url` points at the sub-resource list endpoint. No silent caps. (Carriers review.)
- [ ] **Defaults are a field on the parent** (`default_service_level`, `default_order_unit`), never an `is_default` flag on children. (Carriers review.)
- [ ] **Created-date filters are `created_after` / `created_before`** on every list (replacing `starts_at`/`ends_at` where they mean "created between"). `starts_at`/`ends_at` stay only for analytics/reporting periods. (Items review.)
- [ ] **Request mirrors response.** Same field names and nesting in both directions; a related object is written as `<field>_id` where the response returns the object; PATCH merges section fields individually. (Customers review.)
- [ ] **Inherited defaults live in a `defaults` section** at every level of an inheritance chain (customer, account group, account), with the same field names, so the chain reads the same everywhere. (Account groups review.)
- [ ] **Fixed platform enums are fields, not resources.** No list endpoint, ID or `owner` for a value set the platform defines; reference by code. If one later becomes a resource, it keeps `code` as its stable identifier, so existing fields still hold the string. (Account groups review.)
- [ ] **Every endpoint is namespaced** (`/v1/{namespace}/{resource}`); the only exception is `/healthz`. All `/v1` routes already comply. (Addresses review.)
- [ ] **Documents snapshot the master data they print.** Orders, invoices and shipments copy addresses (and similar printed values) at creation; editing the source affects only future documents. (Addresses review.)
- [ ] **One `Discount` value shape** (`percent_off` whole-number percent, or `amount_off` Money) wherever something is taken off; price adjustments relative to base use `percent_of_base` (no signs). (Pricing review.)
- [ ] **Related records are lightweight `Record` stubs** (`{ id, object: "record", type, number, status, metadata }`) grouped in `related`, not full expanded sub-objects; clients fetch the full record by ID when they need it. (Sales orders review.)
- [ ] **Lifecycle transitions are `POST …/actions/{verb}`**, one verb per transition, named for the status it produces; `status` is read-only. (Sales orders review.)
- [ ] **US English spelling** in every field name (`acknowledgment`). (Sales orders review.)
- [ ] **Accounting documents are immutable.** Invoices and credit notes are finalized and never edited or deleted; corrections are new documents (credit notes, manual invoices) and status changes (void, uncollectible). Money movements (payments, refunds) are reversed, never deleted. (AR review.)
- [ ] **Two-party documents are one row.** Orders, invoices, credit notes and payments between two accounts are stored once with both parties (`seller`/`buyer`, `payer`/`payee`, `owner`); each party reads it through its own collection and perspective (sales order / purchase order, invoice / bill, credit note / supplier credit), with per-party data (`metadata`, internal instructions, workflow state) in a party table. Never mirror rows per party. (AP review.)
- [ ] **Separate who from what.** Records caused by someone carry an `actor` (`user | api_key | agent | device | system`; `scanning_station` → `device` per the auth review); their `type` describes what happened, never who did it, so agents and API keys fit without new values. (Inventory review.)
- [ ] **Definitions that drive work are versioned.** A recipe-like definition (manufacturing method) is edited as a draft and activated; active and archived versions are immutable, and work in progress keeps the version it started with. (Production review.)
- [ ] **Two reference patterns only.** A field naming one resource is expandable and `null` unless included; links to other documents are `related` record stubs. No always-present `{id, name}` stubs and no API shapes built for one screen — the dashboard composes standard resources. (Production review.)
- [ ] **Private beta is internal, not public.** Areas still being designed ship as internal routes and internal fields (excluded from the public OpenAPI); exposing them later is additive. (Planning review.)
- [ ] **No booleans in responses.** Use an enum for any state or mode, so new values can be added later without another field. Every public resource has now been reviewed against this; none keep a boolean. (Shipping terms review.)

## Payment terms — `/v1/finance/payment-terms` ✅ finalized

Endpoints: list, create, retrieve, update, delete. Namespace `finance` stays.

Resource
- [ ] `status` enum: `active | archived` (rename `inactive`; DB `is_active` mapping unchanged).
- [ ] `owner` stays expandable (`null` unless `include[]=owner`).
- [ ] Docstrings per the table below.

Create
- [ ] Add `status` (`field.Optional[constants.PaymentTermStatus]`, default `active`).

Update
- [ ] Add `status` (`field.Optional[constants.PaymentTermStatus]`).

List
- [ ] Add `statuses[]` filter; omitted = all statuses.
- [ ] `q`: case-insensitive substring match on `name` (`LIKE '%q%'`, kept — table is bounded per tenant).
- [ ] Limit default 25 / max 100 (cross-cutting).

Delete
- [ ] 409 `resource_in_use` when referenced by `account_relation.payment_term_id` (customers) or `sales_order.payment_term_id`. Both have indexes; two `EXISTS` probes.
- [ ] Return `{id, object: "payment_term", deleted: true}`.

Referencing endpoints
- [ ] Customer create/update and sales order create/update reject an archived payment term.

Schema
- [ ] Drop unused `FULLTEXT payment_term_name_idx`.

Deferred
- `days_until_due` / due-date math — the real feature is more involved; not part of forge.1.

Docstrings

| Where | Text (summary ¶ description) |
|---|---|
| List | Returns a list of payment terms. ¶ Includes the terms your account created and the system defaults every account shares. Terms are sorted by creation date, newest first. |
| List · `q` | Case-insensitive substring match on `name`. |
| List · `statuses` | Only return payment terms with one of these statuses. Omit to return all statuses. |
| Create | Creates a payment term. ¶ The new term belongs to your account. |
| Create · `status` | Initial status. Defaults to `active`; create as `archived` to set a term up before offering it. |
| Retrieve | Retrieves a payment term. ¶ Works for your account's terms and for system defaults. |
| Update | Updates a payment term. ¶ Only fields you send are changed. System defaults can't be updated. |
| Delete | Deletes a payment term. ¶ Only terms your account created can be deleted, and only while no customer or sales order uses them. Archive a term that is in use by setting `status` to `archived`. |
| `name` | Display name, such as `Net 30`. ¶ Unique (case-insensitive) among your account's terms and the system defaults. |
| `status` | Whether the term can be assigned to customers and sales orders. ¶ `archived` terms stay attached to the records already using them. |
| `owner` | Who owns this payment term. ¶ `system` terms are shared defaults and are read-only; `account` terms belong to your account. |
| `created_at` | Time the payment term was created. |
| `updated_at` | Time the payment term was last changed. |

## Shipping terms — `/v1/operations/shipping-terms` ✅ finalized

Endpoints: list, create, retrieve, update, delete. Namespace `operations` stays.

Resource
- [ ] Add `status: active | archived` (new `is_active` column, schema migration), settable on create and update; `statuses[]` list filter.
- [ ] `type` enum values: `free | flat_rate | carrier_rate` (were `free_freight | flat_rate_freight | carrier_rate_freight`).
- [ ] Replace top-level `minimum_order_value` + `free_shipping_service_levels` with `free_shipping`: `null` or `{ minimum_order_value: Money, service_level_scope: all | selected, service_levels: expandable List }`.
- [ ] Include keys become `free_shipping.service_levels`, `owner`, `owner.account` (money needs no unit include).
- [ ] `flat_rate` and `free_shipping.minimum_order_value` are `Money`.
- [ ] Quantities lose `id` (cross-cutting).

Create / update
- [ ] `flat_rate` (`Money`) required when `type = flat_rate`, rejected otherwise (422). On update, changing `type` away from `flat_rate` clears it; changing to `flat_rate` requires it in the same request.
- [ ] `free_shipping` request: `{ minimum_order_value: Money (required), service_level_ids?: [] }` (omitted = all). `field.Clearable` on update, replaced as a whole. Rejected when `type = free`.
- [ ] `name` unique (case-insensitive) among the account's terms and system defaults → 409 `resource_exists`.
- [ ] Add `status` on create (default `active`) and update.

List
- [ ] `q`: case-insensitive substring match on `name`. `statuses[]` filter.

Delete
- [ ] 409 `resource_in_use` when referenced by `account_relation.shipping_term_id` or `sales_order.shipping_term_id` (both indexed).
- [ ] Return `{id, object: "shipping_term", deleted: true}`.

Referencing endpoints
- [ ] Customer and sales order create/update reject an archived shipping term.

Schema
- [ ] Add `shipping_term.is_active`. Drop unused `FULLTEXT shipping_term_name_idx`.

Docstrings

| Where | Text (summary ¶ description) |
|---|---|
| List | Returns a list of shipping terms. ¶ Includes the terms your account created and the system defaults every account shares. Terms are sorted by creation date, newest first. |
| List · `q` | Case-insensitive substring match on `name`. |
| List · `statuses` | Only return shipping terms with one of these statuses. Omit to return all statuses. |
| Create | Creates a shipping term. ¶ The new term belongs to your account. It affects freight quotes once it's assigned to a customer or sales order. |
| Retrieve | Retrieves a shipping term. ¶ Works for your account's terms and for system defaults. |
| Update | Updates a shipping term. ¶ Only fields you send are changed. System defaults can't be updated. Freight already recorded on orders isn't recalculated. |
| Delete | Deletes a shipping term. ¶ Only terms your account created can be deleted, and only while no customer or sales order uses them. Archive a term that is in use by setting `status` to `archived`. |
| `name` | Display name, such as `Free over $500`. ¶ Unique (case-insensitive) among your account's terms and the system defaults. |
| `type` | How freight is charged. ¶ `free`: never charged. `flat_rate`: charged `flat_rate` on every order. `carrier_rate`: charged the carrier's quoted rate for the order's service level. |
| `status` | Whether the term can be assigned to customers and sales orders. ¶ `archived` terms stay attached to the records already using them. |
| `flat_rate` | Freight charged on every order. ¶ Present only when `type` is `flat_rate`. |
| `free_shipping` | Waives freight on orders over a minimum value. ¶ `null` when the term has no threshold. Not allowed when `type` is `free`. |
| `free_shipping.minimum_order_value` | Order total that must be exceeded for freight to be waived. |
| `free_shipping.service_level_scope` | Which service levels qualify for free freight. ¶ `all`: every service level. `selected`: only the levels in `service_levels`. |
| `free_shipping.service_levels` | Service levels that qualify for free freight when `service_level_scope` is `selected`. Empty when it is `all`. |
| Request · `service_level_ids` | Service levels that qualify for free freight. Omit to waive freight on every service level. |
| `owner` / `created_at` / `updated_at` | Same text as payment terms. |

## Units — `/v1/catalog/units` ✅ finalized

Endpoints: list, create, retrieve, update, delete, bulk upsert (public). `actions/export` and `actions/validate` are internal and out of scope.

Resource
- [ ] `type` → `dimension`; values `quantity | time | mass | volume | length | temperature | area` (`currency` removed — see money).
- [ ] Remove `is_base_unit` (response and bulk upsert input).
- [ ] No `status`. Units aren't archived; availability is controlled elsewhere (see unit groups review).

Create / update
- [ ] `abbreviation` max length 32 on create, update and bulk upsert.
- [ ] Reject `dimension: currency`.

List
- [ ] `type` → `dimensions[]`. `unit_group_ids[]` removed (unit groups retired).
- [ ] `q`: case-insensitive substring on `name` or `abbreviation` (`LIKE '%q%'` for every length; drop the `MATCH` path).

Delete
- [ ] In use (quantity, rate, or demand adjustment) → 409 `resource_in_use` (today `resource_conflict`).
- [ ] Return `{id, object: "unit", deleted: true}`.

Bulk upsert
- [ ] Match rows on `abbreviation` only (case-insensitive); unmatched rows create; a name collision → `resource_exists` on that row. Max stays 1000.

Schema
- [ ] Drop `FULLTEXT unit_name_idx`, `unit_abbreviation_idx`, `unit_name_abbreviation_idx`.

Docstrings

| Where | Text (summary ¶ description) |
|---|---|
| List | Returns a list of units. ¶ Includes the units your account created and the system units every account shares. Units are sorted by creation date, newest first. |
| List · `q` | Case-insensitive substring match on `name` or `abbreviation`. |
| List · `dimensions` | Only return units in one of these dimensions. |
| Create | Creates a unit. ¶ The new unit belongs to your account. Its conversion ratio is relative to the base unit of its dimension. |
| Retrieve | Retrieves a unit. ¶ Works for your account's units and for system units. |
| Update | Updates a unit. ¶ Only fields you send are changed. System units can't be updated, and `dimension` can't change after creation. |
| Delete | Deletes a unit. ¶ Only units your account created can be deleted, and only while no quantity, rate or price is recorded in it and no unit group uses it as its base unit. To stop offering a unit that is in use, remove it from its unit groups. |
| Bulk upsert | Creates or updates units in bulk. ¶ Each row is matched to an existing unit by `abbreviation` (case-insensitive). Unmatched rows create new units. The work runs in the background; the response is an async job to poll. |
| `name` | Display name, such as `Kilogram`. ¶ Unique (case-insensitive) among your account's units and the system units. |
| `abbreviation` | Short label, such as `kg`. ¶ Unique (case-insensitive) among your account's units and the system units. |
| `dimension` | What the unit measures. ¶ Values convert only between units of the same dimension. `quantity` is for countable things, such as each, pair or case. |
| `ratio_numerator` / `ratio_denominator` | The ratio that converts a value in this unit to its dimension's base unit. ¶ A value converts as `value × ratio_numerator ÷ ratio_denominator + offset_numerator ÷ offset_denominator`. A kilogram, with grams as the base, is `1000` / `1`. The denominator can't be zero. |
| `offset_numerator` / `offset_denominator` | An offset added after the ratio, for scales with different zero points, such as temperature. ¶ `0` / `1` for units with no offset. The denominator can't be zero. |
| Money · `amount` / `currency` | Decimal amount, as a string to keep precision. / Three-letter ISO 4217 currency code, lowercase. Only `usd` is supported. |
| Quantity · `value` / `unit` | Decimal value, as a string to keep precision. / Unit the value is measured in. |
| Rate · `per_unit` | Unit the rate is measured per, such as hours in `100 kg / hr`. |

## Unit groups — `/v1/catalog/unit-groups` ✅ retired, replaced by order units

Decided
- [ ] Remove all 12 public unit group endpoints, the internal export, and the `unit_group` / `unit_group_unit` objects.
- [ ] Remove `item_category.unit_group` / `unit_group_id`, `product_line.unit_group` / `unit_group_id`, the `unit_group_ids` filter on list units, and `unit_group` on the analytics material.
- [ ] Remove the 3 system groups. The dashboard's import uses the "time" and "price" groups for lead-time unit and currency: currency becomes `usd`; lead time gets an explicit unit when items are reviewed.
- [ ] Each item gets `stocking_unit` (set at create). Migration: `item.stocking_unit_id` = the category group's base unit. Hot paths (scan floor, inventory, analytics) join `item → unit` directly.
- [ ] Stocking unit must be correctable without delete/re-create (mechanism pending).
- [ ] Orderable units carry a default unit (auto-selected on order lines) and a portal visibility; hidden units don't appear in the customer portal at all.
- [ ] Purchase orders follow the same model as sales orders ("sales orders in reverse").
- [ ] Pack discounts (e.g. a 12-pr pack costs a little less per pair than a 6-pr pack) are a real use case and must survive in some form; float storage goes.
- [ ] Product-line analytics rollups stop depending on groups (target unit decided in the analytics review).

Replacement (decided round 2)

Items (materials, parts, products, generic items)
- [ ] `stocking_unit` (expandable Unit). `stocking_unit_id` required on create; not updatable by PATCH.
- [ ] `default_order_unit` (expandable Unit). `default_order_unit_id` optional on create/update; must be the stocking unit or an order unit; defaults to the stocking unit.
- [ ] `order_units` (expandable List): `{ unit, portal_visibility: visible | hidden, discount }`. Request `order_units: [{ unit_id, portal_visibility?, discount? }]`; on update `field.Optional`, replaces the list. Every order unit shares the stocking unit's dimension. Stored per item (`item_order_unit`, PK `(item_id, unit_id)`).
- [ ] `discount`: `null | { type: "percent_off", percent_off: "decimal" } | { type: "amount_off", amount_off: Money }`. Applied when a line's price defaults: price converted to the ordered unit, then discounted. Same on sales and purchase orders. Decimal/Money storage, no floats.
- [ ] Sales order lines and purchase order lines: unit must be the item's stocking unit or one of its order units; the line's unit defaults to `default_order_unit`.
- [ ] `hidden` order units never appear in the customer portal; staff can still use them.
- [ ] Purchasing (receiving, inventory adjustments) uses the same allowed set as orders.

New endpoints
- [ ] `POST /v1/catalog/items/actions/set-order-units` — `{ filter: { item_ids | product_line_ids | category_ids | attribute filters }, order_units, default_order_unit_id }` → 202 job. Replaces both on every matching item.
- [ ] `POST /v1/catalog/items/{id}/actions/change-stocking-unit` — `{ stocking_unit_id, mode: convert | relabel }` → **always 202 + async job** (some items restate 100K+ rows). `convert`: same dimension only; restates levels, lots, rates by ratio. `relabel`: numbers kept, any dimension; order units of another dimension are removed and reported in the async job result. One audit event on the item, mode in metadata.

Later, additive
- Supplier-specific order units (`supplier_material.order_units`).
- Pack prices as `UnitPrice` per pack (prices review); may take precedence over the order-unit discount.

Migration
- [ ] `item.stocking_unit_id` = category group's base unit.
- [ ] `order_units` = product line group members (products) or category group members (other items), carrying `is_visible` → `portal_visibility` and the one fixed discount → `amount_off`.
- [ ] `default_order_unit_id` = stocking unit.
- [ ] Then drop `unit_group`, `unit_group_unit`, and the `unit_group_id` columns.

Docstrings

| Where | Text (summary ¶ description) |
|---|---|
| `stocking_unit` | Unit this item's inventory, costs and usage are kept in. ¶ Quantities in other units of the same dimension are converted to it. To correct it, use Change Stocking Unit. |
| `default_order_unit` | Unit selected by default when this item is added to a sales or purchase order. ¶ Always the stocking unit or one of `order_units`. |
| `order_units` | Units this item can be ordered in, besides its stocking unit. ¶ Applies to both sales and purchase orders. Every unit measures the same dimension as the stocking unit. |
| `order_units[].portal_visibility` | Whether customers can see and pick this unit in the customer portal. ¶ `hidden` units can still be used on orders your team enters. |
| `order_units[].discount` | Discount applied to the line's default price when the item is ordered in this unit, such as a lower per-pair price for a 12-pair pack. ¶ `null` when there is none. |
| `discount.type` | How the discount is expressed. ¶ `percent_off`: `percent_off` percent off the converted price. `amount_off`: `amount_off` subtracted from the converted price. |
| Set Order Units | Sets orderable units on many items at once. ¶ Replaces `order_units` and `default_order_unit` on every item matching the filter. The work runs in the background; the response is an async job to poll. |
| Change Stocking Unit | Changes the unit an item is stocked in. ¶ `convert` restates inventory, lots and costs into the new unit of the same dimension. `relabel` keeps the numbers and only changes the unit, to fix one entered in error. The work runs in the background; the response is an async job to poll. |

## Carriers & service levels — `/v1/operations/carriers` ✅ finalized

Endpoints: carriers list/create/retrieve/update/delete; service levels list/create/retrieve/update/delete under `/carriers/{carrier_id}/service-levels`. Internal (out of scope): `actions/sync-options`, `actions/initiate-oauth`, `oauth-status`.

Carrier
- [ ] `code` → required `type`: `fedex | ups | usps | ltl | will_call | local_delivery | other`. Migration: `ltl1` → `ltl`, `delivery` → `local_delivery`, `null` → `other`; drop `freight_collect` (unused; a freight-billing concept for shipping terms/orders later). Fixed after create. EDI integrations that used `ltl` vs `ltl1` to tell carriers apart move to `scac`.
- [ ] Remove `deleted_at`.
- [ ] Add `scac` (optional; 2–4 uppercase letters, Standard Carrier Alpha Code) on create/update; not unique. EDI integrations send `carrier.scac` instead of mapping carrier types to SCACs. Backfill `scac` on existing carriers from the integrations' current mappings.
- [ ] Add `status: active | archived` (settable on create/update, `statuses[]` filter); archived carriers can't be newly set on orders or customer defaults.
- [ ] `customer_portal_visibility` → `portal_visibility`.
- [ ] `default_service_level` (expandable ServiceLevel, nullable); `default_service_level_id` on create/update (clearable); must belong to the carrier and be active.
- [ ] `service_levels` include: first 10 + `page_info.next_page_url` to the sub-resource list.
- [ ] Delete: 409 `resource_in_use` if referenced by `account_relation.default_carrier_id`, `sales_order.carrier_id`, `shipment.carrier_id`, `shipping_case.carrier_id`; returns the stub. Shippo deactivation stays outside the DB transaction.
- [ ] `q`: substring on `name`. Drop FULLTEXT `carrier_name_idx`, `carrier_description_idx`, `carrier_name_description_idx`.

Service level
- [ ] `service_level_token` → `code` (one field, unique per carrier). **DB: `service_level_token` holds the data we want; `code` is legacy.** Copy token into the kept column, then drop the legacy one (prod: identical on all 16 rows, but migrate token → code semantics, not the reverse).
- [ ] Remove `is_default` (replaced by `carrier.default_service_level`; migrate the 2 flagged rows).
- [ ] `default_transit_days` → `estimated_transit_days`.
- [ ] `customer_portal_visibility` → `portal_visibility`; add `status: active | archived` + `statuses[]`.
- [ ] Delete: 409 `resource_in_use` if referenced by `account_relation.default_carrier_option_id`, `sales_order.carrier_option_id`, `shipment.carrier_option_id`, `shipping_term_free_shipping_rule.carrier_option_id`, or it's the carrier's default; returns the stub.
- [ ] Sync archives synced service levels the carrier no longer offers instead of deleting them; manually created ones are untouched. Shippo calls stay outside any transaction.
- [ ] `q`: substring on `name`. Drop FULLTEXT `carrier_option_name_idx`, `carrier_option_code_idx`, `carrier_option_name_code_idx`.

Docstrings

| Where | Text (summary ¶ description) |
|---|---|
| List carriers | Returns a list of carriers. ¶ Includes the carriers your account created and the system carriers every account shares. Carriers are sorted by creation date, newest first. |
| · `q` / `statuses` | Case-insensitive substring match on `name`. / Only return carriers with one of these statuses. Omit to return all statuses. |
| Create carrier | Creates a carrier. ¶ A `fedex`, `ups` or `usps` carrier is connected through Shippo for live rates and labels, and gets a service level for each service the carrier offers, hidden from the customer portal until you show it. This needs an active Shippo integration and is skipped for sandbox accounts. |
| Retrieve carrier | Retrieves a carrier. ¶ Works for your account's carriers and for system carriers. |
| Update carrier | Updates a carrier. ¶ Only fields you send are changed. `type` and `account_number` can't change after creation, and system carriers can't be updated. |
| Delete carrier | Deletes a carrier and its service levels. ¶ Only carriers your account created can be deleted, and only while no customer, order or shipment uses them. Archive a carrier that is in use by setting `status` to `archived`. A Shippo-connected carrier's Shippo account is deactivated. |
| `name` | Display name, such as `FedEx`. ¶ Unique (case-insensitive) among your account's carriers and the system carriers. |
| `type` | What kind of carrier this is. ¶ `fedex`, `ups`, `usps`: connected through Shippo for live rates and labels. `ltl`: less-than-truckload freight. `will_call`: the customer picks up. `local_delivery`: your own vehicles. `other`: any other carrier you manage yourself. |
| `scac` | Standard Carrier Alpha Code, such as `FDEG`. ¶ The 2–4 letter code that identifies the carrier on EDI documents and bills of lading. `null` when not set. |
| `account_number` | Your account number with the carrier. ¶ Required for `ups` and `usps`, which connect using it. FedEx connects through OAuth instead. |
| `default_service_level` | Service level selected by default when this carrier is chosen. ¶ `null` when the carrier has no default. |
| `service_levels` | Service levels this carrier offers. ¶ Returns the first 10. When there are more, `page_info.next_page_url` fetches the rest. |
| `portal_visibility` | Whether customers can see and pick this carrier (or service level) at checkout in the customer portal. |
| List service levels | Returns a list of a carrier's service levels. ¶ Sorted by creation date, newest first. |
| Create service level | Adds a service level to a carrier. ¶ Use it for carriers you manage yourself, or for a service a connected carrier doesn't publish. Service levels you add are never archived by a sync. |
| Delete service level | Deletes a service level. ¶ Only while no order, shipment or shipping term uses it, and it isn't its carrier's default. Archive it instead by setting `status` to `archived`. |
| `code` | Code identifying the service, such as `fedex_ground`. ¶ Unique among the carrier's service levels. For connected carriers it's the carrier's own code, used for rates and labels. |
| `estimated_transit_days` | Business days this service usually takes, used when the carrier can't quote a lane. ¶ Works an order's ship-by date back from its promised date. `null` means unknown. |

## Item categories — `/v1/catalog/item-categories` ✅ finalized

Endpoints: list, create, retrieve, update, delete, bulk upsert. Removed: `PUT/DELETE /{id}/properties/{property_id}`, `POST /{id}/properties`, `PUT /{id}/unit-groups/{unit_group_id}`. Internal export out of scope.

Resource
- [ ] `type` values `material_category | product_category` → `material | product` (a `product` category holds products and parts). Fixed after create.
- [ ] Remove `unit_group` and its include keys (`unit_group`, `unit_group.base_unit`, `unit_group.associated_units`, `unit_group.associated_units.unit`).
- [ ] No `status` (fails the status test: retire a category by moving its items and deleting it).
- [ ] `properties` include: first 10 + `next_page_url` → `GET /v1/catalog/properties?item_category_ids[]=…` (properties review adds the filter).

Create / update
- [ ] `property_ids` on create (optional) and update (`field.Optional`, replaces the list). Duplicate property names in one category → 409.
- [ ] Remove `notes` (no-notes convention).
- [ ] Remove `unit_group_id`.

List
- [ ] `type` → `types[]`. `q`: substring on `name`. (No `statuses[]`.)

Delete
- [ ] 409 `resource_in_use` if referenced by `item.item_category_id` or a price-list rule's category scope (both indexed). Today delete orphans items. Returns the stub.

Bulk upsert
- [ ] Remove `unit_group` from rows. `property_names` keeps import semantics (attach by name, create missing, never detach), documented.

Schema
- [ ] Drop `item_category.unit_group_id` (after the order-units migration) and FULLTEXT `item_category_name_idx`.

Docstrings

| Where | Text (summary ¶ description) |
|---|---|
| List | Returns a list of item categories. ¶ Includes the categories your account created and the system categories every account shares. Categories are sorted by creation date, newest first. |
| · `q` / `types` | Case-insensitive substring match on `name`. / Only return categories of these types. |
| Create | Creates an item category. ¶ The new category belongs to your account. |
| Retrieve | Retrieves an item category. ¶ Works for your account's categories and for system categories. |
| Update | Updates an item category. ¶ Only fields you send are changed. `type` can't change after creation, and system categories can't be updated. |
| Delete | Deletes an item category. ¶ Only categories your account created can be deleted, and only while no item or price-list rule uses them. Move a category's items to another category before deleting it. |
| Bulk upsert | Creates or updates item categories in bulk. ¶ Rows are matched to existing categories by `name` (case-insensitive). `property_names` attaches properties by name, creating any that don't exist, and never removes a property already attached. The work runs in the background; the response is an async job to poll. |
| `name` | Display name, such as `Fasteners`. ¶ Unique (case-insensitive) among your account's categories. |
| `type` | What kind of items the category holds. ¶ `material`: materials. `product`: products and parts. An item can only be in a category of its own kind. Can't change after creation. |
| `properties` | Properties the category's items vary by, such as `Color` or `Size`. ¶ Also shown in the customer catalog. Returns the first 10. When there are more, `page_info.next_page_url` fetches the rest. |
| Request · `property_ids` | Properties the category's items vary by. On update, replaces the whole list. Property names must be unique within the category. |

## Properties & attributes — `/v1/catalog/properties` ✅ finalized

Endpoints: properties list/create/retrieve/update/delete/bulk upsert; attributes list/create/retrieve/update/delete under `/properties/{property_id}/attributes`. Internal export out of scope. Account-only (no system rows, no `owner`).

Property
- [ ] No `status` on properties (fails the status test). Attributes keep it.
- [ ] `attributes` include: first 10 + `next_page_url` → attributes list.
- [ ] List: add `item_category_ids[]` (via the category–property link; index the category side). `q`: substring on `name`. (No `statuses[]`.)
- [ ] Delete: 409 `resource_in_use` if an item category references it or any of its attributes is in use; returns the stub. Today it cascades and strips attributes from items.

Attribute
- [ ] `value` unique (case-insensitive) **within its property**, not account-wide. Anything resolving attributes by bare value (imports, bulk upserts of items) resolves by `(property, value)`.
- [ ] `property` is a normal expandable field everywhere (`include[]=property`; on items `attributes.property`, on price list rules `scope.attributes.property`). Today it's populated only under items and volume discounts.
- [ ] Omitted `color` defaults to `default` (today: random).
- [ ] Add `status: active | archived` (create/update, `statuses[]`); archived attributes stay on items but can't be newly assigned.
- [ ] Delete: 409 `resource_in_use` if referenced by `_item_attributes` or a price-list rule's attribute scope; returns the stub.
- [ ] `q`: substring on `value` (`attribute.text`).

Schema
- [ ] Drop FULLTEXT `property_name_idx`, `attribute_text_idx`. Replace account-wide attribute-value uniqueness with `(property_id, text)`.

Docstrings

| Where | Text (summary ¶ description) |
|---|---|
| List properties | Returns a list of properties. ¶ Properties are sorted by creation date, newest first. |
| · `q` / `item_category_ids` | Case-insensitive substring match on `name`. / Only return properties attached to at least one of these item categories. |
| Create property | Creates a property. ¶ It starts with no attributes; add them with Create Attribute. |
| Retrieve / Update property | Retrieves a property. / Updates a property. ¶ Only fields you send are changed. |
| Delete property | Deletes a property and its attributes. ¶ Only while no item category, item or price-list rule uses it or any of its attributes. |
| Bulk upsert | Creates or updates properties and their attributes in bulk. ¶ Rows are matched to existing properties by `name` (case-insensitive). Attributes are added by value and appended in order. Existing attributes are never changed or removed. The work runs in the background; the response is an async job to poll. |
| List attributes | Returns a list of a property's attributes. ¶ Sorted by `sort_order`, first to last. |
| · `q` / `statuses` | Case-insensitive substring match on `value`. / Only return attributes with one of these statuses. |
| Create attribute | Adds an attribute to a property. ¶ Attributes after the chosen `sort_order` shift down by one. |
| Update attribute | Updates an attribute. ¶ Only fields you send are changed. A new `value` shows everywhere the attribute is assigned. |
| Delete attribute | Deletes an attribute. ¶ Only while no item or price-list rule uses it. Archive it instead by setting `status` to `archived`. Later attributes shift up to keep `sort_order` contiguous. |
| `property.name` | Display name, such as `Size`. ¶ Unique (case-insensitive) among your account's properties. |
| `property.attributes` | The property's attributes, in `sort_order`. ¶ Returns the first 10. When there are more, `page_info.next_page_url` fetches the rest. |
| `attribute.value` | The value, such as `Large`. ¶ Unique (case-insensitive) within its property. |
| `attribute.color` | Swatch color shown for the attribute. ¶ Defaults to `default`, a neutral swatch. |
| `attribute.sort_order` | Position within the property, starting at `1`. ¶ Positions stay contiguous: adding, moving or deleting an attribute shifts the others. |
| `attribute.property` | The property this attribute belongs to. |

## Product lines — `/v1/catalog/product-lines` ✅ finalized

Endpoints: list, create, retrieve, update, delete, bulk upsert. Internal export out of scope.

Resource
- [ ] Remove `unit_group` / `unit_group_id` (create, update, bulk) and the include key.
- [ ] `commission_policy`: `commission_applied | commission_exempt` → `applied | exempt`; `freight_policy`: `billed_freight | free_freight` → `billed | free`. Shared enums: also customers, account groups, account-group/customer product-line access.
- [ ] `default_lot_size` is a `Quantity` value: not expandable (remove `default_lot_size` / `default_lot.unit` include keys); still `null` for portal users.
- [ ] Remove `notes` (see "Notes → instructions"; commentary is the `note` resource, AI A6). `description` becomes writable: optional on create, `field.Clearable` on update; visible to portal users.
- [ ] ~~Read-only `type` (L4)~~ **superseded by the items review**: shipping/credit become order line types, so the reserved lines and system products go away. No `type` field.
- [ ] Keep `status: active | archived` (create/update, `statuses[]`); archived lines can't be assigned to new products or granted to customers.

`default_lot_size` rule
- [ ] Unit must share the dimension of the stocking unit of every product currently in the line (one indexed query at write). At planning, convert to each product's stocking unit; a product in another dimension skips the line's lot and falls back (line it feeds into → account default). Moving a product into the line doesn't re-check.

Delete
- [ ] 409 `resource_in_use` if referenced by `product`, a price-list rule's product-line scope, `product_line_target`, or `territory`. `account_group_product_line`, `account_relation_product_line` rows for the line are removed with it. Returns the stub.

Data cleanup
- [ ] Delete the reserved lines (4 system: Shipping, Service, Credit, Tax; 29 per-account Credit/Shipping copies) once shipping/credit products are migrated to order line types and the one service product moves to a standard line.

List
- [ ] `q`: substring on `name`. Drop FULLTEXT `product_line_name_idx`, `product_line_description_idx`, `product_line_name_description_idx`.

Docstrings

| Where | Text (summary ¶ description) |
|---|---|
| List | Returns a list of product lines. ¶ Includes your account's lines and the system lines every account shares. Lines are sorted by creation date, newest first. |
| · `q` / `statuses` | Case-insensitive substring match on `name`. / Only return lines with one of these statuses. Omit to return all statuses. |
| Create | Creates a product line. ¶ The new line belongs to your account and starts with no products. Set a product's line to add it. Customers and account groups can only be granted lines your account owns. |
| Retrieve | Retrieves a product line. ¶ Works for your account's lines and for system lines. |
| Update | Updates a product line. ¶ Only fields you send are changed. |
| Delete | Deletes a product line. ¶ Only lines your account created can be deleted, and only while no product, price-list rule, sales target or territory uses them. Archive a line that is in use by setting `status` to `archived`. Customer access to the line is removed with it. |
| `name` | Display name, such as `Outdoor`. ¶ Unique (case-insensitive) among your account's lines and the system lines. |
| `description` | Description of the line, shown to customers in the portal. |
| `commission_policy` | Whether sales of this line's products earn commission. ¶ `applied` or `exempt`. A customer's or account group's policy can override it. Always `null` for portal users. |
| `freight_policy` | Whether this line's products add a freight charge. ¶ `billed` or `free`. A customer's or account group's policy can override it. |
| `default_lot_size` | The lot this line's products are made in, such as one doff or one pallet. ¶ Sizes the lots a production plan proposes, and defaults the quantity of a batch added to a production run. An item's own lot takes precedence. Without one, planning uses the lot of the line the item feeds into, then the account default. Always `null` for portal users. |
| `fulfillment_policy` | How this line's products are produced when they don't say for themselves. ¶ `make_to_stock`: built to forecast, holding safety stock. `make_to_order`: built only against orders. `null` uses the account default. Always `null` for portal users. |

## Items, part 1 — structure ✅ decided (X1)

One public resource, `item`, with one ID (the item ID). Fields are reviewed in part 2.

- [ ] `type: material | part | product | service`, fixed at create. `material`, `part`, `product` are physical: stocked (inventory, `stocking_unit`), picked/packed/shipped, produced or consumed. `service` is never stocked, picked, packed, shipped or produced; it has no `stocking_unit` and its `order_units` follow `default_order_unit`'s dimension.
- [ ] Sections: `sales: { product_line, portal_visibility } | null` (present on sellable items: `product`, `service`); `material: { order_point, lead_time } | null`. Parts have no section (addable later). Sending a section that doesn't apply to the type → 422.
- [ ] **`product.type` removed.** Migration: `sale` → item `product`; `service` (1) → item `service`; `shipping` (30 system products, 18,501 order lines) and `credit` (30, 426 lines) → order line types (sales orders review); `tax` (1, unused) and `return` (0) dropped; returns and tax come back as their own flows later.
- [ ] Pick/pack/ship rule is structural: only `item` lines whose item type is physical are picked and packed. Shipping lines, credit lines and services are skipped.
- [ ] Routes: `GET/POST /v1/catalog/items`, `GET/PATCH/DELETE /v1/catalog/items/{id}`, `POST /items/actions/bulk-upsert` (typed rows), `POST /items/actions/set-order-units`, `POST /items/{id}/actions/change-stocking-unit`, lot-default sub-resource (part 2). Stock changes are inventory movements (Inventory N1).
- [ ] Remove `/v1/catalog/materials/*`, `/v1/catalog/parts/*`, `/v1/catalog/products/*`, `PUT /items/{id}/category/{category_id}` (→ PATCH `category_id`), `PUT /products/{id}/product-line/{product_line_id}` (→ PATCH `sales.product_line_id`), `PUT/DELETE /items/{id}/attributes/{attribute_id}` (→ PATCH `attribute_ids`, replaces the list).
- [ ] Data: `sales_order_line.product_id` dropped (it already has `item_id`); `supplier_material.material_id` → `item_id`; `product` and `material` stay as 1:1 extension tables keyed by `item_id` with their own IDs unexposed; drop the `part` table.
- [ ] List filters that only apply to sellable items (`product_line_ids`, `portal_visibilities`) match only products and services.
- [ ] Perf: one `item` row + 1:1 `LEFT JOIN`s on unique `item_id`; type-filtered lists use `item_account_type_created_idx`.

## Items, part 2 — fields ✅ finalized

Fields
- [ ] Remove `notes`. `description` clearable on update for every type.
- [ ] `unit_value` → `sales.unit_price` (`UnitPrice`; base sales-order price before price lists apply). Request and response both say `unit_price`.
- [ ] `unit_cost` (`UnitPrice`; internal + `costs:read`) and `burn_rate` (`Rate`; internal; read-only, computed) are always populated (value shapes aren't expandable). Every type can set `unit_cost` and `sales.unit_price` on create/update.
- [ ] `category_id` required on create, updatable; category `type` must match (`material` → materials; `product` → parts, products, services). Changing category no longer restates rates.
- [ ] `attribute_ids` on create/update (replaces the list); at most one attribute per property (422; prod has 0 violations); archived attributes rejected.
- [ ] `sales: { unit_price, product_line_id (clearable), portal_visibility (default hidden) }`; `material: { order_point, lead_time }` (each clearable; `lead_time` in a `time` unit).
- [ ] `status: active | archived`; archived items can't be added to new orders, production or purchasing.

Delete (replaces soft delete)
- [ ] Hard delete only while nothing references the item (order lines, inventory movements, lots, batches, production, planned orders, price-list rules, supplier materials — all indexed `item_id` probes); else 409 `resource_in_use`. Returns the stub.
- [ ] Migration: the 131 soft-deleted items (`deleted_at` set) become `archived`; then drop `item.deleted_at`.

List
- [ ] `q`: case-insensitive substring on `sku` or `description`, ranked exact SKU → SKU whole-word → SKU prefix → other, newest first within a rank (documented as contract). Drop FULLTEXT `item_sku_idx`, `item_description_idx`, `item_sku_description_idx`.
- [ ] New `skus[]` exact filter (case-insensitive, ≤100) on the unique `(account_id, sku)` key.
- [ ] Keep `types[]`, `category_ids[]`, `attribute_ids[]`, `product_line_ids[]`, `customer_ids[]` (the latter two match sellable items only). `portal_visibility` → `portal_visibilities[]`; `supplier_id` → `supplier_ids[]`; new `statuses[]`; `starts_at`/`ends_at` → `created_after`/`created_before`. `subassembly_filter` kept for now (revisit with production flows).

Bulk upsert
- [ ] One `POST /items/actions/bulk-upsert` replaces the three per-type ones: rows carry `type`, matched by `sku`, same fields as create (sections by type); attributes resolve by `(property, value)`. Async 202 + async job.

Deferred to the inventory review
- `GET/PATCH /items/{id}/inventory`, `GET /items/{id}/lot-default`, `POST /items/actions/bulk-reconcile`. Decided now: one vocabulary, `operation: adjust | reconcile` (bulk's `reconcile_type: addition | force` → `operation: adjust | reconcile`).

Docstrings

| Where | Text (summary ¶ description) |
|---|---|
| List | Returns a list of items. ¶ Sorted by creation date, newest first, or by search rank when `q` is set. |
| · `q` | Case-insensitive substring match on `sku` or `description`. ¶ Ranked: exact SKU first, then a SKU containing the term as a whole word, then a SKU starting with it, then all other matches. Newest first within each rank. |
| · `skus` | Only return items with exactly these SKUs (case-insensitive). At most 100. |
| · `customer_ids` | Only return sellable items at least one of these customers may order, through access to the item's product line. |
| Create | Creates an item. ¶ `type` can't change later. Send the `sales` section for products and services, and `material` for materials. |
| Retrieve | Retrieves an item. |
| Update | Updates an item. ¶ Only fields you send are changed. To change the stocking unit, use Change Stocking Unit. |
| Delete | Deletes an item. ¶ Only while nothing references it: no order, inventory, production, price or supplier record. Archive an item that's in use by setting `status` to `archived`. |
| `type` | What kind of item this is. ¶ `material`: bought and consumed in production. `part`: produced and used to make other items. `product`: a physical item you sell. `service`: something you sell that is never stocked, picked or shipped, such as installation. Can't change after creation. |
| `status` | Whether the item can be added to new orders, production and purchasing. ¶ `archived` items stay on the records already using them. |
| `sku` | Your stock keeping unit, unique (case-insensitive) within your account. |
| `description` | Description of the item. ¶ Shown to customers in the portal for visible products. |
| `category` | The category the item is classified under. |
| `attributes` | The item's attributes, at most one per property, such as `Size: Large`. |
| `unit_cost` | What one stocking unit of the item costs you. ¶ For items made by a production flow, it's restated from the flow when anything the item is built from changes. `null` without `costs:read`, and always `null` for portal users. |
| `burn_rate` | How fast the item is consumed, such as `120 pr / day`. ¶ Calculated from demand; read-only. Always `null` for portal users. |
| `sales` | Selling details, for products and services. `null` for materials and parts. |
| `sales.unit_price` | Base price on sales orders, before price lists apply. |
| `sales.product_line` | The product line the item is sold under. ¶ Customers are granted access to whole lines, so an item with no line is never shown in the customer portal. The line also sets the default commission and freight policies. |
| `sales.portal_visibility` | Whether customers with access to the item's product line can see and order it in the portal. ¶ Defaults to `hidden`. |
| `material.order_point` | Reorder when on-hand stock falls to this quantity. ¶ `null` when there is no order point. |
| `material.lead_time` | How long the material usually takes to arrive after it's ordered, in a time unit. ¶ `null` when unknown. |

## Customers — `/v1/sales/customers` ✅ finalized

Endpoints: list, create, retrieve, update, delete, merge, lead-time. Internal (out of scope): bulk-delete, export, frequently-ordered-products. Notification recipients: see Messaging M2.

Shape (request mirrors response — C1)
- [ ] Response: `status` (active | archived), `standing`, `name`, `number`, `relationship_type` (read-only), `instructions` (internal), `commission_policy` (internal), `freight_policy`, `credit_limit` (Money), `contact { email, phone, url }`, `carrier_billing { type: sender | third_party, account_number }`, `defaults { carrier, service_level, payment_term, shipping_term, sales_rep, bill_to_address, ship_to_address, receive_calendar, lead_time_days, fulfillment_policy (internal) }`, `group`, `price_lists`, `parent_customer`, `child_customers`, timestamps.
- [ ] Sections (`contact`, `carrier_billing`, `defaults`) are always populated, with no `object`; related objects inside them stay expandable.
- [ ] Request uses the same names: `defaults.carrier_id`, `group_id`, `price_list_ids`, etc. PATCH merges section fields individually. Create alone takes inline `bill_to_address` / `ship_to_address` (they become `defaults.*_address`).

Field changes
- [ ] `status` (`normal | preferred | hold_shipment | hold_all`) → `standing: normal | hold_shipment | hold_all` (drop `preferred`: 0 rows; prod: normal 3,464, hold_all 131, hold_shipment 1). Holds stay advisory.
- [ ] New lifecycle `status: active | archived`; archived customers can't receive new orders.
- [ ] `type` (an AccountGroup) → `group`; `customer_type_group_id` → `group_id`; `price_groups` / `customer_price_group_ids` → `price_lists` / `price_list_ids` (price lists review).
- [ ] `freight_preferences.status` → top-level `freight_policy` (`billed | free`); `freight_preferences.carrier` / `.service_level` → `defaults.carrier` / `.service_level`; `billing_type` / `billing_account` → `carrier_billing { type, account_number }`; drop the `freight_preferences` section.
- [ ] `contact_info` → `contact`; `parent_account` / `child_accounts` → `parent_customer` / `child_customers`; `bill_to_address` / `ship_to_address` → `defaults.*`; `defaults.receive_calendar_id` → `defaults.receive_calendar` (expandable).
- [ ] `default_priority` removed (see "Priority removed" under Fixed lookups). Urgency is `defaults.lead_time`.
- [ ] `credit_limit`: Quantity → Money, not expandable.
- [ ] `note` → `instructions` (notes decision).
- [ ] Remove `notification_preferences` (single derived boolean; replaced by `notification_recipients`, Messaging M2).

List
- [ ] `q`: substring on the displayed name (`alias`, falling back to the account name) or `number`. Remove `ar.notes` and the per-row `account_branding.support_email` subquery from the search.
- [ ] `status_codes` → `standings[]`; new `statuses[]`; `customer_group_ids` → `group_ids[]`; `pricing_group_ids` → `price_list_ids[]`; `commission_status_codes` → `commission_policies[]`; `freight_status_codes` → `freight_policies[]`; `parent_account_status` → `relationship_types[]`; `starts_at`/`ends_at` → `created_after`/`created_before`. Keep `sales_rep_ids`, `shipping_term_ids`, `payment_term_ids`, `carrier_ids`, `service_level_ids`, `city`, `state`, `postal_code` (document: match any of the customer's addresses).

Merge
- [ ] Returns 202 + async job (today sync 200); the target customer is in the async job result.

Delete
- [ ] 409 `resource_in_use` (today `resource_conflict`) when sales orders, invoices, shipments or child customers reference it (the customer's own price list is deleted with it); returns the stub.

Docstrings

| Where | Text (summary ¶ description) |
|---|---|
| List | Returns a list of customers. ¶ Sorted by creation date, newest first. |
| · `q` | Case-insensitive substring match on `name` or `number`. |
| · `city` / `state` / `postal_code` | Only return customers with at least one address in this city / state / postal code. |
| Create | Creates a customer. ¶ Give its first billing and shipping addresses inline. They become the customer's default addresses. If `number` is omitted, the next number in your sequence is assigned. |
| Update | Updates a customer. ¶ Only fields you send are changed, including fields inside `contact`, `carrier_billing` and `defaults`. Send `null` to clear a clearable field. |
| Delete | Deletes a customer. ¶ Only while no order, invoice, shipment or child customer references it. Its own price list is deleted with it. Archive it instead, or merge it into another customer. |
| Merge | Merges customers into this one. ¶ Orders, invoices, shipments and deliveries of the source customers move to this customer. Group memberships, product line access, addresses and users are combined; the source customers' own price lists are discarded, and the sources' child customers move under this one. Notification recipients of the sources are discarded. This customer keeps its own name, number and defaults. The source customers are deleted. Runs in the background; the response is a job to poll. |
| `status` | Whether the customer can receive new orders. ¶ `archived` customers stay on their existing records. |
| `standing` | The customer's credit standing. ¶ `normal`: no restrictions. `hold_shipment`: hold their shipments; orders can still be placed. `hold_all`: hold all activity. Holds flag orders but never reject them. |
| `relationship_type` | Where the customer sits in an account hierarchy. ¶ `standalone`, `parent` (has `child_customers`) or `child` (has a `parent_customer`). Read-only. |
| `freight_policy` | Whether this customer is billed for freight. ¶ Freight is waived if this, the customer's group, or the product line of an ordered item is `free`. |
| `credit_limit` | Credit extended to this customer. ¶ Orders are flagged once the outstanding balance passes it, never rejected. `null` means no limit. |
| `carrier_billing` | Who pays the carrier. ¶ `sender`: you pay. `third_party`: the carrier bills `account_number`. |
| `defaults` | Values copied onto a new order for this customer when the order doesn't set them. |
| `defaults.lead_time` | Business days (on your working calendar) from an order being issued to it being due to ship. ¶ `null` inherits from the parent customer, then the group, then your account default. Get Customer Lead Time shows the result. |
| `group` / `price_lists` | The account group that classifies this customer, such as `Distributors`. / Price lists assigned directly to this customer; they take precedence over its parent's and group's lists on equally specific rules. |

## Account groups — `/v1/sales/account-groups` ✅ finalized

Endpoints: list, create, retrieve, update, delete. Product-line access for groups is internal (out of scope).

- [ ] ~~`type` → `classification | pricing`~~ **superseded by price lists**: pricing groups are removed, so account groups are classification-only and the `type` field (and `types[]` filter) is removed.
- [ ] Add `price_lists` (expandable List; request `price_list_ids`), applying to every customer in the group.
- [ ] `default_lead_time_days` → `defaults.lead_time` (mirrors `customer.defaults.lead_time_days`).
- [ ] Drop group-level receive calendar and fulfillment policy from the inheritance chains (columns `receive_calendar_id`, `fulfillment_policy_code` are unexposed and unused: 0 rows). Customer chains become: customer → account default. Update the customer docstrings to drop the group step for those two fields.
- [ ] Policies take the L1 values (`applied | exempt`, `billed | free`).
- [ ] Delete: 409 `resource_in_use` (today a validation error) while the group is any customer's `group`, or backs product-line access or a registration flow. Price list assignments are removed with the group. Returns the stub. No `status`.
- [ ] `q`: substring on `name` or `description` (documented).

Docstrings

| Where | Text (summary ¶ description) |
|---|---|
| List | Returns a list of account groups. ¶ Sorted by creation date, newest first. |
| · `q` | Case-insensitive substring match on `name` or `description`. |
| Create | Creates an account group. |
| Update | Updates an account group. ¶ Only fields you send are changed. New policies apply to every customer already in the group. |
| Delete | Deletes an account group. ¶ Only while no customer has it as their `group`, and it backs no product line access or registration flow. Its price list assignments are removed with it. |
| `name` | Display name, unique (case-insensitive) among your account's groups. |
| `commission_policy` | Whether orders from the group's customers earn commission. ¶ A customer is exempt if it or its group is `exempt`. Always `null` for portal users. |
| `freight_policy` | Whether the group's customers are billed for freight. ¶ Freight is waived if the customer, its group, or the product line of an ordered item is `free`. |
| `defaults.lead_time` | Lead time for customers in this group that don't set their own or inherit one from a parent customer. ¶ Calendar days from an order being issued to it being due to ship. |

## Fixed lookups (E1) ✅ decided

- [ ] Remove `GET /v1/sales/account-statuses[/{id}]` (replaced by the customer `standing` enum) and `GET /v1/sales/priorities[/{id}]`.
- [ ] **Priority removed (decided 2026-10-08).** `priority` is dropped from customers (`defaults.priority`), sales orders and purchase orders, with no enum replacement. It only labeled records (shown on picks, shipments, invoices, the acknowledgment PDF and the customer export); nothing scheduled or sorted by it. "Ship this customer's orders in 1 day" is the customer's `defaults.lead_time`, and an order's urgency is its `commitment` (`promised_at`, `lead_time_override`, `ship_by_override_date`, giving a ship-by date). Documents and lists show the ship-by date where they showed priority. The `priority` table and columns are dropped after the migration.
- [ ] Revisit in their domain reviews: `/v1/sales/sales-order-statuses`, `/v1/finance/adjustment-types`, `/v1/finance/transaction-methods`, `/v1/finance/transaction-types`, `/v1/operations/demand-override-types`, `/v1/operations/machine-downtime-reasons`, `/v1/operations/schedule-deviation-types`, `/v1/operations/location-types[/{id}]`, `/v1/operations/machine-status`. Remove each unless accounts can add values.

## Addresses & contacts ✅ finalized

Endpoints after: `GET/POST /v1/core/addresses`, `GET/PATCH/DELETE /v1/core/addresses/{id}`, `POST /v1/core/addresses/actions/validate`, `POST /v1/sales/contacts/actions/find-by-email`. Internal: `GET /v1/core/addresses/suggestions` (was public), `GET /v1/core/addresses/details/{id}`.

Address
- [ ] Flat shape, one vocabulary: `{ id, object, name, type, line_1, line_2, city, state, postal_code, country, phone, email, receive_calendar, created_at, updated_at }`. Remove the nested `geolocation` object (and its `id`/`object`); `street_line_1/2` → `line_1/2`, `locality` → `city`. Same names on create, update, validate and every embedded address. The `geolocation` table stays behind it as storage (24,037 addresses / 22,335 geolocations).
- [ ] `country`: uppercase ISO 3166-1 alpha-2, validated (422). All prod rows already comply. `state`: free text, documented as the two-letter code in the US and Canada.
- [ ] Move CRUD from `/v1/sales/addresses` to `/v1/core/addresses` (one namespace with validation).
- [ ] `receive_calendar_id` → expandable `receive_calendar`; chain: address → customer → account (group step dropped per G2).
- [ ] Delete: 409 `resource_in_use` when an order, invoice or shipment, or any customer/supplier `defaults.*_address`, uses it; returns the stub. No `status`.
- [ ] List: `type` → `types[]`; `q`: substring on `name`, `line_1`, `line_2`, `city`, `state`, `postal_code`, `country` (≤ ~1,200 rows per account; EDI address matching relies on it).
- [ ] Validate request takes the address field names (`address_line_1` → `line_1`, etc.).
- [ ] `GET /v1/core/addresses/suggestions` becomes internal (paid Google Places proxy).

Documents snapshot their address (AD7, option A)
- [ ] Sales orders, invoices and shipments (and purchase orders / receiving where they carry addresses) embed the address as a value with the address fields, copied at document creation; they keep an expandable `address` reference only as the source. Editing an address affects future documents only. Today `sales_order.billing_address_id` / `shipping_address_id`, `invoice.billing_address_id`, `shipment.shipping_address_id` reference the live row, so edits rewrite history. Backfill snapshots from current rows (already-edited history can't be recovered). Exact shape is settled in the sales orders / invoices / shipments reviews.

Contacts
- [ ] Keep `POST /v1/sales/contacts/actions/find-by-email` (email stays out of the URL); `relationships[]` filter stays. Contact shape reviewed with account users; contacts get no portal access or email unless invited (Q1).

Docstrings

| Where | Text (summary ¶ description) |
|---|---|
| List | Returns a list of addresses. ¶ Returns the addresses of the account you're acting in, newest first. |
| · `q` / `types` | Case-insensitive substring match on `name`, `line_1`, `line_2`, `city`, `state`, `postal_code` or `country`. / Only return addresses of these types. |
| Create | Creates an address. ¶ Saved to the account you're acting in: yours, or a customer's or supplier's you manage. Use it as a billing or shipping address on orders, invoices and shipments. |
| Update | Updates an address. ¶ Only fields you send are changed. Documents already created keep the address as it was when they were created. |
| Delete | Deletes an address. ¶ Only while no order, invoice or shipment uses it and it isn't a customer's or supplier's default. |
| Validate | Checks an address and returns a standardized version. ¶ Nothing is saved. Use it before creating or updating an address to catch missing or misspelled parts. |
| `name` | A label for the address, such as `Main warehouse`. |
| `type` | How the address is used. ¶ `standard`: a normal billing or shipping address. `drop_ship`: shipped to directly on the customer's behalf, such as an end customer. |
| `line_1` / `line_2` | Street address. / Apartment, suite, unit or building. |
| `city` / `state` / `postal_code` | City or locality. / State, province or region; in the US and Canada, the two-letter code. / Postal or ZIP code. |
| `country` | Two-letter country code (ISO 3166-1 alpha-2), such as `US`. |
| `receive_calendar` | Days this location accepts freight. ¶ Set it when one site keeps different days from the rest. `null` uses the customer's calendar, then your account default. |

## Pricing ✅ finalized (redesigned: price lists)

End state: **price lists** (`/v1/sales/price-lists`) and **order discounts** (`/v1/sales/order-discounts`) are the only pricing resources. Account prices, pricing account groups and volume discounts are removed.

Shared
- [ ] **One `Discount` value shape** for things taken off: `{ type: "percent_off", percent_off: "decimal" }` (whole-number percent, `"5"` = 5%, 0–100) or `{ type: "amount_off", amount_off: Money }`. Used by order-unit discounts and order discounts. Migration multiplies today's fractions (`"0.05"`) by 100.

Price lists
- [ ] `price_list`: `{ id, object, status (active | archived), name, description, rules (expandable List; first 10 + next_page_url), created_at, updated_at }`.
- [ ] `price_list_rule` (sub-resource with IDs): `{ id, object, scope { items, product_lines, categories, attributes } (expandable Lists; empty = any; item must have every attribute), price, created_at, updated_at }`. Requests: `scope.item_ids`, `product_line_ids`, `category_ids`, `attribute_ids`.
- [ ] `price` types: `{ type: "fixed", unit_price: UnitPrice }`; `{ type: "percent_of_base", percent_of_base: "decimal > 0" }` (`"80"` = 20% off, `"235.3"` = markup; no signs); `{ type: "tiered", unit: Unit, tiers: [{ min_quantity, percent_of_base | unit_price }] }` (quantity = total across every order line in the rule's scope, converted into `unit`; highest reached tier sets the price; tiers sorted, max 20).
- [ ] Assignment: `customer.price_lists` (request `price_list_ids`, replaces `price_groups`), `account_group.price_lists` (applies to every member), and a parent customer's lists apply to its child customers.
- [ ] Pricing pipeline: base `sales.unit_price` → **one** price-list rule (most specific scope: item > attributes and/or category > product line > any; tie → customer's own list > parent's > group's; tie → newest rule) → convert to the ordered unit + order-unit discount → order discount on the total. Matches today's one-winner behavior (an account price replaced everything).
- [ ] Endpoints: `GET/POST /v1/sales/price-lists`, `GET/PATCH/DELETE /v1/sales/price-lists/{id}`, `GET/POST /v1/sales/price-lists/{id}/rules`, `GET/PATCH/DELETE /v1/sales/price-lists/{id}/rules/{rule_id}`. Later, additive: bulk upsert of rules (price-sheet imports), an account-level "applies to all customers" default list. The internal price-list export moves here.
- [ ] List filters: price lists `statuses[]`, `customer_ids[]` (applying directly or inherited), `q` on `name`; rules `item_ids[]`, `product_line_ids[]`.
- [ ] Delete a price list: 409 `resource_in_use` while assigned to a customer or group; archive instead (archived lists stop applying immediately). Rules delete freely (orders keep their price). Stub responses.
- [ ] Perf: a customer's rules load once per order with the existing pricing bundle (hundreds of rules in prod); no per-line queries.
- [x] Confirmed: the precedence above (no list `priority`), and rules as a sub-resource with IDs.

Migration
- [ ] 152 account prices (62 customers) → one customer-specific list per customer with `fixed` rules (scope: product line, categories now counting, attributes).
- [ ] 3 pricing groups (default policies, no product-line access) → price lists of the same name, assigned to their members; their 20 flat single-tier (threshold 0) volume discounts → `percent_of_base` rules with `100 × (1 − fraction)` (6 are markups, e.g. −135.3% → `"235.3"`).
- [ ] Flat discounts scoped to classification groups → lists assigned to those groups.
- [ ] 2 real tiered volume discounts → `tiered` rules (`unit` = their carton unit; tier percents rewritten to today's cumulative effect so prices don't change).
- [ ] Drop `account_price*`, `quantity_discount*` (incl. 50 orphaned unit rows), pricing account groups and their memberships.

Order discounts
- [ ] `discount_type` + `percentage` + `amount` → `discount` (Discount shape; `amount_off` is Money).
- [ ] Add `status: active | archived`; archived codes can't be applied to new orders (find-by-code returns 404 to portal users).
- [ ] `order_count` → `times_redeemed`, from a stored counter.
- [ ] Keep `POST actions/find-by-code`. List: `statuses[]`; `q` substring on `name` or `code`. Later, additive: `expires_at`, `max_redemptions`.

Docstrings

| Where | Text (summary ¶ description) |
|---|---|
| Price list | A set of pricing rules you assign to customers or account groups. ¶ A customer's line price comes from the most specific matching rule across the lists that apply to it: its own, its parent customer's, and its group's. |
| `status` | Whether the list's rules apply. ¶ `archived` lists stop applying to new orders immediately. Orders already priced keep their prices. |
| Rule | A price for a scope of items. ¶ Empty scope lists match any item. An item must have every listed attribute. |
| `rule.price` | The price this rule sets. ¶ `fixed`: exactly `unit_price`. `percent_of_base`: that percent of the item's `sales.unit_price`, so `"80"` is 20% off and `"120"` is a 20% markup. `tiered`: the price depends on the total quantity ordered across every line in the rule's scope, counted in `unit`; the highest tier reached applies. |
| `customer.price_lists` | Price lists assigned directly to this customer. ¶ They take precedence over its parent's and its group's lists on equally specific rules. |
| `account_group.price_lists` | Price lists that apply to every customer in this group. |
| Order discount | A code a buyer enters to take money off an order. ¶ Codes are unique within your account and matched case-insensitively. |
| `discount` | How much comes off. ¶ `percent_off`: a whole-number percent of the order total. `amount_off`: a fixed amount. |
| `times_redeemed` | How many orders this code has been applied to. |

## Sales orders, part 1 — the order ✅ finalized

Lines, totals, quotes, checkout, production run and bulk endpoints are part 2.

Shape
- [ ] **One name per concept (S1):** `buyer_account_id` (create) / `customer_id` (update) → `customer_id`; `bill_to_address_id` / `billing_address_id` / `bill_to_address` / `billing_address` → `bill_to` section (same for ship-to); `priority_code` removed (priority dropped; see Fixed lookups); `carrier_id`, `service_level_id`, `carrier_billing_type`, `carrier_billing_account_number` → `freight { carrier_id, service_level_id, carrier_billing { type, account_number } }`; `promised_at`, `lead_time_override`, `ship_by_override_date` → `commitment { … }`; `acknowledgement_*` → `acknowledgment_*` (US spelling everywhere). `customer_id` can change only while `estimate`.
- [ ] `freight.policy` uses `billed | free`; `freight.billing_type` / `billing_account_number` → `freight.carrier_billing { type, account_number }` (matches customers).
- [ ] **Address snapshots (S2):** `bill_to` / `ship_to` = the address fields (`name`, `line_1`, `line_2`, `city`, `state`, `postal_code`, `country`, `phone`, `email`) copied when set, plus `address` (expandable reference to the saved source; `null` for inline). Request: `{ address_id }` or inline fields. Shipments copy `ship_to`, invoices copy `bill_to`, each at creation.
- [ ] **(S3)** `note` → internal `instructions` (copied from the customer when omitted on create). `contacts` → `email_recipients: [{ account_user (expandable), documents: [acknowledgment | invoice | shipment] }]`, the same shape in requests and responses and the same as the customer's `notification_recipients` (Messaging M2), which it is copied from when omitted.
- [ ] **Keep `related` with lightweight `Record` stubs (S4 rejected):** `related { pick, jobs, shipments, invoices }` stays, returning small `record` objects instead of full resources, so clients don't pull large sub-objects. (Convention below.)
- [ ] Omitted values on create come from the customer's `defaults` and are copied onto the order.

Lifecycle actions (S5)
- [ ] `PUT …/actions/issue|unissue|close|open` → `POST …/actions/issue | unissue | cancel` (`draft → issued`, `issued → draft`; `fulfilled` is derived from shipments, Q5). `issue` keeps `notify_customer` (request boolean is fine). `status` read-only.

Fields (S6)
- [ ] `completed_at` → `fulfilled_at`; `first_ship_at` → `first_shipped_at`; `expired_at` → `expires_at`.
- [ ] Remove `payment_intent_ids` (payments reviewed with checkout). Remove the unreferenced `SalesOrderType` / `SalesOrderStatusDetail` shapes; `/v1/sales/sales-order-statuses` removed (E1).

Delete
- [ ] **Not deletable once any shipment or invoice exists** (409 `resource_in_use`); unissue or fulfill instead. Today delete is blocked only when fulfilled and removes shipment and invoice lines. Bulk delete follows the same rule. Returns the stub.

List
- [ ] `q`: **exact** match on `number` or `customer_purchase_order_number` (documented).
- [ ] `status_codes` → `statuses[]`; `customer_group_ids` → `group_ids[]`; `starts_at`/`ends_at` → `created_after`/`created_before`; keep `item_ids`, `product_line_ids`, `customer_ids`, `sales_rep_ids`, `ship_by_after`, `ship_by_before`, `past_due`; new `payment_statuses[]`.

Docstrings

| Where | Text (summary ¶ description) |
|---|---|
| List · `q` | Exact match on `number` or `customer_purchase_order_number`. |
| Create | Creates a sales order as a `draft`. ¶ Anything you omit is copied from the customer's `defaults`: addresses, carrier, terms, lead time, sales rep and instructions. Issue it to start fulfillment. |
| Delete | Deletes a sales order. ¶ Only while it has no shipment or invoice. Releases any reserved inventory. |
| `status` | Where the order is in its lifecycle. ¶ `draft`: not yet committed. `issued`: being fulfilled. `fulfilled`: every line shipped or closed (set automatically). `canceled`: stopped before anything shipped. Issue, unissue and cancel are the only actions; `fulfilled` follows the shipments. |
| `bill_to` / `ship_to` | The address as it was when set on this order. ¶ `address` is the saved address it came from, if any. Later changes to that address don't affect this order. |
| `payment_status` | How much of the order has been paid. ¶ Derived from invoices, payments and settlements; read-only. |
| `acknowledgment_status` | Whether an order acknowledgment was sent. ¶ Becomes `sent` when the order is issued with `notify_customer` and has acknowledgment recipients. Set it yourself when you acknowledged the order outside OpenMRP. |
| `commitment` | When the order is due to ship: what was promised or pinned, the resulting ship-by date, and which rule decided it. |
| `related` | Lightweight references to the records produced from this order: its pick, production run, shipments and invoices. ¶ Retrieve a record by its ID for full details. |
| Issue / Unissue / Cancel | Issues the order: creates its pick and reserves inventory. / Returns it to `draft`: deletes the pick and releases inventory. / Stops the order: closes whatever hasn't shipped and releases its inventory. Becomes `canceled` if nothing shipped, otherwise `fulfilled`. |

## Sales orders, part 2 — lines, totals, helpers ✅ finalized

Lines (`/v1/sales/sales-orders/{id}/lines`)
- [ ] `type: item | shipping` (no `credit`; the order discount is `totals.discount`, not a negative line).
- [ ] Remove `product`; requests send `item_id`. `product_sku` → `sku`, `product_description` → `description` (copied from the item, editable), `quantity_ordered` → `quantity`, `line_item_number` → `line_number`.
- [ ] `quantity` (Quantity), `unit_price` (UnitPrice), `unit_cost` (UnitPrice; internal + `costs:read`), `amount` (Money) are always populated, never expandable.
- [ ] Unit must be the item's stocking unit or an order unit; defaults to `default_order_unit`. Services allowed and never picked.
- [ ] `totals` (money stages + `completion` floats) → `fulfillment { picked, packed, shipped, invoiced }` as Quantities.
- [ ] Shipping lines: `{ type: "shipping", description, amount }`, set from a freight quote or by hand; always sorted below item lines.
- [ ] `unit_price` on item lines comes from price lists; staff may override on create/update, portal users can't.
- [ ] Delete: 409 `resource_in_use` once any quantity is packed or shipped; remove the admin-only override after shipping. Editing a line never changes invoices (snapshots). Adding a line to an issued order adds pick work.

Totals
- [ ] Order `totals` (always populated; Money): `subtotal` (item lines), `shipping` (shipping lines), `discount` (order discount), `tax` (0 until tax exists), `total`, `invoiced`, `paid` (from the order's invoices). Replaces `ordered` + per-stage `{amount, completion}` strings.
- [ ] Shipping charges stay **lines** (`type: shipping`; several allowed, e.g. per shipment), mirrored onto invoices; no order-level `freight.amount`.

Helper endpoints
- [ ] `POST /sales-orders/price-quote` → `POST /sales-orders/actions/preview-prices` `{ customer_id, lines: [{ item_id, quantity }] }`; response lines return `unit_price` (UnitPrice, was `ComputedRate`) in request order.
- [ ] `POST …/{id}/actions/estimate-freight` returns `amount` (Money) for the shipping line, not a `ComputedRate`.
- [ ] `POST …/actions/preview-commitment` request uses order names: `customer_id`, `ship_to { address_id }`, `freight { carrier_id, service_level_id }`, `commitment { … }`, `issued_at`.
- [ ] `POST …/{id}/checkout {email} → {checkout_url}` → `POST …/{id}/actions/create-checkout-session {email} → {url, expires_at}`; successful payments become `payment` records.
- [ ] Remove internal `POST /v1/sales/checkout-sessions` (takes `order_total_cents` from the client; amounts must be server-computed).
- [ ] `POST …/{id}/actions/create-production-run` → `create-jobs` (returns a List of jobs; see Jobs, work orders and traceability).
- [ ] `POST …/actions/bulk-delete` returns a list of deleted stubs; max 100 IDs; atomic; each order must pass the delete rule.
- [ ] `POST …/{id}/lines/actions/reorder` returns the reordered lines (List); shipping lines stay at the bottom.

Docstrings

| Where | Text (summary ¶ description) |
|---|---|
| `line.type` | What the line charges for. ¶ `item`: an item at a quantity and unit price. `shipping`: a freight charge, as an amount. |
| `line.unit_price` | Price per unit for this line. ¶ Set from the customer's price lists when the line is added. Your team can override it; portal users can't. |
| `line.amount` | `quantity` × `unit_price`, or the freight charge for shipping lines. |
| `line.fulfillment` | How much of the line has been picked, packed, shipped and invoiced, in the line's unit. |
| Add line | Adds a line to a sales order. ¶ Item lines go above shipping lines. On an issued order, the line is added to the pick as outstanding work. |
| Delete line | Deletes a line. ¶ Only while none of it has been packed or shipped. |
| `totals` | The order's money totals. ¶ `subtotal` covers item lines, `shipping` covers shipping lines, `discount` is the order discount, and `total` is what the customer owes for the order. `invoiced` and `paid` are drawn from its invoices. |
| Quote Prices | Prices items for a customer without creating an order. ¶ Uses the same price lists as a real order line. |

## Accounts receivable documents ✅ finalized

Replaces the "Accounting documents (direction)" notes. All of these become **public** at forge.1 (every finance route is internal today). Namespace `finance`.

Model
- [ ] **Invoice** (debit, `/v1/finance/invoices`): `status: draft | open | paid | void | uncollectible`; `source: shipment | order | manual`; `customer`; `bill_to` (address snapshot); `issued_at`, `due_at` (from the payment term); `lines[]` (`type: item | shipping`, `sku`, `description`, `quantity`, `unit_price` UnitPrice, `amount` Money, `order_line` reference) snapshotted at issue; stored `totals { subtotal, discount, shipping, tax, total, amount_paid, amount_credited, amount_due }` (Money); `related` record stubs (sales order, shipment); `metadata`.
  - `metadata` (and delivery status) stay updatable on finalized invoices; nothing that affects amounts does. Both are **per party** (stored in `document_party`; see AP).
  - Lifecycle: `draft` (editable) → `POST …/actions/finalize` → `open` (immutable; number assigned) → `paid` (amount_due reaches 0) | `void` (`actions/void`, only with nothing applied, with a reason) | `uncollectible` (`actions/mark-uncollectible`). **Never deleted; no DELETE endpoint.** Shipment invoices are created and finalized in one step.
  - **Invoicing ahead of shipment**: `source: order` bills order lines (or part of their quantity) before shipping; later shipments invoice only the uninvoiced quantity. A deposit is an `order` invoice with one amount line, applied to the final invoice like a payment.
  - **Other debits** (charges with no shipment, upward corrections): `source: manual` invoices. No separate debit-note document.
  - **Numbering**: new invoices use their own per-account invoice sequence; existing numbers (today = shipment number) are kept.
  - Voiding a shipment voids its invoice if nothing is applied; otherwise a credit note is required. It never deletes the invoice.
- [ ] **Credit note** (credit, `/v1/finance/credit-notes`): `status: issued | void` (void only while unapplied); `reason: discount | rebate | return | price_adjustment | short_payment | shipping_discrepancy | customer_fee | write_off | other`; optional `invoice` it corrects; `lines[] { description, amount, invoice_line }`; `total`, `amount_applied`, `amount_remaining`; `issued_at`. Applied to invoices, kept as customer credit, or refunded. Immutable.
- [ ] **Payment** (`/v1/finance/payments`; shared with payables, `direction` relative to the viewer — see AP): `amount` Money, `method` (`cash | check | card | ach | …`), `reference`, `description`, `received_at`, `funds_available_at`, `status: succeeded | reversed`, `source: manual | stripe`. Amount immutable; `actions/reverse` for bounced checks / failed cards (recorded, never deleted). Create accepts `applications[]` and `deductions[]` (each deduction creates a credit note with its reason), replacing settlements.
- [ ] ~~**Refund** resource~~ **folded into payments** (AP review): a refund is a `payment` with `type: refund` (outbound to a customer, inbound from a supplier). Immutable.
- [ ] **Application** (`/v1/finance/applications`): applies a payment or credit note to an invoice; can be unapplied and reapplied, with history kept (audit). Unapplied amounts form the customer's credit balance (computed; replaces the open-credits list).
- [ ] **Receivables**: read-only aging view from `amount_due` and unapplied credit.
- [ ] Stripe webhooks create and reverse payments; they never delete transactions (today `canceled` hard-deletes).
- [ ] `has_been_sent` / `is_edi_sent` booleans → an enum (e.g. `delivery_status: not_sent | sent`) per the no-booleans rule; `is_paid_in_full` / `is_over_paid` replaced by `status` and `totals`. `accepts_invoice_emails` removed (customer `notification_recipients`, Messaging M2).

Sales order effects
- [ ] Order lines are `item | shipping` only; credits are credit notes, not order lines. The order discount is an order-level amount (and an invoice `discount` total), not a negative line.
- [ ] Deleting or editing sales orders and lines never touches invoices (snapshots).

Migration (prod: 125,762 invoices; 28,350 transactions; 37,599 allocations; 15,076 settlements)
- [ ] Invoices: snapshot lines and totals from current order-line prices (history already drifted by repricing can't be recovered); `paid` when balance 0, else `open`; 33 overpaid → `paid` with the excess as customer credit.
- [ ] `payment` transactions (14,710) → payments.
- [ ] `credit_memo` (402), `rebate` (5,874), and `adjustment` `discount` (6,769), `write_off` (211), `short_payment` (44), `shipping_discrepancy` (77) → credit notes with the matching reason.
- [ ] `adjustment` `fee` (196) → credit notes with `reason: customer_fee` (a customer charged a fee for missed contract terms and deducted it).
- [ ] `adjustment` `refund` (67) → payments with `type: refund` (money sent back); verify each one's allocations so invoice balances are preserved.
- [ ] Allocations → applications; settlements → payments (settlement number kept as `reference`).
- [ ] Order `credit` lines (426) → credit notes; order discount negative lines → order-level discount.
- [ ] Drop `transaction`, `transaction_allocation`, `settlement` once migrated; `/v1/finance/transaction-types`, `/transaction-methods`, `/adjustment-types` removed (E1: enums on the new resources).

## Purchasing ✅ finalized

Suppliers, supplier items, purchase orders and receiving orders become **public** at forge.1 (all 30 routes are internal today). Namespace `operations`.

Carried over
- [ ] Suppliers take the customer shape (request mirrors response): `status` (active | archived), `name`, `number`, `instructions` (internal; was `note`), `contact`, `defaults { bill_to_address, ship_to_address, payment_term, shipping_term, carrier, service_level, … }`. Delete blocked while purchase orders or supplier items reference it.
- [ ] Purchase orders take the sales order shape: `bill_to` / `ship_to` snapshots + `address` reference; `instructions` (internal; copied from the supplier when omitted); `freight { carrier, service_level, carrier_billing }`; no `priority`; `contacts` → `email_recipients`; `related` stubs (receiving order, deliveries).
- [ ] Lines: `item_id`; `sku`, `description`, `quantity`, `line_number`; `unit_price` UnitPrice, `amount` Money (not expandable); unit = stocking unit or an order unit, default `default_order_unit`.
- [ ] Remove `GET /v1/operations/purchase-orders/statuses` (E1). Plural filters, `created_after` / `created_before`, deleted stubs apply.

Changes
- [ ] **U2** `status`: `estimate | issued | fulfilled` (sales enum) → `draft | issued | completed`. `PUT …/actions/change-status { status_change, send_email }` → `POST …/actions/issue { notify_supplier } | unissue | complete | reopen`.
- [ ] **U3** `scheduled_at` → `promised_at` (one name in request and response). `totals` (Money): `subtotal`, `total`, `received`. Lines: `fulfillment { received, stocked, rejected }` (Quantities) replaces `quantity_received` (ComputedQuantity); `delivery_lines` → `related`. Delete blocked once any line is received (409).
- [ ] **U4** Supplier materials → **supplier items**: `/v1/operations/suppliers/{supplier_id}/items`, object `supplier_item`, `item` (was `material`), `supplier_sku` (was `supplier_part_number`), `supplier_description`, `status: active | archived` (was `inactive`). Later, additive: supplier-specific price and order units.
- [ ] **U5** Remove `supplier.material_count`.
- [ ] **U6** Receiving actions: `PUT …/actions/receive` → `POST …/actions/receive-all`; `PUT …/lines/{id}/actions/receive` → `POST …/lines/{id}/actions/receive-all`; `PUT …/actions/void` → `POST …/actions/reset`; `PUT …/lines/{id}/actions/void` → `POST …/lines/{id}/actions/reset`; `PATCH …/lines/{id}` and `POST …/actions/stock` kept. **Stocking is final**: stocked lines can't be reset (409; today resetting a completed order allows double-stocking); corrections go through inventory adjustments. Receiving line `quantity` → `received_quantity` (alongside `rejected_quantity`, `ordered_quantity`).

Docstrings

| Where | Text (summary ¶ description) |
|---|---|
| PO `status` | Where the purchase order is in its lifecycle. ¶ `draft`: not yet sent. `issued`: sent to the supplier and being received. `completed`: everything received or closed (set automatically). `canceled`: stopped before anything was received. Issue, unissue and cancel are the only actions. |
| Issue PO | Issues the purchase order. ¶ Opens a receiving order with a line for each PO line, ready to receive. Set `notify_supplier` to email it to the supplier's recipients. |
| `promised_at` | The delivery date the supplier promised. |
| Receive all | Marks everything still outstanding as received. ¶ Received goods aren't in inventory until they're stocked. |
| Stock | Books received quantities into inventory. ¶ Give lot numbers and any rejected quantity per line. Stocking is final; correct mistakes with an inventory adjustment. |
| Reset | Clears receiving progress that hasn't been stocked. ¶ Received quantities return to 0 and the order reopens. Stocked lines can't be reset. |
| `supplier_sku` | The supplier's own code for this item, as it appears on their documents. |

- [ ] **U7** Supplier bills / accounts payable are designed for forge.1 (see "Accounts payable").

## Accounts payable ✅ finalized (two-sided documents)

Payables is receivables read from the buyer's side. **No mirrored tables**: one row per real-world document, read by both parties (the same pattern as `sales_order`, which the seller reads as a sales order and the buyer as a purchase order).

Storage
- [ ] `invoice` (reworked; see AR): `seller_account_id`, `buyer_account_id`, `owner_account_id`, status, snapshotted lines and stored totals.
- [ ] `credit_note` (new): same three party columns.
- [ ] `payment` (new): `payer_account_id`, `payee_account_id`, `owner_account_id`.
- [ ] `application` (new): links a payment or credit note to an invoice; inherits the document's parties.
- [ ] `document_party` (new, small): one row per party per document holding that party's own `metadata`, delivery status (seller), internal instructions, and later per-party workflow such as bill approval (buyer).
- [ ] Replaces `transaction`, `transaction_allocation`, `settlement` (migrated, then dropped).

Views (two collections over one table; same ID from both sides)
- [ ] Seller: `/v1/finance/invoices` (object `invoice`, counterparty field `customer`). Buyer: **`/v1/finance/bills`** (object `bill`, counterparty field `supplier`, `number` shown as `supplier_invoice_number`).
- [ ] Seller: `/v1/finance/credit-notes` (`credit_note`). Buyer: **`/v1/finance/supplier-credits`** (`supplier_credit`).
- [ ] `/v1/finance/payments`: one resource for both sides; `direction: inbound | outbound` is computed relative to the viewer; `type: payment | refund` (**the separate `refunds` resource from the AR review is folded in**). `counterparty { type: customer | supplier, … }`.
- [ ] `/v1/finance/applications`: visible to both parties.
- [ ] `/v1/finance/receivables` and `/v1/finance/payables`: read-only aging views.

Authorship and rights
- [ ] When the counterparty uses OpenMRP, the issuer (seller) authors the invoice and it appears in the buyer's bills automatically (no data entry; both sides always agree).
- [ ] When the counterparty doesn't (a placeholder account, as with most suppliers today), you author the document on their behalf: `owner_account_id` = you, e.g. entering a supplier's invoice as a bill.
- [ ] Only the owner can edit a draft, finalize or void. The counterparty reads, and writes only its own `document_party` data (`metadata` etc.).
- [ ] `metadata` returned to each party comes from that party's `document_party` row (the seller's `edi_sent_at` is invisible to the buyer and can't be overwritten by them).

Bill (buyer's view of an invoice)
- [ ] Same row and lifecycle as invoices, so the same enums: `draft | open | paid | void | uncollectible` (the seller marks it uncollectible; the buyer sees that). Never deleted; `metadata` editable. `source` is one enum for both sides (`shipment | order | receiving | purchase_order | manual`); each side sees the author's value. Credit note and supplier credit `reason` are likewise one union enum.
- [ ] `source: receiving | purchase_order | manual` when you author it (receiving fills lines from what was received; purchase_order bills before receipt; manual has no PO).
- [ ] `supplier_invoice_number` (the issuer's `number`) is unique per supplier, blocking duplicate entry.
- [ ] Lines reference `purchase_order_line`; `related` stubs (purchase order, receiving order). Match-variance reporting later (additive).
- [ ] List filters: `statuses[]`, `supplier_ids[]`, `purchase_order_ids[]`, `due_before`, `created_after` / `created_before`; `q` exact match on `supplier_invoice_number`.

Supplier credit (buyer's view of a credit note)
- [ ] `status: issued | void` (void only while unapplied); `reason: return | price_adjustment | rebate | other` when you author it; optional `bill` it corrects; applied to bills, kept as credit, or received as a refund.

Docstrings

| Where | Text (summary ¶ description) |
|---|---|
| Bill | An invoice a supplier sent you, recording what you owe them. ¶ Once finalized, a bill can't be changed or deleted. Correct it with a supplier credit, or void it while nothing is applied. If the supplier uses OpenMRP, their invoices appear here automatically. |
| `source` | Where the bill's lines came from. ¶ `receiving`: what was received against a purchase order. `purchase_order`: the order itself, billed before receipt. `manual`: entered by hand. |
| `supplier_invoice_number` | The number on the supplier's invoice. ¶ Unique per supplier, so the same bill can't be entered twice. |
| Supplier credit | A credit a supplier gave you, reducing what you owe. ¶ Apply it to a bill, keep it as credit with the supplier, or receive it as a refund. |
| `payment.direction` | Which way the money moved, from your side. ¶ `inbound`: to you. `outbound`: from you. |
| `metadata` (shared documents) | Your own key-value data on this document. ¶ Each party has its own; the other party never sees it. |

## Fulfillment — picks, shipments, shipping cases, deliveries ✅ finalized

Carried over
- [ ] Pick / shipment `note` → read-only `order_instructions`; pick `ship_to` and shipment `shipping_address` → `ship_to` snapshot (copied from the order); `related` stubs stay (pick → sales order, shipments; shipment → sales order, pick, invoice; delivery → purchase order, receiving order); `freight` section, `priority` enum, pick keeps the order's `commitment`; every `PUT …/actions/*` → `POST`.
- [ ] Shipping case `freight_amount` → Money (`freight_weight` stays Quantity); rate results return Money.
- [ ] Pick `totals` (money stages) → per-line quantities.

Changes
- [ ] **F1** Shipments, shipment lines, shipping cases and deliveries become **public** (deliveries read-only). `POST /shipments/actions/estimate-rate` becomes public next to `rate-shop`.
- [ ] **F2** Pick actions: `PUT …/actions/pick` → `POST …/actions/pick-all`; `PUT …/lines/{id}/actions/pick` → `POST …/lines/{id}/actions/pick-all`; `PUT …/actions/void` → `POST …/actions/reset`; `PUT …/lines/{id}/actions/void` → `POST …/lines/{id}/actions/reset`; `PATCH …/lines/{id}` kept; `POST …/actions/pack` kept (202 + job). Pick line `quantity` → `picked_quantity` (with `ordered_quantity`); `finished_at` → `completed_at`.
- [ ] **F3** Remove `is_ready_to_ship` (boolean; code TODO). Shipment `status` gains `ready`, derived (read-only): not shipped, ≥ 1 case, every case has freight weight > 0 (today's rule, `shipment.sql:71-81`). See the shipment lifecycle below.
- [ ] **F4** `POST …/actions/void` → `POST …/actions/unship`: returns a shipped shipment to `packed`; its invoice is **voided** if nothing is applied, else 409 `resource_in_use` (issue a credit note first). Never deletes the invoice. `admin-update-tracking` (shipment and shipping case) → `POST …/actions/correct-tracking` (admin-only; carrier, service level, tracking numbers only).
- [ ] **F5** `shipping_cases`: embedded List (first 10 + `next_page_url`) and a new `GET /shipments/{id}/shipping-cases`. Delete only while not shipped (unpacks pick lines; shipped → 409). `case_count` kept.
- [ ] **F6** Deliveries read-only; `unit_cost` → UnitPrice; `status: accepted | rejected`, `accepted_at`, `rejected_at` kept.

Shipment lifecycle (decided)
- [ ] `status: packed | ready | labeled | shipped` (read-only). `packed`: packed into cases. `ready`: computed (every case weighed). `labeled`: labels bought, waiting for carrier pickup. `shipped`: the carrier has it; **the invoice is created here**.
- [ ] **One `ship` click, as today.** Self-managed carriers (`ltl`, `will_call`, `local_delivery`, `other`): `POST …/actions/ship` → `shipped` directly. Shippo carriers (`fedex`, `ups`, `usps`): the same action buys labels → `labeled`; the first carrier scan (Shippo `track_updated` past `PRE_TRANSIT`, i.e. any later event, including a skipped first scan) moves it to `shipped` with no further click. Staff can still force `shipped` (e.g. delayed scan).
- [ ] New read-only `tracking_status: pre_transit | in_transit | delivered | returned | failure | unknown` from the carrier; `null` for self-managed carriers.
- [ ] **New: Shippo tracking webhook** endpoint and handler (none today; only Stripe webhooks exist).
- [ ] Invoice timing changes for Shippo carriers: created at first carrier scan instead of at the ship click.
- [ ] `unship` works from `labeled` or `shipped`.

Lists
- [ ] Picks: `q` exact on pick number (= order number) or customer PO number (docstring says "exact"); sorted by ship-by date, soonest first. Shipments: `q` exact on `number`, `master_tracking_number` or `bill_of_lading`; `statuses[]`, `customer_ids[]`, `carrier_ids[]`, `shipped_after`, `shipped_before`. Deliveries: `purchase_order_ids[]`, `item_ids[]`, `statuses[]`. Plural filters, `created_after` / `created_before`.

Docstrings

| Where | Text (summary ¶ description) |
|---|---|
| Pick | The warehouse work to gather an issued order's items. ¶ Created when the order is issued. Pack it to create a shipment. |
| Pick all | Marks everything still outstanding on the pick as picked. ¶ Never lowers a quantity; an over-pick is kept as recorded. |
| Reset (pick) | Clears picking work that hasn't been packed. ¶ Picked quantities return to 0. The order is unaffected. |
| Unship | Returns a shipped shipment to `packed` so it can be corrected. ¶ Its invoice is voided. If a payment or credit has been applied to the invoice, issue a credit note first. |
| Delivery | Goods stocked against a purchase order, with their lot, location and cost. ¶ Created by stocking a receiving order; read-only. |

## Inventory ✅ finalized

- [ ] **N1 Movements are created, not PATCHed.** `PATCH /v1/catalog/items/{id}/inventory` (returns `{}`) → `POST /v1/operations/inventory-movements` `{ item_id, operation: adjust | reconcile, quantity, location_id, lot_number, customer_id, reason }` → 201 `inventory_movement`. Movements are immutable (corrections are new movements). `POST /v1/catalog/items/actions/bulk-reconcile` (sync, 1,000 rows) → `POST /v1/operations/inventory-movements/actions/bulk-create` → **202 + job** (rows by `sku` or `item_id`, same `operation`; per-row created/skipped/error in the job result).
- [ ] `reason` is an enum on corrections: `damaged | count_correction | found | scrap | other`.
- [ ] **N2** `/v1/operations/inventory-change-logs` → `/v1/operations/inventory-movements`, object `inventory_movement`: `{ id, object, type, item, quantity, location, lot, actor, related, occurred_at, created_at }`.
  - **Who vs what:** `actor` says who caused it — `user | api_key | agent | device | system` (extend the shared actor type with `device` and `system`; replaces `responsible_user` / `responsible_scanning_station`). `type` says what happened, independent of who: `receipt | shipment | consumption | output | transfer | adjustment | reconciliation | return` (X2; was `scan | user_action | system_action | user_correction`). An agent or API key adjusting stock is `type: adjustment`, `actor.type: agent | api_key`.
  - `location`, `lot`, `related` (source document stubs) where known; more is additive.
  - Filters: `item_ids[]`, `types[]`, `actor_ids[]` (was `changed_by_user_ids`), `actor_types[]`, `location_ids[]`, `occurred_after` / `occurred_before`.
  - Remove the deprecated `GET …/actions/export`; keep `POST …/actions/export` (job).
- [ ] **N3** Public `GET /v1/operations/inventory-levels?item_ids[]&location_ids[]&as_of` (was internal `/inventories`): `{ item (expandable), location (expandable), on_hand, reserved, available_to_promise, short }` (Quantities). `GET /v1/catalog/items/{id}/inventory` kept, same shape.
- [ ] **N4** Lot default: `quantity: float64` + `unit` → `lot: Quantity | null` (`null` = no lot convention; was `0`); `item` / `product_line` (`Entity`) → expandable references; `source` kept.
- [ ] **N5** Locations: remove `/location-types` (E1; `type` enum stays); add `status: active | archived`; delete → 409 `resource_in_use` while stock, movements or child locations reference it; `children` expandable List (first 10 + `next_page_url`); new `parent_ids[]` filter.

Docstrings

| Where | Text (summary ¶ description) |
|---|---|
| Create movement | Records a change to an item's stock. ¶ `adjust` adds `quantity` (negative to remove). `reconcile` sets stock to exactly `quantity`, measured against what's on hand net of uncovered demand. Movements can't be edited or deleted; correct a mistake with another movement. |
| `movement.type` | What kind of movement this is. ¶ `scan`: recorded by a shop-floor scan. `operation`: part of normal work, such as stocking, picking or consumption. `correction`: a manual adjust or reconcile. `automatic`: made by OpenMRP itself, such as an allocation. |
| `movement.actor` | Who caused the movement: a user, API key, agent, device, or OpenMRP itself. |
| `reason` | Why stock was corrected. ¶ `damaged`, `count_correction`, `found`, `scrap` or `other`. |
| Inventory level | An item's stock position. ¶ `on_hand`: in your facility, including customer-owned stock you hold. `reserved`: held for existing orders. `available_to_promise`: free for new orders. `short`: demand not yet covered. |
| `lot` | The quantity this item is made in, such as one doff. ¶ `null` when no lot convention applies. `source` says which rule supplied it. |

## Production structure — manufacturing methods ✅ decided (replaces production steps)

Production steps are replaced by **manufacturing methods**: the bill of materials and routing for making one item. Public at forge.1 (steps and flows are internal today; departments, machines and scanning stations are already public).

Why: in production every step has exactly one output and no item is made by more than one step (5,788 steps), so each step is already "the one-operation method of its output item", and the step graph is a hand-maintained multi-level bill of materials. The method model keeps everything, adds versioning, and derives the graph.

Manufacturing method (`/v1/operations/manufacturing-methods`)
- [ ] `{ id, object: "manufacturing_method", item, version, status: draft | active | archived, operations (List), materials (List), created_at, updated_at }`. One `active` method per item.
- [ ] **Versioned**: edits happen on a `draft`; `POST …/actions/activate` makes it the item's active method (the previous active one becomes `archived`); `POST …/actions/new-version` copies the active method into a new draft. Active and archived methods are immutable.
- [ ] Production runs **copy the method version** they're released with (run operations and materials), so later edits never change a run in progress.

Operations (`…/manufacturing-methods/{id}/operations`; editable only on drafts)
- [ ] `{ id, object: "method_operation", position, sequence: after_previous | with_previous, work_center, machines, scanning_station, setup_time (time Quantity per run, X5), labor_time, machine_time (Rates: time per unit of output), labor_rate, overhead_rate (UnitPrice; default from the work center, overridable; `costs:read`), leveling_percent, allowance_percent (whole numbers), work_instructions }`.
- [ ] Today's step fields map here: labor rate/time, overhead rate, leveling factor (→ percent), allowances (→ percent), machines, scanning station, department (→ work center). `notes` dropped.

Materials (`…/manufacturing-methods/{id}/materials`; editable only on drafts)
- [ ] `{ id, object: "method_material", item, quantity, scrap_quantity, operation (the operation that consumes it), supply: make | stock | buy, work_instructions }`.
- [ ] `supply: make` points at the material's own active method — the multi-level structure. **The production graph is derived** from this; no `in_steps` / `out_steps` / `upstream_step_ids` / `connect-steps`.

Reads
- [ ] `GET /v1/catalog/items/{id}/production-flow` (was `/production-flows/by-item/{item_id}`): read-only tree of active methods from raw materials to the item.
- [ ] List methods: `item_ids[]`, `statuses[]`, `work_center_ids[]`.

Removed
- [ ] `/v1/operations/production-steps/*` (incl. bulk-create, `productions/{id}`), `/production-steps/{id}/consumptions/*`, `/production-flows/*`. Bulk import moves to `POST /manufacturing-methods/actions/bulk-upsert` (job).
- [ ] `PUT /scanning-stations/{id}/production-steps` → stations are set on operations.

Migration
- [ ] Each production step → version 1, `active`, method for its output item: one operation (the step's labor, rates, factors, machines, station, department) and its consumptions as materials (`supply: make` when the consumed item has a method, else `stock`). The 21 outputs with no step and orphaned productions are reviewed by hand. Step links are dropped (implied by materials). References to steps (scanning stations, runs, batches, schedules, OEE) move to methods/operations.

Work centers (was departments), machines, scanning stations
- [ ] `notes` removed (all three). Work center `labor_rate` → UnitPrice. Machines get `status: active | archived`. Scanning station `type` / `operator_requirement` enums stay. Bulk upserts become public (jobs). Delete blocked while in use.
- [ ] **Departments → work centers:** `/v1/operations/departments` → `/v1/operations/work-centers`, object `work_center`; every `department` / `department_ids` field and filter → `work_center` / `work_center_ids` (operations, machines, scanning stations, method lists, OEE/analytics).
- Later, additive: a "process" (kind of work) separate from the work center; outside (subcontracted) operations.

Docstrings

| Where | Text (summary ¶ description) |
|---|---|
| Manufacturing method | How an item is made: the operations it goes through and the materials it consumes. ¶ Methods are versioned. Edit a draft, then activate it; each item has one active method, and production runs keep the version they were released with. |
| `status` | `draft`: editable, not in use. `active`: the method used for new production of this item. `archived`: a previous version, kept for history. |
| Operation | One step of work in a method, done at a work center with its machines and scanning station. ¶ `sequence` says whether it starts after the previous operation or alongside it. |
| `labor_time` | Labor time per unit of output, such as `0.02 hr / pr`. ¶ Effective time is `labor_time × (1 + leveling_percent ÷ 100) × (1 + allowance_percent ÷ 100)`. |
| Material | An item an operation consumes, and how much per unit of output. ¶ `scrap_quantity` is the expected loss on top of `quantity`. `supply` says where it comes from: `make` (its own method), `stock` (inventory) or `buy` (purchased for the run). |
| Production flow | Every method involved in making an item, from raw materials to the item itself. ¶ Read-only, built from the active methods. |

## Production execution — runs, batches, downtime, machine status ✅ finalized

- [ ] **E1 Production runs** (public; **superseded by jobs + work orders, P1/P2**): `status: planned | in_progress | completed` (read-only); `item`, `quantity`; `manufacturing_method` (the version released); the run's copy of `operations` and `materials` (expandable Lists); `responsible_user`; `related { sales_order, batches }` stubs; `started_at`, `completed_at`. `batch_summaries` → batches in `related` + per-operation `progress` quantities. Create + `bulk-create` (job).
- [ ] **E2** Runs, batch reads, batch flow and **all scan-driven actions become public** (design finalized in the scanning review).
- [ ] **E4 Batch fields**: `production_step` → `operation`; `department` → `work_center`; `seconds` → `duration` (time Quantity); `waste` → `scrap_quantity`; `input_batches` / `output_batches` → parent and child batches as `related` stubs (E3); read-only `status: open | closed`; `closed_at`, `scanned_at` kept.
- [ ] **E5 Downtime**: remove `/machine-downtime-reasons` (fixed list); `reason` enum `breakdown | changeover | material_shortage | no_operator | planned_maintenance | minor_stop | quality_hold | no_schedule` + read-only `oee_category`; `source: manual | scanner | inferred | api` → `source: manual | scanner | inferred` (how it was recorded); `reported_by` → `actor` (user, API key, agent, device or system); `note` kept (event explanation); `duration_seconds` → `duration` (time Quantity, read-only); `department` → `work_center`; `production_run` / `batch` / `schedule_line` references per the reference-shape decision.
- [ ] **E6 Machine status**: `week_planned_quantity`, `week_scanned_quantity`, `week_planned_run_hours` (floats) + `unit` (string) → `this_week { planned, scanned, planned_run_time }` (Quantities); `status: running | idle | down` kept.

## References (E3) ✅ decided

- [ ] **Two reference patterns only**: (1) a field naming one resource is **expandable and `null` unless included** (unchanged convention — `null` when not included is never violated); (2) links to other documents/records go in `related` as lightweight `record` stubs. The `Entity` stub (`{ id, name, handle }`, always present) is removed from the public API. No bespoke "view rows with display labels".
- [ ] Batches: `item`, `work_center`, `operation`, `machines`, `scanning_station` → expandable (null unless included); `production_run`, `parent_batches`, `child_batches` → `related` stubs.
- [ ] Downtime events: `machine`, `work_center`, `item` expandable (as today); `production_run`, `batch` → `related` stubs (`schedule_line` dropped; planned orders are internal).
- [ ] Machine status: `machine`, `work_center` → expandable references (the board includes them).
- [ ] `GET /batches/{id}/flow` returns a normal `List[batch]` (edges via `related.parent_batches` / `child_batches`); remove `BatchFlowNode`.
- [ ] Work-in-progress summary (`PUT /analytics/open-batches`) moves to the analytics review (a real aggregate), not a batch view.
- [ ] **Dashboard is redesigned to compose screens from standard resources** (includes + `related`) rather than the API growing view-specific shapes.

## Batch scanning ✅ finalized (public)

- [ ] **B1 Scans are records**: `POST /v1/operations/batch-scans` `{ scanning_station_id, batch_ids, operation_id?, action?: initialize | move | split | merge (defaults from the station's type), split?: { firsts, seconds, scrap (QuantityInputs), close_source }, consume_materials? }` → 201 `batch_scan { id, object, action, status: recorded | reversed, scanning_station, operation (expandable), related { input_batches, output_batches }, actor (user | api_key | agent | device; plus operator), scanned_at }`. Replaces `POST /batches/actions/initialize | move | merge | split`. Renames: `type_override` → `action`, `production_step_id` → `operation_id`, `waste` → `scrap`, `close_batch` → `close_source`. Idempotent (Idempotency-Key) for terminals. `GET /batch-scans` lists history (`scanning_station_ids[]`, `batch_ids[]`, `actions[]`, `actor_ids[]`, `scanned_after` / `scanned_before`). Material consumption and produced inventory stay async; resulting inventory movements carry the scan in `related`.
- [ ] **B2 Preview**: `POST /v1/operations/batch-scans/actions/preview` (same body; nothing saved) → `batch_scan_preview { action, options [{ operation, action }], output_quantity, remaining_to_split, materials [{ item, required, on_hand }] }`. Replaces `POST /batches/{id}/init-steps`, `POST /batches/{id}/next-steps`, `POST /batches/remaining-quantities`, `POST /scanning-stations/{id}/consumptions`.
- [ ] **B3 Reverse**: `POST /batch-scans/{id}/actions/reverse` — same behavior as today's "delete batch" undo (releases consumed batches, reopens the run, reverses the scan's inventory via async message, `batch_service.go:429`) but keeps the scan with `status: reversed` and its reversing movements. Only while output batches haven't been scanned again (else 409). `DELETE /batches/{id}` and bulk delete only for never-scanned (planned) batches; return deleted stubs.
- [ ] **B4 Reads**: `GET /scanning-stations/{id}/batches` → `GET /batches?scanning_station_ids[]=…&statuses[]=open`; `POST /batches/actions/close {batch_id}` → `POST /batches/{id}/actions/close`; `GET /batches/{id}/flow` returns a batch list; `PUT /analytics/open-batches` → analytics review.

Docstrings

| Where | Text (summary ¶ description) |
|---|---|
| Batch scan | Records a batch being scanned at a station. ¶ `initialize` starts a batch at its first operation. `move` advances batches to the next operation. `split` grades part of a batch into firsts, seconds and scrap. `merge` combines batches. Each scan except an initialize creates new output batches. Scans can't be edited; reverse one to undo it. |
| `action` | What the scan does. ¶ Defaults to the scanning station's `type`. |
| Preview | Shows what a scan would do without saving it. ¶ Returns the operations it could perform, the quantity it would produce, how much is left to split, and the materials it would consume against what's on hand. |
| Reverse | Undoes a scan. ¶ Reverses the inventory it recorded and closes its output batches. Only possible while those batches haven't been scanned again. |

## Production planning ✅ redesigned — ships with forge.1 as **private (internal) beta**

Planning is rebuilt around flow principles (below) and lands with forge.1, but **every planning route is internal (`Public: false`)** and excluded from the public contract while it's refined in private beta. Today all 46 planning routes are public; they become internal. Making them public later is additive.

Principles (stated in the docs)
- Little's Law (WIP = throughput × cycle time): WIP, throughput and cycle time are reported together; WIP limits are first-class.
- Variability is buffered by inventory, capacity or time: every item states its buffer.
- The bottleneck sets throughput: plans are built at the constraint work center; other work is derived from it.
- Cycle time grows non-linearly near 100% utilization: work centers carry a target utilization; load above it is flagged.
- Pull over push: release is capped by WIP (CONWIP), **enforced**.
- Lot size trades setup against inventory: lot sizing and changeovers are explicit.

Resources (all internal at forge.1)
- [ ] **Demand**: `GET /v1/operations/demand` (weekly buckets; firm orders + forecast + adjustments) and `/demand-adjustments` (CRUD; was demand overrides; `scope { type, … }`, quantity or percent by `adjustment`, `status: active | archived`, `reason_note`).
- [ ] **Item `planning` section**: `participation: planned | excluded`, `strategy: make_to_stock | make_to_order`, read-only `recommended_strategy` (replaces fulfillment recommendations + apply), `buffer { service_level_percent, safety_stock { mode: computed | fixed, quantity }, reorder_point (read-only) }`, `lot { multiple, minimum, economic_quantity (read-only) }`, `quoted_lead_time`. Replaces item settings, item policies and finished policies. While planning is internal, the public `fulfillment_policy` (product lines, customer defaults) and `material.order_point` stay authoritative; `planning.strategy` and `planning.buffer.reorder_point` take over from them when planning goes public (recorded then as a breaking change).
- [ ] **Work center `planning` section**: `participation`, `role: constraint | standard`, `capacity { shifts_per_day, hours_per_shift, calendar }`, `target_utilization_percent`, `wip_limit` (Quantity), `lead_time`. Replaces resource settings and capacity fields in the settings singleton.
- [ ] **Planning settings** singleton: `GET/PATCH /v1/operations/planning-settings` (horizon, frozen weeks, week start, forecast window, default service level, holding rate).
- [ ] **Production plans** (was schedules): versioned `status: draft | published | archived`, `horizon`, `frozen_weeks` (time fence), typed `settings` snapshot (no `map[string]any`), `failure { code, message }`; generation via `POST /production-plans` → job; `actions/preview`, `actions/publish` (archives the previous), `actions/archive`.
- [ ] **Planned orders** (one shape; replaces lines, derived lines, finishing lines, `greige_*` fields): `{ role: constraint | upstream | downstream, status: planned | firm | released, week_starts_on, sequence, item, operation, work_center, machine, quantity, lots, run_time, changeover_time, projected_on_hand { before, after }, source: solver | manual, related { production_run } }`. Editable only in drafts; manual edits recorded with `source: manual` (replaces deviations).
- [ ] **Load**: per work center per week `{ capacity, planned, utilization_percent, status: under_target | over_target | over_capacity }`.
- [ ] **Exceptions**: one list `{ type: at_risk_order | over_capacity | over_target | below_safety_stock, … }` (replaces at-risk orders).
- [ ] **Release (pull)**: `POST /production-plans/{id}/actions/release { through_week, preview? }` → production runs in sequence **only while each constraint work center's WIP is under its `wip_limit` (enforced)**; held-back orders stay `firm` with the reason. Replaces `release-week` / `week-release-preview`.
- [ ] **Weekly review (plan vs actual)**: the planning meeting reviews last week per machine — `GET /production-plans/{id}/attainment?week_starts_on=…` → per machine and item: planned vs produced quantity and run time, attainment percent, downtime by reason (incl. planned maintenance), and shortfall; links to downtime events in `related`. Feeds the flow metrics (throughput, WIP, cycle time) in analytics.
- [ ] **Time buffer for make-to-order**: the order's commitment (customer / order lead time, decided in customers and sales orders) is the primary time buffer; the item's `quoted_lead_time` is the floor (an order can't be promised faster than its slowest item).
- [ ] Operating calendars stay **public** (customers' and addresses' `receive_calendar` reference them; PL5 changes apply). Downtime and machine status stay public (execution review).

Visibility mechanics (implementation note)
- [ ] The item and work center `planning` sections live on **public** resources but are **internal fields** at forge.1: excluded from the public OpenAPI and from public API-key responses until planning goes public. Needs field-level "internal/beta" visibility in the resource framework (similar to `sensitive:"internal"` for portal users).

Removed (replaced by the above): `/production-schedules/*` (22 routes), `/production-schedule-settings/*`, `/fulfillment-recommendations/*`, `/demand-overrides/*`, `/demand-override-types`, `/schedule-deviation-types`.

## Core ✅ finalized

Audit events (public)
- [ ] Shape kept. `starts_at` / `ends_at` → `occurred_after` / `occurred_before`; `root_resource_type` / `root_resource_id` → `root_resource_types[]` / `root_resource_ids[]`. Remove `GET /audit-events/resource-types` (fixed enum: `resource_type` is the `object` enum).

Request logs (public)
- [ ] **C1** `request_body`, `response_body` (and `query_params`) are removed from the log resource. Bodies are fetched with a **separate call**: `GET /v1/core/request-logs/{id}/bodies` → `{ object: "request_log_bodies", request_body, response_body }` (secrets redacted), gated by a dedicated permission (`request_logs:read_bodies`). Bodies are moving to S3, so this is one object fetch per call and never part of list responses. `query_params` → typed `{ key: value }` map on the log (it's small). Date filters → `occurred_after` / `occurred_before`; `min_latency_us` kept.

Email logs (public)
- [ ] Add `related` (the invoice, order acknowledgment, statement, … it carried). Filters `send_statuses[]`, `created_after` / `created_before`, `related_ids[]`.

Jobs (public)
- [ ] Async jobs move to `/v1/core/async-jobs` (X11): `POST /jobs/{id}/cancel` → `POST /v1/core/async-jobs/{id}/actions/cancel`. New `GET /v1/core/async-jobs` (`types[]`, `statuses[]`).

Sandboxes (public)
- [ ] Kept; gain `seed: empty | sample_factory` (G4).

Search (public)
- [ ] **C2** Results are `{ object: "search_result", type, resource }` with `resource` expandable (`include[]=data.resource`, batch-loaded per type); no `entity` stubs. `customer` filter → `customer_ids[]`. Each type's docstring lists the columns it matches.

Analytics
- [ ] **C3** All analytics are **internal at forge.1**, including the currently public `oee`, `oee-trend`, `delivery-performance` and `schedule-attainment` (attainment joins the planning beta). Internal analytics move off PUT (GET, or POST `actions/query` for complex filters) but aren't part of the contract.
- [ ] **Future (separate design):** replace per-dashboard analytics endpoints with a flexible analytics query model (metrics × dimensions × filters × time buckets) so a new dashboard doesn't need a new endpoint.

Other
- [ ] **C4** `POST /core/records/actions/generate-pack-list` → `POST /v1/operations/shipments/{id}/actions/generate-pack-list`. `/identity/me/tenancy` reviewed with identity.

## Identity & auth (resources) ✅ finalized

- [ ] **I1** Account users: `status: active | disabled | removed` (all visible; `statuses[]` omitted = all; no `removed_scope` hiding). `PUT …/actions/activate|disable|remove` → `POST`. Seat rules unchanged.
- [ ] **I2** No generated passwords: email users get a sign-in link; the API never returns or emails a password. (Any flow relying on the emailed password, e.g. portal onboarding, moves to the link.) Username-only users are replaced by the auth redesign (devices and operators).
- [ ] **I3** `department` → `work_center`; `is_commission_eligible` → `commission_eligibility: eligible | ineligible`; `notification_types` moves to messaging (notification preferences).
- [ ] **I4** `GET /identity/permission-groups` → `GET /v1/identity/permissions` catalog `{ code, name, description, group }` (no id/owner/timestamps); roles reference codes. (Permission model: see below.)
- [ ] **I5** API keys: add read-only `status: active | expired | revoked`; create/rotate response `created_api_key { api_key_secret, api_key_info }` → `{ secret, api_key }` (secret shown once); `DELETE /api-keys/{id}` (which already revokes and keeps the record) → `POST /api-keys/{id}/actions/revoke`; rotate keeps `revoke_at`.
- [ ] **I6** `PUT /identity/accounts/{id}/favicon` becomes internal (account settings all internal at forge.1; a public `GET /identity/account` is additive later).
- [ ] **I7** Customer hierarchy: `parent_customer_id` writable on customer create/update (clearable); remove internal `/identity/child-accounts/*`. Product-line access grants stay internal.

## Authentication & permissions redesign ✅ decided

Actors
- [ ] Every actor is typed: `user` (person with email), `device` (station/terminal/kiosk/tablet), `api_key`, `agent`. The `scanning_station` actor type → `device`. Records made at a device by an identified person carry `actor` (the device) and `operator` (the account user).

People (A1)
- [ ] Passwordless: **passkeys** (WebAuthn; primary; count as both factors); **email link or code** (fallback; first sign-in); **second factor** after email sign-in: passkey or TOTP, with 10 single-use recovery codes.
- [ ] Account policy `security.two_factor: optional | required`; **`required` by default for new accounts**.
- [ ] Passwords removed: no password fields anywhere; existing email users sign in by link and are prompted to add a passkey. SSO (SAML/OIDC) later, additive.
- [ ] Sign-in ceremonies, enrollment, links, recovery codes and the account security policy are internal; public read-only `GET /v1/identity/users/{id}/auth-methods`.

Devices (A2)
- [ ] `POST /v1/identity/devices { name, role_id, work_center_id?, machine_id?, scanning_station_id? }` → `device { status: pending | active | revoked, role, work_center, machine, scanning_station, last_seen_at, paired_at, pairing { code, expires_at } (on create/re-pair only) }`.
- [ ] Links to work center, **machine** (e.g. a tablet mounted at a machine) and scanning station are all **optional and changeable** (PATCH); no assumption that a device stays put.
- [ ] Pairing: single-use code (~10 min); the terminal calls `POST /v1/auth/device-pairings { code, public_key }` and gets a credential bound to its key pair. `POST …/actions/revoke` (immediate), `POST …/actions/re-pair` (new code; replaces the terminal).
- [ ] Migration: the 21 username-only station logins → devices with the same role and station; each re-paired once.

Operators at devices (A3) — designed now, shipped later (additive)
- [ ] Badge (barcode / QR / NFC issued to an account user) **and** PIN, used only on an enrolled device to identify who is working; not a sign-in elsewhere. Operators may lack email. Scanning station `operator_requirement` gains `badge`. `POST /identity/account-users/{id}/actions/issue-badge` (reissue invalidates the old badge). Records carry `operator`. Not part of forge.1; the `operator` field and actor model leave room for it.

Permissions (A4)
- [ ] Role: `{ name, template?, access { catalog, sales, purchasing, inventory, production, finance, settings: none | read | edit | full }, grants [codes], revokes [codes], permissions (read-only effective list), notification_defaults (see messaging N3) }`.
- [ ] Levels: `read` = list/retrieve; `edit` adds create/update; `full` adds delete and the area's sensitive actions. Every permission code belongs to one area and level (recorded in the `/identity/permissions` catalog).
- [ ] Templates (catalog `GET /v1/identity/role-templates`): `owner, admin, sales, customer_service, purchasing, warehouse, production_supervisor, operator, accounting, read_only`; a starting point, not a live link.
- [ ] `POST /v1/identity/roles/actions/preview` returns effective permissions before saving. Devices, users, API keys and agents all get permissions through roles.

Removed
- [ ] `/v1/auth/actions/login` (password), `/v1/auth/passwords/*`, `/v1/auth/scanner-passwords`.

## Settings — integrations & portal domains ✅ finalized

Integrations (public)
- [ ] **T1** `credentials` (a JSON string) → typed object discriminated by `provider`: `stripe { private_key, publishable_key, webhook_secret }`, `shippo { api_key }`, `hubspot { access_token }`. Write-only (never returned or logged); live-vs-test key check kept.
- [ ] **T2** Create no longer upserts: a second integration for a connected provider → 409 `resource_exists`. New `POST /v1/settings/integrations/{id}/actions/rotate-credentials { credentials }` (validated against the provider before saving). `PUT /integrations/{id}` → `PATCH` (`name`, `status`). Delete disconnects and discards credentials; returns the stub.
- [ ] **T3** `status: active | inactive` → `active | disabled` (deliberate exception to `archived`: integrations are paused and resumed, not retired). New read-only `connection_status: connected | invalid_credentials | unknown` from the last provider call.
- Internal (unchanged): Stripe status/publishable-key, HubSpot sync, billing, system properties, support routes.

Portal domains (public)
- [ ] **T4** Shape kept (`domain`, `status: pending | securing | verified | failed`, `dns_records`, `verified_at`; one per account). OpenMRP re-checks pending domains in the background; `POST …/actions/verify` stays as "check now". `dns_records` is a plain array of `{ type, name, value }` (not an expandable List), the same shape as email domains (M1).

Docstrings

| Where | Text (summary ¶ description) |
|---|---|
| Create integration | Connects a provider to your account. ¶ One connection per provider. `credentials` depends on `provider` and is never returned. Sandbox accounts must use test keys, and live accounts live keys. |
| Rotate credentials | Replaces an integration's credentials. ¶ The new credentials are checked against the provider before they're saved. |
| `status` / `connection_status` | Whether OpenMRP uses this integration: `active` or `disabled`. / Whether the last call to the provider succeeded. |
| Portal domain | A custom domain for your customer portal. ¶ Publish `dns_records` at your DNS provider. The domain moves from `pending` to `securing` while a certificate is issued, then to `verified`. It's checked automatically; Verify checks now. |

## Messaging ✅ finalized

Endpoints: conversations (+ participants, messages, attachments, links), groups (+ members), blocks, contacts, notifications, announcements, preferences, email domains, email inboxes, email sender. Actions are already `POST …/actions/{verb}`.

Carried-over conventions
- [ ] `Entity` references (conversation `topic`, message / notification / announcement / conversation-link `resource`) → `related` record stubs.
- [ ] Bare IDs → expandable references: email inbox `group_id` → `group`; email sender `email_domain_id` → `email_domain`. Email sender drops the copied `domain` / `domain_status`.
- [ ] Preference booleans `in_app_enabled` / `email_enabled` / `push_enabled` → `channels: ["in_app", "email", "push"]`.
- [ ] Message `agent_run_failed` + `agent_error_code` → message `status: failed` + `error { code, message }` (AI A4; `agent_run` is internal).
- [ ] Conversation `unread` → `unread_count`.
- [ ] `PUT /messaging/preferences` and `PUT /messaging/email-sender` → `PATCH`.

Decisions
- [ ] **M1** Email domains use the portal-domain shape: `{ domain, status, dns_records: [{ type, name, value }], verified_at }` replaces `dkim_tokens` and the `mail_from_*` strings. Pending domains are re-checked in the background; `actions/verify` is "check now".
- [ ] **M2** Customer document recipients go public as a customer field: `notification_recipients: [{ account_user (expandable), documents: ["invoice", "acknowledgment", "statement", "shipment"] }]` (null unless included), written by customer PATCH, which replaces the list. The internal `/notification-recipients` routes are removed. New orders copy the `acknowledgment`, `invoice` and `shipment` recipients into `email_recipients` (statements are account-level, not per order). Invoices derive whether anyone receives them from this (replaces `accepts_invoice_emails` and the customer's `notification_preferences`). Personal `/messaging/preferences` is separate.
- [ ] **M3** Deleted messages are visible tombstones: `status` gains `deleted`, `body` and `attachments` become `null`, `deleted_at` stays. `actions/redact` does the same to another person's message and records the actor.
- [ ] **M4** Per-person conversation state stays as paired actions (`archive`/`unarchive`, `hide`/`unhide`, `mute`/`unmute`, `read`, `leave`); the caller's state is shown through their participant (`membership`, `notifications`, `read_cursor`). `set-status` and `assign` stay.
- [ ] **M5** Internal: `typing`, `/messaging/support`, `support-availability`. Agent participants, inbox `agent_config` / `agent_trigger_*` and message agent fields move to the AI review. Notification `change_count` → `occurrences`.

Docstrings

| Where | Text (summary ¶ description) |
|---|---|
| Conversation | A thread of messages between people, devices and agents. ¶ A conversation can be about a record, such as an order, which appears in `related`. |
| `unread_count` | Messages the caller hasn't read yet. |
| Notification recipients | Who receives a customer's documents by email. ¶ Each recipient is one of the customer's users and the document types they get. New orders copy these as their `email_recipients`. |
| Email domain | A domain you send email from. ¶ Publish `dns_records` at your DNS provider. Verification runs automatically; Verify checks now. |

### Notification opt-in & opt-out ✅ decided

- [ ] **N1** Subscriptions are stored rows, not derived from the audit trail: `subscription { id, object, record (related stub), reason: creator | editor | assignee | mentioned | manual, status: subscribed | unsubscribed, created_at, updated_at }`. `GET /v1/messaging/subscriptions?record_ids[]&record_types[]&statuses[]`, `POST /v1/messaging/subscriptions { record: { type, id } }` (idempotent follow), `POST …/{id}/actions/unsubscribe`, `POST …/{id}/actions/subscribe`. An `unsubscribed` row blocks automatic resubscription. Followable: sales orders, purchase orders, production runs, invoices, customers and suppliers (covers their orders and invoices). Conversations keep participant mute. Fan-out is one indexed lookup by record (`(account_id, record_type, record_id, status)`).
  - Automatic subscription is a preference: `auto_subscribe: ["created", "assigned", "mentioned"]` (default). `edited` is opt-in. **Editing no longer subscribes by default** (behavior change from today's audit-trail followers). Migration: no backfill of past editors; creators and responsible users are backfilled as subscriptions.
- [ ] **N2** `GET/PATCH /v1/messaging/preferences` → one object: `{ object: "notification_preferences", default: { channels, digest }, types: { "<type>": { channels, digest } }, auto_subscribe, paused_until }`. Notification `type` (was `category`) is a documented fixed enum. PATCH merges per type, and `null` removes an override. `channels: []` turns a type off. `paused_until` holds email and push (in-app still collects).
- [ ] **N3** Defaults at two levels, with the same `{ default, types, auto_subscribe }` shape:
  - Account: `GET/PATCH /v1/settings/notification-defaults`.
  - Role: a `notification_defaults` field on the role, seeded from its template; for example `warehouse` gets shipment emails.
  - Resolution, first match wins: personal type → personal default → role type → role default → account type → account default → built-in (in-app on; email and push off).
- [ ] **N4** Required types (new sign-in, passkey/2FA change, device paired) always email; `channels: []` on them → 422.
- [ ] **N5** Every notification email carries one-click `List-Unsubscribe` headers and footer links (signed, no sign-in): "Stop following {record}" (unsubscribes) and "Stop emails like this" (removes `email` for the type). Customer document emails link to "Stop sending me {document}", which removes that document type from the recipient (M2). These links are internal handlers, not API endpoints.

| Where | Text (summary ¶ description) |
|---|---|
| Subscription | Whether you're notified about a record. ¶ You're subscribed automatically according to your `auto_subscribe` preference. Unsubscribing stays in effect until you subscribe again. |
| `reason` | Why you're subscribed: `creator`, `editor`, `assignee`, `mentioned` or `manual`. |
| Preferences | Where your notifications go. ¶ `default` applies to every type you haven't set in `types`. An empty `channels` list turns a type off. Required types can't be turned off. |
| `auto_subscribe` | When you're subscribed to a record automatically: `created`, `assigned`, `mentioned`, `edited`. |
| `paused_until` | Holds email and push notifications until this time. In-app notifications still arrive. |
| Notification defaults | Your team's starting notification settings. ¶ A role's defaults apply before the account's, and each person's own preferences take precedence. |

## AI — agents, conversations, tool calls, notes ✅ finalized (public with forge.1)

Agents and versions
- [ ] **A1 / A8** Identity on the agent, behavior on versions (same rule as manufacturing methods):
  - `agent { id, object: "agent", owner (`type: system` for OpenMRP's agents, else the account), name, slug, description, status: active | disabled, role (expandable), active_version (expandable), trigger: { type } (read-only, from the active version), model: { tier } (read-only, from the active version), created_at, updated_at }`. Lists need no include to show trigger type and tier.
  - `agent_version { id, object: "agent_version", agent, version, status: draft | active | archived, instructions, model: { tier }, trigger: { type: manual | chat | schedule | event, schedule: { cron, timezone } | null, events: [] }, tools: { access: selected | all, count }, created_at }`. Active and archived versions are immutable.
  - `POST /v1/ai/agents { name, slug, description?, role, instructions, model, trigger, tools? }` creates the agent with an active v1. `PATCH /v1/ai/agents/{id}` (name, slug, description, role, status). `GET/POST /v1/ai/agents/{id}/versions` (POST copies the active version into a new draft), `PATCH …/versions/{version}` (draft only), `POST …/versions/{version}/actions/activate`.
  - Renames and removals: `agent_definition` → `agent`; `definition_type: system | custom` → `owner` (as on other system rows); `editability`, `category_code`, `temperature` removed; the `config` sub-object is removed (its fields live on the version); `system_prompt` → `instructions`; `trigger_type` + `trigger_config` → `trigger`; `status: inactive` → `disabled` and set through PATCH (`PUT …/status` removed); `role_id` → `role`.
  - `instructions` is returned only when included, and is always `null` on system-owned agents (their versions come from OpenMRP).
- [ ] **A2** Tools are never listed inline. A version shows `tools: { access, count }`. `GET /v1/ai/agents/{id}/versions/{version}/tools` (paginated; `types[]`, `effects[]`, `q`) → `[{ tool (id, expandable), review: required | not_required, config }]`; `POST …/tools { tool, review?, config? }`, `PATCH …/tools/{tool} { review, config }` and `DELETE …/tools/{tool}` on drafts. `access: all` stores no list (every tool the role allows; write tools reviewed by default). Replaces `tools[]` + `endpoint_tool_slugs` + `endpoint_tool_review`, `config_json` (string → object validated by `config_schema`), `require_review` bool, `"*"` and `sort_order`.
- [ ] **A3** Catalog `tool { id (slug), object: "tool", type: built_in | api_endpoint, name, description, group, effect: read | write, config_schema (include-only), required_permissions }`. `GET /v1/ai/tools?types[]&groups[]&effects[]&ids[]&q` (paginated; `q` = name, description over the in-memory catalog). `available_tool` → `tool`; `category` → `type`; `mutating` → `effect`; `required_role_type` removed; `/v1/ai/tool-groups` internal.

Conversations replace runs
- [ ] **A4** Every agent is used through conversations. A message or mention works in that conversation; "start a chat" is `POST /v1/messaging/conversations { participants: [{ agent }], message }`; an event about a record uses that record's conversation (an email's thread is its conversation); **each scheduled run starts a new conversation**, with the agent's subscribers (N1) as participants.
  - The agent's work is its messages: `status: streaming | sent | failed`, `error { code, message }`, attached `tool_calls`. Questions are answered by replying. `POST /v1/messaging/conversations/{id}/actions/stop-agent`; `POST /v1/messaging/messages/{id}/actions/retry` on a failed agent message.
  - Conversations gain read-only `origin: person | email | event | schedule` and an `agent_ids[]` filter.
  - Internal: `/v1/ai/runs` (trigger, cancel, retry, continue), run steps, message `agent_run`. Runs remain the execution record for support, cost and debugging.
- [ ] **A5** `agent_action` → `tool_call { id, object: "tool_call", message, conversation, agent_version, tool, label, description, status: pending_review | rejected | running | succeeded | failed, review: { requirement, decision: approved | rejected | null, actor, decided_at }, record (stub), input, output, error, executed_at, created_at }`. `POST /v1/ai/tool-calls/{id}/actions/approve { scope: once | rest_of_conversation }`, `POST …/actions/reject { reason? }`. Replaces `continue` and its four arrays.
- [ ] **A7** Agents are participants: `POST /v1/messaging/conversations/{id}/participants { agent }` (internal `/conversations/{id}/agents` removed). Inbox `agent_config` → `agent`; `agent_trigger_policy` + `agent_trigger_keywords` → `agent_trigger: { policy: always | keyword | mention, keywords } | null` on inboxes and conversations. Message `agent_run_failed` / `agent_error_code` removed.

Notes (replaces memories; brings the deferred comments resource into forge.1)
- [ ] **A6** `note { id, object: "note", record (stub | null = account-wide), body, type: comment | preference | fact | instruction, importance: low | normal | high, status: active | expired, expires_at, actor (user | api_key | agent | system), edited_at, created_at, updated_at }`.
  - `GET /v1/core/notes?record_ids[]&record_types[]&types[]&actor_types[]&statuses[]&q` (`q` = body, LIKE; omitting `statuses[]` returns all, per convention; agents recall only `active`), `POST /v1/core/notes { record: { type, id } | null, body, type? (default comment), importance?, expires_at? }`, `PATCH` (author only), `DELETE` (author or admin; returns the stub).
  - Works on any record: orders, customers, items, users, API keys, agents and the rest. Internal only (never shown to portal users). Records don't embed notes; clients filter by `record_ids[]`.
  - Agents recall the notes on the record they're working on plus account-wide `preference` / `fact` / `instruction` notes, highest importance first, and write what they learn as notes (`actor.type: agent`). `/v1/ai/memories` removed; existing memories migrate to notes (category → type, importance 0–1 → three levels, entity → record).
  - `@mentions` notify and subscribe (N1 reason `mentioned`).
  - `instructions` fields stay (the single standing guidance orders copy); notes are attributed and many.
  - Migration: the unexposed `notes` columns (departments, machines, items, item categories, product lines, production steps/flows, scanning stations) and per-document shipment/invoice notes that differ from their order's become notes with `actor: system`.
  - Index `(account_id, record_type, record_id, created_at)`; recall is one indexed read per run.

Docstrings

| Where | Text (summary ¶ description) |
|---|---|
| Agent | An AI worker that acts in your account. ¶ An agent acts as itself, limited to its `role`'s permissions. You work with it in conversations, whether you message it or it runs on a schedule or event. |
| Agent version | What an agent does: its instructions, model, trigger and tools. ¶ Edit a draft, then activate it. Active and archived versions never change. |
| `model.tier` | How capable, and costly, a model the agent uses. ¶ OpenMRP picks the model within the tier and may change it over time. |
| `tools.access` | Which tools the agent can use: `selected` (only its listed tools) or `all` (every tool its role allows). |
| Tool call | A tool an agent called. ¶ Calls that need review wait in `pending_review` until approved or rejected. A rejected call never runs. |
| Note | A note on a record, or on your whole account. ¶ People, API keys and agents can leave notes. Agents read the notes on the records they work with, so a note is also how you teach an agent. |
| `note.type` | `comment`: a remark. `preference`: how someone likes things done. `fact`: a durable detail. `instruction`: standing guidance to follow. |
| `origin` | What started the conversation: a `person`, an `email`, an `event` or an agent's `schedule`. |

### Starting fresh conversations ✅ decided

- [ ] **C1** Creating a conversation always creates a new one: the per-pair `dm_key` uniqueness is removed. `POST /v1/messaging/conversations { participants: [{ account_user } | { agent }], title?, message?, continues? }`. Find the latest with a person or agent via `participant_ids[]` / `agent_ids[]` filters (`limit=1`). `type: direct_message | group | system` stays descriptive only.
- [ ] **C2** An agent's context is the current conversation plus notes (A6) on the records involved and account-wide notes. Nothing else carries between conversations.
- [ ] **C3** `continues: <conversation id>` links the new conversation to the old one (in `related`). Agents start from a summary of the old conversation, written in the background and posted as the first message.
- [ ] **C4** `title` is settable on create and PATCH. Agent conversations without a title get a generated one after the first exchange; conversations between people stay untitled until named.
- [ ] **C5** `DELETE /v1/messaging/conversations/{id}` (hard delete, returns the stub) is allowed only when the caller is the only person participant (e.g. chats with agents). It returns 409 `resource_in_use` when the conversation is on legal hold or an approved tool call in it changed records; those can only be archived. Conversations with other people are archived or left.
- Note types (A6) stay as four: `comment | preference | fact | instruction`.

| Where | Text (summary ¶ description) |
|---|---|
| Create conversation | Starts a new conversation. ¶ Every call creates a new conversation, even with the same participants. Find an existing one with the `participant_ids[]` or `agent_ids[]` filter. |
| `continues` | A conversation this one picks up from. ¶ Agents start from a summary of it instead of its full history. |
| Delete conversation | Permanently deletes a conversation that only you and agents are in. ¶ Conversations on legal hold, or where an agent changed records, can only be archived. |

## Pre-release decisions (comparison pass) ✅ decided 2026-10-08

These changes supersede the earlier sections wherever they conflict. Each one would be breaking to make after forge.1.

- [ ] **X1 Lot size vs lot.** The size concept is `lot_size` everywhere: product line `default_lot` → `default_lot_size`, lot defaults → lot-size defaults (`/lot-defaults` → `/lot-size-defaults`, field `lot_size`), planning `lot` → `lot_size`. `lot` means only a traceability lot. On inventory movements it's an expandable reference (`lot`, `null` unless included) to a `lot` resource `{ id, object: "lot", item, number, expires_on, status }`, which replaces the `lot` / `lot_number` strings. Item tracking modes and serials are additive later.
- [ ] **X2 Movement `type` is the business event:** `receipt | shipment | consumption | output | transfer | adjustment | reconciliation | return` (replaces `scan | operation | correction | automatic`). How it was recorded is `actor` (a scan is `actor.type: device`). Movements stay one signed row per location. A transfer is two `transfer` rows linked in `related`.
- [ ] **X3 One lead-time shape, in business days.** Every lead time is a time `Quantity` in days, counted in **business days on the account's working calendar** (the same rule as carrier transit days):
  - Customer and group `defaults.lead_time_days` → `defaults.lead_time`.
  - Order `commitment.lead_time_override_days` → `lead_time_override`.
  - `material.lead_time` and planning `quoted_lead_time` already use this shape.
  - Optional `commitment.firmness: soft | hard` (`hard` = the customer can't accept late).
- [ ] **X4 Calendar dates are dates.** A `YYYY-MM-DD` date type for business days, interpreted in the account's timezone. Naming: `_at` = instant (RFC 3339, UTC); `_on` = date. Renames:
  - `promised_at` → `promised_on`
  - `ship_by_date` / `ship_by_override_date` → `ship_by_on` / `ship_by_override_on`
  - invoice and bill `due_at` → `due_on`
  - sales order `expires_at` → `expires_on`
  - purchase order `promised_at` → `promised_on`
- [ ] **X5 Setup time is per run:** `method_operation.setup_time` is a time `Quantity` per run. `labor_time` and `machine_time` stay Rates per unit of output.
- [ ] **X6 Cancel is a state.**
  - Sales orders: `draft | issued | fulfilled | canceled` (`estimate` → `draft`).
  - Purchase orders: `draft | issued | completed | canceled`.
  - Production runs: `planned | in_progress | completed | canceled`.
  - `POST …/actions/cancel` closes the unfulfilled remainder and keeps history. Delete stays for untouched drafts.
  - A `quote` resource is additive later.
- [ ] **X7 Outbound webhooks: not part of forge.1.** Deferred. When they're designed, event types and payloads follow the API version of the subscribing endpoint.
- [ ] **X8 Error object:** `{ type, code, message, param, is_transient, errors: [{ param, code, message }] }`.
  - `param` names the parameter or field the error is about, or is null. `errors` lists every field failure on a 422, and `param` is the first of them; `errors` is empty otherwise.
  - `is_transient` stays (a documented exception to the no-booleans rule).
  - Bulk rows and async job results use the same error object instead of a string.
  - `request_log_url` is removed (the dashboard does not read it). `limit_exceeded` keeps its `quota` member, registered with apikit as an OpenMRP extension.
- [ ] **X9 Applications link any credit to any debit:** `application { source: { type: payment | credit_note, id }, target: { type: invoice | refund, id }, amount }`. Replaces the fixed `invoice` field.
- [ ] **X10 Contract rules** (in the conventions doc):
  - **Decimal scale:** document totals 2 places; unit prices up to 6; Quantity values up to 6.
  - **Tax:** `unit_price` and line `amount` are tax-exclusive.
  - **Currency:** one currency per document, with a document-level `currency` field; price lists carry a `currency`.
  - **Metadata:** 50 keys, keys ≤ 40 characters, values ≤ 500, on customers, suppliers, items, sales orders, purchase orders, invoices, shipments and production runs.
  - **Idempotency:** `Idempotency-Key` on any POST, ≤ 255 characters, kept 24 hours. Reuse with a different body → 422 `idempotency_key_reused`; still in flight → 409.
  - **IDs:** opaque, ≤ 64 characters.
  - **Rate limits:** document the `RateLimit-*` headers and 429 `rate_limited`.
  - **Deprecation:** each stable version is supported at least 24 months after its successor ships, with `Deprecation` / `Sunset` headers.
- [ ] **X11 Background work is an `async_job`.** `job` is the manufacturing job (P1), so the object a long-running action returns is `async_job`: 202 with `Location` pointing at `/v1/core/async-jobs/{id}`. Shape `{ id, object: "async_job", type, resource_type, status: queued | running | completed | failed | canceled, results (List of `async_job_result { index, status: created | updated | failed, resource, sub_resources, error }`), error, started_at, completed_at, failed_at, canceled_at, created_at, updated_at }` plus OpenMRP's `created_by` and `export`. Today's `created` / `started` / `cancelled` become `queued` / `running` / `canceled`. The shape is `object.AsyncJob` in apikit.
- [ ] **Y1** `item.default_manufacturing_method` (expandable) names the method used for new production. "One active method per item" is no longer the contract, so alternates are additive.
- [ ] **Y2** Methods gain `output_quantity` (Quantity, default 1 stocking unit). Material quantities are per that much output.
- [ ] **Y3** Order and invoice lines gain a nullable `discount` (Discount shape). `amount = quantity × unit_price − discount`.
- [ ] **Y4** An order unit can carry its own `quantity` of stocking units for this item (e.g. a "Case" of 24 here, 12 elsewhere), overriding the unit's account-wide ratio. Order, PO and invoice lines snapshot `stocking_quantity`, so ratio edits never rewrite history.
- [ ] **Y5** The `sales` section is allowed on any physical item type (`material`, `part`, `product`) and on `service`. `type` says what the item is; sourcing stays in method `supply` and planning.
- [ ] **Y6** Operation `work_center` stays required; a `process` concept is additive later.
- [ ] **Y7** Inventory movements record read-only `unit_cost` (UnitPrice) at the time of the movement.

Additive roadmap (not forge.1): outbound webhooks, taxes, multi-currency, serial/lot tracking modes and holds, quality, GL and accounting sync, costing methods, quotes, returns and transfers as source documents, warehouse routes and putaway, supplier prices and lead times, item revisions (as method versions; SKU stays unique), by-products and kits, typed custom fields (`custom_fields` reserved).

## Make-to-order readiness ✅ decided 2026-10-08

forge.1 stays make-to-stock first, but these shapes keep make to order additive. They supersede earlier sections where they conflict.

- [ ] **O1 Production work pegs to order lines, with quantities.** An order can have many runs (`sales_order.related.production_runs`, plural). Pegging is `demand: [{ sales_order_line (expandable), quantity }]`; work not pegged to an order line is for stock. `POST /v1/sales/sales-orders/{id}/actions/create-production-run` → `create-production-runs { lines?: [{ line_id, quantity }] }`, which returns a List. Where `demand` lives (run vs run line) is settled in O2.
- [ ] **O3 One sourcing word: `supply: stock | make | buy`.**
  - Item `supply` is public and nullable, inherited item → product line → account default.
  - Customers override it in `defaults.supply` (some customers are made to order, others pull from the stock buffer).
  - Sales order lines get read-only `supply`, resolved when the line is added: customer default, then the item chain.
  - `issue` reserves inventory only for `stock` lines. `make` lines are pegged to production (O1); `buy` lines get purchase orders later.
  - Replaces `fulfillment_policy` (product lines, customer defaults) and planning `strategy`, so the break recorded for when planning goes public is no longer needed.
- [ ] **O4 Method material `supply` meanings:** `make` = production pegged to this work; `stock` = taken from inventory however it was replenished; `buy` = purchased for this work. `customer` (customer-supplied) is a later value.
- [ ] **O5 Free the word "quote":** sales order `actions/quote-prices` → `preview-prices`, `quote-freight` → `estimate-freight`, `quote-commitment` → `preview-commitment`. `/v1/sales/quotes` is reserved for a future quote resource.
- [ ] **O6 Operation `type: inside`** (the only value at forge.1). `work_center` is nullable in the schema and documented as always set for `inside`, so `outside` (supplier, cost, lead time, PO pegging) is additive. Revises Y6.
- [ ] **O7 Order-specific methods:** manufacturing methods gain nullable `sales_order_line` (always `null` at forge.1). List methods returns standard methods by default. `default_manufacturing_method` is "used when nothing more specific applies".
- [ ] **O8 Per-line commitment:** the order's `commitment` is the default for every line; line `commitment` is reserved (`null` = the order's). Production pegged to demand takes its `due_on` and `firmness` from that demand.
- [ ] **O9 Costs:** order line `unit_cost` = "the item's cost when the line was added (an estimate)". Movement `unit_cost` on `output` = "valuation under the account's costing method". Run `costs { estimated, actual }` is reserved for job costing.

Additive later: quotes (quantity breaks, per-break lead time and markup, revisions, convert), customer part numbers on customer items, configurators, outside-processing details and PO pegging, saving an edited run method back as a version, capacity-based lead-time quoting, job-costing actuals, `supply: customer`, run `paused`, item revision labels.

## Jobs, work orders and traceability ✅ decided 2026-10-08

Any planning policy (one stage to finished goods, or several stages separated by buffers; weekly or continuous) is planning configuration, not API shape. Four concepts are kept separate. Supersedes E1 (run shape), O1/O2 (run pegging) and the planning "planned orders → runs" release wherever they conflict.

- [ ] **P1 Jobs are the unit of work** (`/v1/operations/jobs`):
  - Shape: `job { id, object: "job", number, item, quantity, status: planned | released | in_progress | completed | canceled, manufacturing_method (version copied at creation), operations, materials (the job's copies, sub-resources), demand[], work_order (expandable, null when released to queues), due_on, firmness (from its earliest demand), produced_lots (List), related { batches }, costs (reserved), created_at, updated_at }`.
  - Actions: `POST …/{id}/actions/release | cancel`.
  - One item per job. A subassembly is its own job, pegged to its parent.
  - Batches belong to a job (`batch.related.job`).
  - Planning's planned orders become jobs with `status: planned`. The jobs resource is public; planning stays internal beta.
- [ ] **P2 Release: work orders or queues.** A `work_order` is a release of many jobs for a period, e.g. the weekly one (`/v1/operations/work-orders`, `{ id, object: "work_order", number, status, starts_on, ends_on, jobs (List) }`). It replaces `production_run`.
  - In continuous mode a job is released without a work order, and its operations join each work center's queue: `GET /v1/operations/work-centers/{id}/queue`, `POST …/queue/actions/resequence`. Queues are internal beta, with planning.
  - A factory can mix both, per planning scope.
- [ ] **P3 Demand and stages:**
  - `demand: [{ type: order_line, sales_order_line, quantity } | { type: job, job, quantity } | { type: buffer, item, location, quantity }]`.
  - Any item with a planning buffer is a decoupling point (stage). Each buffer refills by reorder point with jobs carrying `buffer` demand, sized from demand history.
  - `stock` order lines are filled from the finished-goods buffer.
  - **Make to order pushes through every stage**: `make` order lines create pegged jobs at every level and never draw from intermediate buffers (buffers are sized from make-to-stock demand history).
  - Several demands can share one job.
  - Planning scopes (internal beta): a production plan covers a set of items or work centers with `cadence: weekly | continuous`.
- [ ] **P4 Traceability follows physical genealogy, not pegging.**
  - Batch genealogy is a graph: a merge has several parents, a split several children (today's batch flow edges).
  - A batch that enters stock gets a lot. Split children keep their parent's lot number unless relabeled; a merge produces a new lot.
  - Every consumption records the lot it drew from (already true for batches). **New:** picks and shipment lines record `lots: [{ lot, quantity }]`.
  - Reads: `GET /v1/operations/lots/{id}/genealogy?direction=upstream|downstream` (a List of lots with depth) and `GET /v1/sales/sales-orders/{id}/lots` (every lot behind the order, down to purchased material and supplier lots).
  - Latency: recording a consumption, merge or split writes ancestor rows (closure table `batch_id, ancestor_batch_id, depth`), so a full upstream walk is one indexed read at any depth.
- Renames from earlier sections:
  - `production_run` → `work_order` (release group).
  - Run item, quantity, method copy and pegging → `job`.
  - Order action `create-production-runs` → `create-jobs { lines?: [{ line_id, quantity }], work_order_id? }`, which returns a List of jobs, optionally added to an existing work order.
  - Order `related.production_runs` → `related.jobs`.
  - Batch and downtime `related.production_run` → `related.job`.

## Unusual designs settled ✅ decided 2026-10-08

These are decisions where our design differs from common ERP practice. They supersede earlier sections where they conflict.

- [ ] **Q1 Contacts stay account users, without implied access.**
  - A customer's or supplier's contacts are account users of that account (one model for people).
  - Adding a contact never grants portal access and never emails them. New read-only `portal_access: none | invited | active`; `POST /v1/identity/account-users/{id}/actions/invite` sends the sign-in link.
  - Contacts receive email only when listed in the customer's `notification_recipients` or an order's `email_recipients`.
  - Replaces "adding users sends a sign-in link" for customer and supplier contacts. Adding your own team members still invites them.
- [ ] **Q2 Sites.**
  - Location `type` gains `site`, a top-level location with an `address` (a plant, warehouse or 3PL). Every account has one default site (`settings`), created from today's top-level location.
  - Nullable, expandable `site` on sales orders (ship-from; defaults to the account's default site), shipments and picks (from the order), purchase orders and receiving (receive-at), jobs and work orders (where made), and inventory levels (filter `site_ids[]`).
  - Additive multi-site later; a single-site account never sets it.
- [ ] **Q3 Commercial vs physical grouping.**
  - **Product lines** hold commercial policy: customer access, portal visibility default, commission and freight policy, sales targets and territories, price-list scope, and `supply` default (O3).
  - **Item categories** hold physical grouping: item `type`, properties and attributes, and production defaults. `default_lot_size` moves from product lines to categories (lot-size defaults inherit item → category → account).
  - Attributes stay as the shared descriptive layer used by both, e.g. in price-list scopes. Docstrings for each resource state its job in one line.
- [ ] **Q4 Promises use the larger lead time, and record it.**
  - An order line's promise is the later of the customer's lead time and the item's own lead time. Items gain nullable `sales.lead_time` (business days; production or purchase time to ship).
  - `commitment` snapshots what was promised: read-only `promised_lead_time` (Quantity) and `promised_lead_time_source: customer | item | override`, alongside `promised_on`. Delivery performance compares shipments against these snapshots, never the customer's current setting.
  - `preview-commitment` returns the same fields.
- [ ] **Q5 `fulfilled` / `completed` are derived.**
  - A sales order becomes `fulfilled` when every line is shipped, and returns to `issued` if a shipment is unshipped. A purchase order becomes `completed` when every line is received.
  - Stopping early is `actions/cancel` on the remainder: `canceled` if nothing shipped; if some lines shipped, the remainder is closed, `fulfillment.canceled` quantities are recorded, and the order becomes `fulfilled`.
  - Removed actions: sales order `fulfill`, `reopen`; purchase order `complete`, `reopen`. Kept: `issue`, `unissue`, `cancel`.

## Agent sign-up and agent experience ✅ decided 2026-10-08 (ships with forge.1)

Goal: an agent goes from nothing to a working, safe API key in one call, and a human takes the account live later.

- [ ] **G1 One-call sign-up into a sandbox.** `POST /v1/auth/agent-sign-ups` (no auth; public) `{ human_email?, company_name?, source?, referrer? }` → 201 `{ object: "agent_sign_up", account, api_key (secret, shown once), status: unverified, sandbox: true, expires_at }`.
  - Creates a free **sandbox account** seeded with `sample_factory` (G4).
  - **Keys expire on their own.** A sign-up key has `expires_at` (30 days). Verifying extends nothing: the human's claim (G2) issues long-lived keys.
  - **Unverified:** everything works inside the sandbox, but nothing leaves it (customer and supplier email, live integrations, invites, going live). Those return 403 `verification_required`.
  - `human_email` is optional.
    - **With it:** a 6-digit code goes to the human. `POST /v1/auth/agent-sign-ups/actions/verify { code }` (authorized by the key) unlocks the sandbox's outbound features. Signing up again with the same email while unverified resends the code and rotates the key.
    - **Without it:** the key can't be recovered, and the sandbox is deleted when the key expires.
  - Abuse limits: per-IP and per-email rate limits, sandbox size caps, and unclaimed sandboxes deleted after their key expires.
- [ ] **G2 A human takes it live.** The verification email (or a claim link the agent can show) opens a claim page: the human signs in, sets up billing (today's registration flow) and creates the live account, with the sandbox kept as its sandbox. Live access reaches the agent only through the human, as a human-created key or an MCP OAuth grant. Agents can never go live or start billing. Free until then.
- [ ] **G3 Hosted MCP server** at `https://mcp.openmrp.ai/mcp`.
  - Auth: OAuth, where a human grants a role, or an API key header.
  - Tools are the public API: generated from the same OpenAPI spec as the SDK, reusing the endpoint-tool catalog and tool discovery. Agents search for tools; the full catalog is never loaded at once.
  - Sign-up and verify are tools too.
  - Permissions are the key's or grant's role.
- [ ] **G4 Seeded sandboxes.** `POST /v1/core/sandboxes { seed: empty | sample_factory }` (default `empty`). The sample factory has customers, suppliers, items with methods, open orders, inventory with lots and a work order in progress.
- [ ] **G5 Errors link to their docs.** The error object (X8) gains `doc_url` (a stable page per `code`). Every public error code gets a docs page before release.
- [ ] **G6 Docs for agents** (public-docs; the API reference is generated from public endpoints, so these routes document themselves):
  - `llms.txt` and `llms-full.txt`, and every page available as `.md`.
  - An agent quickstart: sign up → create an order → issue → ship, in curl, SDK and MCP.
  - Installable skills for Claude Code, Cursor and Codex covering our conventions.
  - An error-code index (G5).
  - A CLI (`openmrp sign-up | verify | request`).
- [ ] **G7 Which agent did it.**
  - API keys gain an optional `client { name, source }` (e.g. `claude-code`), set at sign-up or key creation.
  - API keys gain `expires_at` (nullable; settable on create; always set for sign-up keys).
  - `actor` for an API key carries `client`, so audit events, notes and movements read "API key · Claude Code".

## Order line types ✅ decided (see Sales orders)

- Sales order and invoice lines get a discriminator: `{ type: "item", item, quantity, unit_price }` or `{ type: "shipping", description, amount: Money }`. (`credit` dropped: credits are credit notes — AR review.) Replaces the per-account shipping/credit system products (`synthesizeShippingLine`, `GetSystemProduct(…, "shipping")`).
- Finding freight = filter lines by `type: shipping`.
- Decided: typed lines, mirrored on invoices (Sales orders part 2, AR).

## Notes → instructions (cross-cutting) ✅ decided

Staff guidance (for example "follow the packing SOP" or "send invoices to accounts payable") read by customer service, fulfillment and billing. Internal only; customers never see it. Attributed notes are the separate `note` resource (AI review, A6).

- [ ] **Customer, supplier:** `note` → `instructions` (internal: always `null` for portal users). Default guidance for that account's orders. Migrate `account_relation.notes` (647 rows).
- [ ] **Sales order, purchase order:** `note` → `instructions`, **internal** (today the sales order `note` is *not* internal, so portal customers can read it, including customer notes folded in; this fixes that leak). On create: if the request omits `instructions`, copy the customer's (or supplier's) value once as a snapshot; if it sends a value (including `null`), use it exactly. Remove today's silent concatenation (`sales_order_service.go:434`). Editing the customer later never changes existing orders. Migrate `sales_order.note` as-is (12,870 rows; already contains the folded-in customer notes).
- [ ] **Shipment, invoice, pick, receiving order:** remove `note`; add read-only `order_instructions` (internal), read from the order's `instructions` at read time. No stored copy, so no drift. Zero query cost: shipment and invoice queries already `JOIN sales_order` (`shipment.sql:87,191`, `invoice.sql:68,147`); picks and receiving already read `so.note`. Remove the note from shipment/invoice update requests.
- [ ] Before dropping `shipment.note` / `invoice.note`: review the ~90 + ~90 rows that differ from their order's note; if they're real per-document notes, they migrate into notes (AI A6) with `actor: system`; then drop the columns.
- [ ] **Remove `notes`** (data migrates into the `note` resource, AI A6): department (72), machine (3), item / material / part / product, item category, product line, production step, production flow, scanning station. Unit groups are retired anyway.
- [ ] **Payment descriptions:** transaction and settlement `note` → the payment's `description` (system text such as "Payment captured by Stripe") during the AR migration; the payment shape gains `description`. Allocation copies are dropped.
- [ ] **Event explanations stay:** machine downtime `note`, machine status `note`, demand override `note`, production schedule deviation `reason_note`. They're the event's content; final names are decided in each review.
- Later, additive: `fulfillment_instructions` / `billing_instructions` beside `instructions` if audiences need splitting; a customer-visible message field (portal checkout, EDI partner notes) kept separate from internal instructions.

Docstrings

| Where | Text |
|---|---|
| Customer / supplier `instructions` | Your team's standing instructions for this customer's orders, such as who receives invoices. ¶ Copied onto new orders that don't set their own. Always `null` for customer and supplier portal users. |
| Order `instructions` | Your team's instructions for handling this order. ¶ When an order is created without `instructions`, the customer's (or supplier's) instructions are copied in; later changes to the customer don't affect it. Always `null` for portal users. |
| Request · order `instructions` | Instructions for this order. Omit to copy the customer's (or supplier's) instructions; send `null` for none. |
| `order_instructions` | The order's `instructions`, shown here for convenience. ¶ Read-only; always the order's current value. Always `null` for portal users. |
| Payment `description` | Description of the payment, such as how it was captured. |

## Shared API kit (`apikit`) ✅ decided 2026-10-08

The generic API framework moves to a new repo, `github.com/open-mrp/apikit`, so other APIs can use it. The kit is built to the forge.1 contract from the start, and this repo adopts it in one migration when forge.1 ships, not package by package before. Nearly every kit package returns the kit's forge.1 `APIError`, so a partial adoption would leave two error types in flight. Until then, `api` keeps its own `shared/errors` and the current preview behaviour, with no compatibility transformers for older previews. The first consumer is a new small API, a single HTTP binary.

The cross-cutting framework changes in this review are implemented once in the kit: the error object (X8, G5: no `hint`), `rate_limited` and the idempotency and rate-limit rules (X10), the deleted stub, page size and `page_info`, and `Deprecation` / `Sunset` headers.

- [ ] **Kit contents:**
  - Utilities: `field`, `validate`, `pagination`, `crypto`, `id` generator, `retry`, `cache`, `redact`, `safeconv`, `ptrutil`, `timeutil` (with the `_on` date type) and `fuzzy`.
  - Errors: `apierror`, the forge.1 error object with a code registry.
  - HTTP layer: request binding and responses, `APIEndpoint` and its groups, includes, router, version engine, `sensitive` tag redaction (from `costguard`), generic middleware, an idempotency `Store` interface, and the forge.1 shapes (deleted stub, job, `Money`, `page_info`, metadata limits).
  - Service plumbing: request context (`appctx`), tracing, canonical logs, `db`, `lease`, `querytag`, and the S3, SQS and blobstore wrappers.
  - Tooling: a generic OpenAPI generator (with the agent-tool catalog, optional Stainless config, public-endpoint inventory and error-code index), forge.1 conformance checks, and `txaudit`.
- [ ] **Stays in `api`:**
  - Domain code: constants, protos, endpoint definitions, resources, request inputs, version transforms and include definitions.
  - Auth, platform, sandbox and subscription middleware, plus auth cookies.
  - Migrations and seeds.
  - Business error codes, which register as kit extensions: `limit_exceeded` with `quota`, `payment_required`, `agent_spending_cap_reached` and `registration_closed`.
- [ ] **Deferred until an API needs them:** gRPC contracts and `rpc`, `messaging` (outbox and RabbitMQ), `audit`, and the PlanetScale tools `vtparse` and `schemasplit`. They stay here, and moving them later is additive.
- [ ] **Identity and permissions are app-defined.** The context holds the app's own identity type (`Identity[T]`). An endpoint carries an `Auth` policy value checked by an `Authorizer` the app registers. Our permission fields (`RequiredPermissions`, `CounterpartyPermissions`, `SelfPathParam`, `RequiredRoleType`) become OpenMRP's policy type.
- [ ] **Migration at forge.1:** follow `docs/apikit-migration.md` (steps, rename map, what OpenMRP registers, client-visible changes, verification). In short: switch every import to apikit and delete the local copies. Register OpenMRP's error codes, ID prefixes, versions, sensitive-tag policies, `Authorizer` and identity type. Diff the generated OpenAPI spec and Stainless config against the pre-migration output, so the only changes are the forge.1 ones.

## Dashboard breaking changes

Everything the dashboard must change when it moves to the forge.1 version, grouped by decision. Each resource section above has the detail.

All resources
- `limit` max 100 (was 1000). Any call with `limit` > 100 must paginate (dropdowns, pickers, exports).
- `page_info.has_next_page` and `has_prev_page` are removed. Another page exists when `next_page_url` or `previous_page_url` is present.
- DELETE responses are `{id, object, deleted: true}` (was `{}`).
- Status value `inactive` → `archived` everywhere, except integrations, agents and account users (`disabled`).
- Deleting an in-use resource fails with 409 `resource_in_use` (was allowed for payment and shipping terms; was `resource_conflict` for units). Surface the archive path in the UI.
- Assigning an archived resource fails with 422.
- Money fields are `{amount, currency}` instead of a `Quantity` with a currency unit. Requests send `currency: "usd"` instead of a currency `unit_id`.
- Quantities have no `id`, `object` or `display_value`. `unit` is always populated, and the `include[]=*.unit` keys become invalid.
- Rates: `numerator_unit`/`denominator_unit` → `unit`/`per_unit`. No `id`, `object`, timestamps or `display_value`. Money-per-unit is `UnitPrice {amount, currency, per_unit}`.
- `display_value` is gone. The dashboard needs a shared formatter for money (`Intl.NumberFormat`), quantities and rates.
- `ComputedQuantity` merges into `Quantity`.
- Internal `PATCH /v1/operations/quantities/{id}` and `PATCH /v1/operations/rates/{id}` are removed.

Payment terms
- Create and update accept `status`; list accepts `statuses[]`.

Shipping terms
- `type` values `free_freight | flat_rate_freight | carrier_rate_freight` → `free | flat_rate | carrier_rate`.
- Response `minimum_order_value` + `free_shipping_service_levels` → `free_shipping {minimum_order_value, service_level_scope, service_levels}`. Request `minimum_order_value` + `free_shipping_service_level_ids` → `free_shipping {minimum_order_value, service_level_ids}`. Include key `free_shipping_service_levels` → `free_shipping.service_levels`.
- `flat_rate` required exactly when `type = flat_rate`; free shipping rejected on `free` terms; names must be unique.
- New `status`; create and update accept it; list accepts `statuses[]`.

Unit groups
- Unit group pages (list, detail, import, `EditUnitConversionForm`) are removed; new UI is needed for orderable units, default unit, portal visibility, pack discounts and stocking-unit correction.
- `category.unitGroup.baseUnit` (default lot unit, blank quantities, import rate denominator) → the item's `stocking_unit`.
- Includes like `...unit_group.base_unit` and `unit_group.associated_units` on orders, receiving and purchase orders become invalid.
- Item import: lead-time unit and currency no longer come from the system "time"/"price" groups.
- Analytics client's unit-group mapping (`analytics.api.ts`) must be reworked.
- Item categories and product lines no longer take `unit_group_id` on create.
- Item create requires `stocking_unit_id`; items expose `stocking_unit`, `default_order_unit`, `order_units` (expandable).
- Order-line unit pickers read the item's `order_units` + stocking unit and preselect `default_order_unit` (sales and purchase orders, receiving).
- `customer_portal_visibility` → `portal_visibility`; pack discounts move from unit-group rows to `order_units[].discount` (typed, Money/decimal).
- New UI: per-item order units, bulk "set order units" (job), and "change stocking unit" with convert/relabel (job).

Item categories
- `type` values `material_category | product_category` → `material | product`; list filter `type` → `types[]`.
- `unit_group`/`unit_group_id` and the change-unit-group page action removed.
- Property attach/detach/create-and-attach endpoints removed: send `property_ids` on create/update instead; create properties through the Properties API.
- `notes` removed.
- Delete blocked while items or price-list rules use the category (409).

Items (structure)
- `/materials`, `/parts`, `/products` pages and calls move to `/items` with `type` (+ `types[]` filter). Every `product_id`/`material_id`/`part_id` (order lines, supplier materials, price-list rules, URLs, routes) becomes the item ID. `include[]=item` is gone: item fields are top-level; product/material fields move under `sales`/`material`.
- Item `type` gains `service`; product fields live under `sales` (`sales.product_line`, `sales.portal_visibility`); `product.type` is gone. Shipping and credit charges stop being products: show and edit them as order line types; hide no more system products in pickers.
- `unit_value` → `sales.unit_price`; `unit_cost`/`burn_rate` no longer expandable (drop those include keys). `notes` gone.
- Soft-deleted items reappear as `archived`; deleting an item in use fails (409) — offer archive. New `status`.
- One bulk upsert for all item types; one-attribute-per-property enforced.
- Item list: `starts_at`/`ends_at` → `created_after`/`created_before`, `portal_visibility` → `portal_visibilities[]`, `supplier_id` → `supplier_ids[]`; use `skus[]` for exact lookups.
- Bulk reconcile → `POST /operations/inventory-movements/actions/bulk-create` (Inventory N1), `reconcile_type: addition | force` → `operation: adjust | reconcile`.
- Change category / product line / attributes via `PATCH /items/{id}` (`category_id`, `sales.product_line_id`, `attribute_ids`) instead of the PUT sub-endpoints.

Customers
- Request bodies change shape to mirror the response: `contact {…}`, `carrier_billing {…}`, `defaults { carrier_id, service_level_id, payment_term_id, shipping_term_id, sales_rep_id, bill_to_address_id, ship_to_address_id, receive_calendar_id, lead_time_days, fulfillment_policy }`, `group_id`, `price_list_ids`, top-level `freight_policy`.
- `status` (credit standing) → `standing` (no `preferred`); new lifecycle `status`.
- Response renames: `type` → `group`, `contact_info` → `contact`, `freight_preferences` dissolved (`freight_policy` top-level, carrier/service level under `defaults`, billing under `carrier_billing`), `parent_account`/`child_accounts` → `parent_customer`/`child_customers`, addresses under `defaults`, `receive_calendar_id` → `receive_calendar`.
- `credit_limit` is Money.
- **`notification_preferences` removed.** `src/app/_lib/api/client/customer.api.ts:293,356,406` maps `notification_preferences.accepts_invoice_emails` → `acceptsInvoiceEmails` (and requests the include). Replace with the customer's `notification_recipients` (Messaging M2). Invoices also drop `accepts_invoice_emails`, used to disable "send invoice email" (`InvoiceTableRow.tsx:129`, `invoice.api.ts:172,219`); the button checks the customer's invoice recipients instead so it keeps working.
- Merge returns an async job: poll it, then read the target customer.
- List filter renames: `standings[]`, `statuses[]`, `group_ids[]`, `price_list_ids[]`, `commission_policies[]`, `freight_policies[]`, `relationship_types[]`, `created_after`/`created_before`. `q` no longer matches notes or support email.

Account groups & lookups
- Account group `type` removed (classification only); `default_lead_time_days` → `defaults.lead_time`; new `price_lists`.
- Account statuses and priorities endpoints removed: use the `standing` enum with a dashboard label map. Priority is gone everywhere (customers, orders, POs): show the ship-by date / lead time instead (picks, shipments, invoices, acknowledgment PDF, customer export).

Addresses
- Routes move `/v1/sales/addresses` → `/v1/core/addresses`. Address fields flatten: `geolocation.street_line_1` → `line_1`, `street_line_2` → `line_2`, `locality` → `city`, `state`, `postal_code`, `country` top-level; requests use the same names (validate: `address_line_1` → `line_1`, `address_line_2` → `line_2`). No geolocation `id`.
- `country` must be a valid uppercase ISO code.
- List filter `type` → `types[]`.
- Address autocomplete (`/suggestions`) is internal-only (dashboard may keep using it).
- Documents embed their address snapshot; show the document's own copy, not the live address.

Pricing
- Account prices, volume discounts and pricing account groups are replaced by **price lists** (`/v1/sales/price-lists` + `/rules`): new screens for lists and rules (fixed, percent of base, tiered). Customer `price_groups` → `price_lists`; account groups gain `price_lists` and lose `type`.
- Percentages are whole numbers (`"5"` = 5%, was `"0.05"`); discounts use `{ type, percent_off | amount_off }`.
- Order discounts: `discount` object, `status`, `order_count` → `times_redeemed`.

Sales orders (the order)
- Request fields renamed to match the response: `customer_id`, `bill_to { address_id | inline }`, `ship_to { … }`, `freight { carrier_id, service_level_id, carrier_billing }`, `commitment { promised_at, lead_time_override, ship_by_override_date }`, `email_recipients [{ account_user, documents }]`; `acknowledgement_*` → `acknowledgment_*`.
- Response: `bill_to`/`ship_to` are snapshots (`…_address` → `bill_to.address` reference); `contacts` → `email_recipients`; `note` → `instructions` (internal); `completed_at` → `fulfilled_at`, `first_ship_at` → `first_shipped_at`, `expired_at` → `expires_at`; `payment_intent_ids` removed.
- Status actions: `PUT …/close|open|issue|unissue` → `POST …/actions/issue|unissue|cancel`; `fulfilled` is automatic once every line ships (Q5).
- Orders with a shipment or invoice can't be deleted (409).
- List: `q` is exact match; `status_codes` → `statuses[]`, `customer_group_ids` → `group_ids[]`, `starts_at`/`ends_at` → `created_after`/`created_before`. `/v1/sales/sales-order-statuses` and `/purchase-orders/statuses` removed (enums).
- Order totals are Money: `subtotal`, `shipping`, `discount`, `tax`, `total`, `invoiced`, `paid` (no more `ordered` + stage strings). Price quote → `actions/preview-prices`; checkout → `actions/create-checkout-session` (`url`); `/v1/sales/checkout-sessions` removed.
- Lines: `type` (`item | shipping`), `item_id` (no `product`), `sku`, `description`, `quantity`, `line_number`; `unit_price`/`unit_cost` as UnitPrice and `amount` as Money (not expandable); per-line `fulfillment` quantities replace money stage totals; no credit or negative discount lines; lines can't be deleted once packed/shipped (no admin override).

Settings
- Integration forms send typed `credentials` objects (not JSON strings); reconnecting uses `actions/rotate-credentials` (create on a connected provider → 409); status `inactive` → `disabled`; show `connection_status`. Portal domain verification no longer needs polling.

Messaging
- Conversation/message/notification `topic`/`resource` objects → `related` stubs; `unread` → `unread_count`; preference booleans → `channels`; preferences and email sender use PATCH; email sender's `domain`/`domain_status` → include `email_domain`; email domain `dkim_tokens`/`mail_from_*` → `dns_records`; deleted messages arrive as `status: deleted` with null body; notification `change_count` → `occurrences`.
- Customer "accepts invoice emails" comes from the customer's `notification_recipients` (include it); invoices drop `accepts_invoice_emails`, so the send-invoice-email button checks the customer's invoice recipients.
- Notification settings page moves to the single preferences object (`default`, `types`, `auto_subscribe`, `paused_until`) instead of per-category rows. Records get a Follow/Unfollow control (subscriptions). Admin settings gain account notification defaults, and the role editor gains notification defaults. Editing an order no longer follows it unless the user opts in.

AI
- Agent pages move to agent + versions (draft → activate); agent settings come from the version, `instructions` only when included; tools are a paginated sub-resource edited one at a time (`tools: { access, count }` on the version).
- Run pages and "trigger run" go away: agents are used through conversations (start a chat with an agent; scheduled runs open a new conversation each time; tool-call approvals on agent messages via `/v1/ai/tool-calls/{id}/actions/approve|reject`; stop/retry actions). Conversation list gains `origin` and an agent filter.
- "Message someone" opens the latest conversation with them plus a New conversation button (creating no longer reuses the DM); "Continue in new conversation", generated titles for agent chats, delete for chats only you and agents are in.
- Renames: `available_tool` → `tool` (`category` → `type`, `mutating` → `effect`); `agent_action` → `tool_call`; inbox `agent_config` → `agent`, `agent_trigger_*` → `agent_trigger`; `PUT …/status` removed (PATCH `status`); `system_prompt` → `instructions`; `definition_type` → `owner`.
- Memories page becomes notes (`/v1/core/notes`), and records gain a notes panel filtered by `record_ids[]`.

Pre-release changes (X/Y)
- `lot` sizes → `lot_size` (`default_lot_size`, lot-size defaults); movement `lot` is an expandable lot reference, not a string.
- Movement `type` values change to business events (`receipt | shipment | consumption | output | transfer | adjustment | reconciliation | return`).
- Lead times are business-day Quantities (`defaults.lead_time`, `commitment.lead_time_override`); dates are `YYYY-MM-DD` `_on` fields (`promised_on`, `ship_by_on`, `due_on`, `expires_on`); date pickers send dates, not timestamps.
- Sales order `estimate` → `draft`; new `canceled` status + cancel action on sales orders, POs and runs.
- Operation `setup_time` is per run; methods gain `output_quantity`; items gain `default_manufacturing_method`.
- Errors carry an `errors[]` list for 422s (show every field error); bulk/job row errors are objects.
- Applications use `source` / `target`.
- Lines gain an optional `discount` and snapshot `stocking_quantity`; order units can carry a per-item `quantity`; the `sales` section is allowed on materials and parts.

Make to order (O1–O9)
- Order → runs is plural (`related.production_runs`, `create-production-runs` returns a List); production is pegged to order lines via `demand`.
- `fulfillment_policy` → `supply: stock | make | buy` (items, product lines, customer `defaults.supply`); order lines show resolved `supply`; issue reserves only `stock` lines.
- Order actions `quote-prices` / `quote-freight` / `quote-commitment` → `preview-prices` / `estimate-freight` / `preview-commitment`.
- Operations show `type: inside`.

Jobs and work orders (P1–P4)
- Production runs → **work orders** (weekly release groups, `/v1/operations/work-orders`) and **jobs** (one item each, `/v1/operations/jobs`, with method copy, `demand[]` and `status`). Run detail screens split into work order (list of jobs) and job (operations, materials, batches).
- Order "create production run" → `create-jobs` (can add to an existing work order); order shows `related.jobs`.
- Batch/downtime `related.production_run` → `related.job`.
- New lot genealogy views: lot upstream/downstream and an order's lots; pick and ship screens record lots taken.

Unusual designs (Q1–Q5)
- Adding a customer/supplier contact no longer invites or emails them; an explicit Invite action does (`portal_access` shows the state).
- Sites: location type `site` with address; orders, shipments, POs, receiving, jobs and work orders show a `site` (single-site accounts can ignore it).
- `default_lot_size` moves from product lines to item categories; product line screens keep commercial settings only.
- Commitments show `promised_lead_time` and its source; items gain an optional sales lead time; delivery performance reads the snapshot.
- Fulfill/reopen (orders) and complete/reopen (POs) buttons go away: status updates itself; Cancel closes the remainder.

Agent experience (G1–G7)
- API key screens show `client` and `expires_at`; audit/activity shows the API key's client ("API key · Claude Code").
- New claim page: a human claims an agent-created sandbox, signs in, sets up billing and goes live.
- Sandbox creation offers a sample-factory seed.

Authentication & permissions
- Login is passwordless: passkeys (primary), email link/code, second factor (passkey or TOTP) with recovery codes; 2FA required by default for new accounts. Remove password login, reset and scanner-password screens.
- Stations become devices: device management screens (create → pairing code/QR, revoke, re-pair; optional work center / machine / station links). Station terminal app pairs with a code instead of a username/password.
- Role editor: template picker, per-area access levels, grants/revokes, effective-permission preview.

Identity
- Account user actions → POST; removed users visible with `status: removed`; `department` → `work_center`; `is_commission_eligible` → `commission_eligibility`.
- Adding users sends a sign-in link, never a password.
- Permission groups → `/identity/permissions` catalog; API key `status`, revoke via `actions/revoke`, create/rotate response `{ secret, api_key }`; favicon upload internal; customer parent set via `parent_customer_id` (child-accounts routes removed).

Core
- Request log bodies: read via `GET /core/request-logs/{id}/bodies` (separate call, new permission); list/retrieve no longer carry bodies.
- Search results: `include[]=data.resource` to get labels (no `entity` stubs); `customer` filter → `customer_ids[]`.
- Audit events: `root_resource_type` / `root_resource_id` → `root_resource_types[]` / `root_resource_ids[]`.
- Pack list: `POST /core/records/actions/generate-pack-list` → `POST /v1/operations/shipments/{id}/actions/generate-pack-list`.
- Analytics: all internal (OEE, OEE trend, delivery performance, schedule attainment included) — dashboard keeps using internal routes; PUT → GET/POST query.
- Jobs: cancel → `actions/cancel`; job list available. Audit/request log date filters → `occurred_after`/`occurred_before`; audit resource-types endpoint removed.

Production planning
- Schedules, schedule settings, item/resource settings, fulfillment recommendations and demand overrides are replaced by demand + demand adjustments, item and work center `planning` sections, versioned production plans with planned orders / load / exceptions, pull release under WIP caps, and a weekly plan-vs-actual attainment view. All internal (private beta); the planning screens are rebuilt on them.

Production execution
- Shop-floor scanning client: `batches/actions/initialize|move|merge|split` → `POST /batch-scans` with `action`; step pickers / remaining / consumption previews → one `batch-scans/actions/preview`; undo via `batch-scans/{id}/actions/reverse` (not batch delete); `close` takes the batch ID in the path.
- Batch screens (run detail, flow graph) compose standard resources: request `include[]=item,work_center,operation,machines`, read run and parent/child batch IDs from `related`; the flow graph is rebuilt from `GET /batches/{id}/flow` (a batch list) instead of `BatchFlowNode`. Machine status board includes `machine`, `work_center`. Downtime reads `production_run`/`batch` from `related`.
- Runs: `batch_summaries` → batches in `related` + per-operation `progress`. Batches: `production_step` → `operation`, `seconds` → `duration`, `waste` → `scrap_quantity`. Downtime: `reason` enum (no `/machine-downtime-reasons`), `source` values, `reported_by` → `actor`, `duration_seconds` → `duration`. Machine status: `week_*` floats → `this_week` Quantities.
- The `Entity` stubs (`item.name`, `department.name`, `production_step.id`, `production_run.id`, `input_batches`/`output_batches` without includes in `mapBatchFromSDK`, `production-run.api.ts:84-107`; `machine.name`/`department.name` on the machine status page) are gone.

Production structure
- Production steps, consumptions and flows are replaced by versioned **manufacturing methods** (operations + materials per item; draft → activate). New method editor: edit drafts, activate versions; the production graph is derived from materials (no step linking UI).
- `in_steps`/`out_steps`, connect-steps and `productions/{id}` are gone; scanning stations are set on operations; leveling/allowances are whole-number percents; `notes` removed from work centers, machines, stations. **Departments are renamed work centers** (routes, fields, filters).

Inventory
- Stock changes: `PATCH /items/{id}/inventory` → `POST /operations/inventory-movements` (returns the movement, with `reason` enum); bulk reconcile → `inventory-movements/actions/bulk-create` (202 job).
- Change logs → inventory movements: `action_type` → `type` (`receipt | shipment | consumption | output | transfer | adjustment | reconciliation | return`), `responsible_*` → `actor`; filters `actor_ids`/`actor_types`/`occurred_after`/`occurred_before`; deprecated GET export removed.
- New public `inventory-levels` list for many items/locations; lot default `quantity` float → `lot` Quantity or null; `/location-types` removed; locations archivable.

Fulfillment
- Shipments, shipping cases, deliveries move to public routes; pick `pick` → `pick-all`, `void` → `reset` (POST); pick line `quantity` → `picked_quantity`; `finished_at` → `completed_at`.
- Shipment status `packed | ready | labeled | shipped` (+ `tracking_status`); Shippo shipments become `shipped` on carrier pickup, self-managed on the ship click; `is_ready_to_ship` → `status: ready`; `void` → `unship` (voids the invoice, or 409 if anything is applied); `admin-update-tracking` → `correct-tracking`; `note` → `order_instructions`; `shipping_address` → `ship_to`; shipping cases listable per shipment; `freight_amount` is Money.

Purchasing
- Purchase orders: status `estimate|issued|fulfilled` → `draft|issued|completed`; `change-status` → `POST …/actions/issue|unissue|cancel` (`send_email` → `notify_supplier`; `completed` is automatic, Q5); `scheduled_at` → `promised_at`; request/response renames as sales orders (`bill_to`/`ship_to`, `freight`, no `priority`, `email_recipients`, `instructions`, line `item_id`/`sku`/`description`/`quantity`/`line_number`); totals Money; line `fulfillment` quantities.
- Supplier materials → supplier items (`/suppliers/{id}/items`, `item`, `supplier_sku`, `archived`); `material_count` removed; supplier `note` → `instructions`; supplier shape mirrors customer.
- Receiving: `receive` → `receive-all`, `void` → `reset` (POST); stocked lines can't be reset; line `quantity` → `received_quantity`.

Accounts payable
- New bills and supplier credits screens (buyer views of invoices and credit notes; supplier invoices from OpenMRP suppliers appear automatically); payments show direction; refunds are payments with `type: refund`.

Accounts receivable
- Finance screens move to the new resources: invoices (with status, stored totals, actions finalize/void/mark-uncollectible; no delete), credit notes (replacing credit-memo / rebate / adjustment transactions), payments (with applications and deductions; replacing settlements; refunds are payments with `type: refund`), applications (replacing allocations and open credits).
- Invoices can be created ahead of shipment (`source: order`) and manually (`source: manual`).
- Invoice totals and lines are snapshots: stop computing them from order lines.
- Transaction types / methods / adjustment types endpoints removed (enums).
- Invoice `has_been_sent` / `is_edi_sent` → `delivery_status`; `is_paid_in_full` / `is_over_paid` removed (use `status`, `totals`); `accepts_invoice_emails` removed.

Notes → instructions
- Customer/supplier `note` → `instructions`; sales/purchase order `note` → `instructions` (now internal: null in the portal).
- Creating an order without `instructions` copies the customer's/supplier's; sending a value uses it exactly (no more concatenation). Order forms should prefill from the customer and let staff edit.
- Shipment, invoice, pick, receiving order: `note` gone; show read-only `order_instructions`. Remove note editing on shipments and invoices.
- `notes` removed from departments, machines, items/materials/parts/products, item categories, product lines, production steps/flows, scanning stations; existing values appear in the record's notes panel (AI A6).
- Transaction/settlement `note` → payment `description`; allocation copy removed.

Product lines
- `commission_policy` values → `applied | exempt`, `freight_policy` → `billed | free` (also on customers, account groups, access grants).
- `unit_group` removed; `default_lot_size` no longer expandable (drop `default_lot.unit` include).
- `notes` removed; `description` now editable.
- Reserved Shipping/Service/Credit/Tax lines disappear (no more hiding them by name). New `status`.
- Delete blocked while products, price-list rules, sales targets or territories use the line (409).

Properties & attributes
- Deleting a property or attribute in use now fails (409) instead of stripping it from items; new `status` on attributes.
- `attribute.property` is null unless included (`include[]=property`, or `attributes.property` on items).
- Attribute values may repeat across properties; any lookup by value must include the property.
- Omitted attribute `color` is `default`, not random (pick a color client-side if wanted).

Carriers
- New optional `scac` field (carrier forms).
- `code` → required `type`; values `fedex | ups | usps | ltl | will_call | local_delivery | other` (`delivery` → `local_delivery`, `ltl1` → `ltl`, `null` → `other`, `freight_collect` removed).
- `deleted_at` removed; `customer_portal_visibility` → `portal_visibility`; new `status`.
- `default_service_level` / `default_service_level_id` on the carrier replaces `is_default` on service levels.
- Carrier and service level delete blocked while in use (409); delete returns the stub.
- `service_levels` include: first 10 + `page_info.next_page_url`.

Service levels
- `service_level_token` → `code`; `default_transit_days` → `estimated_transit_days`; `is_default` removed; `customer_portal_visibility` → `portal_visibility`; new `status`.

Units
- `type` → `dimension`; list filter `type` → `dimensions[]`; `currency` is no longer a dimension, so currency units disappear from lists.
- `is_base_unit` removed.
- Bulk upsert matches on `abbreviation` only, and rows no longer send `is_base_unit`.
- `abbreviation` has a max length.

## Private integrations

Breaking changes for private integrations are tracked outside this public repo.

## Industry identifiers backlog

**Deferred until after forge.1 (decided 2026-10-07):** all additive, so none are part of this review. `carrier.scac` is the exception, already adopted. Kept here for later.

| Standard | Lands on | Review | What it replaces / enables |
|---|---|---|---|
| SCAC | `carrier.scac` | carriers ✅ | Hand-maintained carrier-type → SCAC maps in EDI integrations |
| X12 unit-of-measure code (355: `EA`, `PR`, `CA`, `BX`) | `order_units[].edi_unit_code` (per item, because `CA` means a different pack per product) | items | Integration config that hard-codes product-line and SKU-prefix rules to turn `CA`/`EA` into pack units |
| GTIN (UPC-A / EAN-13 / GTIN-14) | `item.gtin` (each) and `order_units[].gtin` (case/pack level, GS1 packaging hierarchy) | items | Matching 850 lines by UPC (`PO1` qualifier `UP`/`UK`) instead of vendor SKU; FDA UDI/GUDID for medical devices |
| NMFC freight class | `item.freight_class` (`50`…`500`) | items | LTL rating |
| HS tariff code + country of origin (ISO 3166-1) | `item.hs_code`, `item.country_of_origin` | items | Customs, commercial invoices, international shipping |
| Freight payment method (X12 `PP`/`CC`/`TP`) / Incoterms 2020 | `shipping_term.freight_payment: prepaid \| collect \| third_party` (+ `incoterm` later) | shipping terms follow-up | Where `freight_collect` belongs; EDI 856 `TD5`/`FOB` |
| Payment terms structure (X12 `ITD`: discount %, discount days, net days) | `payment_term` | payment terms follow-up (deferred) | `2/10 Net 30`, due-date math |
| GLN / DUNS | addresses (ship-to, DC) and accounts | addresses, customers | Partner DC/location codes on EDI `N1` segments (today hand-maintained in integration config) |
| ISO 3166-1 alpha-2 / ISO 3166-2 | `address.country` (validate), `address.state` | addresses | Today `country` is "two-letter country code", with no named standard or validation |
| SSCC-18 | `shipping_case.sscc` | — | Already modeled |
