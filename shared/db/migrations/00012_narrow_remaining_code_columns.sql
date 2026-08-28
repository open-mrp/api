-- +goose NO TRANSACTION
-- +goose Up

-- The rest of the varchar(255) code columns deferred from 00011, each at the width its values actually need rather than a flat 32. Kept as its own migration because request_log is 1M rows and roughly 17GB: PlanetScale copies the table to apply this, so it is the long pole here and should not share a deploy with the nine tables in 00011.
-- Widths are set from the longest value the code can produce, not just what is in the table today. permission.code and role_permission.permission_code hold PermissionDomains values and already reach 31 characters (production_step_transformations), so varchar(32) would leave a single character of headroom. request_log.error_code holds the API error constants, the longest of which is agent_spending_cap_reached at 26, and new codes get added routinely. All four take varchar(64). change_log.model_type holds PascalCase model names, currently up to 17 (TransactionRecord) but a name like ProductionScheduleFinishingLine is 31, so it takes varchar(64) as well; it is a discriminator rather than an enum code, which is why it was held back from 00011.
-- request_log.identity_type and actor_type are the exception: they hold short identity discriminators (user, agent, group, api_key — 7 characters at most) and take varchar(32).
-- None of these columns is covered by a prefix index, so unlike 00011 there is no prefix to restore on the way back down. Nullability is restated on every line because MODIFY COLUMN replaces the whole definition.

ALTER TABLE `request_log`
  MODIFY `error_code` varchar(64) NULL,
  MODIFY `identity_type` varchar(32) NULL,
  MODIFY `actor_type` varchar(32) NULL;

ALTER TABLE `permission`
  MODIFY `code` varchar(64) NOT NULL,
  MODIFY `permission_group_code` varchar(64) NOT NULL;

ALTER TABLE `permission_group`
  MODIFY `code` varchar(64) NOT NULL;

ALTER TABLE `role_permission`
  MODIFY `permission_code` varchar(64) NOT NULL;

ALTER TABLE `change_log`
  MODIFY `model_type` varchar(64) NOT NULL;

-- +goose Down

ALTER TABLE `request_log`
  MODIFY `error_code` varchar(255) NULL,
  MODIFY `identity_type` varchar(255) NULL,
  MODIFY `actor_type` varchar(255) NULL;

ALTER TABLE `permission`
  MODIFY `code` varchar(255) NOT NULL,
  MODIFY `permission_group_code` varchar(255) NOT NULL;

ALTER TABLE `permission_group`
  MODIFY `code` varchar(255) NOT NULL;

ALTER TABLE `role_permission`
  MODIFY `permission_code` varchar(255) NOT NULL;

ALTER TABLE `change_log`
  MODIFY `model_type` varchar(255) NOT NULL;
