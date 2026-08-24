-- +goose NO TRANSACTION
-- +goose Up

-- Bookkeeping for shared/db/data-migrations. goose would create this itself, but creating a table is
-- DDL and prod has safe migrations enabled, so on prod it can only arrive through a deploy request.
-- goose still inserts its own version-0 row on first use; that is DML, which prod does allow.
CREATE TABLE `goose_db_version_data` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `version_id` bigint NOT NULL,
  `is_applied` tinyint(1) NOT NULL,
  `tstamp` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down

DROP TABLE IF EXISTS `goose_db_version_data`;
