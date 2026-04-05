package agentmemoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteMemoryRequest is the request to delete an agent memory.
type DeleteMemoryRequest struct {
	// The ID of the memory to delete.
	ID string `path:"id" validate:"required"`
}

type DeleteMemoryEndpoint struct{}

func (e *DeleteMemoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteMemoryRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteMemoryRequest, *apiresource.EmptyResource]{
		Title:             "Delete Agent Memory",
		Description:       "Deletes an agent memory.",
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
	}
}
