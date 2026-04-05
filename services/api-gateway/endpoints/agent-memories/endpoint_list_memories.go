package agentmemoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ListMemoriesRequest is the request to list agent memories.
type ListMemoriesRequest struct {
	apiresource.PaginationRequest
	// Filter by memory category (e.g. "preference", "fact").
	Category *string `query:"category"`
	// Filter by entity type (e.g. "customer", "product").
	EntityType *string `query:"entity_type"`
}

type ListMemoriesEndpoint struct{}

func (e *ListMemoriesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListMemoriesRequest, *apiresource.List[apiresource.AgentMemory]] {
	return &apiendpoint.APIEndpoint[*ListMemoriesRequest, *apiresource.List[apiresource.AgentMemory]]{
		Title:             "List Agent Memories",
		Description:       "Returns a paginated list of agent memories for the current account.",
		Method:            http.MethodGet,
		Route:             "/v1/ai/memories",
		Request:           &ListMemoriesRequest{},
		Response:          &apiresource.List[apiresource.AgentMemory]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListMemoriesRequest) (*apiresource.List[apiresource.AgentMemory], *apierror.APIError) {
			return svc.(AgentMemorySvc).ListMemories
		},
	}
}
