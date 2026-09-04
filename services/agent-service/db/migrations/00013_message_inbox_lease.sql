-- +goose Up

-- Mirrors shared/db/migrations/00018_message_inbox_lease.sql for agent-service's Postgres inbox; see that file for why the lease exists.
ALTER TABLE message_inbox
  ADD COLUMN failed_at timestamptz DEFAULT NULL,
  ADD COLUMN lock_owner varchar(64) DEFAULT NULL,
  ADD COLUMN lock_expires_at timestamptz DEFAULT NULL;

-- +goose Down

ALTER TABLE message_inbox
  DROP COLUMN lock_expires_at,
  DROP COLUMN lock_owner,
  DROP COLUMN failed_at;
