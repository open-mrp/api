# PUT /v1/core/analytics/production-costs

## Status: Issues Found — Partial Fix Applied, Major Gaps Remain

## What Was Compared

| Aspect | Dashboard | Go | Match? |
|--------|-----------|-----|--------|
| **Route** | `PUT /v1/analytics/production-costs` | `PUT /v1/core/analytics/production-costs` | Yes (expected prefix change) |
| **Request fields** | Optional: startDate, endDate, itemIDs, productLineIDs, departmentIDs, categoryIDs | Same fields (snake_case) | Yes |
| **Actor type** | Internal actor only | Internal actor only | Yes |
| **Permission domain** | `batches` / `read` | ~~`invoices` / `read`~~ → Fixed to `batches` / `read` | **Fixed** |
| **DB query logic** | Complex multi-step: resolve productLines→parts, filter batches by scannedAt/department/item/category, fetch production steps separately | Simple query grouping by item with hardcoded 0 for costs, no filters applied | **No** |
| **Cost calculation** | Sophisticated app-level calculation: labor (with leveling factor + allowances), overhead, material consumption costs, waste/seconds tracking | None — returns hardcoded 0 for total_cost and cost_per_unit | **No** |
| **Response shape** | Array of `{department, category, totalCosts, productiveCosts, wasteCosts, secondsCosts}` where each cost has `{total, labor, materials, overhead, time, quantity}` as BaseQuantity | Protobuf/presenter layer has correct shape, but domain model returns flat item-level data (`ItemID, ProductSku, TotalQuantity, TotalCost, CostPerUnit, Unit`) — mismatch between domain and transport layers | **No** |
| **Aggregation** | Groups by `department.id + category.id` composite key | Groups by item (no department/category aggregation) | **No** |
| **Idempotency** | N/A (PUT, read-only analytics) | N/A | Yes |
| **Side effects** | None | None | Yes |

## Issues Found and Fixed

### 1. Wrong Permission Domain (Fixed)
- **File:** `services/core-service/internal/service/analytics_service.go:101`
- **Was:** `PermissionDomainInvoices` with `ActionRead`
- **Fixed to:** `PermissionDomainBatches` with `ActionRead`
- **Reason:** Dashboard checks `PermissionDomains.batches` / `read` in `BatchSvc.fetchProductionCosts`

## Major Remaining Issues (Not Fixed — Require Significant Reimplementation)

### 2. Domain Model Mismatch
The Go `ProductionCostEntry` struct has flat item-level fields (`ItemID`, `ProductSku`, `ProductDescription`, `ProductLine`, `TotalQuantity`, `TotalCost`, `CostPerUnit`, `Unit`). The Dashboard returns aggregated cost breakdowns grouped by department+category with detailed cost components (labor, materials, overhead, time, quantity) for each of totalCosts, productiveCosts, wasteCosts, and secondsCosts.

The protobuf definitions and presenter layer already have the correct target shape (`ProductionCostEntryProto` with `CostBreakdown` fields), but the domain model and repository return completely different data. The gRPC handler currently only populates `TotalCosts.Total.Measure` from `e.TotalCost` and ignores everything else.

### 3. SQL Query Missing All Filters
The current SQL query (`GetProductionCostEntries`) only filters by `account_id` and `closed_at IS NOT NULL`. It does not apply any of the optional filters:
- `startDate` / `endDate` filtering on `scanned_at`
- `departmentIDs` filtering via scanning station
- `itemIDs` filtering
- `categoryIDs` filtering
- `productLineIDs` → part resolution (requires querying the production flow to find parts)

### 4. No Cost Calculation Logic
The Dashboard performs complex application-level cost calculations:
1. Resolves product lines to parts via `ProductionFlowRepo.findPartsByProduct()`
2. Fetches batches with scanning station + department info
3. Fetches production steps separately using a lightweight adapter
4. Calculates costs per step using: labor time × (leveling factor × allowances corrections) × rate, overhead similarly, plus material consumption costs
5. Tracks productive, waste, and seconds costs separately
6. Aggregates by department+category composite key

This logic cannot be replicated in a single SQL query — it requires multiple queries and significant Go application code (production flow traversal, step cost calculation with normalization factors, etc.).

### 5. Repository Does Not Pass Filter Params
`GetProductionCostEntries` in the repository only passes `params.AccountID` to the SQL query. The `StartDate`, `EndDate`, `ItemIDs`, `ProductLineIDs`, `DepartmentIDs`, and `CategoryIDs` params are all ignored.

## Recommendation

This endpoint needs a full reimplementation of the repository and service layers to match the Dashboard behavior. The transport layers (endpoint definition, protobuf, presenter) are already correctly shaped. The work needed:

1. Rewrite the SQL query to fetch batches with scanning station/department/item/category joins and apply all filters
2. Add a separate query or repository method to resolve product line IDs to part IDs via production flows
3. Add a separate query to fetch lightweight production step data
4. Implement the cost calculation algorithm in Go (labor time correction, overhead, material consumption costs)
5. Implement the aggregation by department+category composite key
6. Update the domain model from flat `ProductionCostEntry` to match the aggregated cost breakdown structure
7. Update the gRPC handler to properly convert the new domain model to protobuf
