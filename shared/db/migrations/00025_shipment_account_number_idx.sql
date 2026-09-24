-- +goose NO TRANSACTION
-- +goose Up
-- Shipment number lookups (duplicate checks when numbering a shipment) filter on (account_id, number), but
-- the only usable index is the account prefix, so each lookup reads the account's whole shipment history.
ALTER TABLE `shipment`
  ADD KEY `shipment_account_id_number_idx` (`account_id`, `number`);

-- +goose Down
ALTER TABLE `shipment`
  DROP KEY `shipment_account_id_number_idx`;
