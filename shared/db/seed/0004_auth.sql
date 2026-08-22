-- 0004_auth.sql
-- Seeds permissions, roles, users, and account-user associations.

-- bcrypt hash of 'Testing123!' (cost 10)
SET @password_hash = '$2a$10$w68CrxLdi9fdVttqNZMAZesPa2dJlsUrGNy39boKJXz81hyvX0m6y';

-- Permissions (using PermissionDomains values as both id and code)
-- Group assignments mirror domainToGroup in dashboard/packages/scripts/src/updatePermissions.ts, the script that syncs deployed databases
INSERT IGNORE INTO permission (id, code, name, permission_group_code, created_at, updated_at) VALUES
    ('self', 'self', 'Self', 'self', NOW(), NOW()),
    ('messaging', 'messaging', 'Messaging', 'self', NOW(), NOW()),
    ('alerts', 'alerts', 'Alerts', 'self', NOW(), NOW()),
    ('deliveries', 'deliveries', 'Deliveries', 'shipping', NOW(), NOW()),
    ('locations', 'locations', 'Locations', 'inventory', NOW(), NOW()),
    ('settlements', 'settlements', 'Settlements', 'payments', NOW(), NOW()),
    ('transactions', 'transactions', 'Transactions', 'payments', NOW(), NOW()),
    ('batches', 'batches', 'Batches', 'production', NOW(), NOW()),
    ('carriers', 'carriers', 'Carriers', 'shipping', NOW(), NOW()),
    ('customer_groups', 'customer_groups', 'Customer Groups', 'customers', NOW(), NOW()),
    ('customers', 'customers', 'Customers', 'customers', NOW(), NOW()),
    ('contacts', 'contacts', 'Contacts', 'customers', NOW(), NOW()),
    ('department_picks', 'department_picks', 'Department Picks', 'picking', NOW(), NOW()),
    ('departments', 'departments', 'Departments', 'departments', NOW(), NOW()),
    ('discounts', 'discounts', 'Discounts', 'pricing', NOW(), NOW()),
    ('edi_locations', 'edi_locations', 'Edi Locations', 'edi', NOW(), NOW()),
    ('edi_runs', 'edi_runs', 'Edi Runs', 'edi', NOW(), NOW()),
    ('email_logs', 'email_logs', 'Email Logs', 'logs', NOW(), NOW()),
    ('error_logs', 'error_logs', 'Error Logs', 'logs', NOW(), NOW()),
    ('products', 'products', 'Products', 'products', NOW(), NOW()),
    ('inventory', 'inventory', 'Inventory', 'inventory', NOW(), NOW()),
    ('inventory_change_logs', 'inventory_change_logs', 'Inventory Change Logs', 'logs', NOW(), NOW()),
    ('inventory_logs', 'inventory_logs', 'Inventory Logs', 'logs', NOW(), NOW()),
    ('invoices', 'invoices', 'Invoices', 'invoices', NOW(), NOW()),
    ('item_categories', 'item_categories', 'Item Categories', 'items', NOW(), NOW()),
    ('jobs', 'jobs', 'Jobs', 'admin', NOW(), NOW()),
    ('machines', 'machines', 'Machines', 'production', NOW(), NOW()),
    ('machine_downtime', 'machine_downtime', 'Machine Downtime', 'production', NOW(), NOW()),
    ('production_schedules', 'production_schedules', 'Production Schedules', 'production', NOW(), NOW()),
    ('demand_overrides', 'demand_overrides', 'Demand Overrides', 'production', NOW(), NOW()),
    ('materials', 'materials', 'Materials', 'items', NOW(), NOW()),
    ('accounts', 'accounts', 'Accounts', 'admin', NOW(), NOW()),
    ('payment_terms', 'payment_terms', 'Payment Terms', 'payments', NOW(), NOW()),
    ('permissions', 'permissions', 'Permissions', 'admin', NOW(), NOW()),
    ('parts', 'parts', 'Parts', 'items', NOW(), NOW()),
    ('picks', 'picks', 'Picks', 'picking', NOW(), NOW()),
    ('receiving_orders', 'receiving_orders', 'Receiving Orders', 'purchasing', NOW(), NOW()),
    ('product_groups', 'product_groups', 'Product Groups', 'products', NOW(), NOW()),
    ('items', 'items', 'Items', 'items', NOW(), NOW()),
    ('production_runs', 'production_runs', 'Production Runs', 'production', NOW(), NOW()),
    ('production_step_transformations', 'production_step_transformations', 'Production Step Transformations', 'production', NOW(), NOW()),
    ('production_steps', 'production_steps', 'Production Steps', 'production', NOW(), NOW()),
    ('product_lines', 'product_lines', 'Product Lines', 'products', NOW(), NOW()),
    ('product_variations', 'product_variations', 'Product Variations', 'products', NOW(), NOW()),
    ('properties', 'properties', 'Properties', 'items', NOW(), NOW()),
    ('purchase_orders', 'purchase_orders', 'Purchase Orders', 'purchasing', NOW(), NOW()),
    ('suppliers', 'suppliers', 'Suppliers', 'purchasing', NOW(), NOW()),
    ('receiving', 'receiving', 'Receiving', 'purchasing', NOW(), NOW()),
    ('relevant_products', 'relevant_products', 'Relevant Products', 'products', NOW(), NOW()),
    ('roles', 'roles', 'Roles', 'admin', NOW(), NOW()),
    ('sales_orders', 'sales_orders', 'Sales Orders', 'sales_orders', NOW(), NOW()),
    ('sales_rep_territories', 'sales_rep_territories', 'Sales Rep Territories', 'sales_reps', NOW(), NOW()),
    ('sales_targets', 'sales_targets', 'Sales Targets', 'sales_reps', NOW(), NOW()),
    ('scanners', 'scanners', 'Scanners', 'scanning', NOW(), NOW()),
    ('scanning_error_logs', 'scanning_error_logs', 'Scanning Error Logs', 'scanning', NOW(), NOW()),
    ('shifts', 'shifts', 'Shifts', 'production', NOW(), NOW()),
    ('shipments', 'shipments', 'Shipments', 'shipping', NOW(), NOW()),
    ('shipping_cases', 'shipping_cases', 'Shipping Cases', 'shipping', NOW(), NOW()),
    ('shipping_terms', 'shipping_terms', 'Shipping Terms', 'shipping', NOW(), NOW()),
    ('supplies', 'supplies', 'Supplies', 'items', NOW(), NOW()),
    ('system_properties', 'system_properties', 'System Properties', 'admin', NOW(), NOW()),
    ('team', 'team', 'Team', 'teams', NOW(), NOW()),
    ('units', 'units', 'Units', 'units', NOW(), NOW()),
    ('unit_groups', 'unit_groups', 'Unit Groups', 'units', NOW(), NOW()),
    ('request_logs', 'request_logs', 'Request Logs', 'logs', NOW(), NOW()),
    ('audit_events', 'audit_events', 'Audit Events', 'admin', NOW(), NOW()),
    ('sandboxes', 'sandboxes', 'Sandboxes', 'admin', NOW(), NOW()),
    ('api_keys', 'api_keys', 'Api Keys', 'admin', NOW(), NOW()),
    ('integrations', 'integrations', 'Integrations', 'admin', NOW(), NOW()),
    ('priorities', 'priorities', 'Priorities', 'admin', NOW(), NOW()),
    ('addresses', 'addresses', 'Addresses', 'customers', NOW(), NOW()),
    ('adjustment_types', 'adjustment_types', 'Adjustment Types', 'inventory', NOW(), NOW()),
    ('product_types', 'product_types', 'Product Types', 'products', NOW(), NOW()),
    ('agents', 'agents', 'Agents', 'teams', NOW(), NOW()),
    ('agent_runs', 'agent_runs', 'Agent Runs', 'teams', NOW(), NOW()),
    ('agent_memories', 'agent_memories', 'Agent Memories', 'teams', NOW(), NOW());

