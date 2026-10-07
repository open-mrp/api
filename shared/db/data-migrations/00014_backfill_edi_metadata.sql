-- +goose NO TRANSACTION
-- +goose Up

-- Copy each order line's partner line number (the 850's PO1-01) into metadata.edi_line_item_id, so
-- integrations read it from the public API. Merges into whatever metadata the line already has.
-- Idempotent: it touches only lines whose metadata still differs from the column, so it can be re-run
-- to catch lines an older image wrote to the column alone. updated_at is left alone: this is a copy,
-- not a change to the line.
--
-- Batched, each batch its own transaction, so no single statement outruns Vitess's transaction
-- limit. Each fills up to 5,000 lines; 10 cover 50,000, against 32,731 in prod (2026-10). Past that,
-- later ones find nothing but still scan the table, so don't pad the count. "" is skipped: in metadata
-- it means "no key".

UPDATE `sales_order_line`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_line_item_id', `edi_line_item_id`)
WHERE `edi_line_item_id` IS NOT NULL AND `edi_line_item_id` <> ''
  AND (`metadata` ->> '$.edi_line_item_id' IS NULL OR `metadata` ->> '$.edi_line_item_id' <> `edi_line_item_id`)
ORDER BY `id`
LIMIT 5000;

UPDATE `sales_order_line`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_line_item_id', `edi_line_item_id`)
WHERE `edi_line_item_id` IS NOT NULL AND `edi_line_item_id` <> ''
  AND (`metadata` ->> '$.edi_line_item_id' IS NULL OR `metadata` ->> '$.edi_line_item_id' <> `edi_line_item_id`)
ORDER BY `id`
LIMIT 5000;

UPDATE `sales_order_line`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_line_item_id', `edi_line_item_id`)
WHERE `edi_line_item_id` IS NOT NULL AND `edi_line_item_id` <> ''
  AND (`metadata` ->> '$.edi_line_item_id' IS NULL OR `metadata` ->> '$.edi_line_item_id' <> `edi_line_item_id`)
ORDER BY `id`
LIMIT 5000;

UPDATE `sales_order_line`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_line_item_id', `edi_line_item_id`)
WHERE `edi_line_item_id` IS NOT NULL AND `edi_line_item_id` <> ''
  AND (`metadata` ->> '$.edi_line_item_id' IS NULL OR `metadata` ->> '$.edi_line_item_id' <> `edi_line_item_id`)
ORDER BY `id`
LIMIT 5000;

UPDATE `sales_order_line`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_line_item_id', `edi_line_item_id`)
WHERE `edi_line_item_id` IS NOT NULL AND `edi_line_item_id` <> ''
  AND (`metadata` ->> '$.edi_line_item_id' IS NULL OR `metadata` ->> '$.edi_line_item_id' <> `edi_line_item_id`)
ORDER BY `id`
LIMIT 5000;

UPDATE `sales_order_line`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_line_item_id', `edi_line_item_id`)
WHERE `edi_line_item_id` IS NOT NULL AND `edi_line_item_id` <> ''
  AND (`metadata` ->> '$.edi_line_item_id' IS NULL OR `metadata` ->> '$.edi_line_item_id' <> `edi_line_item_id`)
ORDER BY `id`
LIMIT 5000;

UPDATE `sales_order_line`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_line_item_id', `edi_line_item_id`)
WHERE `edi_line_item_id` IS NOT NULL AND `edi_line_item_id` <> ''
  AND (`metadata` ->> '$.edi_line_item_id' IS NULL OR `metadata` ->> '$.edi_line_item_id' <> `edi_line_item_id`)
ORDER BY `id`
LIMIT 5000;

UPDATE `sales_order_line`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_line_item_id', `edi_line_item_id`)
WHERE `edi_line_item_id` IS NOT NULL AND `edi_line_item_id` <> ''
  AND (`metadata` ->> '$.edi_line_item_id' IS NULL OR `metadata` ->> '$.edi_line_item_id' <> `edi_line_item_id`)
ORDER BY `id`
LIMIT 5000;

