# Dashboard API Consumer Audit — Expandable-Stub Refactor

> **Why this exists.** The `refactor/eliminate-expandable-stubs` branch changed how the public
> API returns nested sub-resources. Previously many nested objects were **fabricated** and
> returned **always** (e.g. an `order` object with invented `status`/`priority`). Now the API
> follows one rule consistently:
>
> **An expandable sub-resource is `null` unless the caller explicitly requests it via
> `?include=…`. When requested, it is populated with real data. The API never fabricates
> placeholder sub-resources.**
>
> This is a **behavioral (and in places structural) change** to GET/POST/PATCH responses.
> Anywhere the dashboard read a nested object _without_ asking for it, it will now read `null`.
>
> Use this document to walk the `/dashboard` repo endpoint-by-endpoint and confirm we didn't
> introduce regressions. Check the box once a consumer has been verified.

---

## 1. The two things that bite most often

### (a) `quantity.unit` is no longer returned by default

A `Quantity` (`{ id, object, value, display_value, unit }`) and a `Rate`
(`{ …, numerator_unit, denominator_unit }`) now return their **`unit` object as `null`** unless
a nested unit include is explicitly requested.

- ✅ **The unit abbreviation is still available** on every quantity via **`display_value`**
  (e.g. `"$1,234.56"`, `"100 kg"`). Prefer this for display.
