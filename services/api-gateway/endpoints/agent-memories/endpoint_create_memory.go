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

// Request to create an agent memory.
type CreateMemoryRequest struct {
	// Memory category (e.g. "preference", "fact", "instruction").
	Category string `json:"category" validate:"required,max=255"`
	// Text content.
	Content string `json:"content" validate:"required"`
	// JSON metadata.
	Metadata json.RawMessage `json:"metadata,omitempty"`
	// Entity type this memory is scoped to (e.g. "customer", "product").
	EntityType *string `json:"entity_type,omitempty" validate:"omitempty,max=255"`
	// Entity ID.
	EntityID *string `json:"entity_id,omitempty" validate:"omitempty"`
	// Importance score between 0 and 1.
	Importance float64 `json:"importance,omitempty"`
	// ISO 8601 expiration timestamp.
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
		Description:       "Creates an agent memory.",
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
		LocationFunc: func(resp *apiresource.AgentMemory) string {
			return "/v1/ai/memories/" + resp.ID
		},
	}
}
