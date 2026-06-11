package agentmemoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve an agent memory.
type RetrieveMemoryRequest struct {
	// Memory ID.
	ID string `path:"id" validate:"required"`
}

// Returns an agent memory by ID.
type RetrieveMemoryEndpoint struct{}

func (e *RetrieveMemoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveMemoryRequest, *apiresource.AgentMemory] {
	return (&apiendpoint.APIEndpoint[*RetrieveMemoryRequest, *apiresource.AgentMemory]{
		Title:             "Retrieve Agent Memory",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/ai/memories/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAgentMemory,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveMemoryRequest) (*apiresource.AgentMemory, *apierror.APIError) {
			return svc.(AgentMemorySvc).GetMemory
		},
	})
}
