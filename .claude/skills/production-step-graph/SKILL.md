---
name: production-step-graph
description: >-
  `_parent_child_production_steps` column meaning: A is downstream, B is upstream.
  Use when touching production step graphs, ConnectSteps, in_steps/out_steps, flow SQL,
  sandbox seeds, or subassembly_filter=initial_only.
---

# Production step graph

`_parent_child_production_steps` matches Prisma/dashboard — **not** “A = parent.” Human spec: `docs/patterns/production-step-graph-patterns.md`.

| Column | Meaning |
|---|---|
| **`A`** | **Downstream** (later; child consuming upstream output) |
| **`B`** | **Upstream** (earlier; parent feeding `A`) |

Edge direction: **upstream → downstream** = **`B` → `A`**. Row `(A, B)`: `B` is in `A`'s `in` relation; `A` is in `B`'s `out` relation.

## Writes

`ConnectSteps(source, target)` = feed upstream `source` into downstream `target`:

```sql
INSERT INTO _parent_child_production_steps (A, B)
VALUES (target_id, source_id);  -- (downstream, upstream)
```

## Reads

- Upstream parents of `S`: rows where `A = S`; parent IDs are `B`
- Downstream children of `S`: rows where `B = S`; child IDs are `A`
- Roots / initial subassemblies (no upstream): `NOT EXISTS (… WHERE pcps.A = step_id)`

Reversing A/B breaks dashboard flows, seeds (`shared/db/seed/0009_production.sql`), `in_steps`/`out_steps`, and `subassembly_filter=initial_only`.

SQL: `production_flow.sql`, `production_step_query.sql`, `item.sql` (`only_initial_subassemblies`).
