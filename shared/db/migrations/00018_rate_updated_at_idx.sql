-- +goose NO TRANSACTION
-- +goose Up

-- The burn-rate sweeper (core-service) periodically selects the stalest burn-rate rate rows —
-- WHERE updated_at < ? ORDER BY updated_at ASC LIMIT n — to enqueue a bounded recompute batch.
-- Without an index on updated_at that walk is a full scan of `rate` plus a filesort on every
-- tick; with it the scan is an ordered range that stops at the batch limit. The secondary index
-- carries the PK (id) in InnoDB, so the rate side of the join is served index-only.
ALTER TABLE `rate`
  ADD KEY `rate_updated_at_idx` (`updated_at`);

-- +goose Down

ALTER TABLE `rate`
  DROP KEY `rate_updated_at_idx`;
