package repository

import (
	gosql "database/sql"
	"strings"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/apikit/pagination"
)

const productLineListColumns = `pl.id, pl.name, pl.description, pl.notes, pl.is_commission_exempt, pl.is_freight_exempt, ` +
	`pl.unit_group_id, pl.supply_code, pl.is_active, pl.account_id, pl.created_at, pl.updated_at`

// productLineListFilter is one page of the product line listing.
type productLineListFilter struct {
	accountID string
	// search is a LIKE pattern, already escaped and wrapped; empty for none.
	search   string
	isActive *bool
	dir      pagination.Direction
	// cursor is nil on the first page.
	cursorCreatedAt *time.Time
	cursorID        string
	limit           int32
}

// buildProductLineListQuery emits only the predicates the caller supplied, so an absent filter costs nothing. The
// set is the tenant's lines plus the system lines, which number in the dozens, so the name and status filters are
// residual on the account's created-at index.
func buildProductLineListQuery(f productLineListFilter) (string, []any) {
	where := []string{"(pl.account_id = ? OR pl.account_id IS NULL)"}
	args := []any{f.accountID}
	if f.search != "" {
		where, args = append(where, "pl.name LIKE ?"), append(args, f.search)
	}
	if f.isActive != nil {
		where, args = append(where, "pl.is_active = ?"), append(args, *f.isActive)
	}
	backward := f.dir == pagination.DirectionBackward
	if f.cursorCreatedAt != nil {
		if backward {
			where = append(where, "(pl.created_at > ? OR (pl.created_at = ? AND pl.id > ?))")
		} else {
			where = append(where, "(pl.created_at < ? OR (pl.created_at = ? AND pl.id < ?))")
		}
		args = append(args, *f.cursorCreatedAt, *f.cursorCreatedAt, f.cursorID)
	}
	order := " ORDER BY pl.created_at DESC, pl.id DESC"
	if backward {
		order = " ORDER BY pl.created_at ASC, pl.id ASC"
	}
	args = append(args, f.limit)
	return "SELECT " + productLineListColumns + " FROM product_line pl WHERE " + strings.Join(where, " AND ") + order + " LIMIT ?", args
}

// scanProductLineListRows reads rows in productLineListColumns order.
func scanProductLineListRows(rows *gosql.Rows) ([]sqlc.GetProductLineRow, error) {
	var out []sqlc.GetProductLineRow
	for rows.Next() {
		var r sqlc.GetProductLineRow
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.Notes, &r.IsCommissionExempt, &r.IsFreightExempt,
			&r.UnitGroupID, &r.SupplyCode, &r.IsActive, &r.AccountID, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
