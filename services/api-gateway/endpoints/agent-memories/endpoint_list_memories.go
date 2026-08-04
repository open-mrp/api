package agentmemoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list agent memories.
type ListMemoriesRequest struct {
	apiresource.PaginationRequest
	// Filter to memories with this exact category (e.g. `preference`, `fact`).
	Category *string `query:"category"`
	// Filter to memories scoped to this entity type (e.g. `customer`, `product`).
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

// Returns a paginated list of agent memories for the current account, newest first.
//
// Memories whose `expires_at` has passed are excluded. The `q` search term matches against a memory's ID, category, content, and the ID of the record it is scoped to.
type ListMemoriesEndpoint struct{}

func (e *ListMemoriesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListMemoriesRequest, *apiresource.List[apiresource.AgentMemory]] {
	return (&apiendpoint.APIEndpoint[*ListMemoriesRequest, *apiresource.List[apiresource.AgentMemory]]{
		Title:               "List Agent Memories",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/ai/memories",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeAgentMemory,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAgentMemories, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListMemoriesRequest) (*apiresource.List[apiresource.AgentMemory], *apierror.APIError) {
			return svc.(AgentMemorySvc).ListMemories
		},
	})
}
