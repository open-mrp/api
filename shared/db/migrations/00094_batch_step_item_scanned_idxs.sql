-- +goose NO TRANSACTION
-- +goose Up

-- The schedule reads a window of production history twice: the constraint's batches by the steps of
-- its department, and the finishing stages' by the finished goods. With only the step and item keys,
-- each read visited every batch those steps or items ever had and dropped the ones outside the
-- window (79k batch lookups for 12k rows on one plant). Scan time on these keys makes the window a
-- range of each step or item. The item key also serves an item's newest scans in order, so its run-rate
-- history reads only the samples it returns.
--
-- The single-column keys stay: other reads name them in FORCE INDEX and index merges.
ALTER TABLE `batch`
  ADD KEY `batch_step_scanned_idx` (`production_step_id`, `scanned_at`),
  ADD KEY `batch_item_scanned_idx` (`item_id`, `scanned_at`);

-- +goose Down

ALTER TABLE `batch`
  DROP KEY `batch_item_scanned_idx`,
  DROP KEY `batch_step_scanned_idx`;
