-- +goose Up

CREATE TABLE task_leases (
    name        varchar(128) NOT NULL,
    holder      varchar(128) NOT NULL,
    acquired_at timestamptz  NOT NULL,
    expires_at  timestamptz  NOT NULL,
    PRIMARY KEY (name)
);

CREATE INDEX task_leases_expires_at_idx ON task_leases (expires_at);

-- +goose Down

DROP INDEX IF EXISTS task_leases_expires_at_idx;
DROP TABLE IF EXISTS task_leases;
