-- +goose NO TRANSACTION
-- +goose Up

-- Who last set an invoice's paid-in-full flag by hand. Recalculating an invoice's payments decides the
-- flag from what has been applied to it and wins over a value set by hand; when it overturns one, the
-- person who set it is told. NULL when recalculation set the flag last, or when an API key or agent set it
-- and there is no person to tell.
ALTER TABLE `invoice`
  ADD COLUMN `paid_in_full_marked_by_id` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL;

-- +goose Down

ALTER TABLE `invoice`
  DROP COLUMN `paid_in_full_marked_by_id`;