-- Roles
INSERT IGNORE INTO role (id, name, role_type_code, account_id, created_at, updated_at) VALUES
    ('rl_mtg88e6u6fbu', 'Admin', 'admin', NULL, NOW(), NOW()),
    ('rl_hh6mrlkv08n8', 'Sales Rep', 'sales_rep', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW()),
    ('rl_scanner', 'Scanning Station', 'scanner', 'ac_01k0a5smf9ekb8rqg12555zjqa', NOW(), NOW());

-- Admin role permissions (full CRUD on ALL permission domains)
INSERT IGNORE INTO role_permission (id, role_id, permission_code, `create`, `read`, `update`, `delete`, created_at, updated_at) VALUES
    ('rlpm_01seedadm_self000', 'rl_mtg88e6u6fbu', 'self', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_msg0000', 'rl_mtg88e6u6fbu', 'messaging', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_deliv00', 'rl_mtg88e6u6fbu', 'deliveries', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_loc0000', 'rl_mtg88e6u6fbu', 'locations', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_settl00', 'rl_mtg88e6u6fbu', 'settlements', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_trans00', 'rl_mtg88e6u6fbu', 'transactions', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_batch00', 'rl_mtg88e6u6fbu', 'batches', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_carri00', 'rl_mtg88e6u6fbu', 'carriers', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_custgp0', 'rl_mtg88e6u6fbu', 'customer_groups', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_custs00', 'rl_mtg88e6u6fbu', 'customers', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_contac0', 'rl_mtg88e6u6fbu', 'contacts', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_deptpk0', 'rl_mtg88e6u6fbu', 'department_picks', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_depts00', 'rl_mtg88e6u6fbu', 'departments', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_discou0', 'rl_mtg88e6u6fbu', 'discounts', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_ediloc0', 'rl_mtg88e6u6fbu', 'edi_locations', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_edirun0', 'rl_mtg88e6u6fbu', 'edi_runs', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_emllg00', 'rl_mtg88e6u6fbu', 'email_logs', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_errlg00', 'rl_mtg88e6u6fbu', 'error_logs', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_prods00', 'rl_mtg88e6u6fbu', 'products', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_invent0', 'rl_mtg88e6u6fbu', 'inventory', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_invchlg', 'rl_mtg88e6u6fbu', 'inventory_change_logs', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_invlg00', 'rl_mtg88e6u6fbu', 'inventory_logs', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_invoic0', 'rl_mtg88e6u6fbu', 'invoices', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_itmcat0', 'rl_mtg88e6u6fbu', 'item_categories', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_jobs000', 'rl_mtg88e6u6fbu', 'jobs', 0, 1, 0, 1, NOW(), NOW()),
    ('rlpm_01seedadm_machin0', 'rl_mtg88e6u6fbu', 'machines', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_mcdwn00', 'rl_mtg88e6u6fbu', 'machine_downtime', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_prdsch0', 'rl_mtg88e6u6fbu', 'production_schedules', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_dmovr00', 'rl_mtg88e6u6fbu', 'demand_overrides', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_materi0', 'rl_mtg88e6u6fbu', 'materials', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_accoun0', 'rl_mtg88e6u6fbu', 'accounts', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_paytm00', 'rl_mtg88e6u6fbu', 'payment_terms', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_perms00', 'rl_mtg88e6u6fbu', 'permissions', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_parts00', 'rl_mtg88e6u6fbu', 'parts', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_picks00', 'rl_mtg88e6u6fbu', 'picks', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_recvor0', 'rl_mtg88e6u6fbu', 'receiving_orders', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_prodgp0', 'rl_mtg88e6u6fbu', 'product_groups', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_items00', 'rl_mtg88e6u6fbu', 'items', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_prdrun0', 'rl_mtg88e6u6fbu', 'production_runs', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_prsttr0', 'rl_mtg88e6u6fbu', 'production_step_transformations', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_prstep0', 'rl_mtg88e6u6fbu', 'production_steps', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_pdln000', 'rl_mtg88e6u6fbu', 'product_lines', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_pdvar00', 'rl_mtg88e6u6fbu', 'product_variations', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_props00', 'rl_mtg88e6u6fbu', 'properties', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_purord0', 'rl_mtg88e6u6fbu', 'purchase_orders', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_suppl00', 'rl_mtg88e6u6fbu', 'suppliers', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_receiv0', 'rl_mtg88e6u6fbu', 'receiving', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_relprd0', 'rl_mtg88e6u6fbu', 'relevant_products', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_roles00', 'rl_mtg88e6u6fbu', 'roles', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_salord0', 'rl_mtg88e6u6fbu', 'sales_orders', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_salter0', 'rl_mtg88e6u6fbu', 'sales_rep_territories', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_saltgt0', 'rl_mtg88e6u6fbu', 'sales_targets', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_scannr0', 'rl_mtg88e6u6fbu', 'scanners', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_scerlg0', 'rl_mtg88e6u6fbu', 'scanning_error_logs', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_shifts0', 'rl_mtg88e6u6fbu', 'shifts', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_shipmn0', 'rl_mtg88e6u6fbu', 'shipments', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_shcase0', 'rl_mtg88e6u6fbu', 'shipping_cases', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_shterm0', 'rl_mtg88e6u6fbu', 'shipping_terms', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_suppli0', 'rl_mtg88e6u6fbu', 'supplies', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_syspro0', 'rl_mtg88e6u6fbu', 'system_properties', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_team000', 'rl_mtg88e6u6fbu', 'team', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_units00', 'rl_mtg88e6u6fbu', 'units', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_unitgp0', 'rl_mtg88e6u6fbu', 'unit_groups', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_reqlg00', 'rl_mtg88e6u6fbu', 'request_logs', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_audit00', 'rl_mtg88e6u6fbu', 'audit_events', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_sandbo0', 'rl_mtg88e6u6fbu', 'sandboxes', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_apiky00', 'rl_mtg88e6u6fbu', 'api_keys', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_integ00', 'rl_mtg88e6u6fbu', 'integrations', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_prior00', 'rl_mtg88e6u6fbu', 'priorities', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_addres0', 'rl_mtg88e6u6fbu', 'addresses', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_adjtyp0', 'rl_mtg88e6u6fbu', 'adjustment_types', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_prdtyp0', 'rl_mtg88e6u6fbu', 'product_types', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_agents0', 'rl_mtg88e6u6fbu', 'agents', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_agentr0', 'rl_mtg88e6u6fbu', 'agent_runs', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_agentm0', 'rl_mtg88e6u6fbu', 'agent_memories', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedadm_alerts0', 'rl_mtg88e6u6fbu', 'alerts', 1, 1, 1, 1, NOW(), NOW());

