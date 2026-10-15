-- +goose Up

INSERT IGNORE INTO `storage_location_type` (`id`, `code`, `name`, `created_at`, `updated_at`)
VALUES ('sltp_01seedsite00000000', 'site', 'Site', NOW(3), NOW(3));

-- Each account's default site is its top-level location: an existing site first, else the oldest.
UPDATE `account` a
JOIN (
  SELECT ranked.account_id, ranked.id
  FROM (
    SELECT sl.account_id, sl.id,
      ROW_NUMBER() OVER (PARTITION BY sl.account_id
        ORDER BY sl.storage_location_type_code = 'site' DESC, sl.created_at, sl.id) AS rn
    FROM `storage_location` sl
    WHERE sl.parent_id IS NULL
  ) ranked
  WHERE ranked.rn = 1
) pick ON pick.account_id = a.id
SET a.default_site_id = pick.id
WHERE a.default_site_id IS NULL;

-- That location becomes a site, at the account's shipping address when it has none of its own.
UPDATE `storage_location` sl
JOIN `account` a ON a.default_site_id = sl.id
SET sl.storage_location_type_code = 'site',
    sl.address_id = COALESCE(sl.address_id, a.default_shipping_address_id),
    sl.updated_at = NOW(3)
WHERE sl.storage_location_type_code <> 'site';

-- +goose Down

UPDATE `storage_location` SET `storage_location_type_code` = 'building' WHERE `storage_location_type_code` = 'site';
UPDATE `account` SET `default_site_id` = NULL WHERE `default_site_id` IS NOT NULL;
DELETE FROM `storage_location_type` WHERE `code` = 'site';
