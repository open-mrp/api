package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// AgentTokenUsage represents a daily token usage record for an account.
type AgentTokenUsage struct {
	// The unique identifier for this usage record.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=agent_token_usage"`
	// The date of usage (YYYY-MM-DD).
	Date string `json:"date" validate:"required"`
	// Total input tokens consumed.
	InputTokens int64 `json:"input_tokens"`
	// Total output tokens consumed.
	OutputTokens int64 `json:"output_tokens"`
	// Total cost in USD.
	TotalCost float64 `json:"total_cost"`
	// Number of agent runs.
	RunCount int32 `json:"run_count"`
	// When this record was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this record was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleAgentTokenUsage = &AgentTokenUsage{
	ID:           "agtk_01gf7a8200er3ar3pkfrb6kk29",
	Object:       constants.ObjectTypeAgentTokenUsage,
	Date:         sampleCreatedAtTimestamp,
	InputTokens:  100,
	OutputTokens: 200,
	TotalCost:    0.01,
	RunCount:     1,
	CreatedAt:    timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:    timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*AgentTokenUsage) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAgentTokenUsage)
}
