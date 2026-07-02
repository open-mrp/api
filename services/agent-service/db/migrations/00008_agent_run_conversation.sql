-- +goose Up

-- Make an agent run conversation-aware. A chat-triggered run surfaces in (and dialogs with) a
-- conversation; scheduled/event/silent runs legitimately have no conversation (both columns null).
-- trigger_type gains a 'chat' value — it is a plain varchar, so no schema change is needed for that.
-- conversation_id / trigger_message_id are string refs into notification-service's MySQL (no FK;
-- the agent and conversation stores are separate databases).
ALTER TABLE agent_run
    ADD COLUMN conversation_id    varchar(191),
    ADD COLUMN trigger_message_id varchar(191);

-- "Which runs surface in this conversation" (a conversation hosts many runs over time).
CREATE INDEX agent_run_conversation_idx ON agent_run (conversation_id);

-- +goose Down

DROP INDEX IF EXISTS agent_run_conversation_idx;
ALTER TABLE agent_run
    DROP COLUMN IF EXISTS conversation_id,
    DROP COLUMN IF EXISTS trigger_message_id;
