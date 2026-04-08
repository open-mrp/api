package agentmemoryep

import (
	"context"
	"encoding/json"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// CreateMemoryRequest is the request to create a new agent memory.
type CreateMemoryRequest struct {
	// The memory category (e.g. "preference", "fact", "instruction").
	Category string `json:"category" validate:"required,max=255"`
	// The text content of the memory.
	Content string `json:"content" validate:"required"`
	// Optional JSON metadata associated with this memory.
	Metadata json.RawMessage `json:"metadata,omitempty"`
	// The type of entity this memory is scoped to (e.g. "customer", "product").
	EntityType *string `json:"entity_type,omitempty" validate:"omitempty,max=255"`
	// The ID of the entity this memory is scoped to.
	EntityID *string `json:"entity_id,omitempty" validate:"omitempty,max=191"`
	// A numeric importance score between 0 and 1.
	Importance float64 `json:"importance,omitempty"`
	// An ISO 8601 timestamp after which this memory expires.
	ExpiresAt *string `json:"expires_at,omitempty"`
}

var sampleCreateMemoryRequest = &CreateMemoryRequest{
	Category:   "preference",
	Content:    "Customer prefers express shipping on all orders.",
	Metadata:   json.RawMessage(`{"source": "support_ticket"}`),
	Importance: 0.8,
}

func (*CreateMemoryRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateMemoryRequest)
}

type CreateMemoryEndpoint struct{}

func (e *CreateMemoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateMemoryRequest, *apiresource.AgentMemory] {
	return &apiendpoint.APIEndpoint[*CreateMemoryRequest, *apiresource.AgentMemory]{
		Title:             "Create Agent Memory",
		Description:       "Creates a new agent memory for the current account.",
		Method:            http.MethodPost,
		Route:             "/v1/ai/memories",
		ContentType:       "application/json",
		Request:           &CreateMemoryRequest{},
		Response:          &apiresource.AgentMemory{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateMemoryRequest) (*apiresource.AgentMemory, *apierror.APIError) {
			return svc.(AgentMemorySvc).CreateMemory
		},
	}
}
