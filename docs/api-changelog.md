# API changelog

Changes to the public API, by the `OpenMRP-Version` that introduces them. A client pinned to an older supported version keeps the shape described for that version; see [API versioning](patterns/api-versioning-patterns.md).

## 1.0.forge-preview.8

### Breaking changes

- **Invoice search matches the start of each field.** On `GET /v1/finance/invoices`, `q` now matches the start of the invoice number, the sales order number, the customer PO number, and the customer's name, number and alias, ignoring case. It no longer matches the middle of those fields, and it no longer searches the invoice note or the customer's notes. A search for part of a number such as `2410` finds invoice `24109`; one for `4109` no longer does.
  - New query parameter `q_match`: `prefix` (the default) or `contains`. `contains` keeps the previous matching, anywhere in every field and in the notes, and is slower on accounts with many invoices.
  - Older versions: a list pinned to 1.0.forge-preview.7 or earlier that sends `q` without `q_match` is searched with `contains`, as before.
  - Migration: nothing to change for searches by the start of a number or name. To keep matching the middle of a field or the notes, send `q_match=contains`.

## 1.0.forge-preview.7

### Breaking changes

- **Customer `edi_status` is removed.** OpenMRP no longer exchanges EDI documents itself, so it no longer stores whether a customer trades over EDI. `edi_status` is gone from the `Customer` resource and from `POST /v1/sales/customers` and `PATCH /v1/sales/customers/{id}`.
  - Older versions: the field is gone from responses in every version, since there is no value left to return. A request pinned to 1.0.forge-preview.6 or earlier that still sends `edi_status` has it dropped rather than refused.
  - Migration: stop reading and sending `edi_status`. An integration that trades EDI with some customers keeps that list itself.

## Unreleased

Not breaking, so it ships on 1.0.forge-preview.6 and applies to every version.

### Additions

- **`metadata` on sales orders, sales order lines and invoices.** A map of string keys to string values that the client sets and reads back unchanged; OpenMRP never reads it. It is returned on `SalesOrder`, `SalesOrderLine` (also as `lines.order_line` on invoices and `lines.sales_order_line` on shipments and picks) and `Invoice`, and is `{}` when nothing is set.
  - Set on create with `POST /v1/sales/sales-orders` (on the order and on each of `lines`) and `POST /v1/sales/sales-orders/{id}/lines`. A key sent as `null` is not stored.
  - Changed with `PATCH /v1/sales/sales-orders/{id}`, `PATCH /v1/sales/sales-orders/{id}/lines/{line_id}` and `PATCH /v1/finance/invoices/{id}`. Keys left out are kept; a key set to `null` is removed; `"metadata": null` removes every key. An empty string is stored as a value.
  - At most 50 keys per object, counted after the update; keys up to 40 characters without `[` or `]`; values up to 500 characters. An update that would leave more than 50 keys is refused with `400` on `metadata` and changes nothing.

### Fixes

- **Validation errors name a top-level list or map entry by its JSON name.** A failed check on an entry of a top-level array or object field reported `param` with the Go field name, such as `RecipientEmails[0]`; it is now the JSON path, such as `recipient_emails[0]`. Entries of nested fields were already named this way.

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
- **Internal notes, commission settings, user activity and legal holds are withheld from portals.** Customer and supplier portal users read these fields as `null` wherever the resource appears, including through includes; the seller's own users, API keys and agents read them as before:
  - `note` and `commission_policy` on customers.
  - `notes` on items, product lines and item categories, and `commission_policy` on product lines and account groups.
  - `is_commission_eligible` and `last_used_at` on account users.
  - `legal_hold` on conversations.
  - `commission_policy`, `is_commission_eligible` and `legal_hold` were always set and are now nullable. They are also `null` on a customer, account group, product line or account user embedded as a reference that does not load them.
  - Older versions: `commission_policy` and `legal_hold` read `""`, and `is_commission_eligible` reads `false`, wherever they are `null` here. The notes and `last_used_at` were already nullable and read `null` in every version.
  - Migration: read `null` on these fields as "withheld", not as a value.

### Fixes

These apply to every version, since an older version's behavior here was a defect rather than part of its contract.

- **Purchase orders require a bill-to and a ship-to.** `POST /v1/operations/purchase-orders` used to accept an order with no bill-to or ship-to and create blank addresses for it, with no name and no country, which the address resource documents as required. Each address is now required, given by `bill_to_address_id` / `ship_to_address_id`, by the inline `bill_to_address` / `ship_to_address` object, or by the flat `bill_to_*` / `ship_to_*` fields.
  - An order missing one is refused with `400 missing_field` on `bill_to_address_id` or `ship_to_address_id`.
  - Flat fields must include at least the name and the country; one that leaves either out is refused with `400 missing_field` on that field (for example `bill_to_country`).
