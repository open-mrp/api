package repository

import (
	gosql "database/sql"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/apikit/pagination"
)

const locationListColumns = `sl.id, sl.name, sl.storage_location_type_code, sl.parent_id, p.name, p.storage_location_type_code, ` +
	`sl.is_active, sl.address_id, sl.created_at, sl.updated_at`

// locationListFilter is one page of the location listing.
type locationListFilter struct {
	accountID string
	// search is a LIKE pattern, already escaped and wrapped; empty for none.
	search    string
	isActive  *bool
	parentIDs []string
	cursor    *pagination.StringCursor
	limit     int32
}

// buildLocationListQuery emits only the predicates the caller supplied, so an absent filter costs nothing. It pages
// on the account's (account_id, created_at, id) index; name, status and parent filters are residual on it, since an
// account holds a few hundred locations.
func buildLocationListQuery(f locationListFilter) (string, []any) {
	var lf listFilter
	lf.add("sl.account_id = ?", f.accountID)
	if f.search != "" {
		lf.add("sl.name LIKE ?", f.search)
	}
	if f.isActive != nil {
		lf.add("sl.is_active = ?", *f.isActive)
	}
	lf.in("sl.parent_id", f.parentIDs)
	order := lf.keyset(f.cursor, "sl.created_at", "sl.id")
	args := append(lf.args, f.limit)
	return "SELECT " + locationListColumns + "\nFROM storage_location sl\nLEFT JOIN storage_location p ON p.id = sl.parent_id" +
		lf.whereSQL() + "\nORDER BY " + order + "\nLIMIT ?", args
}

// scanLocationListRows reads rows in locationListColumns order.
func scanLocationListRows(rows *gosql.Rows) ([]sqlc.GetLocationRow, error) {
	var out []sqlc.GetLocationRow
	for rows.Next() {
		var r sqlc.GetLocationRow
		if err := rows.Scan(&r.ID, &r.Name, &r.TypeCode, &r.ParentID, &r.ParentName, &r.ParentTypeCode,
			&r.IsActive, &r.AddressID, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