-- Sales Rep role permissions (sales_orders + receive/manage own notifications). jobs:read is the baseline every non-admin role needs to poll the 202 of any async operation; updatePermissions.ts grants it the same way in deployed databases.
INSERT IGNORE INTO role_permission (id, role_id, permission_code, `create`, `read`, `update`, `delete`, created_at, updated_at) VALUES
    ('rlpm_01seedsrep_salord', 'rl_hh6mrlkv08n8', 'sales_orders', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedsrep_msg000', 'rl_hh6mrlkv08n8', 'messaging', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedsrep_jobs00', 'rl_hh6mrlkv08n8', 'jobs', 0, 1, 0, 0, NOW(), NOW());

-- Scanner role permissions
INSERT IGNORE INTO role_permission (id, role_id, permission_code, `create`, `read`, `update`, `delete`, created_at, updated_at) VALUES
    ('rlpm_01seedscnr_batch0', 'rl_scanner', 'batches', 1, 1, 1, 1, NOW(), NOW()),
    ('rlpm_01seedscnr_scannr', 'rl_scanner', 'scanners', 0, 1, 0, 0, NOW(), NOW()),
    ('rlpm_01seedscnr_invent', 'rl_scanner', 'inventory', 0, 1, 1, 0, NOW(), NOW()),
    ('rlpm_01seedscnr_invchl', 'rl_scanner', 'inventory_change_logs', 0, 1, 0, 0, NOW(), NOW()),
    ('rlpm_01seedscnr_invlog', 'rl_scanner', 'inventory_logs', 0, 1, 0, 0, NOW(), NOW()),
    ('rlpm_01seedscnr_self00', 'rl_scanner', 'self', 0, 1, 0, 0, NOW(), NOW()),
    ('rlpm_01seedscnr_jobs00', 'rl_scanner', 'jobs', 0, 1, 0, 0, NOW(), NOW());

