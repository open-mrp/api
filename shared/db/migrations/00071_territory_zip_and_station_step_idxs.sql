-- +goose NO TRANSACTION
-- +goose Up

-- An order's sales rep by ship-to zipcode, without reading every territory the account has: the
-- lookup ranges over start_zipcode and checks end_zipcode in the index. The single-column account
-- key is the exact left prefix of the composite, which serves every lookup it did, so it goes.
ALTER TABLE `territory`
  ADD KEY `territory_account_zip_idx` (`account_id`, `start_zipcode`, `end_zipcode`, `sales_rep_id`),
  DROP KEY `territory_account_id_idx`;

-- A scanning station's steps in name order, read straight off the index instead of sorted per call.
ALTER TABLE `production_step`
  ADD KEY `production_step_account_station_name_idx` (`account_id`, `scanning_station_id`, `name`);

-- +goose Down

ALTER TABLE `production_step`
  DROP KEY `production_step_account_station_name_idx`;

ALTER TABLE `territory`
  ADD KEY `territory_account_id_idx` (`account_id`),
  DROP KEY `territory_account_zip_idx`;
