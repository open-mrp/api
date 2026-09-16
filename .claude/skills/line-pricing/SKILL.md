---
name: line-pricing
description: >-
  How to turn a line's quantity and unit price into money exactly as the dashboard
  does: shared/pricing (UnitConversion, ExtendedPrice, LineTotal), the
  pricing_*_ratio_* columns on line queries, where totals round, and the SQL expression
  for money totals. Use when computing an extended price, line total,
  order/invoice/receivable/receiving total, a Stripe charge, or any quantity × rate, in
  Go or SQL.
---

# Line pricing

A line keeps its quantity in its own unit while its price is a rate per the rate's **denominator** unit, and the two can differ. Three cartons of twelve pairs at $22.50/pair is **$810.00**, not the $67.50 that `quantity.value * rate.value` gives. Never multiply a quantity by a rate directly. Price lines exactly as the dashboard (`dashboard/packages/objects`) does.

## Go: `shared/pricing`

`pricing.ExtendedPrice` is a port of the dashboard's `QuantityUtils.multiplyRate`, which both the legacy API and the frontend use for totals. It makes the same calls in the same order:

1. Normalize the quantity: times its unit's ratio numerator, then divided by its ratio denominator.
2. Divide the price by its denominator unit's factor.
3. Multiply the two.

Every step rounds to 40 significant digits, half away from zero (decimal.js as the dashboard configures it). `testdata/dashboard_cases.json` holds values the dashboard's own code computed, and `TestMatchesTheDashboard` requires an exact match. Regenerate the fixture with `testdata/gen_dashboard_cases.ts` if the dashboard's math changes.

```go
conv, err := line.PriceUnitConversion()          // SalesOrderLine, PurchaseOrderLine, InvoiceLine
amount := pricing.ExtendedPrice(qty, price, conv) // unrounded (receivables balances)
cents  := pricing.LineTotal(qty, price, conv)     // rounded per line: every order, PO and invoice total
```

Round where the dashboard rounds:

| Total | Dashboard | Go / SQL |
|---|---|---|
| Order total, Stripe charge, min-order threshold, discount base, acknowledgement, gateway `totals`, HubSpot amount | `calculateTotalOrdered`: each line `toDP(2)`, then summed | `LineTotal` per line |
| Invoice lines and total | `calculateTotalInvoiced`: each line `toDP(2)` | `LineTotal`; SQL `SUM(ROUND(…, 2))` |
| PO lines and total | `PurchaseOrderLineUtils.calculateTotalOrdered`: each line `toDP(2)` | `LineTotal` |
| Receivables balance | unrounded line sum minus allocations, rounded once | SQL `SUM(…)` unrounded, `ROUND(…, 2)` outside |

- Don't reduce the ratios to one factor, and don't multiply before dividing. At 40 digits the order of operations decides which cent a half-cent tie rounds to.
- The same conversion prices a line's ordered, picked, packed and invoiced quantities.
- A line the dashboard would price differently is refused, never priced unconverted:
  - units of different dimensions (the dashboard throws);
  - a unit with an offset;
  - a price in a non-base currency unit;
  - a base unit whose ratio isn't one.

  No such unit exists today. The line queries return empty ratio terms for these lines, `PriceUnitConversion()` returns an error, and the gRPC line sets `pricing_unavailable`. Fail the document or charge, or return a null total.
- Lines loaded through the repos carry the four `Pricing*Ratio*` terms. Over gRPC they travel on `SalesOrderLineInfo.pricing_*`. A gateway talking to an older core that sends no ratios prices lines as before.
- For a line that hasn't been saved yet (a create request), use `unitPairConversions` / `resolvedLineConversions` (core-service `service/line_pricing.go`). They read the units through `UnitConversionRepo.GetUnitFactors` and apply the same eligibility rules.
- Document builders take a `map[lineID]pricing.UnitConversion` from `salesOrderLineConversions` / `purchaseOrderLineConversions` / `invoiceLineConversions`, and price each line with `conversionFor(convs, line.ID)`.

## SQL

A line query that returns a unit price also returns the four ratio terms. `qu` is the quantity's unit, `up_du` the rate's denominator unit and `up_nu` its numerator unit. Copy the `CASE` block from `GetSalesOrderLines` (`sales_order.sql`) verbatim. It returns `'1'` for a same-unit line, the stored ratios for an eligible mixed-unit line, and `NULL` otherwise.

A money total prices each line in the dashboard's order: normalized quantity times (price divided by the price unit's factor), with a direct multiply when the units match. `ru` is the rate's denominator unit.

```sql
CASE WHEN q.unit_id = r.denominator_unit_id THEN q.value * r.value
     ELSE (q.value * qu.ratio_numerator / qu.ratio_denominator) * (r.value / (ru.ratio_numerator / ru.ratio_denominator)) END
```

- Wrap the expression in `ROUND(…, 2)` inside the `SUM` for invoice money (see the table above).
- MySQL divides at 30 decimal places, not 40 significant digits, so an exact half-cent tie can round the other way. That needs a conversion whose division doesn't terminate (a ratio such as `lb`'s 453592/1000, `min`'s 1/60 or `wk`'s 168) and an amount that lands exactly on a half cent. Money that is stated or charged (documents, Stripe, gateway totals) is computed in Go and matches exactly. Keep it there.
- Existing examples: `invoice.sql` (`total_invoiced`), `receivable.sql`, `receiving_order.sql` (`GetReceivingOrderTotals`).

## Checklist

- [ ] No `qty.Mul(price)` or `q.value * r.value` on a line whose units can differ.
- [ ] New line queries select the four `pricing_*_ratio_*` columns, and the repo maps them onto the domain line.
- [ ] Totals round where the dashboard rounds (table above).
- [ ] Tests include a mixed-unit line (e.g. cartons priced per pair) and assert the converted amount.
