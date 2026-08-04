package agentmemoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
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

func (e *DeleteMemoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteMemoryRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteMemoryRequest, *apiresource.EmptyResource]{
		Title:               "Delete Agent Memory",
		Method:              http.MethodDelete,
		Route:               "/v1/ai/memories/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAgentMemories, Action: types.ActionDelete}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteMemoryRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AgentMemorySvc).DeleteMemory
		},
	})
}
