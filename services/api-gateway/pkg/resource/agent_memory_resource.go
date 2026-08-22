package apiresource

import (
	"encoding/json"
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleAgentMemoryID = "agmm_o7tjkr16gfmh"

// A piece of information an agent has saved for recall in future runs.
type AgentMemory struct {
	// Memory ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=agent_memory"`
	// The kind of information this memory holds, used to group related memories.
	//
	// - `preference`: how someone likes things done, such as a customer who always wants express shipping.
	// - `fact`: a durable detail worth remembering about the account or one of its records, such as a customer's typical order size.
	// - `instruction`: standing guidance for agents to follow, such as always confirming freight before issuing an order.
	Category constants.AgentMemoryCategory `json:"category" validate:"required"`
	// The information itself, written as plain text for an agent to read.
	Content string `json:"content" validate:"required"`
	// Arbitrary metadata as JSON.
	Metadata json.RawMessage `json:"metadata"`
	// The platform record this memory is about (e.g. a specific customer or product).
	Entity *Entity `json:"entity"`
	// Relative importance from `0` to `1`, used to prioritize which memories the agent recalls.
	//
	// An agent takes in only a limited number of memories per run, and the highest-importance ones are recalled first.
	Importance float64 `json:"importance"`
	// When this memory stops being used.
	//
	// Past this time the memory is no longer recalled by agents and is omitted from list results, but it is not deleted and can still be retrieved by ID. A memory with no expiration is used indefinitely.
	ExpiresAt *time.Time `json:"expires_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleAgentMemory = &AgentMemory{
	ID:         SampleAgentMemoryID,
	Object:     constants.ObjectTypeAgentMemory,
	Category:   constants.AgentMemoryCategoryPreference,
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
