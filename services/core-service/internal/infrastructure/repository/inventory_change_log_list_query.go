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

// iclListJoins reach every joined table by primary key from the page's change-log rows.
const iclListJoins = ` JOIN item i ON i.id = icl.item_id` +
	` JOIN quantity q ON q.id = icl.quantity_id` +
	` JOIN unit u ON u.id = q.unit_id` +
	` LEFT JOIN scanning_station ss ON ss.id = icl.scanning_station_id` +
	" LEFT JOIN `user` usr ON usr.id = icl.responsible_user_id"

// buildICLListQuery assembles the inventory change-log listing SQL and its bind args. Only the
// predicates the caller supplied are emitted: an `(? = false OR ...)` guard is not sargable, and left the
// planner driving from quantity and reading millions of rows for a page.
//
// The page is chosen from inventory_change_log alone (keysetPage) and joined after; STRAIGHT_JOIN keeps
// the page first. Forward pages older (DESC), backward pages newer (ASC).
func buildICLListQuery(
	params domain.ListInventoryChangeLogsParams,
	dir pagination.Direction,
	cursorCreatedAt gosql.NullTime,
	cursorID gosql.NullString,
	limit int32,
) (string, []any) {
	page := keysetPage{
		table: "inventory_change_log", alias: "icl", sortColumn: "created_at",
		createdIndex: "inventory_change_log_account_created_idx",
		filters: []keysetFilter{
			{column: "icl.item_id", index: "inventory_change_log_account_id_item_id_created_at_id_idx", values: params.ItemIDs},
			{column: "icl.responsible_user_id", index: "inventory_change_log_acct_resp_user_created_idx", values: params.ChangedByUserIDs},
			{column: "icl.action_type_code", index: "inventory_change_log_acct_action_type_code_created_idx", values: params.ActionTypeCodes},
		},
		where: []string{"icl.account_id = ?"},
		args:  []any{params.AccountID},
		desc:  dir != pagination.DirectionBackward,
		limit: limit,
	}
	if params.StartDate != nil {
		page.where, page.args = append(page.where, "icl.created_at >= ?"), append(page.args, *params.StartDate)
	}
	if params.EndDate != nil {
		page.where, page.args = append(page.where, "icl.created_at <= ?"), append(page.args, *params.EndDate)
	}
	if cursorCreatedAt.Valid {
		if dir == pagination.DirectionBackward {
			page.where = append(page.where, "(icl.created_at > ? OR (icl.created_at = ? AND icl.id > ?))")
		} else {
			page.where = append(page.where, "(icl.created_at < ? OR (icl.created_at = ? AND icl.id < ?))")
		}
		page.args = append(page.args, cursorCreatedAt.Time, cursorCreatedAt.Time, cursorID.String)
	}

	orderBy := " ORDER BY icl.created_at DESC, icl.id DESC"
	if dir == pagination.DirectionBackward {
		orderBy = " ORDER BY icl.created_at ASC, icl.id ASC"
	}
	pageSQL, args := page.sql()
	return "SELECT STRAIGHT_JOIN " + iclListColumns + " FROM (" + pageSQL + ") page" +
		" JOIN inventory_change_log icl ON icl.id = page.id" + iclListJoins + orderBy, args
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
