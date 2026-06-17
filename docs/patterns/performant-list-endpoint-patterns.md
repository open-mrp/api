# Performant List Endpoint Patterns

This document describes how to keep paginated **list endpoints** fast as we add
user-facing filters. It is the generalization of the `sales_order` list fix
(see `services/core-service/internal/infrastructure/queries/sales_order.sql`)
and supersedes the triage notes in `docs/WIP-list-filter-index-audit.md`.

The schema (MySQL on PlanetScale) is owned by Prisma in
`dashboard/packages/db/prisma/schema/schema.prisma`. **All indexes described
here are declared there**, even for tables only read by Go (`sqlc`) services.

---

## The shape of every list query

Our list endpoints are tenant-scoped keyset (cursor) lists. They all share three
fixed pieces:

```sql
WHERE  <scope_col> = ?                      -- mandatory tenant anchor: account_id / owner_account_id
  AND  <optional filters...>                -- status, customer, sales_rep, date range, ...
ORDER BY <time_col> DESC, <id_col> DESC     -- fixed sort (keyset order)
LIMIT  ?                                     -- small page (10–50)
```

The scope equality and the `ORDER BY … LIMIT` never change. The only thing that
varies request-to-request is *which* optional filters are active.

---

## The failure mode: the selective-filter cliff

When an optional filter is present and there is **no index that both pins the
filter and preserves the `ORDER BY`**, the optimizer has two bad options, and it
picks one of them depending on table stats:

1. **Single-column filter index + filesort.** It seeks the rows matching the
   filter (could be tens of thousands), runs *every one* of them through the
   joins, then sorts the whole set to honor `ORDER BY` before applying `LIMIT`.
   This is what made `sales_order` filtered by `status=fulfilled` take **4.6 s
   at `LIMIT 10`** — it read all 121k matching rows and filesorted them.

2. **Scope+time index, filter as a residual.** It walks the `(scope, time)`
   index newest-first and tests the filter row-by-row. `LIMIT` can only
   short-circuit once it has collected a full *page of matches* — so a filter
   value with few or zero matching rows scans the **entire** tenant partition
   before returning. This is the "searching by a value with no rows hangs
   forever" stall.

**Why it looks intermittent:** filtering by a common/dense value is fast (the
page fills near the top); filtering by a rare or zero-match value stalls. Same
query, opposite outcomes.

The tell in `EXPLAIN ANALYZE`: `Using filesort`, or `rows`/`loops` far larger
than the page size, or the chosen `key` being a single-column index when a
composite exists.

---

## The mental model: one filesort-free driving path, the rest are residual

You do **not** need an index per filter combination — that is `2^n` indexes and
a write-throughput disaster. You need exactly this guarantee:

> For every request, there exists at least one index that (a) leads with the
> scope column, (b) can pin the **most selective active filter** as an equality,
> and (c) ends in `(time_col DESC, id_col DESC)` so the sort is free.

Given that, MySQL walks that one index in order and stops at `LIMIT`. Any *other*
active filters become **residual** — applied row-by-row on the small window it's
already reading. Residual filtering is cheap **because `LIMIT` bounds the
window**. The cliff only happens when *no* sort-preserving index exists and it
has to filesort or full-scan.

So the job is not "make every combo optimal." It is: **make every combo bounded,
and the common combos optimal.** Pick a budget — e.g. *no filesort, rows examined
< page × 50* — and engineer to it.

---

## The index recipe

For each filter worth driving on, declare a composite:

```
(scope_col, filter_col, time_col DESC, id_col DESC)
```

Every such index ends in the sort key, so **none of them can ever filesort**.
Together they form a menu of sort-free driving paths; the optimizer (or a
`FORCE INDEX`, below) picks the cheapest valid one per request and residual-filters
the rest.

Plus one **baseline** index for the no-filter path:

```
(scope_col, time_col DESC, id_col DESC)
```

### Prisma declaration

MySQL 8 supports descending key parts; declare them so the index matches the
`ORDER BY` and avoids a filesort.

```prisma
@@index([ownerAccountID, statusCode, createdAt(sort: Desc), id(sort: Desc)], map: "sales_order_owner_status_created_idx")
@@index([ownerAccountID, createdAt(sort: Desc), id(sort: Desc)], map: "sales_order_owner_created_idx")
```

Naming convention: `<table>_<scope>_<filter>_<time>_idx` (e.g.
`sales_order_owner_status_created_idx`). Always set an explicit `map:` name so
the migration and any `FORCE INDEX` hint can reference it.

---

## When the optimizer won't cooperate: `FORCE INDEX`

