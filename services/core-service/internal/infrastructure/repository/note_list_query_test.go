package repository

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/actor"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/pagination"
)

func TestNoteListQuery_EmitsOnlySuppliedFilters(t *testing.T) {
	query, args := buildNoteListQuery(buildNoteListPage(domain.ListNotesParams{AccountID: "ac_1"}, pagination.DirectionForward, nil, 26), pagination.DirectionForward)
	assert.Contains(t, query, "FORCE INDEX (note_account_created_idx)")
	assert.NotContains(t, query, "n.expires_at <")
	assert.NotContains(t, query, "n.expires_at IS NULL")
	assert.NotContains(t, query, "LIKE")
	assert.Equal(t, []any{"ac_1", int32(26)}, args)
	assert.Equal(t, strings.Count(query, "?"), len(args))
}

func TestNoteListQuery_FiltersAndCursor(t *testing.T) {
	q := "50%_off"
	at := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	cursor := &pagination.StringCursor{OccurredAt: at, ID: "nt_9", Direction: pagination.DirectionBackward}
	query, args := buildNoteListQuery(buildNoteListPage(domain.ListNotesParams{
		AccountID:  "ac_1",
		RecordIDs:  []string{"ac_c"},
		Types:      []constants.NoteType{constants.NoteTypeFact, constants.NoteTypeInstruction},
		ActorTypes: []constants.ActorType{constants.ActorTypeAgent},
		Statuses:   []constants.NoteStatus{constants.NoteStatusExpired},
		Query:      &q,
	}, pagination.DirectionBackward, cursor, 11), pagination.DirectionBackward)

	assert.Contains(t, query, "FORCE INDEX (note_account_record_created_idx)", "a record's notes ride its key")
	assert.Contains(t, query, "n.record_id IN (?)")
	assert.Contains(t, query, "n.type IN (?,?)")
	assert.Contains(t, query, "n.actor_type IN (?)")
	assert.Contains(t, query, "n.expires_at <= NOW(3)")
	assert.Contains(t, query, "n.body LIKE ?")
	assert.Contains(t, query, "(n.created_at > ? OR (n.created_at = ? AND n.id > ?))")
	assert.Contains(t, query, "ORDER BY n.created_at ASC, n.id ASC")
	assert.Contains(t, args, `%50\%\_off%`, "a wildcard in the search is matched literally")
	assert.Equal(t, strings.Count(query, "?"), len(args))
}

func TestNoteListQuery_BothStatusesFilterNothing(t *testing.T) {
	query, _ := buildNoteListQuery(buildNoteListPage(domain.ListNotesParams{
		AccountID: "ac_1",
		Statuses:  []constants.NoteStatus{constants.NoteStatusActive, constants.NoteStatusExpired},
	}, pagination.DirectionForward, nil, 26), pagination.DirectionForward)
	assert.NotContains(t, query, "n.expires_at <")
	assert.NotContains(t, query, "n.expires_at IS NULL")
}

func TestRankRecalled_MostImportantThenNewest(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	n := func(id string, i constants.NoteImportance, at time.Time) *domain.Note {
		return &domain.Note{ID: id, Importance: i, CreatedAt: at}
	}
	got := rankRecalled([]*domain.Note{
		n("a", constants.NoteImportanceLow, t0.Add(3*time.Hour)),
		n("b", constants.NoteImportanceHigh, t0),
		n("c", constants.NoteImportanceNormal, t0.Add(time.Hour)),
		n("d", constants.NoteImportanceHigh, t0.Add(2*time.Hour)),
	}, 3)
	ids := make([]string, len(got))
	for i, g := range got {
		ids[i] = g.ID
	}
	assert.Equal(t, []string{"d", "b", "c"}, ids)
}

func TestBuildNoteImport_IsRepeatable(t *testing.T) {
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	query, args := buildNoteImport([]domain.ImportNote{
		{ID: "nt_1", AccountID: "ac_1", Body: "x", Type: constants.NoteTypeFact, Importance: constants.NoteImportanceHigh, Actor: actor.System(), CreatedAt: at, UpdatedAt: at},
		{ID: "nt_2", AccountID: "ac_1", Record: &domain.NoteRecord{Type: constants.RecordTypeItem, ID: "it_1"}, Body: "y", Type: constants.NoteTypeComment, Importance: constants.NoteImportanceLow, Actor: actor.System(), CreatedAt: at, UpdatedAt: at},
	})
	assert.True(t, strings.HasPrefix(query, "INSERT IGNORE INTO note "), "an ID already copied is skipped")
	assert.Equal(t, strings.Count(query, "?"), len(args))
	assert.Equal(t, int32(3), args[6], "importance is stored as its rank")
}

func TestNoteColumnNoteID_IsStablePerRow(t *testing.T) {
	assert.Equal(t, noteColumnNoteID("item", "it_1"), noteColumnNoteID("item", "it_1"))
	assert.NotEqual(t, noteColumnNoteID("item", "it_1"), noteColumnNoteID("machine", "it_1"), "rows of different tables never share a note")
	assert.True(t, strings.HasPrefix(noteColumnNoteID("item", "it_1"), "nt_"))
}
