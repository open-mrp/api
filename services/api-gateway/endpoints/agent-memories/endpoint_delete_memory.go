package agentmemoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete an agent memory.
type DeleteMemoryRequest struct {
	// Memory ID.
	ID string `path:"id" validate:"required"`
}

// Deletes an agent memory.
type DeleteMemoryEndpoint struct{}

func (e *DeleteMemoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteMemoryRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteMemoryRequest, *apiresource.EmptyResource]{
		Title:             "Delete Agent Memory",
		Method:            http.MethodDelete,
		Route:             "/v1/ai/memories/{id}",
		ContentType:       "application/json",
		Request:           &DeleteMemoryRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteMemoryRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AgentMemorySvc).DeleteMemory
		},
	}).WithDocSource(e)
}
