package repository

import (
	"cmp"
	"context"
	gosql "database/sql"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/actor"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/pagination"
	"github.com/open-mrp/apikit/tracing"
)

var noteRepoTracer = tracing.GetTracer("core-service.note_repository")

type noteRepoImpl struct {
	queries *sqlc.Queries
}

func NewNoteRepo(queries *sqlc.Queries) domain.NoteRepo {
	return &noteRepoImpl{queries: queries}
}

func noteCreatedAt(n *domain.Note) time.Time { return n.CreatedAt }
func noteID(n *domain.Note) string           { return n.ID }

// noteRow is one note as stored.
type noteRow struct {
	ID, AccountID        string
	RecordType, RecordID gosql.NullString
	Body, Type           string
	Importance           int8
	ExpiresAt            gosql.NullTime
	ActorType            string
	ActorID              gosql.NullString
	EditedAt             gosql.NullTime
	CreatedAt, UpdatedAt time.Time
}

func (r noteRow) toDomain() (*domain.Note, error) {
	a, err := actor.FromColumns(r.ActorType, r.ActorID)
	if err != nil {
		return nil, fmt.Errorf("note %s: %w", r.ID, err)
	}
	n := &domain.Note{
		ID:         r.ID,
		AccountID:  r.AccountID,
		Body:       r.Body,
		Type:       constants.NoteType(r.Type),
		Importance: constants.NoteImportanceFromRank(int32(r.Importance)),
		Actor:      a,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}
	if r.RecordID.Valid {
		t := constants.RecordType(r.RecordType.String)
		n.RecordType, n.RecordID = &t, &r.RecordID.String
	}
	if r.ExpiresAt.Valid {
		n.ExpiresAt = &r.ExpiresAt.Time
	}
	if r.EditedAt.Valid {
		n.EditedAt = &r.EditedAt.Time
	}
	return n, nil
}

func noteFromSQLC(n sqlc.Note) (*domain.Note, *apierror.APIError) {
	out, err := noteRow(n).toDomain()
	if err != nil {
		return nil, apierror.NewInternalError(err, "Issue reading a note.")
	}
	return out, nil
}

func notesFromSQLC(rows []sqlc.Note) ([]*domain.Note, *apierror.APIError) {
	out := make([]*domain.Note, 0, len(rows))
	for _, r := range rows {
		n, apiErr := noteFromSQLC(r)
		if apiErr != nil {
			return nil, apiErr
		}
		out = append(out, n)
	}
	return out, nil
}

func nullRecordColumns(t *constants.RecordType, recordID *string) (gosql.NullString, gosql.NullString) {
	if t == nil || recordID == nil {
		return gosql.NullString{}, gosql.NullString{}
	}
	return gosql.NullString{String: string(*t), Valid: true}, gosql.NullString{String: *recordID, Valid: true}
}

func nullTime(t *time.Time) gosql.NullTime {
	if t == nil {
		return gosql.NullTime{}
	}
	return gosql.NullTime{Time: *t, Valid: true}
}

func (r *noteRepoImpl) List(ctx context.Context, params domain.ListNotesParams) (*domain.ListNotesResult, *apierror.APIError) {
	ctx, span := noteRepoTracer.Start(ctx, "repository.note.list")
	defer span.End()

	dir := pagination.DirectionForward
	var cursor *pagination.StringCursor
	var cursorDir *pagination.Direction
	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewParameterInvalidError("cursor", "Invalid pagination cursor.")
		}
		cursor, cursorDir, dir = &cur, &cur.Direction, cur.Direction
	}

	query, args := buildNoteListQuery(buildNoteListPage(params, dir, cursor, params.Limit+1), dir)
	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	defer func() { _ = rows.Close() }()
	notes, err := scanNoteRows(rows)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	result, pageInfo := pagination.BuildPageString(notes, params.Limit, cursorDir, noteCreatedAt, noteID)
	return &domain.ListNotesResult{Notes: result, PageInfo: pageInfo}, nil
}

