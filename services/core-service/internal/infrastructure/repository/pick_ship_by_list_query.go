package repository

import (
	gosql "database/sql"
	"strings"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/pagination"
)

// pickListColumns is the SELECT list for the ship-by pick listing, in the exact order
// scanPickShipByRows reads it — which is the field order of sqlc.ListPicksForwardRow, so the scan can
// reuse mapPickForwardRow and never drift from the sqlc-generated projection it mirrors.
const pickListColumns = `p.id, p.number, p.sales_order_id, so.number, ` +
	`ar.counterparty_account_id, ba.name, ar.external_number, ` +
	`so.priority_code, pr.id, pr.name, ` +
	`p.finished_at, p.created_at, p.updated_at, ` +
	`(SELECT COUNT(*) FROM pick_line plc WHERE plc.pick_id = p.id), ` +
	`(SELECT MAX(sh.shipped_at) FROM shipment sh WHERE sh.sales_order_id = so.id), ` +
	`so.promised_at, so.customer_po_number, so.note, ` +
	`so.carrier_id, cr.name, cr.is_portal_enabled, cr.created_at, cr.updated_at, ` +
	`so.carrier_option_id, co.name, co.is_portal_enabled, co.service_level_token, co.created_at, co.updated_at, ` +
	`so.carrier_billing_type, so.carrier_billing_account, ` +
	`so.ship_by_date, so.ship_by_cutoff_at, ` +
	`so.lead_time_days, so.lead_time_source_code, so.transit_days, so.transit_source_code, ` +
	`so.shipping_address_id, addr.name, addr.phone, addr.email, addr.is_drop_ship, ` +
	`ship_geo.id, ship_geo.street_line_1, ship_geo.street_line_2, ship_geo.locality, ship_geo.state, ` +
	`ship_geo.postal_code, ship_geo.country, addr.created_at, addr.updated_at`

// pickListFrom is the join graph, identical to ListPicksForward. Every joined table is reached by
// primary key or a unique key, so the joins are nested-loop lookups over whatever the driving index
// on `pick` yields.
const pickListFrom = ` FROM pick p` +
	` JOIN sales_order so ON so.id = p.sales_order_id` +
	` JOIN account_relation ar ON ar.owner_account_id = so.owner_account_id AND ar.counterparty_account_id = so.buyer_account_id` +
	` JOIN account ba ON ba.id = so.buyer_account_id` +
	` JOIN priority pr ON pr.code = so.priority_code` +
	` LEFT JOIN address addr ON addr.id = so.shipping_address_id` +
	` LEFT JOIN geolocation ship_geo ON ship_geo.id = addr.geolocation_id` +
	` LEFT JOIN carrier cr ON cr.id = so.carrier_id` +
	` LEFT JOIN carrier_option co ON co.id = so.carrier_option_id`

