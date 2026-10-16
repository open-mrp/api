package repository

import (
	"context"
	gosql "database/sql"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/actor"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/apikit/db/backfill"
)

// NoteColumnSource is one free-form notes column whose text becomes a note on its row's record, written by OpenMRP.
type NoteColumnSource struct {
	// Table holds the column; rows are walked by id.
	Table string
	// Column is the text copied.
	Column string
	// RecordType is the kind of record the note lands on.
	RecordType constants.RecordType
	// RecordExpr is the record the note lands on; t.id unless the row's record is elsewhere. NULL makes it account-wide.
	RecordExpr string
	// Join and Where narrow which rows carry a note worth keeping.
	Join, Where string
}

// BackfillName is the source's entry in backfill_progress.
func (s NoteColumnSource) BackfillName() string { return "note_column_" + s.Table }

// NoteColumnSources are the notes columns forge.1 removes from the API. A production step's note lands on the item it
// makes, which its manufacturing method replaces. A shipment's or invoice's note is kept only where it says something
// its order's note doesn't; the rest are copies of the order's instructions.
var NoteColumnSources = []NoteColumnSource{
	{Table: "department", Column: "notes", RecordType: constants.RecordTypeWorkCenter},
	{Table: "machine", Column: "notes", RecordType: constants.RecordTypeMachine},
	{Table: "scanning_station", Column: "notes", RecordType: constants.RecordTypeScanningStation},
	{Table: "item", Column: "notes", RecordType: constants.RecordTypeItem},
	{Table: "item_category", Column: "notes", RecordType: constants.RecordTypeItemCategory},
	{Table: "product_line", Column: "notes", RecordType: constants.RecordTypeProductLine},
	{
		Table: "production_step", Column: "notes", RecordType: constants.RecordTypeItem,
		RecordExpr: "(SELECT MIN(p.item_id) FROM production p WHERE p.production_step_id = t.id)",
	},
	{
		Table: "shipment", Column: "note", RecordType: constants.RecordTypeShipment,
		Join: " JOIN sales_order so ON so.id = t.sales_order_id", Where: " AND NOT (so.note <=> t.note)",
	},
	{
		Table: "invoice", Column: "note", RecordType: constants.RecordTypeInvoice,
		Join: " JOIN sales_order so ON so.id = t.sales_order_id", Where: " AND NOT (so.note <=> t.note)",
	},
}

// NoteColumnBackfill copies one source's notes into notes, walking its rows by id. It reads from read (a replica) and
// writes to write (the primary). Each note's ID derives from its source row, so a re-run batch inserts nothing twice.
type NoteColumnBackfill struct {
	read, write sqlc.DBTX
	src         NoteColumnSource
}

func NewNoteColumnBackfill(read, write *sqlc.Queries, src NoteColumnSource) *NoteColumnBackfill {
	return &NoteColumnBackfill{read: read.DB(), write: write.DB(), src: src}
}

// noteColumnNoteID is the note a source row's text becomes.
func noteColumnNoteID(table, rowID string) string {
	return id.DerivedID(id.NoteIDPrefix, table+":"+rowID)
}

// Batch is a backfill.Batch over the source table's ids.
func (b *NoteColumnBackfill) Batch(ctx context.Context, m *backfill.Meter, cursor string, limit int) (string, int, bool, error) {
	src := b.src
	var ids []string
	if err := m.Time(func() (err error) {
		ids, err = selectStrings(ctx, b.read, "SELECT id FROM "+src.Table+" WHERE id > ? ORDER BY id LIMIT ?", cursor, limit)
		return err
	}); err != nil {
		return cursor, 0, false, err
	}
	if len(ids) == 0 {
		return cursor, 0, true, nil
	}

	var notes []domain.ImportNote
	if err := m.Time(func() (err error) {
		notes, err = b.readNotes(ctx, ids)
		return err
	}); err != nil {
		return cursor, 0, false, err
	}
	if len(notes) > 0 {
		query, args := buildNoteImport(notes)
		if err := m.Time(func() error {
			_, err := b.write.ExecContext(ctx, query, args...)
			return err
		}); err != nil {
			return cursor, 0, false, err
		}
	}
	return ids[len(ids)-1], len(ids), len(ids) < limit, nil
}

func (b *NoteColumnBackfill) readNotes(ctx context.Context, ids []string) ([]domain.ImportNote, error) {
	src := b.src
	record := src.RecordExpr
	if record == "" {
		record = "t.id"
	}
	query := "SELECT t.id, t.account_id, " + record + ", t." + src.Column + ", t.updated_at FROM " + src.Table + " t" + src.Join +
		" WHERE t.id IN (" + placeholders(len(ids)) + ") AND t.account_id IS NOT NULL" +
		" AND t." + src.Column + " IS NOT NULL AND TRIM(t." + src.Column + ") <> ''" + src.Where
	rows, err := b.read.QueryContext(ctx, query, stringsToAny(ids)...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []domain.ImportNote
	for rows.Next() {
		var rowID, accountID, body string
		var recordID gosql.NullString
		var updatedAt time.Time
		if err := rows.Scan(&rowID, &accountID, &recordID, &body, &updatedAt); err != nil {
			return nil, err
		}
		n := domain.ImportNote{
			ID:         noteColumnNoteID(src.Table, rowID),
			AccountID:  accountID,
			Body:       body,
			Type:       constants.NoteTypeComment,
			Importance: constants.NoteImportanceNormal,
			Actor:      actor.System(),
			CreatedAt:  updatedAt,
			UpdatedAt:  updatedAt,
		}
		if recordID.Valid && recordID.String != "" {
			n.Record = &domain.NoteRecord{Type: src.RecordType, ID: recordID.String}
		}
		out = append(out, n)
	}
	return out, rows.Err()
}
