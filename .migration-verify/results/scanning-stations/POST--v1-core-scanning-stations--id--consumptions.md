# Migration Verification: POST /v1/core/scanning-stations/{id}/consumptions

## Status: Issues found and fixed

## What was compared

- **Validation rules**: Required fields, format constraints
- **Permission checks**: Internal actor, batches:read permission, target account ID
- **Routing logic**: Type-based branching (initBatch, moveBatch, splitBatch, mergeBatch) and multi-part/multi-batch routing
- **DB queries and logic**: Furthest-right-batch lookup, next-step-quantity calculation, inventory fetch, consumption demand calculation
- **Error handling**: Not-found, validation, closed-batch errors
- **Side effects**: None (read-only endpoint) — no idempotency keys needed despite POST method
- **Response shape**: ScanningConsumption fields (sku, demand_measure, demand_unit, inventory_measure, inventory_unit, instructions). Go adds `object` field per API conventions and wraps in `List` — both intentional.

## Issues found and fixed

### 1. Move batch: missing closedAt validation
- **Dashboard**: Checks `furthestBatch.closedAt` and rejects closed batches with "Batch is closed."
- **Go (before fix)**: No closedAt check after finding furthest right batch
- **Fix**: Added `if furthestBatch.ClosedAt != nil { return nil, apierror.NewValidationError("Batch is closed.") }` in `getMoveBatchConsumption`

### 2. Split batch: missing isMultiPart/multi-batch routing
- **Dashboard**: For non-initBatch types, checks `isMultiPart || mergeBatch || batchIDs.length > 1` and routes to `getManyBatches` path. This applies to split batch too.
- **Go (before fix)**: `getSplitBatchConsumption` always did direct split calculation regardless of multi-part or multi-batch scenarios.
- **Fix**: Added `isMultiPart` and `len(params.BatchIDs) > 1` check in `getSplitBatchConsumption` to route to `getManyBatchConsumption` when applicable.

### 3. Split batch: missing furthest-right-batch lookup and closedAt check
- **Dashboard**: For single split batch, finds furthest right batch in flow and validates it's not closed before calculating.
- **Go (before fix)**: No furthest-right-batch lookup or closedAt validation in split path.
- **Fix**: Added `FindFurthestRightBatchInFlow` call and closedAt check in the single-batch split path.

### 4. getManyBatchConsumption: only used first item block's output quantity
- **Dashboard**: Calls `calculateNextStepQuantities` for each batch individually and sums all output quantities.
- **Go (before fix)**: Grouped batches by item block, called `CalculateNextStepQuantities` per group, but only used the first group's output quantity (ignoring subsequent groups).
- **Fix**: Changed to iterate over each source batch individually, calling `CalculateNextStepQuantities` per batch and summing all output quantities, matching Dashboard behavior.

### 5. getManyBatchConsumption: missing duplicate and count validation
- **Dashboard**: Checks for duplicate batches (`uniqueBatches.size !== batches.length`) and validates all requested batches were found (`batches.length !== batchIDs.length`).
- **Go (before fix)**: No duplicate or count validation.
- **Fix**: Added duplicate check using a seen map, and count validation comparing `len(sourceBatches)` to `len(params.BatchIDs)`.

## Remaining concerns

- **Unit conversion**: The Dashboard uses `QuantityUtils.updateUnit()` to convert quantities to the production step's unit before calculating the execution multiplier in single-batch cases (init, move, split). The Go implementation divides measures directly. For the move case, the output of `CalculateNextStepQuantities` should already be in the correct unit. For init and split cases, if the batch/split quantity unit differs from the production step's production quantity unit, the Go calculation may differ. This depends on whether unit mismatches actually occur in practice. If they do, a unit conversion step would be needed in the Go service layer.
