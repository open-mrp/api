-- +goose NO TRANSACTION
-- +goose Up

-- A supplier's items list newest first within the supplier, and a supplier's delete probes for its items.
ALTER TABLE `supplier_material`
  ADD KEY `supplier_material_owner_supplier_created_idx` (`owner_account_id`, `supplier_account_id`, `created_at` DESC, `id` DESC);

-- Supplier item search is a substring match on the supplier's code and description; nothing reads these.
ALTER TABLE `supplier_material` DROP INDEX `supplier_material_supplier_part_number_supplier_description_idx`;
ALTER TABLE `supplier_material` DROP INDEX `supplier_material_supplier_description_idx`;
ALTER TABLE `supplier_material` DROP INDEX `supplier_material_supplier_part_number_idx`;

-- +goose Down

ALTER TABLE `supplier_material` ADD FULLTEXT KEY `supplier_material_supplier_part_number_idx` (`supplier_part_number`);
ALTER TABLE `supplier_material` ADD FULLTEXT KEY `supplier_material_supplier_description_idx` (`supplier_description`);
ALTER TABLE `supplier_material` ADD FULLTEXT KEY `supplier_material_supplier_part_number_supplier_description_idx` (`supplier_part_number`, `supplier_description`);
ALTER TABLE `supplier_material` DROP INDEX `supplier_material_owner_supplier_created_idx`;
