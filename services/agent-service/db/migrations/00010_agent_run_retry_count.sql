-- +goose Up

-- Tracks how many times a run has been re-attempted after a failure. A failed run is no longer a
-- terminal dead end: it can be resumed (manually via the retry action, or automatically for transient
-- pre-side-effect failures), and this counter bounds those attempts so a persistently-failing run
-- can't loop forever.
ALTER TABLE agent_run
    ADD COLUMN retry_count int NOT NULL DEFAULT 0;

-- +goose Down

ALTER TABLE agent_run
    DROP COLUMN IF EXISTS retry_count;
