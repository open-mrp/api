-- +goose Up

-- A chat-triggered run can take "off-conversation" turns: free-text typed directly into the
-- agent-run console is a private fork whose turns enter the run's transcript but are never posted
-- back into the conversation. Once that happens, replying to the agent's earlier message in the
-- conversation must NOT continue the run (it would replay the private fork's context); it should
-- start a fresh run instead. This sticky flag records that a run has diverged this way.
ALTER TABLE agent_run
    ADD COLUMN diverged_from_conversation boolean NOT NULL DEFAULT false;

-- +goose Down

ALTER TABLE agent_run
    DROP COLUMN IF EXISTS diverged_from_conversation;
