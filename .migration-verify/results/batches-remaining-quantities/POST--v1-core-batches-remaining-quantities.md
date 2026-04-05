# POST /v1/core/batches/remaining-quantities

**Status:** Issues found and fixed

## What was compared

- **Validation rules:** Request body fields (`batch_ids`, `production_step_id`), empty array check
- **Permission checks:** Internal actor, batches:read permission, target account required
- **Business logic:** Single-part vs multi-part branching, batch flow traversal, quantity calculation, output batch summing
- **Error handling:** Empty batch IDs, missing required parts, quantity mismatch across blocks
- **Response shape:** Quantity resource with measure and unit

## Issues found and fixed

### 1. Missing empty batchIDs validation
- **Dashboard:** Throws bad request `"No batches to split."` when `batchIDs.length === 0`
- **Go (before):** No check — empty array fell through to multi-part path
- **Fix:** Added `len(batchIDs) == 0` check returning a validation error

### 2. Multi-part path incorrectly summed seconds and waste
- **Dashboard:** Multi-part path only sums `b.quantity` for output batches (does NOT include `seconds` or `waste`)
- **Go (before):** Multi-part path included `ob.Seconds.Measure` and `ob.Waste.Measure` in the total used
- **Fix:** Removed seconds/waste from the multi-part output batch sum. Note: the single-part path correctly includes seconds and waste, matching the Dashboard.

### 3. Missing consumption validation in multi-part path
- **Dashboard:** Fetches the production step, then verifies every consumption has corresponding batches. Throws `"Missing required part: {sku}"` if a consumption is unmatched.
- **Go (before):** Skipped this validation entirely
- **Fix:** Added `stepRepo.Find()` call to load the production step, then iterates `productionStep.Consumptions` to verify each has a matching batch group. Returns a validation error with the SKU if missing.

## Remaining concerns

- None — the Go implementation now matches the Dashboard behavior for both single-part and multi-part paths.
