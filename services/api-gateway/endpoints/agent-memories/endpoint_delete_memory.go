package agentmemoryep

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

// Request to delete an agent memory.
type DeleteMemoryRequest struct {
	// Memory ID.
	ID string `path:"id" validate:"required"`
}

// Permanently deletes an agent memory so it is no longer recalled.
//
// Deleting a memory that has already been deleted succeeds rather than returning an error.
type DeleteMemoryEndpoint struct{}

func (e *DeleteMemoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteMemoryRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteMemoryRequest, *object.Deleted]{
		Title:             "Delete Agent Memory",
		Method:            http.MethodDelete,
		Route:             "/v1/ai/memories/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeAgentMemory,
		Public:            false,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainNotes, Action: types.ActionDelete}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteMemoryRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(AgentMemorySvc).DeleteMemory, constants.ObjectTypeAgentMemory, "id")
		},
	})
}
