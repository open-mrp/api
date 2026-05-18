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

// Request to update an agent memory.
type UpdateMemoryRequest struct {
	// Memory ID.
	ID string `path:"id" validate:"required"`
	// Memory category (e.g. "preference", "fact", "instruction").
	Category string `json:"category,omitempty" validate:"max=255"`
	// Text content.
	Content string `json:"content,omitempty"`
	// JSON metadata.
	Metadata json.RawMessage `json:"metadata,omitempty"`
	// Entity type this memory is scoped to (e.g. "customer", "product").
	EntityType *string `json:"entity_type,omitempty" nullable:"true" validate:"omitempty,max=255"`
	// Entity ID.
	EntityID *string `json:"entity_id,omitempty" nullable:"true" validate:"omitempty"`
	// Importance score between 0 and 1.
	Importance float64 `json:"importance,omitempty"`
	// ISO 8601 expiration timestamp.
	ExpiresAt *string `json:"expires_at,omitempty" nullable:"true"`
}

var sampleUpdateMemoryRequest = &UpdateMemoryRequest{
	Content:    "Customer prefers next-day shipping on all orders.",
	Importance: 0.9,
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
		Request:           &UpdateMemoryRequest{},
		Response:          &apiresource.AgentMemory{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateMemoryRequest) (*apiresource.AgentMemory, *apierror.APIError) {
			return svc.(AgentMemorySvc).UpdateMemory
		},
	}).WithDocSource(e)
}
