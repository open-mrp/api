-- 0002_plans.sql
-- Seeds account plans, plan limits, and plan features.

-- Account plans (5 plans)
INSERT IGNORE INTO account_plan (type_id, name, plan_type_code, version, price_per_seat, price_per_month, seat_minimum, display_features, display_order, is_highlighted, button_text, includes_previous_plan, stripe_pricing_plan_id, effective_at, created_at, updated_at) VALUES
    ('acpl_01seed000free00plan000000', 'Free', 'free', 1, 0, NULL, NULL, '["Full API Access","Order Management","Purchasing Module","Production Module","Order Fulfillment","Shipping Module","BOM Management","Inventory Management","Request Tracking"]', 0, 0, 'Get Started', NULL, NULL, '2024-01-01 00:00:00.000', NOW(), NOW()),
    ('acpl_01seed000starter0plan000', 'Starter', 'starter', 1, 35, NULL, NULL, '["Up to 10 Users","Increased Quotas"]', 1, 0, 'Get Started', 'Free', 'bpp_test_61UHuP8kzanXoV2HY16UHtw6IgSQueoEsXV6K51aiDge', '2024-01-01 00:00:00.000', NOW(), NOW()),
    ('acpl_01seed000pro000plan00000', 'Pro', 'pro', 1, 100, NULL, 5, '["Priority Support","Customer Order Portal","Commission Tracking"]', 2, 1, 'Get Started', 'Starter', 'bpp_test_61UHuR7nOYBlL5j9v16UHtw6IgSQueoEsXV6K51ai3qq', '2024-01-01 00:00:00.000', NOW(), NOW()),
    ('acpl_01seed000enterprise0000', 'Enterprise', 'enterprise_template', 1, 0, NULL, NULL, '["Implementation Support","Hands-on Migration Support","In-Person Training","Custom Development","EDI Support","ITAR Compliance","SSO/SAML","ISO 13485 Compliance","ISO 9001 Compliance","Private Cloud Integration"]', 3, 0, 'Contact Sales', 'Pro', 'bpp_test_61UHuT67fBLeDRhGo16UHtw6IgSQueoEsXV6K51ai5NQ', '2024-01-01 00:00:00.000', NOW(), NOW()),
    ('acpl_01seed000enterprise00002', 'Enterprise', 'enterprise', 1, 0, NULL, NULL, '["Implementation Support","Hands-on Migration Support","In-Person Training","Custom Development","EDI Support","ITAR Compliance","SSO/SAML","ISO 13485 Compliance","ISO 9001 Compliance","Private Cloud Integration"]', 3, 0, 'Contact Sales', 'Pro', 'bpp_test_61UHuT67fBLeDRhGo16UHtw6IgSQueoEsXV6K51ai5NQ', '2024-01-01 00:00:00.000', NOW(), NOW());

-- Plan limits
-- Free
INSERT IGNORE INTO account_plan_limit (account_plan_id, `key`, value, created_at, updated_at) VALUES
    ('acpl_01seed000free00plan000000', 'seats_maximum', 1, NOW(), NOW()),
    ('acpl_01seed000free00plan000000', 'invoices_maximum', 10, NOW(), NOW()),
    ('acpl_01seed000free00plan000000', 'sandboxes_maximum', 1, NOW(), NOW()),
    ('acpl_01seed000free00plan000000', 'batches_maximum', 30, NOW(), NOW());

-- Starter
INSERT IGNORE INTO account_plan_limit (account_plan_id, `key`, value, created_at, updated_at) VALUES
    ('acpl_01seed000starter0plan000', 'seats_maximum', 10, NOW(), NOW()),
    ('acpl_01seed000starter0plan000', 'invoices_maximum', 50, NOW(), NOW()),
    ('acpl_01seed000starter0plan000', 'sandboxes_maximum', 3, NOW(), NOW()),
    ('acpl_01seed000starter0plan000', 'batches_maximum', 150, NOW(), NOW());

