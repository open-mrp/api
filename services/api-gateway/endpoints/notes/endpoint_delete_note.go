package noteep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
)

// Request to delete a note.
type DeleteNoteRequest struct {
	// Note ID.
	ID string `path:"id" validate:"required"`
}

// Deletes a note.
//
// Only the note's author or an admin can delete it.
type DeleteNoteEndpoint struct{}

func (e *DeleteNoteEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteNoteRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteNoteRequest, *object.Deleted]{
		Title:             "Delete Note",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             notesRoute + "/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeNote,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainNotes, Action: types.ActionDelete}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteNoteRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(NoteSvc).DeleteNote, constants.ObjectTypeNote, "id")
		},
	})
}
