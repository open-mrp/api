-- +goose Up

-- GetInboxBacklogStats: WHERE status IN ('received', 'discarded') AND service_name = $1, run by every replica on every metrics export. message_inbox has no status index, so without this each export scans the whole retained processed history. Partial, so it holds only the rows the gauges report.
CREATE INDEX IF NOT EXISTS message_inbox_backlog_idx
    ON message_inbox (service_name, handler, received_at)
    WHERE status IN ('received', 'discarded');

-- +goose Down

DROP INDEX IF EXISTS message_inbox_backlog_idx;
