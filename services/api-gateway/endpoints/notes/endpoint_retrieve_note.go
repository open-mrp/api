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
)

// Request to retrieve a note.
type RetrieveNoteRequest struct {
	// Note ID.
	ID string `path:"id" validate:"required"`
}

// Retrieves a note.
type RetrieveNoteEndpoint struct{}

func (e *RetrieveNoteEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveNoteRequest, *apiresource.Note] {
	return (&apiendpoint.APIEndpoint[*RetrieveNoteRequest, *apiresource.Note]{
		Title:             "Retrieve Note",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             notesRoute + "/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeNote,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainNotes, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveNoteRequest) (*apiresource.Note, *apierror.APIError) {
			return svc.(NoteSvc).GetNote
		},
	})
}
