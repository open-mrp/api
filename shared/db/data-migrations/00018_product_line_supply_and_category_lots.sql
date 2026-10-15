-- +goose Up

-- A line's fulfillment policy becomes its default supply: built to stock is stock, built to order is make.
UPDATE `product_line`
SET `supply_code` = CASE `fulfillment_policy_code` WHEN 'make_to_stock' THEN 'stock' WHEN 'make_to_order' THEN 'make' END
WHERE `fulfillment_policy_code` IS NOT NULL AND `supply_code` IS NULL;

-- Lots move from product lines to item categories. Each account category takes the line lot most of its items were
-- made in: an item's own line, or for an intermediate item, the line of what it is produced into. The category points
-- at the line's quantity row; category edits write a new row, so sharing it is safe. System categories are shared by
-- every account and take no lot.
UPDATE `item_category` ic
JOIN (
  SELECT ranked.item_category_id, ranked.default_lot_id
  FROM (
    SELECT votes.item_category_id, votes.default_lot_id,
      ROW_NUMBER() OVER (PARTITION BY votes.item_category_id ORDER BY SUM(votes.items) DESC, votes.default_lot_id) AS rn
    FROM (
      SELECT i.item_category_id, pl.default_lot_id, COUNT(*) AS items
      FROM `product_line` pl
      JOIN `product` p ON p.product_line_id = pl.id
      JOIN `item` i ON i.id = p.item_id
      WHERE pl.default_lot_id IS NOT NULL AND i.deleted_at IS NULL
      GROUP BY i.item_category_id, pl.default_lot_id
      UNION ALL
      SELECT gi.item_category_id, pl.default_lot_id, COUNT(DISTINCT gi.id) AS items
      FROM `product_line` pl
      JOIN `product` p ON p.product_line_id = pl.id
      JOIN `production` pr ON pr.item_id = p.item_id
      JOIN `consumption` c ON c.production_step_id = pr.production_step_id
      JOIN `item` gi ON gi.id = c.item_id
      LEFT JOIN `product` gp ON gp.item_id = gi.id
      WHERE pl.default_lot_id IS NOT NULL AND gi.deleted_at IS NULL AND gp.product_line_id IS NULL
      GROUP BY gi.item_category_id, pl.default_lot_id
    ) votes
    GROUP BY votes.item_category_id, votes.default_lot_id
  ) ranked
  WHERE ranked.rn = 1
) pick ON pick.item_category_id = ic.id
SET ic.default_lot_size_id = pick.default_lot_id
WHERE ic.account_id IS NOT NULL AND ic.default_lot_size_id IS NULL;

-- +goose Down

UPDATE `item_category` SET `default_lot_size_id` = NULL WHERE `default_lot_size_id` IS NOT NULL;
UPDATE `product_line` SET `supply_code` = NULL WHERE `supply_code` IS NOT NULL;
