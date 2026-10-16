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

// Request to update a note.
type UpdateNoteRequest struct {
	// Note ID.
	ID string `path:"id" validate:"required"`
	// What the note says, as plain text. Maximum 10,000 characters. Changing it sets `edited_at`.
	Body field.Optional[string] `json:"body,omitzero" validate:"omitempty,max=10000"`
	// What kind of note this is.
	Type field.Optional[constants.NoteType] `json:"type,omitzero"`
	// How much the note matters.
	Importance field.Optional[constants.NoteImportance] `json:"importance,omitzero"`
	// When the note stops being used. Send `null` for a note that never expires.
	ExpiresAt field.Clearable[time.Time] `json:"expires_at,omitzero"`
	// Account users this edit @mentions, up to 25. Each is notified, and subscribed to the note's record when they
	// subscribe on mentions. Mentions aren't stored, so only the people you list here are notified.
	Mentions []string `json:"mentions,omitzero" validate:"omitempty,max=25"`
}

var sampleUpdateNoteRequest = &UpdateNoteRequest{
	Body:       field.Some("Confirm freight with the buyer before issuing any order over $5,000."),
	Importance: field.Some(constants.NoteImportanceNormal),
}

func (*UpdateNoteRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateNoteRequest)
}

// Updates a note.
//
// Only the fields you send change. Only the note's author can edit it.
type UpdateNoteEndpoint struct{}

func (e *UpdateNoteEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateNoteRequest, *apiresource.Note] {
	return (&apiendpoint.APIEndpoint[*UpdateNoteRequest, *apiresource.Note]{
		Title:             "Update Note",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             notesRoute + "/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeNote,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainNotes, Action: types.ActionUpdate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateNoteRequest) (*apiresource.Note, *apierror.APIError) {
			return svc.(NoteSvc).UpdateNote
		},
	})
}