-- Pro
INSERT IGNORE INTO account_plan_limit (account_plan_id, `key`, value, created_at, updated_at) VALUES
    ('acpl_01seed000pro000plan00000', 'seats_maximum', NULL, NOW(), NOW()),
    ('acpl_01seed000pro000plan00000', 'invoices_maximum', 500, NOW(), NOW()),
    ('acpl_01seed000pro000plan00000', 'sandboxes_maximum', 5, NOW(), NOW()),
    ('acpl_01seed000pro000plan00000', 'batches_maximum', 1500, NOW(), NOW());

-- Enterprise Template
INSERT IGNORE INTO account_plan_limit (account_plan_id, `key`, value, created_at, updated_at) VALUES
    ('acpl_01seed000enterprise0000', 'seats_maximum', NULL, NOW(), NOW()),
    ('acpl_01seed000enterprise0000', 'invoices_maximum', NULL, NOW(), NOW()),
    ('acpl_01seed000enterprise0000', 'sandboxes_maximum', 10, NOW(), NOW()),
    ('acpl_01seed000enterprise0000', 'batches_maximum', NULL, NOW(), NOW());

-- Enterprise Account
INSERT IGNORE INTO account_plan_limit (account_plan_id, `key`, value, created_at, updated_at) VALUES
    ('acpl_01seed000enterprise00002', 'seats_maximum', NULL, NOW(), NOW()),
    ('acpl_01seed000enterprise00002', 'invoices_maximum', NULL, NOW(), NOW()),
    ('acpl_01seed000enterprise00002', 'sandboxes_maximum', 10, NOW(), NOW()),
    ('acpl_01seed000enterprise00002', 'batches_maximum', NULL, NOW(), NOW());

-- Plan features
-- Free
INSERT IGNORE INTO account_plan_feature (account_plan_id, `key`, enabled, created_at, updated_at) VALUES
    ('acpl_01seed000free00plan000000', 'customer_portal', 0, NOW(), NOW()),
    ('acpl_01seed000free00plan000000', 'sales_rep_dashboards', 0, NOW(), NOW()),
    ('acpl_01seed000free00plan000000', 'commission_tracking', 0, NOW(), NOW());

-- Starter
INSERT IGNORE INTO account_plan_feature (account_plan_id, `key`, enabled, created_at, updated_at) VALUES
    ('acpl_01seed000starter0plan000', 'customer_portal', 0, NOW(), NOW()),
    ('acpl_01seed000starter0plan000', 'sales_rep_dashboards', 0, NOW(), NOW()),
    ('acpl_01seed000starter0plan000', 'commission_tracking', 0, NOW(), NOW());

-- Pro
INSERT IGNORE INTO account_plan_feature (account_plan_id, `key`, enabled, created_at, updated_at) VALUES
    ('acpl_01seed000pro000plan00000', 'customer_portal', 1, NOW(), NOW()),
    ('acpl_01seed000pro000plan00000', 'sales_rep_dashboards', 1, NOW(), NOW()),
    ('acpl_01seed000pro000plan00000', 'commission_tracking', 1, NOW(), NOW());

-- Enterprise Template
INSERT IGNORE INTO account_plan_feature (account_plan_id, `key`, enabled, created_at, updated_at) VALUES
    ('acpl_01seed000enterprise0000', 'customer_portal', 1, NOW(), NOW()),
    ('acpl_01seed000enterprise0000', 'sales_rep_dashboards', 1, NOW(), NOW()),
    ('acpl_01seed000enterprise0000', 'commission_tracking', 1, NOW(), NOW());

-- Enterprise Account
INSERT IGNORE INTO account_plan_feature (account_plan_id, `key`, enabled, created_at, updated_at) VALUES
    ('acpl_01seed000enterprise00002', 'customer_portal', 1, NOW(), NOW()),
    ('acpl_01seed000enterprise00002', 'sales_rep_dashboards', 1, NOW(), NOW()),
    ('acpl_01seed000enterprise00002', 'commission_tracking', 1, NOW(), NOW());
