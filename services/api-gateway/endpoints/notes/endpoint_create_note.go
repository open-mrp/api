package noteep

import (
	"context"
	"net/http"
	"time"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

// Request to create a note.
type CreateNoteRequest struct {
	// The record the note is on, such as a customer, an item or a sales order. Omit it for a note on your whole account.
	Record field.Optional[apiresource.RecordInput] `json:"record,omitzero"`
	// What the note says, as plain text. Maximum 10,000 characters.
	Body string `json:"body" validate:"required,max=10000"`
	// What kind of note this is. Defaults to `comment`.
	//
	// - `comment`: a remark.
	// - `preference`: how someone likes things done.
	// - `fact`: a durable detail.
	// - `instruction`: standing guidance to follow.
	Type field.Optional[constants.NoteType] `json:"type,omitzero"`
	// How much the note matters. Agents read the most important notes first. Defaults to `normal`.
	Importance field.Optional[constants.NoteImportance] `json:"importance,omitzero"`
	// When the note stops being used. Omit it for a note that never expires.
	ExpiresAt field.Optional[time.Time] `json:"expires_at,omitzero"`
	// Account users the note @mentions, up to 25. Each is notified, and subscribed to the note's record when they
	// subscribe on mentions.
	Mentions []string `json:"mentions,omitzero" validate:"omitempty,max=25"`
}

var sampleCreateNoteRequest = &CreateNoteRequest{
	Record:     field.Some(apiresource.RecordInput{Type: constants.RecordTypeCustomer, ID: apiresource.SampleCustomerID}),
	Body:       "Always confirm freight with the buyer before issuing an order.",
	Type:       field.Some(constants.NoteTypeInstruction),
	Importance: field.Some(constants.NoteImportanceHigh),
}

func (*CreateNoteRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateNoteRequest)
}

// Creates a note on a record, or on your whole account.
//
// You are the note's author: only you can edit it, and only you or an admin can delete it.
type CreateNoteEndpoint struct{}

func (e *CreateNoteEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateNoteRequest, *apiresource.Note] {
	return (&apiendpoint.APIEndpoint[*CreateNoteRequest, *apiresource.Note]{
		Title:             "Create Note",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             notesRoute,
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeNote,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainNotes, Action: types.ActionCreate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateNoteRequest) (*apiresource.Note, *apierror.APIError) {
			return svc.(NoteSvc).CreateNote
		},
		LocationFunc: func(resp *apiresource.Note) string {
			return notesRoute + "/" + resp.ID
		},
	})
}
