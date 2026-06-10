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

// Request to create an agent memory.
type CreateMemoryRequest struct {
	// Memory category (e.g. "preference", "fact", "instruction").
	Category string `json:"category" validate:"required,max=255"`
	// Text content.
	Content string `json:"content" validate:"required"`
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
		Title:             "Create Agent Memory",
		Method:            http.MethodPost,
		Route:             "/v1/ai/memories",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAgentMemory,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateMemoryRequest) (*apiresource.AgentMemory, *apierror.APIError) {
			return svc.(AgentMemorySvc).CreateMemory
		},
		LocationFunc: func(resp *apiresource.AgentMemory) string {
			return "/v1/ai/memories/" + resp.ID
		},
	})
}
