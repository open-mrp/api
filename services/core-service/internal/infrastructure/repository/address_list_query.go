package repository

import (
	gosql "database/sql"
	"strings"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/apikit/pagination"
)

// addressListColumns is the SELECT list scanAddressListRows reads, in order.
const addressListColumns = `a.id, a.name, a.phone, a.email, a.is_drop_ship, a.receive_calendar_id, a.created_at, a.updated_at, ` +
	`g.id, g.street_line_1, g.street_line_2, g.locality, g.state, g.postal_code, g.country, g.google_place_id, g.latitude, g.longitude, g.timezone`

// addressSearchColumns are the columns `q` matches by substring.
var addressSearchColumns = []string{"a.name", "g.street_line_1", "g.street_line_2", "g.locality", "g.state", "g.postal_code", "g.country"}

// buildAddressListQuery is one page of an account's addresses. Addresses carry no account, so no key yields one
// account's in list order: the page is chosen from the account's links (STRAIGHT_JOIN, so neither address nor
// geolocation, which hold every tenant's rows, is scanned) and its rows joined after. Only the filters the caller gave
// are emitted, so an absent one costs nothing. Forward pages older (DESC), backward pages newer (ASC).
func buildAddressListQuery(params domain.ListAddressesParams, dir pagination.Direction, cursor *pagination.StringCursor, limit int32) (string, []any) {
	where := []string{"aa.account_id = ?"}
	args := []any{params.AccountID}
	if search := buildAddressSearchParams(params.Query); search.Valid {
		likes := make([]string, len(addressSearchColumns))
		for i, c := range addressSearchColumns {
			likes[i] = c + " LIKE ?"
			args = append(args, search.String)
		}
		where = append(where, "("+strings.Join(likes, " OR ")+")")
	}
	if params.DropShip != nil {
		where, args = append(where, "a.is_drop_ship = ?"), append(args, *params.DropShip)
	}
	order := "DESC"
	if dir == pagination.DirectionBackward {
		order = "ASC"
	}
	if cursor != nil {
		cmp := "<"
		if dir == pagination.DirectionBackward {
			cmp = ">"
		}
		where = append(where, "(a.created_at "+cmp+" ? OR (a.created_at = ? AND a.id "+cmp+" ?))")
		args = append(args, cursor.OccurredAt, cursor.OccurredAt, cursor.ID)
	}
	args = append(args, limit)
	orderBy := " ORDER BY a.created_at " + order + ", a.id " + order
	return "SELECT " + addressListColumns + " FROM (" +
		"SELECT STRAIGHT_JOIN a.id FROM account_address aa" +
		" JOIN address a ON a.id = aa.address_id" +
		" JOIN geolocation g ON g.id = a.geolocation_id" +
		" WHERE " + strings.Join(where, " AND ") + orderBy + " LIMIT ?" +
		") page JOIN address a ON a.id = page.id JOIN geolocation g ON g.id = a.geolocation_id" + orderBy, args
}

// scanAddressListRows reads rows of buildAddressListQuery.
func scanAddressListRows(rows *gosql.Rows) ([]*domain.Address, error) {
	var out []*domain.Address
	for rows.Next() {
		var (
			id, name, geoID, country           string
			phone, email, receiveCalendarID    gosql.NullString
			isDropShip                         bool
			createdAt, updatedAt               gosql.NullTime
			line1, line2, locality, state, zip gosql.NullString
			placeID, timezone                  gosql.NullString
			latitude, longitude                gosql.NullFloat64
		)
		if err := rows.Scan(&id, &name, &phone, &email, &isDropShip, &receiveCalendarID, &createdAt, &updatedAt,
			&geoID, &line1, &line2, &locality, &state, &zip, &country, &placeID, &latitude, &longitude, &timezone); err != nil {
			return nil, err
		}
		out = append(out, mapAddressRow(id, name, phone, email, isDropShip, createdAt.Time, updatedAt.Time,
			geoID, line1, line2, locality, state, zip, country, placeID, latitude, longitude, timezone, receiveCalendarID))
	}
	return out, rows.Err()
}