-- Global Customer role (assigned to customer-portal users). Global (account_id
-- NULL), role_type 'user', resolved by its fixed id (constants.GlobalCustomerRoleID).
-- Keep in sync with cmd/backfill-customer-role, which creates the identical rows in
-- production.
INSERT IGNORE INTO role (id, name, role_type_code, account_id, created_at, updated_at) VALUES
    ('rl_7vafmsquekgt', 'Customer', 'user', NULL, NOW(), NOW());

-- Customer role permissions. Functionally REQUIRED by the portal:
-- addresses:create/read/update (own-account address book), purchase_orders:create
-- (order entry), and the own-account account-user reads used by the order
-- notification-recipient picker (listing/searching your own team runs as an internal
-- actor on the buyer's own account): team:read (the account-user list), plus roles:read
-- and departments:read (the ?include=role,department the picker requests). The remaining
-- reads are inert (relation-scoped portal calls never consult them) and kept only as
-- future-proofing. Keep in sync with cmd/backfill-customer-role.
INSERT IGNORE INTO role_permission (id, role_id, permission_code, `create`, `read`, `update`, `delete`, created_at, updated_at) VALUES
    ('rlpm_customer_addr000', 'rl_7vafmsquekgt', 'addresses', 1, 1, 1, 0, NOW(), NOW()),
    ('rlpm_customer_purord0', 'rl_7vafmsquekgt', 'purchase_orders', 1, 1, 0, 0, NOW(), NOW()),
    ('rlpm_customer_salord0', 'rl_7vafmsquekgt', 'sales_orders', 0, 1, 0, 0, NOW(), NOW()),
    ('rlpm_customer_prods00', 'rl_7vafmsquekgt', 'products', 0, 1, 0, 0, NOW(), NOW()),
    ('rlpm_customer_invoic0', 'rl_7vafmsquekgt', 'invoices', 0, 1, 0, 0, NOW(), NOW()),
    ('rlpm_customer_ship000', 'rl_7vafmsquekgt', 'shipments', 0, 1, 0, 0, NOW(), NOW()),
    ('rlpm_customer_disc000', 'rl_7vafmsquekgt', 'discounts', 0, 1, 0, 0, NOW(), NOW()),
    ('rlpm_customer_msg0000', 'rl_7vafmsquekgt', 'messaging', 0, 1, 0, 0, NOW(), NOW()),
    ('rlpm_customer_team000', 'rl_7vafmsquekgt', 'team', 0, 1, 0, 0, NOW(), NOW()),
    ('rlpm_customer_roles00', 'rl_7vafmsquekgt', 'roles', 0, 1, 0, 0, NOW(), NOW()),
    ('rlpm_customer_dept000', 'rl_7vafmsquekgt', 'departments', 0, 1, 0, 0, NOW(), NOW());

-- Users
-- us_fltactor3 is a third member used only as a distinct request-log actor so the
-- request-log actor_ids filter tests can prove that filtering by two members
-- (User1 + User2) includes both and excludes a third (see crud_request_logs_test.go).
-- image_url mirrors what UploadUserPhoto persists: a relative path that is the
-- authoritative "avatar exists" signal (resolveImageURL presigns the matching
-- {account_id}/{user_id}.png object). The bytes are uploaded by
-- scripts/seed-user-photos.sh (make seed-user-photos). us_fltactor3 has no avatar.
INSERT IGNORE INTO user (id, name, username, email, hashed_password, email_verified, image_url, created_at, updated_at) VALUES
    ('us_1wjfmmbwg8l7', 'John Doe', 'jdoe', 'dane@augno.com', @password_hash, NOW(), '/v1/core/users/us_1wjfmmbwg8l7/photo', NOW(), NOW()),
    ('us_2ndadmin0000', 'Mike Johnson', 'mjohnson', 'mjohnson@openmrp.ai', @password_hash, NOW(), '/v1/core/users/us_2ndadmin0000/photo', NOW(), NOW()),
    ('us_6p7460uuwibz', 'Sarah Martinez', 'user2', 'user2@openmrp.ai', @password_hash, NOW(), '/v1/core/users/us_6p7460uuwibz/photo', NOW(), NOW()),
    ('us_fltactor3', 'Filter Test User 3', 'ftuser3', 'ftuser3@openmrp.ai', @password_hash, NOW(), NULL, NOW(), NOW()),
    ('us_e2esrep0flag000', 'E2E Sales Rep Stale Flag', 'e2esrep0flag', 'e2e-srep-noflag@test.openmrp.ai', @password_hash, NOW(), NULL, NOW(), NOW());

-- Account-user associations. The admin account-user (SeedAccountUserID) is
-- pinned to the Knitting department so `?include=department` resolves on the
-- seeded account_user GET/LIST responses.
INSERT IGNORE INTO account_user (id, user_id, role_id, account_id, department_id, is_commission_eligible, last_used_at, created_at, updated_at) VALUES
    ('acus_s83fjhyfmqen', 'us_1wjfmmbwg8l7', 'rl_mtg88e6u6fbu', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'dp_01k0a5r01yfx3sj1vy9qgv3dc0', 0, NOW(), NOW(), NOW()),
    ('acus_2ndadmin000', 'us_2ndadmin0000', 'rl_mtg88e6u6fbu', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'dp_01k0a5r01yfx3sj1vy9qgv3dc0', 0, NOW(), NOW(), NOW()),
    ('acus_ubdx4zebgl6p', 'us_6p7460uuwibz', 'rl_hh6mrlkv08n8', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'dp_01k0a5r01yfx3sj1vy9qgv3dc0', 1, NOW(), NOW(), NOW()),
    ('acus_fltactor300', 'us_fltactor3', 'rl_hh6mrlkv08n8', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'dp_01k0a5r01yfx3sj1vy9qgv3dc0', 1, NOW(), NOW(), NOW()),
    ('acus_e2esrep0flag00', 'us_e2esrep0flag000', 'rl_hh6mrlkv08n8', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'dp_01k0a5r01yfx3sj1vy9qgv3dc0', 0, NOW(), NOW(), NOW());

-- API keys
-- HMAC computed as: createHmac('sha256', 'pepper').update(secret).digest()
INSERT IGNORE INTO api_key (type_id, key_id, name, secret_hash, redacted_value, owner_account_id, role_id, created_at, updated_at) VALUES
    ('apky_pajbskcck3cabxajdh8h8', 'u6Xh5ZpaUruMAU12EPAs4z', 'Admin API Key', UNHEX('166b905c32542e43efd8ec0b79077cb48df5430a17fb436c48f2ca12d5fc2b32'), 'mrp_sk_prod_****UCZu', 'ac_01k0a5smf9ekb8rqg12555zjqa', 'rl_mtg88e6u6fbu', NOW(), NOW()),
    ('apky_sandbox_pajbskcck3cabxajdh8h9', 'AM4Bjb7xBLrmM0EZ3ADvlv', 'Sandbox Admin API Key', UNHEX('3228020129fc6ffd68dd835338fffaadc5cc45958a7ec5dd89f6c90961d5f071'), 'mrp_sk_test_****WNXD', 'ac_sandbox_01k0a5smf9ekb8rqg12555zjqb', 'rl_01seedsb_admin000000', NOW(), NOW());
