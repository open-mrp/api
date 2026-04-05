# GET /v1/core/items/{id}/costs

## Result: PARITY CONFIRMED

No issues found. The Go implementation correctly replicates the Dashboard's business logic.

## What was compared

### Permission checks
- Both require internal actor + items:read permission + target account set. **Match.**

### Algorithm (cost calculation)
- **Production flow discovery:** Both find the production step that produces the target item, then BFS backward through the step edge graph to find all contributing steps. **Match.**
- **Per-step cost calculation:** Both compute:
  - `correctiveFactor = levelingFactor * allowances + levelingFactor + allowances + 1`
  - `correctedLaborTime = laborTime * correctiveFactor`
  - `totalLaborTime = prodQty * correctedLaborTime`
  - `laborCost = totalLaborTime * laborRate`
  - `overheadCost = totalLaborTime * overheadRate`
  - `materialCost = sum((consumptionQty + wasteQty) * unitCost)` excluding parts and products
  - **Match.**
- **Normalization factors:** Both compute `1/targetProdQty` for the target step, then propagate backward via `childNorm * consumedQty / parentProdQty`. The Dashboard uses a two-phase init+backward-pass approach while Go computes directly during BFS, but the results are equivalent. **Match.**
- **Forward pass aggregation:** Both sum `stepCost * normalizationFactor` across all steps. **Match.**

### Side effects
- Both update the item's unit cost rate in the DB (rate value + denominator unit). **Match.**
- Both clear the item's `is_dirty` flag. **Match.**

### Error handling
- Both return 404 when no production flow is found for the item. **Match.**
- Both handle zero production quantity. **Match.**

### Response shape
- Dashboard returns `CostSummary` with 4 Rate objects (unitCost, labor, materials, overhead), each containing measure + numeratorUnit (currency) + denominatorUnit.
- Go returns `ItemCosts` with flat decimal strings (total_cost, direct_labor_cost, direct_material_cost, overhead_cost) + a Unit subresource.
- This is an intentional redesign following Go API resource conventions. The same cost data is present; the currency unit is omitted as it's implicit. **Acceptable divergence.**

### DB queries
- `GetCostFlowStepConsumptions`: Fetches consumption item type, consumption quantity, waste quantity, and unit cost per production step. Matches Dashboard's Prisma query semantics. **Match.**
- `UpdateItemUnitCostRate`: Updates rate value and denominator unit for the item's unit_cost rate. Matches Dashboard's `itemRepo.updateCost`. **Match.**
- `ClearItemDirtyFlag`: Clears `is_dirty` on the item. Matches Dashboard's `isDirty: false` update. **Match.**

## No issues found
