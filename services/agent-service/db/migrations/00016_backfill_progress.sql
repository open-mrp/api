-- +goose Up

-- Mirrors shared/db/migrations/00076_backfill_progress.sql for agent-service's Postgres: where each paced backfill
-- stopped, so a restart resumes there, and whether it finished, so a finished one never runs again.
CREATE TABLE backfill_progress (
    name         varchar(191) NOT NULL,
    cursor_value varchar(512) NOT NULL DEFAULT '',
    rows_done    bigint NOT NULL DEFAULT 0,
    completed_at timestamptz DEFAULT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (name)
);

-- +goose Down

DROP TABLE backfill_progress;