func (r *noteRepoImpl) Get(ctx context.Context, accountID, id string) (*domain.Note, *apierror.APIError) {
	ctx, span := noteRepoTracer.Start(ctx, "repository.note.get")
	defer span.End()

	row, err := r.queries.GetNote(ctx, sqlc.GetNoteParams{ID: id, AccountID: accountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	n, apiErr := noteFromSQLC(row)
	return n, tracing.Trace(span, apiErr)
}

func (r *noteRepoImpl) GetByIDs(ctx context.Context, accountID string, ids []string) ([]*domain.Note, *apierror.APIError) {
	ctx, span := noteRepoTracer.Start(ctx, "repository.note.get_by_ids")
	defer span.End()

	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.queries.GetNotesByIDs(ctx, sqlc.GetNotesByIDsParams{AccountID: accountID, Ids: ids})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	notes, apiErr := notesFromSQLC(rows)
	return notes, tracing.Trace(span, apiErr)
}

func (r *noteRepoImpl) Create(ctx context.Context, n *domain.Note) *apierror.APIError {
	ctx, span := noteRepoTracer.Start(ctx, "repository.note.create")
	defer span.End()

	recordType, recordID := nullRecordColumns(n.RecordType, n.RecordID)
	actorType, actorID := n.Actor.Columns()
	err := r.queries.InsertNote(ctx, sqlc.InsertNoteParams{
		ID:         n.ID,
		AccountID:  n.AccountID,
		RecordType: recordType,
		RecordID:   recordID,
		Body:       n.Body,
		Type:       string(n.Type),
		Importance: int8(n.Importance.Rank()),
		ExpiresAt:  nullTime(n.ExpiresAt),
		ActorType:  actorType,
		ActorID:    actorID,
	})
	return tracing.Trace(span, db.MapSQLError(err))
}

func (r *noteRepoImpl) Update(ctx context.Context, n *domain.Note) *apierror.APIError {
	ctx, span := noteRepoTracer.Start(ctx, "repository.note.update")
	defer span.End()

	recordType, recordID := nullRecordColumns(n.RecordType, n.RecordID)
	err := r.queries.UpdateNote(ctx, sqlc.UpdateNoteParams{
		ID:         n.ID,
		AccountID:  n.AccountID,
		Body:       n.Body,
		Type:       string(n.Type),
		Importance: int8(n.Importance.Rank()),
		ExpiresAt:  nullTime(n.ExpiresAt),
		RecordType: recordType,
		RecordID:   recordID,
		EditedAt:   nullTime(n.EditedAt),
	})
	return tracing.Trace(span, db.MapSQLError(err))
}

func (r *noteRepoImpl) Delete(ctx context.Context, accountID, id string) *apierror.APIError {
	ctx, span := noteRepoTracer.Start(ctx, "repository.note.delete")
	defer span.End()

	return tracing.Trace(span, db.MapSQLError(r.queries.DeleteNote(ctx, sqlc.DeleteNoteParams{ID: id, AccountID: accountID})))
}

// Import inserts the notes in one statement by primary key; INSERT IGNORE skips IDs already present, so repeating a
// batch after a lost progress save writes nothing twice.
func (r *noteRepoImpl) Import(ctx context.Context, notes []domain.ImportNote) (int64, *apierror.APIError) {
	ctx, span := noteRepoTracer.Start(ctx, "repository.note.import")
	defer span.End()

	if len(notes) == 0 {
		return 0, nil
	}
	query, args := buildNoteImport(notes)
	res, err := r.queries.DB().ExecContext(ctx, query, args...)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, tracing.Trace(span, apierror.NewInternalError(err, "Issue importing notes."))
	}
	return n, nil
}

func buildNoteImport(notes []domain.ImportNote) (string, []any) {
	var b strings.Builder
	b.WriteString("INSERT IGNORE INTO note (id, account_id, record_type, record_id, body, type, importance, expires_at, actor_type, actor_id, created_at, updated_at) VALUES ")
	args := make([]any, 0, len(notes)*12)
	for i, n := range notes {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString("(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		var recordType, recordID gosql.NullString
		if n.Record != nil {
			recordType = gosql.NullString{String: string(n.Record.Type), Valid: true}
			recordID = gosql.NullString{String: n.Record.ID, Valid: true}
		}
		actorType, actorID := n.Actor.Columns()
		args = append(args, n.ID, n.AccountID, recordType, recordID, n.Body, string(n.Type), n.Importance.Rank(),
			nullTime(n.ExpiresAt), actorType, actorID, n.CreatedAt, n.UpdatedAt)
	}
	return b.String(), args
}

// Recall reads account-wide guidance and the notes on the given records, then keeps the most important across both.
func (r *noteRepoImpl) Recall(ctx context.Context, params domain.RecallNotesParams) ([]*domain.Note, *apierror.APIError) {
	ctx, span := noteRepoTracer.Start(ctx, "repository.note.recall")
	defer span.End()

	rows, err := r.queries.RecallAccountNotes(ctx, sqlc.RecallAccountNotesParams{AccountID: params.AccountID, Limit: params.Limit})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if len(params.RecordIDs) > 0 {
		recordRows, err := r.queries.RecallRecordNotes(ctx, sqlc.RecallRecordNotesParams{
			AccountID: params.AccountID,
			RecordIds: nullStrings(params.RecordIDs),
			Limit:     params.Limit,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		rows = append(rows, recordRows...)
	}
	notes, apiErr := notesFromSQLC(rows)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return rankRecalled(notes, int(params.Limit)), nil
}

// rankRecalled orders recalled notes most important first, newest first within an importance, and keeps limit.
func rankRecalled(notes []*domain.Note, limit int) []*domain.Note {
	slices.SortStableFunc(notes, func(a, b *domain.Note) int {
		if c := cmp.Compare(b.Importance.Rank(), a.Importance.Rank()); c != 0 {
			return c
		}
		if c := b.CreatedAt.Compare(a.CreatedAt); c != 0 {
			return c
		}
		return cmp.Compare(b.ID, a.ID)
	})
	if limit > 0 && len(notes) > limit {
		notes = notes[:limit]
	}
	return notes
}

func nullStrings(values []string) []gosql.NullString {
	out := make([]gosql.NullString, len(values))
	for i, v := range values {
		out[i] = gosql.NullString{String: v, Valid: true}
	}
	return out
}