- ⚠️ The structured `unit` object (id, ratios, type, etc.) is **only** returned when a nested
  `?include=<field>.unit` is requested — **and not every quantity-bearing endpoint advertises
  that nested include today.** If a consumer needs the full `unit` object, confirm the endpoint
  accepts the nested include (a 400 `parameter_invalid` means it doesn't) or fetch the unit
  separately.

**Resources/fields that carry a `Quantity`/`Rate` (audit every place these render a unit):**

| Resource | Quantity/Rate fields |
|---|---|
| Customer | `credit_limit` |
| Material | `order_point`, `lead_time` |
| Item inventory | `on_hand`, `reserved`, `available_to_promise`, `short` |
| Sales order line | `quantity_ordered`, `unit_price` (Rate), `unit_cost` (Rate) |
| Shipping term | `flat_rate`, `minimum_order_value` |
| Shipping case | `freight_amount`, `freight_weight` |
| Pick line / Receiving order line / Invoice line / Delivery line | their quantity fields |
| Inventory change log | `quantity` |

### (b) Expandable references are `null` unless included

Any field tagged "Expandable via `include[]=…`" returns `null` unless requested. The dashboard
must pass `?include=` (comma-separated, nested with dots, e.g.
`?include=customer,lines,lines.product`) for anything it renders.

---

## 2. Breaking structural changes (field renames / removals / type changes)

These are **not** just "now null" — the shape changed. Consumers referencing the old names/types
will break regardless of includes.

### Sales Order (`/v1/sales/sales-orders…`) — **largest change**

The `SalesOrder` summary and detail shapes were unified and restructured:

| Old | New |
|---|---|
| `customer_po` | **`customer_purchase_order_number`** (renamed) |
| `is_acknowledgment_sent` (bool) | **`acknowledgment_status`** (enum string) |
| `status` (object `SalesOrderStatusDetail`) | **`status`** (enum code string) |
| `type` (object `SalesOrderType`) | **removed** from the order body |
| `priority` (object `Priority`) | **`priority`** (enum code string) |
| `carrier`, `service_level`, `carrier_billing_type`, `carrier_billing_account` | consolidated into **`freight`** (new object, expandable) |
| `sales_rep` (always present) | **`sales_rep`** now **expandable** (`Actor`) |
| `production_run`, `pick` (top-level) | moved into **`related`** → `{ pick, production_run, shipments }` (each a `Record`, expandable) |
| `lines` (`SalesOrderLineDetail`) | `lines` (`SalesOrderLine`) |
| — | new **`payment_status`** (enum), **`totals`** (expandable), **`related`** (container, always present; members expandable) |

New value objects to be aware of: **`Freight`**, **`SalesOrderTotals`**, **`SalesOrderRelated`**, **`Record`** (a lightweight `{ id, object, type, number?, status?, metadata? }` reference used by `related.*`).

### Sales Order Line

| Old | New |
|---|---|
| `item` (`Item`) | **`product`** (`Product`, expandable) |
| `quantity_ordered`, `unit_price`, `unit_cost` (always present) | now **expandable** |
| `quantity_picked`, `quantity_packed`, `quantity_invoiced` | **removed** (see `totals` / pick/shipment resources instead) |
| `edi_line_item_id` | **removed** |
| `completed_at` | **removed** (column no longer exists) |
| — | new **`totals`** (expandable) |

### Receiving Order (`/v1/operations/receiving-orders…`)

| Old | New |
|---|---|
| `purchase_order` (`*SalesOrderDetail`) | **`purchase_order`** (`*PurchaseOrderDetail`) — type fix |
| `supplier` (always present) | **`supplier`** now expandable |

### Invoice / Shipment / Pick / Delivery / Transaction / Production

- **Invoice** `order` / `order_line`: type `SalesOrderDetail`/`SalesOrderLineDetail` → `SalesOrder`/`SalesOrderLine`; expandable (null unless included).
- **Shipment** `sales_order` / `order_line`: type updated to `SalesOrder`/`SalesOrderLine`; `sales_order`, `customer`, `carrier`, `service_level`, `shipping_address`, `shipped_by`, `invoice`, `pick`, `lines`, `shipping_cases` all expandable.
- **Pick** `sales_order`, `customer`, `lines`, `departments` expandable (**`priority` stays always-present**).
- **Delivery / Transaction / Production-flow / Production-step**: nested `sales_order` / `sales_order_line` references retyped to `SalesOrder`/`SalesOrderLine` and made expandable.

---

## 3. Action endpoints no longer accept `?include=`

The sales-order status actions are now plain commands that return the updated order **without**
include expansion. If the dashboard relied on the old combined status-change endpoint returning
expanded data, it must re-fetch via `GET /v1/sales/sales-orders/{id}?include=…`.

- `POST/PUT /v1/sales/sales-orders/{id}/actions/issue`
- `…/actions/unissue`
- `…/actions/open`
- `…/actions/close`

(The previous single `change-sales-order-status` endpoint was removed and split into these four.)

---

## 4. Known backend gaps (currently return `null` even when requested)

These includes are advertised but cannot yet resolve — tracked as TODOs in the backend. Do **not**
rely on them in the dashboard until fixed:

- **Sales order** `?include=related.shipments` — `SalesOrderInfo` does not expose linked shipment ids yet.
- **Receiving order** `?include=supplier` — supplier is a cross-account (seller) account; loader is caller-scoped. (Use `purchase_order.supplier` once that path is wired, or display nothing.)
- **Shipment** `?include=shipping_address` — address belongs to the customer account; loader is caller-scoped.

---

## 5. Endpoint checklist

For each endpoint, confirm the dashboard either (a) passes the right `?include=` for everything it
renders, and/or (b) reads `display_value`/scalar fields rather than expecting nested objects, and
(c) uses the new field names/types from §2.

### Sales (`/v1/sales/…`)
- [ ] `GET /sales/sales-orders` (list) — new field names, `freight`/`sales_rep`/`totals`/`related.*`/`lines` all expandable
- [ ] `GET /sales/sales-orders/{id}` (retrieve) — same; verify includes requested
- [ ] `POST /sales/sales-orders` (create) — response uses new shape; pass `?include=` if response is consumed
- [ ] `PATCH /sales/sales-orders/{id}` (update) — same
- [ ] `GET/POST /sales/sales-orders/{id}/lines`, `…/lines/{line_id}` — `product` (was `item`), expandable quantities/rates, removed line fields
- [ ] `…/actions/issue|unissue|open|close` — no `?include=` support; re-fetch for expanded data
- [ ] `…/actions/create-production-run`, `…/checkout`, `…/actions/bulk-delete`
- [ ] `GET /sales/customers`, `…/{id}` — `credit_limit.unit` null by default (use `display_value`)
- [ ] `…/customers/{id}/frequently-ordered-products`, `…/actions/merge`, `…/actions/bulk-delete`

### Operations (`/v1/operations/…`)
- [ ] `GET /operations/picks`, `…/{id}` — `customer`/`sales_order`/`lines`/`departments` expandable; `priority` always present
- [ ] `…/picks/{id}/actions/pick|pack|void`, `…/picks/{id}/shipments`, `…/picks/{pick_id}/lines/{id}…`
- [ ] `GET /operations/receiving-orders`, `…/{id}` — `purchase_order` is now `PurchaseOrderDetail`; `supplier` expandable (currently a backend gap, see §4)
- [ ] `…/receiving-orders/{id}/actions/receive|stock|void`, `…/lines/{id}…`
- [ ] `GET /operations/shipments`, `…/{id}` — many expandable refs; `sales_order`/`order_line` retyped
- [ ] `…/shipments/{id}/actions/ship|void`, `…/shipments/{shipment_id}/lines…`, `…/estimate-rate`, `…/rate-shop`
- [ ] `GET /operations/shipping-cases/{id}` — `freight_amount`/`freight_weight` units null by default
- [ ] `GET /operations/shipping-terms`, `…/{id}` — `flat_rate`/`minimum_order_value` units null by default
- [ ] `GET /operations/deliveries`, `…/{id}` — nested sales order/line refs retyped + expandable
- [ ] `GET /operations/inventories`, `GET /catalog/items/{id}/inventory` — `on_hand`/`reserved`/`available_to_promise`/`short` units null by default
- [ ] `GET /operations/quantities/{id}` — `unit` expandable
- [ ] `GET /operations/production-steps`, `…/{id}`, `…/production-flows/by-item/{item_id}`, `…/connect-steps` — nested SO refs retyped + expandable

### Catalog (`/v1/catalog/…`)
- [ ] `GET /catalog/items`, `…/{id}` — verify any embedded quantities/units
- [ ] `GET /catalog/materials`, `…/{id}` — `order_point`/`lead_time` units null by default

### Finance (`/v1/finance/…`)
- [ ] `GET /finance/invoices`, `…/{id}`, `/accounts/{account_id}/invoices` — `order`/`order_line`/`shipment`/`payment_term`/etc. expandable + retyped
- [ ] `GET /finance/transactions`, `…/{id}`, `/accounts/{account_id}/transactions` — nested refs expandable + retyped

---

## 6. Migration cheat-sheet for the dashboard

1. **Add `?include=`** for every nested object the UI renders. Nest with dots:
   `?include=customer,lines,lines.product,freight,totals,related.pick`.
2. **Read `display_value`** for units instead of `quantity.unit.abbreviation`.
3. **Rename** `customer_po → customer_purchase_order_number`, `item → product` (on order lines).
4. **Status/priority/acknowledgment** are now enum **strings**, not nested objects
   (`status`, `priority`, `acknowledgment_status`, `payment_status`).
5. **Carrier/freight billing** now live under the `freight` object (expandable).
6. **`production_run`/`pick`/shipments** on a sales order now live under `related` (expandable `Record`s).
7. **Removed line fields**: `quantity_picked/packed/invoiced`, `edi_line_item_id`, `completed_at` —
   source equivalents from `totals`, pick, shipment, or invoice resources instead.
8. **Status actions** don't expand — re-`GET` the order with includes if you need the full object.
