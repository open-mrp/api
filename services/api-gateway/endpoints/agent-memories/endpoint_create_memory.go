package agentmemoryep

import (
	"context"
	"encoding/json"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to create an agent memory.
type CreateMemoryRequest struct {
	// Category used to group related memories.
	Category string `json:"category" validate:"required,oneof=preference fact instruction"`
	// Text content.
	Content string `json:"content" validate:"required"`
	// Arbitrary metadata as JSON.
	Metadata json.RawMessage `json:"metadata,omitzero"`
	// Type of platform record this memory is scoped to (e.g. `customer`, `product`).
	//
	// Provide together with `entity_id` to scope the memory to a specific record; omit both for an account-wide memory.
	EntityType field.Optional[string] `json:"entity_type,omitzero" validate:"omitempty,max=255"`
	// ID of the platform record this memory is scoped to.
	//
	// Provide together with `entity_type`.
	EntityID field.Optional[string] `json:"entity_id,omitzero" validate:"omitempty"`
	// Relative importance from `0` to `1` in increments of `0.1`, used to prioritize which memories the agent recalls.
	//
	// Higher is more important. When omitted, the memory is stored at the lowest priority (`0`).
	Importance field.Optional[float64] `json:"importance,omitzero" validate:"omitempty,min=0,max=1,multiple_of=0.1"`
	// Expiration timestamp in ISO 8601 format (e.g. `2026-01-02T15:04:05Z`).
	//
	// Omit for a memory that never expires. Expired memories are excluded from list results and are no longer recalled by agents.
	ExpiresAt field.Optional[string] `json:"expires_at,omitzero"`
}

var sampleCreateMemoryRequest = &CreateMemoryRequest{
	Category:   "preference",
	Content:    "Customer prefers express shipping on all orders.",
	Metadata:   json.RawMessage(`{"source": "support_ticket"}`),
	Importance: field.Some(0.8),
}

func (*CreateMemoryRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateMemoryRequest)
}

// Creates an agent memory.
type CreateMemoryEndpoint struct{}

func (e *CreateMemoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateMemoryRequest, *apiresource.AgentMemory] {
	return (&apiendpoint.APIEndpoint[*CreateMemoryRequest, *apiresource.AgentMemory]{
		Title:               "Create Agent Memory",
		Method:              http.MethodPost,
		Route:               "/v1/ai/memories",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusCreated,
		Public:              false,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeAgentMemory,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAgentMemories, Action: types.ActionCreate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateMemoryRequest) (*apiresource.AgentMemory, *apierror.APIError) {
			return svc.(AgentMemorySvc).CreateMemory
		},
		LocationFunc: func(resp *apiresource.AgentMemory) string {
			return "/v1/ai/memories/" + resp.ID
		},
	})
}
