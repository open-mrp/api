-- +goose NO TRANSACTION
-- +goose Up

-- Copied sales_order_line.edi_line_item_id into metadata.edi_line_item_id and invoice.is_edi_sent into
-- metadata.edi_sent = "true". It ran in production before schema migration 00070 dropped both columns;
-- every schema migration runs before any backfill, so on a new database the columns are already gone,
-- and there is nothing to copy.
SELECT 1;

-- +goose Down

-- Not reversible.
SELECT 1;
