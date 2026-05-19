# Production step graph (`_parent_child_production_steps`)

The implicit join table `_parent_child_production_steps` links production steps in a directed process graph. **Column meaning matches Prisma / dashboard**, not an alternate “A = parent” convention.

## Invariants (do not revert)

| Column | Meaning |
|--------|---------|
| **`A`** | **Downstream** step (later in the flow; “child” step consuming upstream output). |
| **`B`** | **Upstream** step (earlier in the flow; “parent” step feeding into `A`). |

Direction of the edge is **upstream → downstream**, i.e. **`B` → `A`**.

Prisma documents this as: row `(A, B)` means `B` is in `A`’s `in` relation and `A` is in `B`’s `out` relation (see `services/core-service/internal/infrastructure/queries/sandbox_seed.sql`, SECTION 26).

## Writes (`ConnectSteps`)

`ConnectSteps(source_step_id, target_step_id)` means **feed from upstream `source` into downstream `target`**.  
Persist as:

```sql
INSERT INTO _parent_child_production_steps (A, B)
VALUES (target_id, source_id);  -- (downstream, upstream)
```

Implementations: [`production_flow.sql`](../../services/core-service/internal/infrastructure/queries/production_flow.sql) (`ConnectSteps`, `ConnectStepsIdempotent`, `FlowDisconnectSteps`).

## Reads (common shapes)

- **Upstream parents of step `S`**: rows where **`A = S`**; parent IDs are **`B`** (`GetProductionStepInputSteps` in [`production_step_query.sql`](../../services/core-service/internal/infrastructure/queries/production_step_query.sql)).
- **Downstream children of step `S`**: rows where **`B = S`**; child IDs are **`A`** (`GetProductionStepOutputSteps`, `GetProductionStepChildSteps`).
- **Root / first-step steps** (no upstream): step never appears as **`A`** on an edge (nothing downstream of them as the `A` endpoint of that row—but roots are **upstream-only**, so they only appear as **`B`** when they feed a downstream step). Equivalently for catalog filtering: **`NOT EXISTS (SELECT 1 … WHERE pcps.A = step_id)`** means “this step is never the downstream end of a link,” i.e. it has **no upstream parent** in the graph— suitable for **initial subassembly** filtering in [`item.sql`](../../services/core-service/internal/infrastructure/queries/item.sql).

## Failure mode if reversed

Treating **`A` as parent and `B` as child** breaks alignment with:

- Dashboard Prisma schema and any flows written from the UI.
- Seed files that already encode **`A` = downstream** (e.g. [`shared/db/seed/0009_production.sql`](../../shared/db/seed/0009_production.sql)).

Symptoms include wrong `in_steps` / `out_steps`, broken flow traversal, and incorrect **`subassembly_filter=initial_only`** on list items.

## Related code

- Flow mutations / edges: `services/core-service/internal/infrastructure/queries/production_flow.sql`
- Step queries (inputs, outputs, “last” step, filters): `services/core-service/internal/infrastructure/queries/production_step_query.sql`
- Initial-subassembly item list: `services/core-service/internal/infrastructure/queries/item.sql` (`only_initial_subassemblies`)
