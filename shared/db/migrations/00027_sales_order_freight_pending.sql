-- +goose NO TRANSACTION
-- +goose Up
-- Set when an order was created without its freight line because no carrier quote was cached; the
-- freight consumer adds the line and clears it. Checkout finishes a still-pending order's freight
-- before charging, so the Stripe amount never omits shipping.
ALTER TABLE `sales_order`
  ADD COLUMN `freight_pending_since` datetime(3) NULL;

-- +goose Down
ALTER TABLE `sales_order`
  DROP COLUMN `freight_pending_since`;