UPDATE `sales_order_line`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_line_item_id', `edi_line_item_id`)
WHERE `edi_line_item_id` IS NOT NULL AND `edi_line_item_id` <> ''
  AND (`metadata` ->> '$.edi_line_item_id' IS NULL OR `metadata` ->> '$.edi_line_item_id' <> `edi_line_item_id`)
ORDER BY `id`
LIMIT 5000;

UPDATE `sales_order_line`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_line_item_id', `edi_line_item_id`)
WHERE `edi_line_item_id` IS NOT NULL AND `edi_line_item_id` <> ''
  AND (`metadata` ->> '$.edi_line_item_id' IS NULL OR `metadata` ->> '$.edi_line_item_id' <> `edi_line_item_id`)
ORDER BY `id`
LIMIT 5000;


-- Record each invoice the dashboard marked sent over EDI as metadata.edi_sent = "true", so integrations
-- read it from metadata and is_edi_sent can be dropped. The send time was never stored, so none is
-- invented: an invoice sent from now on records metadata.edi_sent_at instead. Idempotent and
-- batched the same way; 26 batches cover 130,000 invoices, against 113,530 flagged of 125,718 in prod
-- (2026-10).

UPDATE `invoice`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_sent', 'true')
WHERE `is_edi_sent` = 1
  AND (`metadata` ->> '$.edi_sent' IS NULL OR `metadata` ->> '$.edi_sent' <> 'true')
ORDER BY `id`
LIMIT 5000;

UPDATE `invoice`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_sent', 'true')
WHERE `is_edi_sent` = 1
  AND (`metadata` ->> '$.edi_sent' IS NULL OR `metadata` ->> '$.edi_sent' <> 'true')
ORDER BY `id`
LIMIT 5000;

UPDATE `invoice`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_sent', 'true')
WHERE `is_edi_sent` = 1
  AND (`metadata` ->> '$.edi_sent' IS NULL OR `metadata` ->> '$.edi_sent' <> 'true')
ORDER BY `id`
LIMIT 5000;

UPDATE `invoice`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_sent', 'true')
WHERE `is_edi_sent` = 1
  AND (`metadata` ->> '$.edi_sent' IS NULL OR `metadata` ->> '$.edi_sent' <> 'true')
ORDER BY `id`
LIMIT 5000;

UPDATE `invoice`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_sent', 'true')
WHERE `is_edi_sent` = 1
  AND (`metadata` ->> '$.edi_sent' IS NULL OR `metadata` ->> '$.edi_sent' <> 'true')
ORDER BY `id`
LIMIT 5000;

UPDATE `invoice`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_sent', 'true')
WHERE `is_edi_sent` = 1
  AND (`metadata` ->> '$.edi_sent' IS NULL OR `metadata` ->> '$.edi_sent' <> 'true')
ORDER BY `id`
LIMIT 5000;

UPDATE `invoice`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_sent', 'true')
WHERE `is_edi_sent` = 1
  AND (`metadata` ->> '$.edi_sent' IS NULL OR `metadata` ->> '$.edi_sent' <> 'true')
ORDER BY `id`
LIMIT 5000;

UPDATE `invoice`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_sent', 'true')
WHERE `is_edi_sent` = 1
  AND (`metadata` ->> '$.edi_sent' IS NULL OR `metadata` ->> '$.edi_sent' <> 'true')
ORDER BY `id`
LIMIT 5000;

UPDATE `invoice`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_sent', 'true')
WHERE `is_edi_sent` = 1
  AND (`metadata` ->> '$.edi_sent' IS NULL OR `metadata` ->> '$.edi_sent' <> 'true')
ORDER BY `id`
LIMIT 5000;

UPDATE `invoice`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_sent', 'true')
WHERE `is_edi_sent` = 1
  AND (`metadata` ->> '$.edi_sent' IS NULL OR `metadata` ->> '$.edi_sent' <> 'true')
ORDER BY `id`
LIMIT 5000;

UPDATE `invoice`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_sent', 'true')
WHERE `is_edi_sent` = 1
  AND (`metadata` ->> '$.edi_sent' IS NULL OR `metadata` ->> '$.edi_sent' <> 'true')
ORDER BY `id`
LIMIT 5000;

UPDATE `invoice`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_sent', 'true')
WHERE `is_edi_sent` = 1
  AND (`metadata` ->> '$.edi_sent' IS NULL OR `metadata` ->> '$.edi_sent' <> 'true')
