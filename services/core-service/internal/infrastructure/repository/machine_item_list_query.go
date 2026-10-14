package repository

import (
	"context"
	gosql "database/sql"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/apikit/pagination"
)

// machineItemPageIndex holds a machine's items in list order, so a page walks it and stops at the limit.
const machineItemPageIndex = "machine_item_account_machine_created_idx"

// buildMachineItemListQuery assembles one page of a machine's items. Only the predicates the caller
// supplied are emitted: an `(? IS NULL OR ...)` guard is not sargable. Forward pages older (DESC),
// backward pages newer (ASC).
func buildMachineItemListQuery(params domain.ListMachineItemsParams, search gosql.NullString, cur *pagination.StringCursor, limit int32) (string, []any) {
	backward := cur != nil && cur.Direction == pagination.DirectionBackward
	query := `SELECT STRAIGHT_JOIN i.id, i.sku, i.description, mi.created_at, mi.updated_at
FROM machine_item mi FORCE INDEX (` + machineItemPageIndex + `)
JOIN item i ON i.id = mi.item_id AND i.account_id = mi.account_id
WHERE mi.account_id = ? AND mi.machine_id = ? AND i.deleted_at IS NULL`
	args := []any{params.AccountID, params.MachineID}
	if search.Valid {
		query += ` AND (i.sku LIKE ? OR i.description LIKE ?)`
		args = append(args, search.String, search.String)
	}
	if cur != nil {
		if backward {
			query += ` AND (mi.created_at > ? OR (mi.created_at = ? AND mi.item_id > ?))`
		} else {
			query += ` AND (mi.created_at < ? OR (mi.created_at = ? AND mi.item_id < ?))`
		}
		args = append(args, cur.OccurredAt, cur.OccurredAt, cur.ID)
	}
	if backward {
		query += ` ORDER BY mi.created_at ASC, mi.item_id ASC`
	} else {
		query += ` ORDER BY mi.created_at DESC, mi.item_id DESC`
	}
	return query + ` LIMIT ?`, append(args, limit)
}

func (r *machineRepoImpl) listMachineItemPage(ctx context.Context, query string, args []any) ([]*domain.MachineItem, error) {
	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var items []*domain.MachineItem
	for rows.Next() {
		var item domain.MachineItem
		var description gosql.NullString
		if err := rows.Scan(&item.ItemID, &item.SKU, &description, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		if description.Valid {
			item.Description = &description.String
		}
		items = append(items, &item)
	}
	return items, rows.Err()
}
