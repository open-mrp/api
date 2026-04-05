# Verification: PUT /v1/core/receiving-orders/{id}/actions/void

## Result: Issue found and fixed

## What was compared

- **Permission checks**: Both check internal actor + `receivingOrders` domain with `update` action + target account ID required. ✅ Match
- **Transaction operations**: Both perform the same 3 steps in the same order:
  1. Void all lines (zero quantities, clear stocked_at)
  2. Delete duplicate receiving order lines
  3. Mark receiving order as incomplete (set completed_at to NULL)
- **Delete duplicate logic**: Both group by sales_order_line_id (Dashboard: `orderLineID`), keep the first/earliest, delete the rest. ✅ Match
- **Mark incomplete**: Both set `completed_at = NULL`. ✅ Match
- **Response shape**: Both return the full receiving order with all lines after the transaction. ✅ Match
- **Idempotency**: PUT endpoint — no idempotency keys needed (idempotent by design). ✅ Match
- **Side effects**: No async messages, no inventory mutations, no emails in either implementation. ✅ Match

## Issue found and fixed

**VoidAllReceivingOrderLines SQL query missing `completed_at IS NULL` filter**

The Dashboard `voidAllLines` method filters receiving order lines by `completedAt: null` on the receiving order, meaning it only resets line quantities and stocking status for **incomplete** receiving orders. The Go SQL query was missing this filter, meaning it would void lines even on already-completed receiving orders.

**Fix**: Added `AND ro.completed_at IS NULL` to the `VoidAllReceivingOrderLines` SQL query in `receiving_order.sql`, then regenerated sqlc.

This preserves the Dashboard behavior where voiding a completed receiving order will mark it incomplete but will NOT reset the line quantities/stocking (since `voidAllLines` runs before `markIncomplete` in the transaction).

## No remaining concerns
