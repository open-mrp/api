package apiresource

import (
	"encoding/json"
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleAgentMemoryID = "agmm_018731bdaf4ab04bd5bff1b65c"

// Agent memory resource.
type AgentMemory struct {
	// Memory ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=agent_memory"`
	// Memory category.
	Category string `json:"category" validate:"required"`
	// Memory content.
	Content string `json:"content" validate:"required"`
	// Arbitrary metadata as JSON.
	Metadata json.RawMessage `json:"metadata"`
	// Associated entity.
	Entity *Entity `json:"entity"`
	// Importance score (0–1 scale).
	Importance float64 `json:"importance"`
	// Expiration timestamp. Null means it never expires.
	ExpiresAt *time.Time `json:"expires_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleAgentMemory = &AgentMemory{
	ID:         SampleAgentMemoryID,
	Object:     constants.ObjectTypeAgentMemory,
	Category:   "preference",
	Content:    "Customer prefers express shipping for all orders.",
	Metadata:   json.RawMessage(`{}`),
	Entity:     SampleCustomerEntity,
	Importance: 0.8,
	ExpiresAt:  timeutil.TimestampToTimePtr(sampleExpiresAtTimestamp),
	CreatedAt:  timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:  timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*AgentMemory) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAgentMemory)
}
