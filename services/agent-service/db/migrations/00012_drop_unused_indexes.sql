-- +goose Up

-- Drop indexes that no query uses (verified against internal/infrastructure/queries and the Go
-- repo — none of these columns appear in a WHERE/JOIN/ORDER BY) or that are strictly redundant
-- with another index's leftmost prefix. Kept deliberately: agent_run_conversation_idx (00008, for
-- the not-yet-implemented "runs in this conversation" surface), the *_expires_at / *_lock_expires_at
-- reaper/lock-cleanup indexes, and the *_request_id / *_parent_message_id message-tracing indexes.

-- Redundant: strict leftmost prefix of UNIQUE (account_id, agent_definition_id) on agent_account_status.
DROP INDEX IF EXISTS agent_account_status_account_id_idx;

-- Unused: triggered_by_actor_id is only ever inserted/selected, never filtered.
DROP INDEX IF EXISTS agent_run_triggered_by_actor_id_idx;

-- Unused: message_type is never filtered (inbox lookups go by (handler, message_id); the purge by
-- (status, processed_at)).
DROP INDEX IF EXISTS message_inbox_message_type_idx;

-- Unused: idempotency_key is never filtered (lookups use type_id or (service_name, scope_hash)).
DROP INDEX IF EXISTS service_idempotency_key_idempotency_key_idx;

-- +goose Down

CREATE INDEX service_idempotency_key_idempotency_key_idx ON service_idempotency_key (idempotency_key);
CREATE INDEX message_inbox_message_type_idx ON message_inbox (message_type);
CREATE INDEX agent_run_triggered_by_actor_id_idx ON agent_run (triggered_by_actor_id);
CREATE INDEX agent_account_status_account_id_idx ON agent_account_status (account_id);
