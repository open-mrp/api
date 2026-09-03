-- +goose NO TRANSACTION
-- +goose Up
-- Denormalized sort key: COALESCE(sales_order.ship_by_date, '9999-12-31') for the pick's order. The
-- default pick list sorts by ship-by date, which lives on sales_order, so the list had to filesort
-- every one of an account's picks (10-16s on the largest account). Carrying the value on the pick,
-- indexed with account_id, lets ListPicksShipBy{Forward,Backward} read a page straight from the
-- index in order. Sentinel 9999-12-31 = "no commitment"; it sorts last, as the old COALESCE did.
-- Kept in sync by CreatePick and SetPickShipByDateForOrder; existing rows filled by data-migration.
ALTER TABLE `pick`
  ADD COLUMN `ship_by_sort_date` date NOT NULL DEFAULT '9999-12-31',
  ADD KEY `pick_account_ship_by_idx` (`account_id`, `ship_by_sort_date`, `id`);

-- +goose Down
ALTER TABLE `pick`
  DROP KEY `pick_account_ship_by_idx`,
  DROP COLUMN `ship_by_sort_date`;
