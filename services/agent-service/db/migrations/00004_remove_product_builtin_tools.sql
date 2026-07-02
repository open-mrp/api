-- +goose Up

-- The built-in search_products/list_products tools are redundant with the generated
-- api-gateway endpoint-tool catalog (list_products exposes the same product listing with
-- richer filters + free-text `q` search). Remove the seeded definitions, any agent links to
-- them, and the now-empty Product Tools group.

DELETE FROM agent_definition_tool WHERE tool_definition_id IN (
    'tdef_01k0b1seed0searchproduct0',
    'tdef_01k0b1seed0listproducts00'
);

DELETE FROM tool_definition WHERE id IN (
    'tdef_01k0b1seed0searchproduct0',
    'tdef_01k0b1seed0listproducts00'
);

DELETE FROM tool_group WHERE id = 'tgrp_01k0b1seed0product000000';

-- +goose Down

INSERT INTO tool_group (id, name, slug, icon, sort_order)
VALUES
    ('tgrp_01k0b1seed0product000000', 'Product Tools', 'product_tools', 'inventory', 1)
ON CONFLICT (id) DO NOTHING;

INSERT INTO tool_definition (id, display_name, description, slug, config_schema, input_schema, category, tool_group_id, required_permissions)
VALUES
    ('tdef_01k0b1seed0searchproduct0', 'Search Products',
     'Search for products by keyword or phrase',
     'search_products',
     '{}',
     '{"type":"object","properties":{"query":{"type":"string","description":"Search query for products"}},"required":["query"]}',
     'built_in',
     'tgrp_01k0b1seed0product000000',
     '["products:read"]'),

    ('tdef_01k0b1seed0listproducts00', 'List Products',
     'List all products in the account catalog',
     'list_products',
     '{}',
     '{"type":"object","properties":{}}',
     'built_in',
     'tgrp_01k0b1seed0product000000',
     '["products:read"]')
ON CONFLICT (id) DO NOTHING;
