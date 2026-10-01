-- +goose Up

-- Purchase orders created by the dashboard's legacy API recorded their creator in change_log rather
-- than as a `create` audit event, so their `created_by` include resolved to system. Copy each such
-- creator into a create event. Idempotent: the type_id is derived from the change_log row and the
-- insert skips orders that already have a create event, so it can be re-run to catch orders the
-- legacy API created after an earlier run.
INSERT IGNORE INTO `audit_event` (
  `type_id`, `actor_id`, `actor_type`, `identity_type`, `account_id`, `target_account_id`,
  `action`, `resource_type`, `resource_id`, `service_name`, `occurred_at`,
  `root_resource_id`, `root_resource_type`
)
SELECT
  CONCAT('auev_', SUBSTRING(MD5(cl.id), 1, 19)),
  cl.responsible_user_id, 'internal', 'user', cl.account_id, cl.account_id,
  'create', 'purchase_order', cl.record_id, 'dashboard-api', cl.created_at,
  cl.record_id, 'purchase_order'
FROM `change_log` cl
JOIN `sales_order` so ON so.id = cl.record_id
  AND so.owner_account_id = cl.account_id
  AND so.sales_order_type_code = 'purchase_order'
WHERE cl.model_type = 'purchaseOrder'
  AND cl.action_type_code = 'create_record'
  AND cl.responsible_user_id IS NOT NULL
  AND NOT EXISTS (
    SELECT 1 FROM `audit_event` ae
    WHERE ae.resource_type = 'purchase_order'
      AND ae.resource_id = cl.record_id
      AND ae.action = 'create'
  );

-- +goose Down

DELETE FROM `audit_event`
WHERE `resource_type` = 'purchase_order'
  AND `action` = 'create'
  AND `service_name` = 'dashboard-api';