// buildPickShipByListQuery assembles the default (ship-by) pick listing and its bind args. It exists
// because the sqlc ListPicksForward/Backward serve this sort through a CASE-wrapped ORDER BY and a
// wall of `? IS NULL OR ...` / `? = false OR ...` filter guards. Neither is sargable: the CASE cannot
// be index-ordered and the guards make the optimizer abandon the composite, so the list filesorted
// every one of an account's picks and read ~1.3M rows to return 51 (10-16s, past the RPC deadline).
//
// Emitting only the predicates the caller actually set, over the denormalized p.ship_by_sort_date
// (COALESCE(so.ship_by_date, '9999-12-31'), sentinel sorts last), lets a bare ORDER BY land on
// pick_account_ship_by_idx (account_id, ship_by_sort_date, id): the scan reads in order and stops at
// LIMIT, and every other table is a primary-/unique-key lookup. STRAIGHT_JOIN pins `p` as the driver
// so the index supplies the sort directly rather than MySQL starting from a small joined table and
// filesorting; see inventory_change_log_list_query.go for the same pattern and its measured win.
//
// Direction matches the sqlc queries this path replaced: forward pages ascending (soonest ship-by
// first), backward descending, reversed by BuildPageString. The cursor predicate is emitted only when
// a cursor was supplied, so the first page is a clean range scan.
func buildPickShipByListQuery(
	accountID string,
	searchLike gosql.NullString,
	status *string,
	customerIDs []string,
	customerGroupIDs []string,
	productLineIDs []string,
	startDate gosql.NullTime,
	endDate gosql.NullTime,
	dir pagination.Direction,
	cursorShipBy gosql.NullTime,
	cursorID gosql.NullString,
	limit int32,
) (string, []any) {
	args := make([]any, 0, 8+len(customerIDs)+len(customerGroupIDs)+len(productLineIDs))

	var b strings.Builder
	b.WriteString("SELECT STRAIGHT_JOIN ")
	b.WriteString(pickListColumns)
	b.WriteString(pickListFrom)
	b.WriteString(" WHERE p.account_id = ?")
	args = append(args, accountID)

	// Short (< ngram token size) terms reach the list as a LIKE; longer terms take the ngram path and
	// never call this builder. The picker often has the customer in hand rather than a number, so the
	// box matches who the order is for as well as what it is.
	if searchLike.Valid {
		b.WriteString(" AND (p.number LIKE ? OR so.customer_po_number LIKE ? OR ba.name LIKE ? OR ar.external_number LIKE ?)")
		args = append(args, searchLike.String, searchLike.String, searchLike.String, searchLike.String)
	}
	if status != nil {
		if *status == "closed" {
			b.WriteString(" AND p.finished_at IS NOT NULL")
		} else {
			b.WriteString(" AND p.finished_at IS NULL")
		}
	}
	if len(customerIDs) > 0 {
		b.WriteString(" AND so.buyer_account_id IN (")
		b.WriteString(iclPlaceholders(len(customerIDs)))
		b.WriteString(")")
		for _, id := range customerIDs {
			args = append(args, id)
		}
	}
	if len(customerGroupIDs) > 0 {
		b.WriteString(" AND ar.account_group_id IN (")
		b.WriteString(iclPlaceholders(len(customerGroupIDs)))
		b.WriteString(")")
		for _, id := range customerGroupIDs {
			args = append(args, id)
		}
	}
	if len(productLineIDs) > 0 {
		b.WriteString(" AND EXISTS (SELECT 1 FROM pick_line pl2" +
			" JOIN sales_order_line sol2 ON sol2.id = pl2.sales_order_line_id" +
			" JOIN product prod ON prod.id = sol2.product_id" +
			" WHERE pl2.pick_id = p.id AND prod.product_line_id IN (")
		b.WriteString(iclPlaceholders(len(productLineIDs)))
		b.WriteString("))")
		for _, id := range productLineIDs {
			args = append(args, id)
		}
	}
	if startDate.Valid {
		b.WriteString(" AND p.created_at >= ?")
		args = append(args, startDate.Time)
	}
	if endDate.Valid {
		b.WriteString(" AND p.created_at <= ?")
		args = append(args, endDate.Time)
	}

	// Keyset over (ship_by_sort_date, id). CAST keeps the param a DATE while the column stays bare, so
	// the composite is still usable.
	if cursorShipBy.Valid {
		if dir == pagination.DirectionBackward {
			b.WriteString(" AND (p.ship_by_sort_date < CAST(? AS DATE) OR (p.ship_by_sort_date = CAST(? AS DATE) AND p.id < ?))")
		} else {
			b.WriteString(" AND (p.ship_by_sort_date > CAST(? AS DATE) OR (p.ship_by_sort_date = CAST(? AS DATE) AND p.id > ?))")
		}
		args = append(args, cursorShipBy.Time, cursorShipBy.Time, cursorID.String)
	}

	if dir == pagination.DirectionBackward {
		b.WriteString(" ORDER BY p.ship_by_sort_date DESC, p.id DESC")
	} else {
		b.WriteString(" ORDER BY p.ship_by_sort_date ASC, p.id ASC")
	}
	b.WriteString(" LIMIT ?")
	args = append(args, limit)

	return b.String(), args
}

// scanPickShipByRows reads rows from buildPickShipByListQuery into domain picks. It scans into the
// sqlc row type so the projection, the scan, and mapPickForwardRow share one source of truth for
// column order and types; pickListColumns must stay in sqlc.ListPicksForwardRow field order.
func scanPickShipByRows(rows *gosql.Rows) ([]*domain.Pick, error) {
	var out []*domain.Pick
	for rows.Next() {
		var row sqlc.ListPicksForwardRow
		if err := rows.Scan(
			&row.ID, &row.Number, &row.SalesOrderID, &row.SalesOrderNumber,
			&row.CustomerID, &row.CustomerName, &row.CustomerNumber,
			&row.PriorityCode, &row.PriorityID, &row.PriorityName,
			&row.FinishedAt, &row.CreatedAt, &row.UpdatedAt,
			&row.LineCount, &row.LastShippedAt,
			&row.PromisedAt, &row.CustomerPoNumber, &row.Note,
			&row.CarrierID, &row.CarrierName, &row.CarrierIsPortalEnabled, &row.CarrierCreatedAt, &row.CarrierUpdatedAt,
			&row.ServiceLevelID, &row.ServiceLevelName, &row.ServiceLevelIsPortalEnabled, &row.ServiceLevelToken, &row.ServiceLevelCreatedAt, &row.ServiceLevelUpdatedAt,
			&row.CarrierBillingType, &row.CarrierBillingAccount,
			&row.ShipByDate, &row.ShipByCutoffAt,
			&row.LeadTimeDays, &row.LeadTimeSourceCode, &row.TransitDays, &row.TransitSourceCode,
			&row.ShippingAddressID, &row.ShippingAddressName, &row.ShippingAddressPhone, &row.ShippingAddressEmail, &row.ShippingAddressIsDropShip,
			&row.ShippingAddressGeolocationID, &row.ShippingAddressStreetLine1, &row.ShippingAddressStreetLine2, &row.ShippingAddressLocality, &row.ShippingAddressState,
			&row.ShippingAddressPostalCode, &row.ShippingAddressCountry, &row.ShippingAddressCreatedAt, &row.ShippingAddressUpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, mapPickForwardRow(row))
	}
	return out, rows.Err()
}
