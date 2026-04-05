# Migration Verification: POST /v1/core/batches/actions/merge

## Status: Issues found and fixed

## What was compared

- **Validation rules**: Required fields, empty batch list, duplicate batch IDs, multi-part consumptions
- **Permission checks**: Internal actor, batches:create permission, target account ID
- **Business logic**: Single-part vs multi-part branching, quantity calculation, item grouping
- **DB operations**: Batch creation, batch flow connections, source batch closing, close-if-last-step
- **Side effects**: executeProductionStep (fire-and-forget in Dashboard → outbox event in Go)
- **Error handling**: Error types and messages
- **Response shape**: Returns created batch with 201 status
- **Idempotency**: Go correctly uses idempotency keys (improvement over Dashboard)

## Issues found and fixed

### 1. Missing duplicate batch ID validation

**Dashboard**: Checks for duplicate batch IDs in the result set:
```typescript
const uniqueBatches = new Set(batches.map(b => b.id));
if (uniqueBatches.size !== batches.length) {
    throw HttpError.badRequest('Duplicate batches provided.');
}
```

**Go (before fix)**: No duplicate check. `FindAvailableBatchesInFlow` iterates over each ID individually, so passing the same ID twice would fetch the same batch twice.

**Fix**: Added duplicate batch ID check in `MergeBatches` before dispatching to single/multi-part helpers. Uses a `map[string]struct{}` to detect duplicates early.

### 2. Missing multi-part consumptions validation

**Dashboard**: Fetches the production step with its consumptions and verifies every required consumed item has corresponding batches:
```typescript
for (const consumption of productionStep.consumptions) {
    const blockBatches = batchesByBlock.get(consumption.consumedItem.itemID);
    if (!blockBatches || blockBatches.batches.length === 0) {
        throw HttpError.badRequest(`Missing required part: ${consumption.consumedItem.sku}`);
    }
}
```

**Go (before fix)**: Skipped this validation entirely — just grouped batches by item and calculated output quantities without verifying all required parts were present.

**Fix**: Added `stepRepo.Find()` call to fetch the production step detail with consumptions, then iterates over consumptions to verify each consumed item has a corresponding batch group. Returns a validation error with the missing part's SKU if not found.

## Confirmed parity (no issues)

- **Request shape**: Both accept `batch_ids`, `scanning_station_id`, `production_step_id`
- **Permission checks**: Both check internal actor + batches:create
- **Scanning station validation**: Both verify station is in account
- **Empty batch check**: Both validate at least one batch ID
- **Single-part same-item check**: Both verify all batches have the same item
- **Multi-part output quantity consistency**: Both verify calculated output quantities match across blocks
- **Transaction atomicity**: Both create batch + connect + close in a transaction
- **Production step execution**: Dashboard fires post-response; Go enqueues via outbox (reliable improvement)
- **Response**: Both return 201 with the created batch

## Remaining concerns

None. The Go implementation now matches Dashboard business logic with the added benefits of idempotency keys and transactional event enqueuing.
