-- +goose Up

-- Built-in agent tools now live entirely in code (agents.BuiltinTools) rather than as
-- seeded tool_definition/tool_group rows. agent_definition_tool references a tool by its
-- slug instead of an opaque tool_definition_id, and the tool catalog tables are dropped.

-- 1. Add the new slug column and backfill it from the soon-to-be-dropped tool_definition.slug.
ALTER TABLE agent_definition_tool ADD COLUMN tool_slug varchar(100);

UPDATE agent_definition_tool adt
SET tool_slug = td.slug
FROM tool_definition td
WHERE adt.tool_definition_id = td.id;

-- Drop any links that could not be mapped to a slug (e.g. dangling references).
DELETE FROM agent_definition_tool WHERE tool_slug IS NULL;

-- 2. Enforce the new column and unique key. Dropping tool_definition_id also drops the
--    old UNIQUE (agent_definition_id, tool_definition_id) that depended on it.
ALTER TABLE agent_definition_tool ALTER COLUMN tool_slug SET NOT NULL;
ALTER TABLE agent_definition_tool DROP COLUMN tool_definition_id;
ALTER TABLE agent_definition_tool ADD CONSTRAINT agent_definition_tool_agent_def_slug_key UNIQUE (agent_definition_id, tool_slug);

-- 3. Drop the now-unused catalog tables (read-only, seed-populated; replaced by code).
DROP TABLE tool_definition;
DROP TABLE tool_group;

-- +goose Down

CREATE TABLE tool_group (
    id          varchar(191) NOT NULL,
    name        varchar(255) NOT NULL,
    description text,
    slug        varchar(100) NOT NULL,
    icon        varchar(100),
    sort_order  int DEFAULT 0,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (id)
);
CREATE UNIQUE INDEX tool_group_slug_idx ON tool_group (slug);

CREATE TABLE tool_definition (
    id                   varchar(191) NOT NULL,
    display_name         varchar(255) NOT NULL,
    description          text,
    config_schema        jsonb NOT NULL,
    slug                 varchar(100) DEFAULT NULL,
    input_schema         jsonb DEFAULT NULL,
    category             varchar(50) NOT NULL,
    tool_group_id        varchar(191) DEFAULT NULL,
    required_permissions jsonb NOT NULL DEFAULT '[]',
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (id)
);
CREATE INDEX tool_definition_category_idx ON tool_definition (category);
CREATE UNIQUE INDEX tool_definition_slug_idx ON tool_definition (slug) WHERE slug IS NOT NULL;
CREATE INDEX tool_definition_group_id_idx ON tool_definition (tool_group_id);

ALTER TABLE agent_definition_tool DROP CONSTRAINT agent_definition_tool_agent_def_slug_key;
ALTER TABLE agent_definition_tool ADD COLUMN tool_definition_id varchar(191);
-- Best-effort reverse: slug is left in tool_definition_id; rows cannot be mapped back to
-- the original opaque ids without the dropped catalog, so callers re-seeding must reconcile.
UPDATE agent_definition_tool SET tool_definition_id = tool_slug WHERE tool_definition_id IS NULL;
ALTER TABLE agent_definition_tool ALTER COLUMN tool_definition_id SET NOT NULL;
ALTER TABLE agent_definition_tool ADD CONSTRAINT agent_definition_tool_agent_definition_id_tool_definition_id_key UNIQUE (agent_definition_id, tool_definition_id);
ALTER TABLE agent_definition_tool DROP COLUMN tool_slug;
