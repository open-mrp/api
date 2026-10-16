package service

import (
	"context"
	"time"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
	"github.com/open-mrp/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/apikit/db/backfill"
)

// MemoryNoteBackfillName is the memory copy's entry in backfill_progress.
const MemoryNoteBackfillName = "agent_memory_notes"

// MemoryNoteBackfill copies agent memories into core-service notes once, walking memories by ID. Each note's ID derives
// from its memory's, and core skips IDs it already has, so a re-run batch copies nothing twice.
type MemoryNoteBackfill struct {
	page func(ctx context.Context, afterID string, limit int) ([]sqlc.ListAgentMemoriesAfterRow, error)
	core domain.CoreClient
}

func NewMemoryNoteBackfill(page func(ctx context.Context, afterID string, limit int) ([]sqlc.ListAgentMemoriesAfterRow, error), core domain.CoreClient) *MemoryNoteBackfill {
	return &MemoryNoteBackfill{page: page, core: core}
}

// Batch is a backfill.Batch over agent_memory ids.
func (b *MemoryNoteBackfill) Batch(ctx context.Context, m *backfill.Meter, cursor string, limit int) (string, int, bool, error) {
	var rows []sqlc.ListAgentMemoriesAfterRow
	if err := m.Time(func() (err error) {
		rows, err = b.page(ctx, cursor, limit)
		return err
	}); err != nil {
		return cursor, 0, false, err
	}
	if len(rows) == 0 {
		return cursor, 0, true, nil
	}
	notes := make([]domain.ImportedNote, len(rows))
	for i, r := range rows {
		notes[i] = memoryAsNote(r)
	}
	if err := m.Time(func() error {
		_, err := b.core.ImportNotes(ctx, notes)
		return err
	}); err != nil {
		return cursor, 0, false, err
	}
	return rows[len(rows)-1].ID, len(rows), len(rows) < limit, nil
}

// memoryAsNote is the note a memory becomes: its category is the note type, its 0–1 importance one of three levels, and
// its entity the record. A memory on the account itself, or on an entity no record type names, is account-wide.
// Memories have no author, so OpenMRP writes the note.
func memoryAsNote(r sqlc.ListAgentMemoriesAfterRow) domain.ImportedNote {
	n := domain.ImportedNote{
		ID:         id.DerivedID(id.NoteIDPrefix, "agent_memory:"+r.ID),
		AccountID:  r.AccountID,
		Body:       r.Content,
		Type:       string(constants.NoteTypeFact),
		Importance: string(constants.NoteImportanceFromScore(r.Importance)),
		CreatedAt:  r.CreatedAt.Time,
		UpdatedAt:  r.UpdatedAt.Time,
	}
	if t := constants.NoteType(r.Category); t.IsValid() {
		n.Type = string(t)
	}
	if r.ExpiresAt.Valid {
		t := r.ExpiresAt.Time
		n.ExpiresAt = &t
	}
	if r.EntityType.Valid && r.EntityID.Valid && r.EntityID.String != "" {
		if t, ok := memoryEntityRecordType(r.EntityType.String); ok {
			n.RecordType, n.RecordID = string(t), r.EntityID.String
		}
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now().UTC()
	}
	if n.UpdatedAt.IsZero() {
		n.UpdatedAt = n.CreatedAt
	}
	return n
}

func memoryEntityRecordType(entityType string) (constants.RecordType, bool) {
	switch entityType {
	case "account", "":
		return "", false
	case "product", "material", "part":
		return constants.RecordTypeItem, true
	}
	t := constants.RecordType(entityType)
	return t, t.IsValid()
}
