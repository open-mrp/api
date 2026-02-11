package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleSandboxID = "sbac_abc123xyz789"
const SampleSandboxName = "Integration Testing"
const SampleSandboxAccountID = "ac_xyz789abc123"

var SampleSandbox = &Sandbox{
	ID:        SampleSandboxID,
	Object:    constants.ObjectTypeSandbox,
	Name:      SampleSandboxName,
	AccountID: SampleSandboxAccountID,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

// Sandbox represents an isolated testing environment for an account
type Sandbox struct {
	// The unique identifier for the sandbox
	ID string `json:"id"`
	// The object type, always "sandbox"
	Object constants.ObjectType `json:"object" validate:"enum=sandbox"`
	// The display name of the sandbox
	Name string `json:"name"`
	// The ID of the account this sandbox belongs to
	AccountID string `json:"account_id"`
	// The timestamp when the sandbox was created
	CreatedAt time.Time `json:"created_at"`
	// The timestamp when the sandbox was last updated
	UpdatedAt time.Time `json:"updated_at"`
}

func (*Sandbox) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSandbox)
}
