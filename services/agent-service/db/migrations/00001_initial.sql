-- +goose Up

CREATE TABLE agent_definition (
    id              varchar(191) NOT NULL,
    account_id      varchar(191) DEFAULT NULL,
    name            varchar(255) NOT NULL,
    slug            varchar(255) NOT NULL,
    description     text,
    definition_type varchar(20) NOT NULL DEFAULT 'system',
    category_code   varchar(50) NOT NULL,
    trigger_type    varchar(50) NOT NULL,
    is_active       boolean NOT NULL DEFAULT true,
    config          jsonb NOT NULL,
    role_id         varchar(191) DEFAULT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (id),
    UNIQUE (account_id, slug)
);

CREATE INDEX agent_definition_name_idx ON agent_definition USING gin (to_tsvector('english', name));
CREATE INDEX agent_definition_description_idx ON agent_definition USING gin (to_tsvector('english', description));

CREATE TABLE tool_group (
    id           varchar(191) NOT NULL,
    name         varchar(255) NOT NULL,
    description  text,
    slug         varchar(100) NOT NULL,
    icon         varchar(100) DEFAULT NULL,
    sort_order   int NOT NULL DEFAULT 0,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
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

CREATE TABLE agent_definition_tool (
    id                  varchar(191) NOT NULL,
    agent_definition_id varchar(191) NOT NULL,
    tool_definition_id  varchar(191) NOT NULL,
    config              jsonb NOT NULL,
    sort_order          int NOT NULL DEFAULT 0,
    require_review      boolean NOT NULL DEFAULT false,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (id),
    UNIQUE (agent_definition_id, tool_definition_id)
);

CREATE TABLE agent_config (
    id                  varchar(191) NOT NULL,
    account_id          varchar(191) NOT NULL,
    agent_definition_id varchar(191) NOT NULL,
    is_enabled          boolean NOT NULL DEFAULT true,
    config              jsonb NOT NULL,
    schedule            varchar(255) DEFAULT NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (id)
);

CREATE TABLE agent_run (
    id                         varchar(191) NOT NULL,
    account_id                 varchar(191) NOT NULL,
    agent_definition_id        varchar(191) NOT NULL,
    agent_config_id            varchar(191) DEFAULT NULL,
    status_code                varchar(50) NOT NULL,
    trigger_type               varchar(50) NOT NULL,
    input                      jsonb NOT NULL,
    output                     jsonb NOT NULL,
    error_message              text,
    started_at                 timestamptz DEFAULT NULL,
    completed_at               timestamptz DEFAULT NULL,
    duration_ms                int DEFAULT NULL,
    total_input_tokens         bigint NOT NULL DEFAULT 0,
    total_output_tokens        bigint NOT NULL DEFAULT 0,
    triggered_by_actor_id      varchar(191) DEFAULT NULL,
    triggered_by_identity_type varchar(50) DEFAULT NULL,
    triggered_by_actor_name    varchar(255) DEFAULT NULL,
    allowed_tool_slugs         jsonb NOT NULL DEFAULT '[]'::jsonb,
    created_at                 timestamptz NOT NULL DEFAULT now(),
    updated_at                 timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (id)
);

CREATE INDEX agent_run_triggered_by_actor_id_idx ON agent_run (triggered_by_actor_id);

CREATE TABLE agent_run_event (
    id              varchar(191) NOT NULL,
    agent_run_id    varchar(191) NOT NULL,
    account_id      varchar(191) NOT NULL,
    step_type       varchar(30) NOT NULL,
    title           varchar(255) NOT NULL,
    content         text,
    sequence        int NOT NULL,
    duration_ms     int,
    agent_action_id varchar(191),
    metadata        jsonb NOT NULL DEFAULT '{}',
    actor_id        varchar(191) DEFAULT NULL,
    actor_type      varchar(50) DEFAULT NULL,
    actor_name      varchar(255) DEFAULT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (id)
);

CREATE INDEX agent_run_event_run_id_seq_idx ON agent_run_event (agent_run_id, sequence);

CREATE TABLE agent_action (
    id              varchar(191) NOT NULL,
    account_id      varchar(191) NOT NULL,
    agent_run_id    varchar(191) NOT NULL,
    tool_slug       varchar(255) NOT NULL,
    status_code     varchar(50) NOT NULL,
    label           varchar(255) DEFAULT NULL,
    description     text,
    input           jsonb NOT NULL,
    output          jsonb NOT NULL,
    error_message   text,
    entity_type     varchar(255) DEFAULT NULL,
    entity_id       varchar(255) DEFAULT NULL,
    requires_review boolean NOT NULL DEFAULT false,
    reviewed_at     timestamptz DEFAULT NULL,
    reviewed_by              varchar(255) DEFAULT NULL,
    reviewed_by_actor_type   varchar(50) DEFAULT NULL,
    reviewed_by_actor_name   varchar(255) DEFAULT NULL,
    executed_at     timestamptz DEFAULT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (id)
);

CREATE TABLE agent_artifact (
    id              varchar(191) NOT NULL,
    account_id      varchar(191) NOT NULL,
    agent_action_id varchar(191) NOT NULL,
    artifact_type   varchar(255) NOT NULL,
    name            varchar(255) NOT NULL,
    content         text,
    metadata        jsonb NOT NULL,
    s3_key          varchar(255) DEFAULT NULL,
    mime_type       varchar(255) DEFAULT NULL,
    size_bytes      bigint DEFAULT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (id)
);

CREATE TABLE agent_memory (
    id          varchar(191) NOT NULL,
    account_id  varchar(191) NOT NULL,
    category    varchar(255) NOT NULL,
    content     text NOT NULL,
    metadata    jsonb NOT NULL,
    entity_type varchar(255) DEFAULT NULL,
    entity_id   varchar(255) DEFAULT NULL,
    importance  double precision NOT NULL DEFAULT 0,
    expires_at  timestamptz DEFAULT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (id)
);

CREATE INDEX agent_memory_content_idx ON agent_memory USING gin (to_tsvector('english', content));

CREATE TABLE agent_token_usage (
    id             varchar(191) NOT NULL,
    account_id     varchar(191) NOT NULL,
    date           date NOT NULL,
    input_tokens   bigint NOT NULL DEFAULT 0,
    output_tokens  bigint NOT NULL DEFAULT 0,
    total_cost     double precision NOT NULL DEFAULT 0,
    run_count      int NOT NULL DEFAULT 0,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (id),
    UNIQUE (account_id, date)
);

CREATE TABLE agent_account_status (
    id                  varchar(191) NOT NULL,
    account_id          varchar(191) NOT NULL,
    agent_definition_id varchar(191) NOT NULL,
    status_code         varchar(50) NOT NULL DEFAULT 'inactive',
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (id),
    UNIQUE (account_id, agent_definition_id)
);

CREATE INDEX agent_account_status_account_id_idx ON agent_account_status (account_id);

CREATE TABLE message_outbox (
    id                bigserial PRIMARY KEY,
    message_id        varchar(255) NOT NULL,
    service_name      varchar(255) NOT NULL,
    message_type      varchar(255) NOT NULL,
    destination       varchar(255) NOT NULL,
    routing_key       varchar(255) DEFAULT NULL,
    headers           jsonb DEFAULT NULL,
    payload           jsonb NOT NULL,
    status            varchar(50) NOT NULL DEFAULT 'pending',
    attempts          int NOT NULL DEFAULT 0,
    max_attempts      int NOT NULL DEFAULT 25,
    next_run_at       timestamptz NOT NULL DEFAULT now(),
    locked_at         timestamptz DEFAULT NULL,
    lock_owner        varchar(64) DEFAULT NULL,
    lock_expires_at   timestamptz DEFAULT NULL,
    last_error        text,
    published_at      timestamptz DEFAULT NULL,
    request_id        varchar(255) DEFAULT NULL,
    parent_message_id varchar(255) DEFAULT NULL,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    UNIQUE (message_id)
);

CREATE INDEX message_outbox_status_next_run_at_idx ON message_outbox (status, next_run_at);
CREATE INDEX message_outbox_lock_expires_at_idx ON message_outbox (lock_expires_at);
CREATE INDEX message_outbox_request_id_idx ON message_outbox (request_id);
CREATE INDEX message_outbox_parent_message_id_idx ON message_outbox (parent_message_id);

CREATE TABLE message_inbox (
    id                bigserial PRIMARY KEY,
    message_id        varchar(255) NOT NULL,
    service_name      varchar(255) NOT NULL,
    handler           varchar(128) NOT NULL,
    message_type      varchar(255) NOT NULL,
    request_id        varchar(255) DEFAULT NULL,
    parent_message_id varchar(255) DEFAULT NULL,
    status            varchar(50) NOT NULL DEFAULT 'received',
    attempts          int NOT NULL DEFAULT 0,
    last_error        text,
    received_at       timestamptz NOT NULL DEFAULT now(),
    processed_at      timestamptz DEFAULT NULL,
    UNIQUE (handler, message_id)
);

CREATE INDEX message_inbox_message_type_idx ON message_inbox (message_type);
CREATE INDEX message_inbox_processed_at_idx ON message_inbox (processed_at);
CREATE INDEX message_inbox_request_id_idx ON message_inbox (request_id);
CREATE INDEX message_inbox_parent_message_id_idx ON message_inbox (parent_message_id);

CREATE TABLE service_idempotency_key (
    id              bigserial PRIMARY KEY,
    type_id         varchar(255) NOT NULL,
    service_name    varchar(255) NOT NULL,
    handler         varchar(128) NOT NULL,
    idempotency_key varchar(255) NOT NULL,
    actor_id        varchar(255) DEFAULT NULL,
    identity_type   varchar(255) NOT NULL,
    scope_hash      char(64) NOT NULL,
    response_code   int DEFAULT NULL,
    response_body   jsonb DEFAULT NULL,
    recovery_point  varchar(255) NOT NULL,
    locked_at       timestamptz DEFAULT NULL,
    lock_owner      varchar(64) DEFAULT NULL,
    lock_expires_at timestamptz DEFAULT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    last_run_at     timestamptz DEFAULT NULL,
    expires_at      timestamptz DEFAULT NULL,
    UNIQUE (type_id)
);

CREATE INDEX service_idempotency_key_scope_hash_idx ON service_idempotency_key (scope_hash);
CREATE INDEX service_idempotency_key_idempotency_key_idx ON service_idempotency_key (idempotency_key);
CREATE INDEX service_idempotency_key_lock_expires_at_idx ON service_idempotency_key (lock_expires_at);
CREATE INDEX service_idempotency_key_expires_at_idx ON service_idempotency_key (expires_at);

CREATE TABLE deleted_record (
    id            bigserial PRIMARY KEY,
    deleted_at    timestamptz NOT NULL DEFAULT now(),
    resource_type text        NOT NULL,
    resource_id   text        NOT NULL,
    data          jsonb       NOT NULL
);

CREATE INDEX deleted_record_resource_type_resource_id_idx ON deleted_record (resource_type, resource_id);
CREATE INDEX deleted_record_deleted_at_idx ON deleted_record (deleted_at);

-- +goose Down

DROP TABLE IF EXISTS deleted_record;
DROP TABLE IF EXISTS service_idempotency_key;
DROP TABLE IF EXISTS message_inbox;
DROP TABLE IF EXISTS message_outbox;
DROP TABLE IF EXISTS agent_account_status;
DROP TABLE IF EXISTS agent_token_usage;
DROP TABLE IF EXISTS agent_memory;
DROP TABLE IF EXISTS agent_artifact;
DROP TABLE IF EXISTS agent_action;
DROP TABLE IF EXISTS agent_run_event;
DROP TABLE IF EXISTS agent_run;
DROP TABLE IF EXISTS agent_config;
DROP TABLE IF EXISTS agent_definition_tool;
DROP TABLE IF EXISTS tool_definition;
DROP TABLE IF EXISTS tool_group;
DROP TABLE IF EXISTS agent_definition;
