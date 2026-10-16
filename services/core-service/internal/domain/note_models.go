package domain

import (
	"time"

	"github.com/open-mrp/api/shared/actor"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/pagination"
)

// Note is attributed commentary on one record, or on the whole account when RecordID is nil. Agents recall notes, so a
// note is also how people teach them.
type Note struct {
	ID        string
	AccountID string
	// RecordType and RecordID name the record the note is on; both nil for an account-wide note.
	RecordType *constants.RecordType    `audit:"record_type"`
	RecordID   *string                  `audit:"record_id"`
	Body       string                   `audit:"body"`
	Type       constants.NoteType       `audit:"type"`
	Importance constants.NoteImportance `audit:"importance"`
	ExpiresAt  *time.Time               `audit:"expires_at"`
	Actor      actor.Actor
	// EditedAt is when the body last changed, nil until it does.
	EditedAt  *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Status is expired once ExpiresAt has passed.
func (n *Note) Status(now time.Time) constants.NoteStatus {
	if n.ExpiresAt != nil && !n.ExpiresAt.After(now) {
		return constants.NoteStatusExpired
	}
	return constants.NoteStatusActive
}

// NoteRecord names the record a note is on.
type NoteRecord struct {
	Type constants.RecordType
	ID   string
}

type ListNotesParams struct {
	AccountID   string
	Cursor      *string
	Limit       int32
	RecordIDs   []string
	RecordTypes []constants.RecordType
	Types       []constants.NoteType
	ActorTypes  []constants.ActorType
	Statuses    []constants.NoteStatus
	// Query matches the body by substring.
	Query *string
}

type ListNotesResult struct {
	Notes    []*Note
	PageInfo pagination.PageInfo
}

type CreateNoteParams struct {
	// Record is nil for an account-wide note.
	Record     *NoteRecord
	Body       string
	Type       *constants.NoteType
	Importance *constants.NoteImportance
	ExpiresAt  *time.Time
	// Mentions are account users the note @mentions: each is notified and subscribed to the record.
	Mentions []string
}

// UpdateNoteParams changes only what is set. Record changes serve the preview.9 memory routes, whose entity fields are
// writable; the note API leaves the record fixed.
type UpdateNoteParams struct {
	NoteID         string
	Body           *string
	Type           *constants.NoteType
	Importance     *constants.NoteImportance
	ExpiresAt      *time.Time
	ClearExpiresAt bool
	Record         *NoteRecord
	ClearRecord    bool
	// Mentions are account users this edit @mentions: each is notified and subscribed to the record.
	Mentions []string
}

// ImportNote is a note copied from elsewhere (agent memories) with its own ID, author and timestamps.
type ImportNote struct {
	ID         string
	AccountID  string
	Record     *NoteRecord
	Body       string
	Type       constants.NoteType
	Importance constants.NoteImportance
	ExpiresAt  *time.Time
	Actor      actor.Actor
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// RecallNotesParams asks for the notes an agent starts from: account-wide guidance plus the notes on RecordIDs.
type RecallNotesParams struct {
	AccountID string
	RecordIDs []string
	Limit     int32
}
