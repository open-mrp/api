package repository

import (
	gosql "database/sql"
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

// iclListFrom is the join graph shared by every filter combination. Each joined table is reached by primary key, so the joins are nested-loop lookups over whatever rows the driving index on inventory_change_log yields.
const iclListFrom = ` FROM inventory_change_log icl` +
	` JOIN item i ON i.id = icl.item_id` +
	` JOIN quantity q ON q.id = icl.quantity_id` +
	` JOIN unit u ON u.id = q.unit_id` +
	` LEFT JOIN scanning_station ss ON ss.id = icl.scanning_station_id` +
	" LEFT JOIN `user` usr ON usr.id = icl.responsible_user_id"

// buildICLListQuery assembles the inventory change-log listing SQL and its bind args. Predicates are omitted entirely when the caller did not supply a value, rather than being wrapped in an `OR <sentinel> = false` guard.
//
// The guard form is why this listing reached a 51s p50 in production. `(? = false OR icl.item_id IN (...))` is not sargable: the optimizer could not tell that account_id and created_at narrowed anything, abandoned the (account_id, created_at, id) composite, and drove the join from quantity via inventory_change_log_quantity_id_key instead — 7,034,759 rows read to return 11. Emitting only the predicates that actually narrow the set lets each filter combination land on the composite built for it (account_created, account_id_item_id_created_at_id, acct_action_type_code_created, acct_resp_user_created).
//
// STRAIGHT_JOIN is required on top of that. With sargable predicates but a free join order, MySQL still starts at the smallest table (unit, 113 rows), fans out through quantity, reaches inventory_change_log by quantity_id, and pays "Using temporary; Using filesort" — the ORDER BY then has to sort the account's whole history before LIMIT can apply. Forcing inventory_change_log to drive lets its composite supply the sort order directly, so the scan stops at LIMIT and every other table is a primary-key eq_ref lookup. STRAIGHT_JOIN pins only the join order, not the index, so each filter combination still picks its own best composite.
//
// Direction semantics match the sqlc queries this replaced: forward pages older (DESC), backward pages newer (ASC). The cursor predicate is emitted only when a cursor was supplied, so the first page is a clean range scan.
func buildICLListQuery(
	params domain.ListInventoryChangeLogsParams,
	dir pagination.Direction,
	cursorCreatedAt gosql.NullTime,
	cursorID gosql.NullString,
	limit int32,
) (string, []any) {
	args := make([]any, 0, 8+len(params.ItemIDs)+len(params.ActionTypeCodes)+len(params.ChangedByUserIDs))

	var b strings.Builder
	b.WriteString("SELECT STRAIGHT_JOIN ")
	b.WriteString(iclListColumns)
	b.WriteString(iclListFrom)
	b.WriteString(" WHERE icl.account_id = ?")
	args = append(args, params.AccountID)

	if len(params.ItemIDs) > 0 {
		b.WriteString(" AND icl.item_id IN (")
		b.WriteString(iclPlaceholders(len(params.ItemIDs)))
		b.WriteString(")")
		for _, id := range params.ItemIDs {
			args = append(args, id)
		}
	}
	if len(params.ActionTypeCodes) > 0 {
		b.WriteString(" AND icl.action_type_code IN (")
		b.WriteString(iclPlaceholders(len(params.ActionTypeCodes)))
		b.WriteString(")")
		for _, code := range params.ActionTypeCodes {
			args = append(args, code)
		}
	}
	if len(params.ChangedByUserIDs) > 0 {
		b.WriteString(" AND icl.responsible_user_id IN (")
		b.WriteString(iclPlaceholders(len(params.ChangedByUserIDs)))
		b.WriteString(")")
		for _, id := range params.ChangedByUserIDs {
			args = append(args, id)
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

	if dir == pagination.DirectionBackward {
		b.WriteString(" ORDER BY icl.created_at ASC, icl.id ASC")
	} else {
		b.WriteString(" ORDER BY icl.created_at DESC, icl.id DESC")
	}
	b.WriteString(" LIMIT ?")
	args = append(args, limit)

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
