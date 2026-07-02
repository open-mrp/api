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

// Request to update an agent memory.
type UpdateMemoryRequest struct {
	// Memory ID.
	ID string `path:"id" validate:"required"`
	// Category used to group related memories.
	Category field.Optional[string] `json:"category,omitzero" validate:"omitempty,oneof=preference fact instruction"`
	// Text content.
	Content field.Optional[string] `json:"content,omitzero"`
	// Arbitrary metadata as JSON.
	Metadata json.RawMessage `json:"metadata,omitzero"`
	// Type of platform record this memory is scoped to (e.g. `customer`, `product`).
	//
	// Provide together with `entity_id` to scope the memory to a specific record; send `null` (on either entity field) to unscope the memory.
	EntityType field.Clearable[string] `json:"entity_type,omitzero" validate:"omitempty,max=255"`
	// ID of the platform record this memory is scoped to.
	//
	// Provide together with `entity_type`; send `null` to unscope the memory.
	EntityID field.Clearable[string] `json:"entity_id,omitzero" validate:"omitempty"`
	// Relative importance from `0` to `1` in increments of `0.1`, used to prioritize which memories the agent recalls.
	//
	// Higher is more important.
	Importance field.Optional[float64] `json:"importance,omitzero" validate:"omitempty,min=0,max=1,multiple_of=0.1"`
	// Expiration timestamp in ISO 8601 format (e.g. `2026-01-02T15:04:05Z`).
	//
	// Expired memories are excluded from list results and are no longer recalled by agents. Send `null` to make the memory permanent (never expires).
	ExpiresAt field.Clearable[string] `json:"expires_at,omitzero"`
}

var sampleUpdateMemoryRequest = &UpdateMemoryRequest{
	Content:    field.Some("Customer prefers next-day shipping on all orders."),
	Importance: field.Some(0.9),
}

func (*UpdateMemoryRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateMemoryRequest)
}

// Partially updates an agent memory.
type UpdateMemoryEndpoint struct{}

func (e *UpdateMemoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateMemoryRequest, *apiresource.AgentMemory] {
	return (&apiendpoint.APIEndpoint[*UpdateMemoryRequest, *apiresource.AgentMemory]{
		Title:               "Update Agent Memory",
		Method:              http.MethodPatch,
		Route:               "/v1/ai/memories/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeAgentMemory,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAgentMemories, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateMemoryRequest) (*apiresource.AgentMemory, *apierror.APIError) {
			return svc.(AgentMemorySvc).UpdateMemory
		},
	})
}
