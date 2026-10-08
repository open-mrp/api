---
name: performant-lists
description: >-
  Keyset list-endpoint indexing: scope-leading composites that preserve ORDER BY,
  FORCE INDEX when the optimizer filesorts, and how to add a user-facing filter
  without a table scan. Use when adding or changing a list endpoint, filter, sort,
  cursor pagination query, or an index on a list table.
---

# Performant list endpoints

Every query must stay under **100 ms** worst case. A list that filesorts or scans a tenant partition is a bug — add or fix the index before merging. Human spec: `docs/patterns/performant-list-endpoint-patterns.md`. Indexes are declared in goose schema migrations (`shared/db/migrations`).

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

Always name the index (`sales_order_owner_status_created_idx`). Other active filters are residual — cheap because `LIMIT` bounds the window. You do **not** need `2^n` indexes.

If the optimizer still picks a single-column filter index + filesort, `FORCE INDEX` the **entire** sort-free set. Do not list single-column filter indexes. `IGNORE INDEX` of the bad one is not enough; `STRAIGHT_JOIN` does not fix index choice.

Composites are **ascending** (`(scope, filter, time, id)`): InnoDB scans an ascending key both ways, a descending one only forward, so on a `DESC` key the previous page (read oldest first) sorts the whole range.

`FORCE INDEX` picks the key, not the join order. An **inner** join to a tiny lookup table (types, statuses) lets the planner drive from the lookup and probe the base table once per row of it, then sort; `LEFT JOIN` lookups the row always has.

Choose the page from the base table alone and join after (`FROM (SELECT t.id FROM t … ORDER BY … LIMIT ?) page JOIN t …`): a filter no key serves in order then reads only its matches, not its matches times every join's fan-out.

Resolve a filter on a joined table to base-table IDs first (`customer_group_ids` → customer IDs) so a base-table key serves it.

## Which filters earn an index

| Kind | Index? |
|---|---|
| Low-cardinality, heavily used (`status`) | yes |
| High-cardinality, common (`customer`, `sales_rep`) | yes |
| Date range on the sort column | no — existing `(scope, …, time)` handles it |
| Rare / admin-only | usually residual |
| `EXISTS` / child-table | index the **child**; parent composite cannot help |
| Filter on a joined table | denormalize onto the base table, or rewrite; parent composite cannot help |
| `LIKE '%term%'` | not a B-tree. Fine on small per-tenant tables (tenant-scoped scan); otherwise prefix `LIKE 'q%'` or ngram FULLTEXT. See `public-api-design` |

High-insert tables (`sales_order`, `transaction`, `request_log`, `audit_event`, `inventory_change_log`, `batch`): only composites for filters the UI actually exposes.

## Before merging

Add a plan test (`//go:build plans`, `make test-plans`, core-service `repository/`): a `listPlanSuite` (`plan_harness_test.go`) over a seeded production-shaped corpus — one tenant, prod row widths, a **dense** and a **rare/zero** value per filter — runs every filter pair on the first page and deep pages both directions, under **analyzed** and **production** statistics (`testdata/plan_stats/<table>.json`, a snapshot of prod's `mysql.innodb_index_stats`; prod's are sampled badly and are what produce prod's bad plans). It fails a request that reads far more than the best forced index, or when even the best reads far more than a page. `transaction_list_plan_test.go` is the worked example. No skips: a filter no key can serve in order (FULLTEXT, a range on a non-sort column) is held to its own match count (`floor`).

Reports and totals use `aggregatePlanSuite`: an aggregate cannot stop at a page, so each table is held to twice its `floor` — a direct `COUNT` of the rows the request's scope covers (tenant, window, the narrowest single filter), over the source that should answer it (the rollup when it can). Pin results with `checkPlanResults`. `sales_report_plan_test.go` is the worked example.

PlanetScale Insights is the production backstop.
