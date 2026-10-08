-- +goose NO TRANSACTION
-- +goose Up

-- Superseded by message_outbox_status_service_next_run_at_idx (00073) once every pod claims per service.
ALTER TABLE `message_outbox`
  DROP KEY `message_outbox_status_next_run_at_idx`;

-- +goose Down

ALTER TABLE `message_outbox`
  ADD KEY `message_outbox_status_next_run_at_idx` (`status`, `next_run_at`);
