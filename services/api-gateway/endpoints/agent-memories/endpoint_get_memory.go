package agentmemoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get an agent memory.
type GetMemoryRequest struct {
	// Memory ID.
	ID string `path:"id" validate:"required"`
}

type GetMemoryEndpoint struct{}

func (e *GetMemoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetMemoryRequest, *apiresource.AgentMemory] {
	return &apiendpoint.APIEndpoint[*GetMemoryRequest, *apiresource.AgentMemory]{
		Title:             "Get Agent Memory",
		Description:       "Returns an agent memory by ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/ai/memories/{id}",
		Request:           &GetMemoryRequest{},
		Response:          &apiresource.AgentMemory{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetMemoryRequest) (*apiresource.AgentMemory, *apierror.APIError) {
			return svc.(AgentMemorySvc).GetMemory
		},
	}
}
