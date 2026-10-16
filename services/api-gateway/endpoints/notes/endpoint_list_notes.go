package noteep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list notes.
type ListNotesRequest struct {
	object.ListRequest
	// Case-insensitive substring match on `body`.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Only return notes on one of these records.
	RecordIDs []string `query:"record_ids" validate:"omitempty,max=100"`
	// Only return notes on records of one of these types, such as `customer` or `sales_order`.
	RecordTypes []constants.RecordType `query:"record_types"`
	// Only return notes of one of these types.
	Types []constants.NoteType `query:"types"`
	// Only return notes written by one of these kinds of actor.
	ActorTypes []constants.ActorType `query:"actor_types"`
	// Only return notes with one of these statuses. Omit to return all statuses.
	Statuses []constants.NoteStatus `query:"statuses"`
}

// Returns a list of your notes, newest first.
//
// Filter by `record_ids[]` for the notes on a record. Notes on your whole account have no record.
type ListNotesEndpoint struct{}

func (e *ListNotesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListNotesRequest, *apiresource.List[apiresource.Note]] {
	return (&apiendpoint.APIEndpoint[*ListNotesRequest, *apiresource.List[apiresource.Note]]{
		Title:             "List Notes",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             notesRoute,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeNote,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainNotes, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListNotesRequest) (*apiresource.List[apiresource.Note], *apierror.APIError) {
			return svc.(NoteSvc).ListNotes
		},
	})
}
