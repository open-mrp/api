-- +goose Up

INSERT INTO tool_group (id, name, slug, icon, sort_order)
VALUES
    ('tgrp_01k0b1seed0customer00000', 'Customer Tools', 'customer_tools', 'people', 0),
    ('tgrp_01k0b1seed0product000000', 'Product Tools', 'product_tools', 'inventory', 1),
    ('tgrp_01k0b1seed0memory0000000', 'Memory Management', 'memory_management', 'memory', 2),
    ('tgrp_01k0b1seed0general000000', 'General', 'general', 'settings', 3),
    ('tgrp_01k0b1seed0knowledge0000', 'Knowledge', 'knowledge', 'book', 4)
ON CONFLICT (id) DO NOTHING;

INSERT INTO tool_definition (id, display_name, description, slug, config_schema, input_schema, category, tool_group_id, required_permissions)
VALUES
    ('tdef_01k0b1seed0savememory0000', 'Save Memory',
     'Save an observation about a customer or product for future reference',
     'save_memory',
     '{}',
     '{"type":"object","properties":{"category":{"type":"string","description":"Memory category (e.g., customer_preference, ordering_pattern, product_alias)"},"content":{"type":"string","description":"The observation to remember"},"entity_type":{"type":"string","description":"Type of entity this relates to (e.g., account_relation, product)"},"entity_id":{"type":"string","description":"ID of the related entity"},"importance":{"type":"number","description":"Importance score from 0.0 to 1.0"}},"required":["category","content","importance"]}',
     'built_in',
     'tgrp_01k0b1seed0memory0000000',
     '[]'),

    ('tdef_01k0b1seed0createalert000', 'Create Alert',
     'Create an alert that requires human attention',
     'create_alert',
     '{}',
     '{"type":"object","properties":{"severity":{"type":"string","enum":["info","warning","urgent","critical"],"description":"Alert severity level"},"title":{"type":"string","description":"Short alert title"},"message":{"type":"string","description":"Detailed alert message"}},"required":["severity","title","message"]}',
     'built_in',
     'tgrp_01k0b1seed0general000000',
     '[]'),

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
     '["products:read"]'),

    ('tdef_01k0b1seed0lookupcustomer0', 'Lookup Customer',
     'Look up a customer by their email address',
     'lookup_customer',
     '{}',
     '{"type":"object","properties":{"email":{"type":"string","description":"Customer email address"}},"required":["email"]}',
     'built_in',
     'tgrp_01k0b1seed0customer00000',
     '["customers:read"]'),

    ('tdef_01k0b1seed0createartifact', 'Create Artifact',
     'Create an artifact such as a report, document, or data export',
     'create_artifact',
     '{}',
     '{"type":"object","properties":{"artifact_type":{"type":"string","description":"Type of artifact (e.g., report, document, csv)"},"name":{"type":"string","description":"Artifact name"},"content":{"type":"string","description":"Artifact content"},"mime_type":{"type":"string","description":"MIME type of the content (e.g., text/plain, text/csv, application/json)"}},"required":["artifact_type","name","content","mime_type"]}',
     'built_in',
     'tgrp_01k0b1seed0general000000',
     '[]'),

    ('tdef_01k0b1seed0updatememory00', 'Update Memory',
     'Update an existing memory entry',
     'update_memory',
     '{}',
     '{"type":"object","properties":{"memory_id":{"type":"string","description":"ID of the memory to update"},"category":{"type":"string","description":"Memory category"},"content":{"type":"string","description":"Updated memory content"},"importance":{"type":"number","description":"Importance score from 0.0 to 1.0"},"entity_type":{"type":"string","description":"Type of entity this relates to"},"entity_id":{"type":"string","description":"ID of the related entity"}},"required":["memory_id","category","content","importance"]}',
     'built_in',
     'tgrp_01k0b1seed0memory0000000',
     '[]'),

    ('tdef_01k0b1seed0deletememory00', 'Delete Memory',
     'Delete a memory entry that is no longer relevant',
     'delete_memory',
     '{}',
     '{"type":"object","properties":{"memory_id":{"type":"string","description":"ID of the memory to delete"}},"required":["memory_id"]}',
     'built_in',
     'tgrp_01k0b1seed0memory0000000',
     '[]'),

    ('tdef_01k0b1seed0readdoc000000', 'Read Doc',
     'Read the content of an Augno documentation page. To find the right page, first fetch https://docs.augno.com/llms.txt which lists all available pages with descriptions.',
     'read_doc',
     '{}',
     '{"type":"object","properties":{"url":{"type":"string","description":"The full URL of the documentation page to read (must be from docs.augno.com)"}},"required":["url"]}',
     'built_in',
     'tgrp_01k0b1seed0knowledge0000',
     '[]'),

    ('tdef_01k0b1seed0fetchurl00000', 'Fetch URL',
     'Fetch the content of a public URL and return the response body as text',
     'fetch_url',
     '{}',
     '{"type":"object","properties":{"url":{"type":"string","description":"The HTTPS URL to fetch"}},"required":["url"]}',
     'built_in',
     'tgrp_01k0b1seed0knowledge0000',
     '[]')
ON CONFLICT (id) DO NOTHING;

-- +goose Down

DELETE FROM tool_definition WHERE id IN (
    'tdef_01k0b1seed0savememory0000',
    'tdef_01k0b1seed0createalert000',
    'tdef_01k0b1seed0searchproduct0',
    'tdef_01k0b1seed0listproducts00',
    'tdef_01k0b1seed0lookupcustomer0',
    'tdef_01k0b1seed0createartifact',
    'tdef_01k0b1seed0updatememory00',
    'tdef_01k0b1seed0deletememory00',
    'tdef_01k0b1seed0readdoc000000',
    'tdef_01k0b1seed0fetchurl00000'
);

DELETE FROM tool_group WHERE id IN (
    'tgrp_01k0b1seed0customer00000',
    'tgrp_01k0b1seed0product000000',
    'tgrp_01k0b1seed0memory0000000',
    'tgrp_01k0b1seed0general000000',
    'tgrp_01k0b1seed0knowledge0000'
);
