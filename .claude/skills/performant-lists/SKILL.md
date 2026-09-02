---
name: performant-lists
description: >-
  Keyset list-endpoint indexing: scope-leading composites that preserve ORDER BY,
  FORCE INDEX when the optimizer filesorts, and how to add a user-facing filter
  without a table scan. Use when adding or changing a list endpoint, filter, sort,
  cursor pagination query, or a Prisma @@index on a list table.
---

# Performant list endpoints

Every query must stay under **100 ms** worst case. A list that filesorts or scans a tenant partition is a bug — add or fix the index before merging. Human spec: `docs/patterns/performant-list-endpoint-patterns.md`. Indexes are declared in `dashboard/packages/db/prisma/schema/schema.prisma` even when only Go/sqlc reads the table.

## The query shape

```sql
WHERE  <scope_col> = ?                 -- account_id / owner_account_id
  AND  <optional filters...>
ORDER BY <time_col> DESC, <id_col> DESC
LIMIT  ?
```

Scope equality and `ORDER BY … LIMIT` never change. Only optional filters vary.

## The failure

No index that both pins the active filter **and** preserves `ORDER BY` → either filesort of every match, or a `(scope, time)` walk that cannot short-circuit on a rare/zero-match filter. Dense values look fast; rare values stall. `EXPLAIN`: `Using filesort`, or `rows`/`loops` ≫ page size.

## The recipe

Guarantee: for every request, some index (a) leads with scope, (b) can pin the most selective active **equality** filter, (c) ends in `(time_col DESC, id_col DESC)`.

```
(scope_col, filter_col, time_col DESC, id_col DESC)   -- per driving filter
(scope_col, time_col DESC, id_col DESC)               -- baseline, no filter
```

Prisma: always set `map:` (`sales_order_owner_status_created_idx`). Other active filters are residual — cheap because `LIMIT` bounds the window. You do **not** need `2^n` indexes.

If the optimizer still picks a single-column filter index + filesort, `FORCE INDEX` the **entire** sort-free set. Do not list single-column filter indexes. `IGNORE INDEX` of the bad one is not enough; `STRAIGHT_JOIN` does not fix index choice.

## Which filters earn an index

| Kind | Index? |
|---|---|
| Low-cardinality, heavily used (`status`) | yes |
| High-cardinality, common (`customer`, `sales_rep`) | yes |
| Date range on the sort column | no — existing `(scope, …, time)` handles it |
| Rare / admin-only | usually residual |
| `EXISTS` / child-table | index the **child**; parent composite cannot help |
| Filter on a joined table | denormalize onto the base table, or rewrite; parent composite cannot help |
| `LIKE '%term%'` | not a B-tree — FULLTEXT, exact, or prefix |

High-insert tables (`sales_order`, `transaction`, `request_log`, `audit_event`, `inventory_change_log`, `batch`): only composites for filters the UI actually exposes.

## Before merging

`EXPLAIN ANALYZE` on a production-scale tenant for a **dense** value and a **zero/rare** value. Healthy: no filesort/temporary/`type: ALL`, chosen key is the composite, rows ≈ page size. PlanetScale Insights is the production backstop.
