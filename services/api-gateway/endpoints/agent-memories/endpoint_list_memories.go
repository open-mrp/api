package agentmemoryep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
)

// Request to list agent memories.
type ListMemoriesRequest struct {
	object.ListRequest
	// Case-insensitive substring match on `content`.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Filter to memories with this exact category (e.g. `preference`, `fact`).
	Category *constants.AgentMemoryCategory `query:"category"`
	// Filter to memories scoped to this entity type (e.g. `customer`, `product`).
	EntityType *string `query:"entity_type"`
}

var _ apiexample.Documented = (*ListMemoriesRequest)(nil)

func (*ListMemoriesRequest) SchemaExample() any {
	category := constants.AgentMemoryCategoryPreference
	entityType := "customer"
	m := apiexample.ValidateAndMarshalToMap(&ListMemoriesRequest{
		ListRequest: object.ListRequest{},
		Category:    &category,
		EntityType:  &entityType,
	})
	for k, v := range apiresource.ListRequestSchemaExample() {
		m[k] = v
	}
	return m
}

// Returns a paginated list of agent memories for the current account, newest first.
//
// Memories whose `expires_at` has passed are excluded.
type ListMemoriesEndpoint struct{}

func (e *ListMemoriesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListMemoriesRequest, *apiresource.List[apiresource.AgentMemory]] {
	return (&apiendpoint.APIEndpoint[*ListMemoriesRequest, *apiresource.List[apiresource.AgentMemory]]{
		Title:             "List Agent Memories",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/ai/memories",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		ObjectType:        constants.ObjectTypeAgentMemory,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainNotes, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListMemoriesRequest) (*apiresource.List[apiresource.AgentMemory], *apierror.APIError) {
			return svc.(AgentMemorySvc).ListMemories
		},
	})
}
