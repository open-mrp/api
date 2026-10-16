package repository

import (
	gosql "database/sql"
	"strings"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/pagination"
)

// noteColumns is the SELECT list scanNoteRows reads, in order.
const noteColumns = `n.id, n.account_id, n.record_type, n.record_id, n.body, n.type, n.importance, n.expires_at, ` +
	`n.actor_type, n.actor_id, n.edited_at, n.created_at, n.updated_at`

// buildNoteListPage chooses one page of an account's notes from note alone. record_ids[] rides its own key; every other
// filter is residual. Only the filters the caller gave are emitted. Forward pages older (DESC), backward pages newer.
func buildNoteListPage(params domain.ListNotesParams, dir pagination.Direction, cursor *pagination.StringCursor, limit int32) keysetPage {
	page := keysetPage{
		table: "note", alias: "n", sortColumn: "created_at",
		createdIndex: "note_account_created_idx",
		filters: []keysetFilter{
			{column: "n.record_id", index: "note_account_record_created_idx", values: params.RecordIDs},
		},
		where: []string{"n.account_id = ?"},
		args:  []any{params.AccountID},
		desc:  dir != pagination.DirectionBackward,
		limit: limit,
	}
	in := func(column string, values []string) {
		if len(values) == 0 {
			return
		}
		page.where = append(page.where, column+" IN ("+placeholders(len(values))+")")
		page.args = append(page.args, stringArgs(values)...)
	}
	in("n.record_type", enumStrings(params.RecordTypes))
	in("n.type", enumStrings(params.Types))
	in("n.actor_type", enumStrings(params.ActorTypes))

	active, expired := false, false
	for _, s := range params.Statuses {
		switch s {
		case constants.NoteStatusActive:
			active = true
		case constants.NoteStatusExpired:
			expired = true
		}
	}
	switch {
	case active && !expired:
		page.where = append(page.where, "(n.expires_at IS NULL OR n.expires_at > NOW(3))")
	case expired && !active:
		page.where = append(page.where, "n.expires_at <= NOW(3)")
	}

	if params.Query != nil && strings.TrimSpace(*params.Query) != "" {
		page.where = append(page.where, "n.body LIKE ?")
		page.args = append(page.args, "%"+db.EscapeLike(strings.TrimSpace(*params.Query))+"%")
	}
	if cursor != nil {
		cmp := "<"
		if dir == pagination.DirectionBackward {
			cmp = ">"
		}
		page.where = append(page.where, "(n.created_at "+cmp+" ? OR (n.created_at = ? AND n.id "+cmp+" ?))")
		page.args = append(page.args, cursor.OccurredAt, cursor.OccurredAt, cursor.ID)
	}
	return page
}

// buildNoteListQuery wraps a page from buildNoteListPage in the note columns, in list order.
func buildNoteListQuery(page keysetPage, dir pagination.Direction) (string, []any) {
	orderBy := " ORDER BY n.created_at DESC, n.id DESC"
	if dir == pagination.DirectionBackward {
		orderBy = " ORDER BY n.created_at ASC, n.id ASC"
	}
	pageSQL, args := page.sql()
	return "SELECT " + noteColumns + " FROM (" + pageSQL + ") page JOIN note n ON n.id = page.id" + orderBy, args
}

func enumStrings[T ~string](values []T) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = string(v)
	}
	return out
}

// scanNoteRows reads rows selecting noteColumns.
func scanNoteRows(rows *gosql.Rows) ([]*domain.Note, error) {
	var out []*domain.Note
	for rows.Next() {
		var r noteRow
		if err := rows.Scan(&r.ID, &r.AccountID, &r.RecordType, &r.RecordID, &r.Body, &r.Type, &r.Importance, &r.ExpiresAt,
			&r.ActorType, &r.ActorID, &r.EditedAt, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		n, err := r.toDomain()
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}
