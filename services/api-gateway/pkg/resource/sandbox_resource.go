package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleSandboxID = "sbac_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleSandboxName = "Integration Testing"
const SampleSandboxAccountID = "ac_01jm4r6700g2bz7y4c6e8f1jrm"

var SampleSandbox = &Sandbox{
	ID:        SampleSandboxID,
	Object:    constants.ObjectTypeSandbox,
	Name:      SampleSandboxName,
	AccountID: SampleSandboxAccountID,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

// Sandbox represents an isolated testing environment for an account.
type Sandbox struct {
	// The unique identifier for the sandbox.
	ID string `json:"id" validate:"required"`
	// The object type.
	Object constants.ObjectType `json:"object" validate:"required,enum=sandbox"`
	// The display name of the sandbox.
	Name string `json:"name" validate:"required"`
	// The ID of the account this sandbox belongs to.
	AccountID string `json:"account_id" validate:"required"`
	// When this sandbox was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this sandbox was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

func (*Sandbox) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSandbox)
}
