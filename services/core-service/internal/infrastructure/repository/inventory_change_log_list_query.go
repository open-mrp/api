package repository

import (
	gosql "database/sql"
	"slices"
	"strings"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/pagination"
)

// iclListColumns is the SELECT list for the change-log listing, in the exact order scanICLListRows reads it. Kept as one constant so the projection and the scanner cannot drift.
const iclListColumns = `icl.id, icl.action_type_code, icl.account_id, icl.created_at, icl.updated_at, ` +
	`i.id, i.sku, i.item_type_code, i.created_at, i.updated_at, ` +
	`q.id, q.value, ` +
	`u.id, u.name, u.abbreviation, u.unit_dimension_code, ` +
	`u.ratio_numerator, u.ratio_denominator, u.offset_numerator, u.offset_denominator, u.created_at, u.updated_at, ` +
	`icl.scanning_station_id, ss.name, ss.scanning_station_type_code, ss.created_at, ss.updated_at, ` +
	`icl.responsible_user_id, usr.name, usr.created_at, usr.updated_at`

// iclListJoins reach every joined table by primary key from the page's change-log rows.
const iclListJoins = ` JOIN item i ON i.id = icl.item_id` +
	` JOIN quantity q ON q.id = icl.quantity_id` +
	` JOIN unit u ON u.id = q.unit_id` +
	` LEFT JOIN scanning_station ss ON ss.id = icl.scanning_station_id` +
	" LEFT JOIN `user` usr ON usr.id = icl.responsible_user_id"

// iclCreatedIndex yields an account's change log in list order.
const iclCreatedIndex = "inventory_change_log_account_created_idx"

// iclMaxArms is the most values of one filter the list reads as separate arms. Each arm reads at most a
// page, so this bounds a request to a few pages; a longer list (a SKU search) is one IN.
const iclMaxArms = 8

// uniqueStrings is v sorted, without repeats.
func uniqueStrings(v []string) []string {
	return slices.Compact(slices.Sorted(slices.Values(v)))
}

// iclListFilter is one IN filter of the list and its key, which yields one value's entries in list order.
type iclListFilter struct {
	column, index string
	values        []string
}

// iclListPlan is how a list reads its page: the keys it may use, and the filter (if any) it reads as one
// arm per value.
type iclListPlan struct {
	filters []iclListFilter
	indexes []string
	split   int
}

// planICLList picks the keys a list may be read from. Left free, the planner reaches for single-column
// keys (created_at spans every account), or ranges a multi-valued filter's key and sorts every match.
//   - A single-valued filter's key yields its entries in list order, so it is offered.
//   - A filter with a few values is yielded in order by no key. When it is the most selective filter
//     (filters run from most to least), it is read as one arm per value, each stopping at a page on that
//     value's key, and the arms merged. Otherwise it is residual to a more selective filter and its key
//     withheld: an arm whose value the other filter excludes would read a whole range to find nothing.
//   - A long list (a SKU search) keeps its key: ranging just its entries and sorting them is the best
//     plan when it is sparse, walking another key when it is dense.
func planICLList(params domain.ListInventoryChangeLogsParams) iclListPlan {
	p := iclListPlan{
		filters: []iclListFilter{
			{"icl.item_id", "inventory_change_log_account_id_item_id_created_at_id_idx", uniqueStrings(params.ItemIDs)},
			{"icl.responsible_user_id", "inventory_change_log_acct_resp_user_created_idx", uniqueStrings(params.ChangedByUserIDs)},
			{"icl.action_type_code", "inventory_change_log_acct_action_type_code_created_idx", uniqueStrings(params.ActionTypeCodes)},
		},
		indexes: []string{iclCreatedIndex},
		split:   -1,
	}
	for i, f := range p.filters {
		if n := len(f.values); n > 1 && n <= iclMaxArms {
			p.split = i
		}
		if len(f.values) > 0 {
			break
		}
	}
	for i, f := range p.filters {
		if n := len(f.values); n == 1 || n > iclMaxArms || i == p.split {
			p.indexes = append(p.indexes, f.index)
		}
	}
	return p
}

