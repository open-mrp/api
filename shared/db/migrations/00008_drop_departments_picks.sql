-- +goose NO TRANSACTION
-- +goose Up

-- Picks are never assigned to a department: nothing reads this join table, and the pick API no longer exposes a department relation or a department_ids filter.
DROP TABLE `_departments_picks`;

-- +goose Down

CREATE TABLE `_departments_picks` (
  `A` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `B` varchar(191) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  UNIQUE KEY `_departments_picks_AB_unique` (`A`,`B`),
  KEY `_departments_picks_B_index` (`B`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
