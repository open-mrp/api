package agentmemoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list agent memories.
type ListMemoriesRequest struct {
	apiresource.PaginationRequest
	// Category filter (e.g. "preference", "fact").
	Category *string `query:"category"`
	// Entity type filter (e.g. "customer", "product").
	EntityType *string `query:"entity_type"`
}

var _ contracts.DocumentedType = (*ListMemoriesRequest)(nil)

func (*ListMemoriesRequest) SchemaExample() any {
	category := "preference"
	entityType := "customer"
	m := apiexample.ValidateAndMarshalToMap(&ListMemoriesRequest{
		PaginationRequest: apiresource.PaginationRequest{},
		Category:          &category,
		EntityType:        &entityType,
	})
	for k, v := range (&apiresource.PaginationRequest{}).SchemaExample().(map[string]any) {
		m[k] = v
	}
	return m
}

// Returns a paginated list of agent memories.
type ListMemoriesEndpoint struct{}

func (e *ListMemoriesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListMemoriesRequest, *apiresource.List[apiresource.AgentMemory]] {
	return (&apiendpoint.APIEndpoint[*ListMemoriesRequest, *apiresource.List[apiresource.AgentMemory]]{
		Title:             "List Agent Memories",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/ai/memories",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListMemoriesRequest) (*apiresource.List[apiresource.AgentMemory], *apierror.APIError) {
			return svc.(AgentMemorySvc).ListMemories
		},
	})
}
