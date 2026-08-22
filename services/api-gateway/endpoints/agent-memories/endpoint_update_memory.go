package agentmemoryep

import (
	"context"
	"encoding/json"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to update an agent memory.
type UpdateMemoryRequest struct {
	// Memory ID.
	ID string `path:"id" validate:"required"`
	// The kind of information this memory holds, used to group related memories.
	//
	// - `preference`: how someone likes things done, such as a customer who always wants express shipping.
	// - `fact`: a durable detail worth remembering about the account or one of its records, such as a customer's typical order size.
	// - `instruction`: standing guidance for agents to follow, such as always confirming freight before issuing an order.
	Category field.Optional[constants.AgentMemoryCategory] `json:"category,omitzero"`
	// The information to remember, written as plain text for an agent to read.
	Content field.Optional[string] `json:"content,omitzero"`
	// Arbitrary metadata as JSON.
	//
	// Replaces the stored metadata outright rather than merging into it.
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
	// An agent takes in only a limited number of memories per run and recalls the highest-importance ones first.
	Importance field.Optional[float64] `json:"importance,omitzero" validate:"omitempty,min=0,max=1,multiple_of=0.1"`
	// When this memory should stop being used, as an ISO 8601 timestamp (e.g. `2026-01-02T15:04:05Z`).
	//
	// Past this time the memory is no longer recalled by agents and is omitted from list results, but it is not deleted. Send `null` so the memory is used indefinitely.
	ExpiresAt field.Clearable[string] `json:"expires_at,omitzero"`
}

var sampleUpdateMemoryRequest = &UpdateMemoryRequest{
	Content:    field.Some("Customer prefers next-day shipping on all orders."),
	Importance: field.Some(0.9),
}

func (*UpdateMemoryRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateMemoryRequest)
}

// Updates an agent memory.
//
// Only the fields included in the request are changed; everything else keeps its current value.
type UpdateMemoryEndpoint struct{}

func (e *UpdateMemoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateMemoryRequest, *apiresource.AgentMemory] {
	return (&apiendpoint.APIEndpoint[*UpdateMemoryRequest, *apiresource.AgentMemory]{
		Title:               "Update Agent Memory",
		Method:              http.MethodPatch,
		Route:               "/v1/ai/memories/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeAgentMemory,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAgentMemories, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateMemoryRequest) (*apiresource.AgentMemory, *apierror.APIError) {
			return svc.(AgentMemorySvc).UpdateMemory
		},
	})
}
