-- +goose NO TRANSACTION
-- +goose Up

-- Each service's enqueuer claims only the pending rows it wrote, so the claim needs the service between
-- the status and the due time. This supersedes message_outbox_status_next_run_at_idx, which deployed pods
-- still claim through, so that one is dropped in a later release.
ALTER TABLE `message_outbox`
  ADD KEY `message_outbox_status_service_next_run_at_idx` (`status`, `service_name`, `next_run_at`);

-- +goose Down

ALTER TABLE `message_outbox`
  DROP KEY `message_outbox_status_service_next_run_at_idx`;