// buildICLListQuery assembles the inventory change-log listing SQL and its bind args. Only the
// predicates the caller supplied are emitted: an `(? = false OR ...)` guard is not sargable, and left the
// planner driving from quantity and reading millions of rows for a page.
//
// The page is chosen from inventory_change_log alone, on the keys planICLList picks, and joined after
// (STRAIGHT_JOIN keeps the page first). Forward pages older (DESC), backward pages newer (ASC).
func buildICLListQuery(
	params domain.ListInventoryChangeLogsParams,
	dir pagination.Direction,
	cursorCreatedAt gosql.NullTime,
	cursorID gosql.NullString,
	limit int32,
) (string, []any) {
	plan := planICLList(params)
	filters, split := plan.filters, plan.split

	orderBy := " ORDER BY icl.created_at DESC, icl.id DESC"
	if dir == pagination.DirectionBackward {
		orderBy = " ORDER BY icl.created_at ASC, icl.id ASC"
	}

	var args []any
	// arm writes the page's candidates with filters[split] pinned to value.
	arm := func(b *strings.Builder, value string) {
		b.WriteString("SELECT icl.id, icl.created_at FROM inventory_change_log icl FORCE INDEX (")
		b.WriteString(strings.Join(plan.indexes, ", "))
		b.WriteString(") WHERE icl.account_id = ?")
		args = append(args, params.AccountID)
		for i, f := range filters {
			switch {
			case i == split:
				b.WriteString(" AND " + f.column + " = ?")
				args = append(args, value)
			case len(f.values) > 0:
				b.WriteString(" AND " + f.column + " IN (" + iclPlaceholders(len(f.values)) + ")")
				for _, v := range f.values {
					args = append(args, v)
				}
			}
		}
		if params.StartDate != nil {
			b.WriteString(" AND icl.created_at >= ?")
			args = append(args, *params.StartDate)
		}
		if params.EndDate != nil {
			b.WriteString(" AND icl.created_at <= ?")
			args = append(args, *params.EndDate)
		}
		if cursorCreatedAt.Valid {
			if dir == pagination.DirectionBackward {
				b.WriteString(" AND (icl.created_at > ? OR (icl.created_at = ? AND icl.id > ?))")
			} else {
				b.WriteString(" AND (icl.created_at < ? OR (icl.created_at = ? AND icl.id < ?))")
			}
			args = append(args, cursorCreatedAt.Time, cursorCreatedAt.Time, cursorID.String)
		}
		b.WriteString(orderBy + " LIMIT ?")
		args = append(args, limit)
	}

	var b strings.Builder
	b.WriteString("SELECT STRAIGHT_JOIN ")
	b.WriteString(iclListColumns)
	b.WriteString(" FROM (")
	if split < 0 {
		arm(&b, "")
	} else {
		for i, v := range filters[split].values {
			if i > 0 {
				b.WriteString(" UNION ALL ")
			}
			b.WriteString("(")
			arm(&b, v)
			b.WriteString(")")
		}
		b.WriteString(strings.ReplaceAll(orderBy, "icl.", "") + " LIMIT ?")
		args = append(args, limit)
	}
	b.WriteString(") page JOIN inventory_change_log icl ON icl.id = page.id")
	b.WriteString(iclListJoins)
	b.WriteString(orderBy)

	return b.String(), args
}

// scanICLListRows reads rows produced by buildICLListQuery into domain objects. The scan order must match iclListColumns exactly.
func scanICLListRows(rows *gosql.Rows) ([]*domain.InventoryChangeLog, error) {
	var out []*domain.InventoryChangeLog
	for rows.Next() {
		var icl domain.InventoryChangeLog
		var itemTypeCode string
		var stationID, stationName, stationType gosql.NullString
		var stationCreatedAt, stationUpdatedAt gosql.NullTime
		var responsibleUserID, responsibleUserName gosql.NullString
		var responsibleUserCreatedAt, responsibleUserUpdatedAt gosql.NullTime

		if err := rows.Scan(
			&icl.ID, &icl.ActionTypeCode, &icl.AccountID, &icl.CreatedAt, &icl.UpdatedAt,
			&icl.ItemID, &icl.ItemSKU, &itemTypeCode, &icl.ItemCreatedAt, &icl.ItemUpdatedAt,
			&icl.QuantityID, &icl.QuantityValue,
			&icl.QuantityUnitID, &icl.QuantityUnitName, &icl.QuantityUnitAbbreviation, &icl.QuantityUnitType,
			&icl.QuantityUnitRatioNumerator, &icl.QuantityUnitRatioDenominator,
			&icl.QuantityUnitOffsetNumerator, &icl.QuantityUnitOffsetDenominator,
			&icl.QuantityUnitCreatedAt, &icl.QuantityUnitUpdatedAt,
			&stationID, &stationName, &stationType, &stationCreatedAt, &stationUpdatedAt,
			&responsibleUserID, &responsibleUserName, &responsibleUserCreatedAt, &responsibleUserUpdatedAt,
		); err != nil {
			return nil, err
		}

		icl.ItemTypeCode = &itemTypeCode
		if stationID.Valid {
			icl.ScanningStationID = &stationID.String
		}
		if stationName.Valid {
			icl.ScanningStationName = &stationName.String
		}
		if stationType.Valid {
			icl.ScanningStationType = &stationType.String
		}
		if stationCreatedAt.Valid {
			icl.ScanningStationCreatedAt = &stationCreatedAt.Time
		}
		if stationUpdatedAt.Valid {
			icl.ScanningStationUpdatedAt = &stationUpdatedAt.Time
		}
		if responsibleUserID.Valid {
			icl.ResponsibleUserID = &responsibleUserID.String
		}
		if responsibleUserName.Valid {
			icl.ResponsibleUserName = &responsibleUserName.String
		}
		if responsibleUserCreatedAt.Valid {
			icl.ResponsibleUserCreatedAt = &responsibleUserCreatedAt.Time
		}
		if responsibleUserUpdatedAt.Valid {
			icl.ResponsibleUserUpdatedAt = &responsibleUserUpdatedAt.Time
		}

		out = append(out, &icl)
	}
	return out, rows.Err()
}

func iclPlaceholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat("?, ", n-1) + "?"
}
