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

// UpdateMemoryRequest is the request to update an existing agent memory.
type UpdateMemoryRequest struct {
	// The ID of the memory to update.
	ID string `path:"id" validate:"required"`
	// The memory category (e.g. "preference", "fact", "instruction").
	Category string `json:"category,omitempty"`
	// The text content of the memory.
	Content string `json:"content,omitempty"`
	// Optional JSON metadata associated with this memory.
	Metadata json.RawMessage `json:"metadata,omitempty"`
	// The type of entity this memory is scoped to (e.g. "customer", "product").
	EntityType *string `json:"entity_type,omitempty" nullable:"true"`
	// The ID of the entity this memory is scoped to.
	EntityID *string `json:"entity_id,omitempty" nullable:"true"`
	// A numeric importance score between 0 and 1.
	Importance float64 `json:"importance,omitempty"`
	// An ISO 8601 timestamp after which this memory expires.
	ExpiresAt *string `json:"expires_at,omitempty" nullable:"true"`
}

var sampleUpdateMemoryRequest = &UpdateMemoryRequest{
	Content:    "Customer prefers next-day shipping on all orders.",
	Importance: 0.9,
}

func (*UpdateMemoryRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateMemoryRequest)
}

type UpdateMemoryEndpoint struct{}

func (e *UpdateMemoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateMemoryRequest, *apiresource.AgentMemory] {
	return &apiendpoint.APIEndpoint[*UpdateMemoryRequest, *apiresource.AgentMemory]{
		Title:             "Update Agent Memory",
		Description:       "Partially updates an agent memory.",
		Method:            http.MethodPatch,
		Route:             "/v1/ai/memories/{id}",
		ContentType:       "application/json",
		Request:           &UpdateMemoryRequest{},
		Response:          &apiresource.AgentMemory{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateMemoryRequest) (*apiresource.AgentMemory, *apierror.APIError) {
			return svc.(AgentMemorySvc).UpdateMemory
		},
	}
}
