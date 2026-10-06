# API changelog

Changes to the public API, by the `OpenMRP-Version` that introduces them. A client pinned to an older supported version keeps the shape described for that version; see [API versioning](patterns/api-versioning-patterns.md).

## 1.0.forge-preview.6

### Breaking changes

Clients pinned to 1.0.forge-preview.5 or earlier keep the previous behavior.

- **Registration flow option lists.** `PATCH /v1/sales/registration-flows/{id}` replaces an option list (`customer_group_ids`, `payment_term_ids`, `shipping_term_ids`) whenever it is sent, and `[]` removes every option. An omitted list is left unchanged. The `has_customer_group_ids`, `has_payment_term_ids` and `has_shipping_term_ids` flags are gone.
  - Older versions: a list is replaced only when its `has_<list>` flag is `true`, and is ignored otherwise.
  - Migration: drop the flags, and send a list only when you mean to replace it.
- **Cost figures need `costs:read`.** The seller's cost and margin figures are returned only to its own users and API keys whose role holds `costs:read` (admins always do). For anyone else, including customer and supplier portal users, they are `null`. These fields were plain numbers and are now nullable:
  - `changeover_labor_rate` on production schedule settings (`GET` and `PUT /v1/operations/production-schedule-settings`).
  - `unit_cost`, `setup_cost` and `holding_cost` on each of `policies` in a production schedule preview (`PUT /v1/operations/production-schedules/actions/preview`), and on production schedule item policies (`GET /v1/operations/production-schedules/{id}/item-policies`).
  - `annual_cogs` on fulfillment recommendations (`GET /v1/operations/fulfillment-recommendations` and `POST /v1/operations/fulfillment-recommendations/actions/apply`).
  - Older versions: these fields stay numbers, and read `0` where the caller cannot read costs. A caller who holds `costs:read` gets the same figures in every version.
  - Migration: read `null` as "withheld", not as zero, and grant `costs:read` to the roles that plan with these figures.
  - In every version, the fields that were already nullable are `null` for such callers too: `labor_rate` on departments, `labor_rate` and `overhead_rate` on production steps and production flow steps, and the `unit_cost` include on items, sales order lines and delivery lines. Saving production schedule settings without `costs:read` keeps the stored `changeover_labor_rate` rather than replacing it with the value sent.

### Fixes

These apply to every version, since an older version's behavior here was a defect rather than part of its contract.

- **Purchase orders require a bill-to and a ship-to.** `POST /v1/operations/purchase-orders` used to accept an order with no bill-to or ship-to and create blank addresses for it, with no name and no country, which the address resource documents as required. Each address is now required, given by `bill_to_address_id` / `ship_to_address_id`, by the inline `bill_to_address` / `ship_to_address` object, or by the flat `bill_to_*` / `ship_to_*` fields.
  - An order missing one is refused with `400 missing_field` on `bill_to_address_id` or `ship_to_address_id`.
  - Flat fields must include at least the name and the country; one that leaves either out is refused with `400 missing_field` on that field (for example `bill_to_country`).
