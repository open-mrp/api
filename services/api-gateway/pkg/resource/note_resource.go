package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

const SampleNoteID = "nt_q7m2kx9w4bja"

// A note on a record, or on your whole account.
//
// People, API keys and agents can leave notes. Agents read the notes on the records they work with, so a note is also how you teach an agent.
type Note struct {
	// Note ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=note"`
	// The record the note is on, such as a customer, an item or a sales order.
	//
	// `null` for a note on your whole account.
	Record *Record `json:"record"`
	// What the note says, as plain text.
	Body string `json:"body" validate:"required"`
	// What kind of note this is.
	//
	// - `comment`: a remark.
	// - `preference`: how someone likes things done.
	// - `fact`: a durable detail.
	// - `instruction`: standing guidance to follow.
	Type constants.NoteType `json:"type" validate:"required"`
	// How much the note matters. Agents read the most important notes first.
	Importance constants.NoteImportance `json:"importance" validate:"required"`
	// Whether the note is still in use: `expired` once `expires_at` has passed. Agents read only `active` notes.
	Status constants.NoteStatus `json:"status" validate:"required"`
	// When the note stops being used. `null` for a note that never expires.
	ExpiresAt *time.Time `json:"expires_at"`
	// Who wrote the note. Notes copied from older fields are written by `system`.
	Actor *ActorRef `json:"actor" validate:"required"`
	// When the body last changed. `null` until it is edited.
	EditedAt *time.Time `json:"edited_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last update timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleNote = &Note{
	ID:     SampleNoteID,
	Object: constants.ObjectTypeNote,
	Record: &Record{
		ID:       SampleCustomerID,
		Object:   constants.ObjectTypeRecord,
		Type:     constants.RecordTypeCustomer,
		Metadata: map[string]string{},
	},
	Body:       "Always confirm freight with the buyer before issuing an order.",
	Type:       constants.NoteTypeInstruction,
	Importance: constants.NoteImportanceHigh,
	Status:     constants.NoteStatusActive,
	ExpiresAt:  nil,
	Actor:      SampleActorRef,
	EditedAt:   nil,
	CreatedAt:  timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:  timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Note) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleNote)
}
