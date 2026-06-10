package agentmemoryep

import (
	"context"
	"encoding/json"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to update an agent memory.
type UpdateMemoryRequest struct {
	// Memory ID.
	ID string `path:"id" validate:"required"`
	// Memory category (e.g. "preference", "fact", "instruction").
	Category field.Optional[string] `json:"category,omitzero" validate:"omitempty,max=255"`
	// Text content.
	Content field.Optional[string] `json:"content,omitzero"`
	// JSON metadata.
	Metadata json.RawMessage `json:"metadata,omitzero"`
	// Entity type this memory is scoped to (e.g. "customer", "product").
	EntityType field.Optional[string] `json:"entity_type,omitzero" validate:"omitempty,max=255"`
	// Entity ID.
	EntityID field.Optional[string] `json:"entity_id,omitzero" validate:"omitempty"`
	// Importance score between 0 and 1.
	Importance field.Optional[float64] `json:"importance,omitzero"`
	// ISO 8601 expiration timestamp.
	ExpiresAt field.Optional[string] `json:"expires_at,omitzero"`
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
		Title:             "Update Agent Memory",
		Method:            http.MethodPatch,
		Route:             "/v1/ai/memories/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAgentMemory,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateMemoryRequest) (*apiresource.AgentMemory, *apierror.APIError) {
			return svc.(AgentMemorySvc).UpdateMemory
		},
	})
}
