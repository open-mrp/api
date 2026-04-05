package apiresource

import (
	"encoding/json"
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleAgentMemoryID = "agmm_01jm4r6700f8nwq3v5hx2d9ktp"

// AgentMemory represents a piece of agent memory stored for contextual recall.
type AgentMemory struct {
	// The unique identifier for the memory.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=agent_memory"`
	// The category of memory.
	Category string `json:"category" validate:"required"`
	// The content of the memory.
	Content string `json:"content" validate:"required"`
	// Arbitrary metadata as JSON.
	Metadata json.RawMessage `json:"metadata"`
	// The entity this memory relates to.
	Entity *Entity `json:"entity"`
	// How important this memory is (0-1 scale).
	Importance float64 `json:"importance"`
	// When this memory expires. Null means it never expires.
	ExpiresAt *time.Time `json:"expires_at"`
	// When this memory was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this memory was last updated.
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