ORDER BY `id`
LIMIT 5000;

UPDATE `invoice`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_sent', 'true')
WHERE `is_edi_sent` = 1
  AND (`metadata` ->> '$.edi_sent' IS NULL OR `metadata` ->> '$.edi_sent' <> 'true')
ORDER BY `id`
LIMIT 5000;

UPDATE `invoice`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_sent', 'true')
WHERE `is_edi_sent` = 1
  AND (`metadata` ->> '$.edi_sent' IS NULL OR `metadata` ->> '$.edi_sent' <> 'true')
ORDER BY `id`
LIMIT 5000;

UPDATE `invoice`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_sent', 'true')
WHERE `is_edi_sent` = 1
  AND (`metadata` ->> '$.edi_sent' IS NULL OR `metadata` ->> '$.edi_sent' <> 'true')
ORDER BY `id`
LIMIT 5000;

UPDATE `invoice`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_sent', 'true')
WHERE `is_edi_sent` = 1
  AND (`metadata` ->> '$.edi_sent' IS NULL OR `metadata` ->> '$.edi_sent' <> 'true')
ORDER BY `id`
LIMIT 5000;

UPDATE `invoice`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_sent', 'true')
WHERE `is_edi_sent` = 1
  AND (`metadata` ->> '$.edi_sent' IS NULL OR `metadata` ->> '$.edi_sent' <> 'true')
ORDER BY `id`
LIMIT 5000;

UPDATE `invoice`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_sent', 'true')
WHERE `is_edi_sent` = 1
  AND (`metadata` ->> '$.edi_sent' IS NULL OR `metadata` ->> '$.edi_sent' <> 'true')
ORDER BY `id`
LIMIT 5000;

UPDATE `invoice`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_sent', 'true')
WHERE `is_edi_sent` = 1
  AND (`metadata` ->> '$.edi_sent' IS NULL OR `metadata` ->> '$.edi_sent' <> 'true')
ORDER BY `id`
LIMIT 5000;

UPDATE `invoice`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_sent', 'true')
WHERE `is_edi_sent` = 1
  AND (`metadata` ->> '$.edi_sent' IS NULL OR `metadata` ->> '$.edi_sent' <> 'true')
ORDER BY `id`
LIMIT 5000;

UPDATE `invoice`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_sent', 'true')
WHERE `is_edi_sent` = 1
  AND (`metadata` ->> '$.edi_sent' IS NULL OR `metadata` ->> '$.edi_sent' <> 'true')
ORDER BY `id`
LIMIT 5000;

UPDATE `invoice`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_sent', 'true')
WHERE `is_edi_sent` = 1
  AND (`metadata` ->> '$.edi_sent' IS NULL OR `metadata` ->> '$.edi_sent' <> 'true')
ORDER BY `id`
LIMIT 5000;

UPDATE `invoice`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_sent', 'true')
WHERE `is_edi_sent` = 1
  AND (`metadata` ->> '$.edi_sent' IS NULL OR `metadata` ->> '$.edi_sent' <> 'true')
ORDER BY `id`
LIMIT 5000;

UPDATE `invoice`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_sent', 'true')
WHERE `is_edi_sent` = 1
  AND (`metadata` ->> '$.edi_sent' IS NULL OR `metadata` ->> '$.edi_sent' <> 'true')
ORDER BY `id`
LIMIT 5000;

UPDATE `invoice`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_sent', 'true')
WHERE `is_edi_sent` = 1
  AND (`metadata` ->> '$.edi_sent' IS NULL OR `metadata` ->> '$.edi_sent' <> 'true')
ORDER BY `id`
LIMIT 5000;

UPDATE `invoice`
SET `metadata` = JSON_SET(COALESCE(`metadata`, JSON_OBJECT()), '$.edi_sent', 'true')
WHERE `is_edi_sent` = 1
  AND (`metadata` ->> '$.edi_sent' IS NULL OR `metadata` ->> '$.edi_sent' <> 'true')
ORDER BY `id`
LIMIT 5000;

-- +goose Down

-- Not reversible: the column is dropped by its schema migration's Down.
SELECT 1;
