-- +goose Up

-- Backs FindOpenIssuesForItemPaged / FindOpenIssuesForItem: equality on (account_id, item_id,
-- status_code) then the created_at ordering, so each page is a range scan on the (created_at, id)
-- keyset rather than a filesort over the item's whole open set (InnoDB appends the primary key id).
ALTER TABLE `inventory_issue`
  ADD KEY `inventory_issue_open_paging_idx` (`account_id`, `item_id`, `status_code`, `created_at`);

-- +goose Down

ALTER TABLE `inventory_issue`
  DROP KEY `inventory_issue_open_paging_idx`;