Having the right index is necessary but **not always sufficient**. With a wide
join set, the optimizer's cost model can still pick the single-column filter
index + filesort over your composite (this is exactly what happened to
`sales_order`). When that happens, restrict its choice to the sort-free menu:

```sql
FROM sales_order so
  FORCE INDEX (sales_order_owner_created_idx, sales_order_owner_status_created_idx)
```

List **every** sort-free driving index for that query. Because they all end in
the sort key, the optimizer can never filesort — it just picks the best driving
path for the active filter (the composite when a filter is present, the baseline
when not). Do **not** list single-column filter indexes here — excluding them is
the entire point.

Notes:
- `IGNORE INDEX` of the bad index alone is **not** reliable — the optimizer just
  falls to the *next* non-sort-preserving index and still filesorts. Confirmed on
  `sales_order`. Prefer `FORCE INDEX` of the good set.
- `STRAIGHT_JOIN` pins join *order*, not index *choice*. It does not prevent this
  failure mode on its own.

---

## Which filters earn an index

Score each filter by **selectivity × frequency of use**, and check storage/write
cost on high-insert tables. Not every filter deserves a composite.

| Filter kind | Index it? | How |
|---|---|---|
| Low-cardinality but heavily used (e.g. `status`) | **Yes** | residual-scanning a common status is huge; composite `(scope, status, time, id)` |
| High-cardinality + common (e.g. `customer`, `sales_rep`) | **Yes** | very selective; composite per filter |
| Date range on the sort column | **No new index** | a range on `time_col` is handled natively by any `(scope, …, time)` index |
| Rare / admin-only filters | **Usually no** | leave residual — bounded by `LIMIT` anyway |
| `EXISTS` / child-table filters (e.g. `item_id`, `product_line_id` via order lines) | **Different** | index the **child** table's join+filter columns; the outer query keeps driving on `(scope, time)`. A composite on the parent can't help. |
| Filters on a **joined** table (e.g. paginate `invoice`, filter by `sales_order.buyer_account_id`) | **Composite won't help** | needs denormalization of the filter column onto the base table, or a filter-first subquery rewrite |
| `LIKE '%term%'` free-text (`name`, `subject`) | **Not a B-tree** | leading-wildcard is non-sargable; use `FULLTEXT … MATCH/AGAINST`, an exact-match seek, or a prefix anchor — not a composite |

High-insert tables (`sales_order`, `transaction`, `request_log`, `audit_event`,
`inventory_change_log`, `batch`) pay the index tax on every write. On those,
**only add composites for filters the UI actually exposes and users actually
use** — do not speculatively add the full matrix.

---

## Verifying *before* production

Turn "hope it's fast" into a gate. For each filter and each common combination:

1. **`EXPLAIN ANALYZE` on a production-sized tenant**, with two values:
   - a **dense** value (many matches) — confirms the common path
   - a **zero/rare** value — this is where the cliff hides
2. **Assert the plan is healthy:**
   - no `Using filesort`, no `Using temporary`, no `type: ALL`
   - chosen `key` is the intended composite
   - `rows` / `loops` are within the budget (≈ page size, not the whole tenant)
3. **Keep a large, realistically-skewed seed tenant** (one account with 100k+
   rows and a lopsided filter distribution) so these checks are repeatable. The
   real `ac_01gf…` account (121k sales orders, 99.95% `fulfilled`) is the canonical
   stress fixture.
4. **Make it a CI gate** — a query-plan regression test that fails the moment a
   new filter is added without its sort-free index. PlanetScale branches make
   this cheap: branch the prod schema, run the suite, tear down.

PlanetScale **schema recommendations** and **Insights** (sort by total time and
rows-read) are the production backstop for anything that slips through.

---

## Checklist for a new (or newly-filtered) list endpoint

- [ ] Scope column leads; `ORDER BY (time DESC, id DESC)` is fixed and matches a baseline index.
- [ ] Every UI-exposed, selective filter has a `(scope, filter, time DESC, id DESC)` composite — or is a documented residual.
- [ ] `EXISTS`/joined/`LIKE` filters handled by their own mechanism (child index / rewrite / FULLTEXT), not a parent composite.
- [ ] `EXPLAIN ANALYZE` verified on a big tenant for dense **and** zero-match values — no filesort, bounded rows.
- [ ] If the optimizer mis-picks, `FORCE INDEX` lists exactly the sort-free driving set.
- [ ] Write cost considered on high-insert tables; only used filters indexed.
- [ ] Index declared in `dashboard/packages/db/prisma/schema/schema.prisma` with an explicit `map:` name.
```