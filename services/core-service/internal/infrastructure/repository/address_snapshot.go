package repository

import (
	"context"
	gosql "database/sql"
	"fmt"
	"strings"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/apikit/db/backfill"
)

// A document stores each address snapshot in columns named with its role's prefix (bill_to, ship_to), all
// nullable; {prefix}_country is NULL only when the document has no such address:
//
//	`{prefix}_address_id` varchar(191)  -- the saved address it was copied from; NULL when given inline
//	`{prefix}_name` varchar(255)
//	`{prefix}_line_1`, `{prefix}_line_2`, `{prefix}_city`, `{prefix}_state`, `{prefix}_postal_code` varchar(255)
//	`{prefix}_country` char(2)
//	`{prefix}_phone`, `{prefix}_email` varchar(255)

// AddressSnapshotColumns is a snapshot as its columns hold it, for mapping sqlc params and rows.
type AddressSnapshotColumns struct {
	AddressID, Name, Line1, Line2, City, State, PostalCode, Country, Phone, Email gosql.NullString
}

// AddressSnapshotToColumns is s as its columns store it; nil is every column NULL.
func AddressSnapshotToColumns(s *domain.AddressSnapshot) AddressSnapshotColumns {
	if s == nil {
		return AddressSnapshotColumns{}
	}
	return AddressSnapshotColumns{
		AddressID:  toNullString(s.AddressID),
		Name:       gosql.NullString{String: s.Name, Valid: true},
		Line1:      toNullString(s.Line1),
		Line2:      toNullString(s.Line2),
		City:       toNullString(s.City),
		State:      toNullString(s.State),
		PostalCode: toNullString(s.PostalCode),
		Country:    gosql.NullString{String: s.Country, Valid: true},
		Phone:      toNullString(s.Phone),
		Email:      toNullString(s.Email),
	}
}

// Snapshot is the snapshot the columns hold, or nil when the document has none.
func (c AddressSnapshotColumns) Snapshot() *domain.AddressSnapshot {
	if !c.Country.Valid {
		return nil
	}
	return &domain.AddressSnapshot{
		AddressID:  nullStringPtr(c.AddressID),
		Name:       c.Name.String,
		Line1:      nullStringPtr(c.Line1),
		Line2:      nullStringPtr(c.Line2),
		City:       nullStringPtr(c.City),
		State:      nullStringPtr(c.State),
		PostalCode: nullStringPtr(c.PostalCode),
		Country:    c.Country.String,
		Phone:      nullStringPtr(c.Phone),
		Email:      nullStringPtr(c.Email),
	}
}

// addressSnapshotSources pairs each snapshot column suffix with what it copies from a saved address (a) and its
// geolocation (g).
var addressSnapshotSources = [][2]string{
	{"address_id", "a.id"},
	{"name", "a.name"},
	{"line_1", "g.street_line_1"},
	{"line_2", "g.street_line_2"},
	{"city", "g.locality"},
	{"state", "g.state"},
	{"postal_code", "g.postal_code"},
	{"country", "g.country"},
	{"phone", "a.phone"},
	{"email", "a.email"},
}

// AddressSnapshotBackfill is a backfill.Batch that fills one snapshot's columns on the rows written before the table
// had them, copying the saved address each row's source column references. Rows that already have a snapshot are
// left alone, so a batch can rerun safely.
type AddressSnapshotBackfill struct {
	read, write sqlc.DBTX
	table       string
	update      string
}

// NewAddressSnapshotBackfill backfills prefix's columns on table (keyed by `id`) from sourceColumn, e.g.
// ("sales_order", "billing_address_id", "bill_to"). Reads go to read, updates to write.
func NewAddressSnapshotBackfill(read, write *sqlc.Queries, table, sourceColumn, prefix string) (*AddressSnapshotBackfill, error) {
	for _, ident := range []string{table, sourceColumn, prefix} {
		if !identifierPattern.MatchString(ident) {
			return nil, fmt.Errorf("address snapshot backfill: invalid identifier %q", ident)
		}
	}
	sets := make([]string, len(addressSnapshotSources))
	for i, s := range addressSnapshotSources {
		sets[i] = fmt.Sprintf("d.`%s_%s` = %s", prefix, s[0], s[1])
	}
	update := fmt.Sprintf("UPDATE `%s` d JOIN address a ON a.id = d.`%s` JOIN geolocation g ON g.id = a.geolocation_id SET %s WHERE d.`%s_country` IS NULL AND d.id IN ",
		table, sourceColumn, strings.Join(sets, ", "), prefix)
	return &AddressSnapshotBackfill{read: read.DB(), write: write.DB(), table: table, update: update}, nil
}

// Batch is a backfill.Batch over the table's ids.
func (b *AddressSnapshotBackfill) Batch(ctx context.Context, m *backfill.Meter, cursor string, limit int) (string, int, bool, error) {
	var ids []string
	if err := m.Time(func() (err error) {
		ids, err = selectStrings(ctx, b.read, fmt.Sprintf("SELECT id FROM `%s` WHERE id > ? ORDER BY id LIMIT ?", b.table), cursor, limit)
		return err
	}); err != nil {
		return cursor, 0, false, err
	}
	if len(ids) == 0 {
		return cursor, 0, true, nil
	}
	if err := m.Time(func() error {
		_, err := b.write.ExecContext(ctx, b.update+"("+placeholders(len(ids))+")", stringsToAny(ids)...)
		return err
	}); err != nil {
		return cursor, 0, false, err
	}
	return ids[len(ids)-1], len(ids), len(ids) < limit, nil
}
