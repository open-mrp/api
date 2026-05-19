package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleAgentTokenUsageID = "agtk_01gf7a8200er3ar3pkfrb6kk29" // #nosec G101 -- sample ID, not a credential

// Daily agent token usage record.
type AgentTokenUsage struct {
	// Usage record ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=agent_token_usage"`
	// Date of usage (YYYY-MM-DD).
	Date string `json:"date" validate:"required"`
	// Total input tokens consumed.
	InputTokens int64 `json:"input_tokens"`
	// Total output tokens consumed.
	OutputTokens int64 `json:"output_tokens"`
	// Total cost in USD.
	TotalCost float64 `json:"total_cost"`
	// Number of agent runs.
	RunCount int32 `json:"run_count"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleAgentTokenUsage = &AgentTokenUsage{
	ID:           SampleAgentTokenUsageID,
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
