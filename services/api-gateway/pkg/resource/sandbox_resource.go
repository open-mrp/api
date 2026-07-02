package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleSandboxID = "sbac_01ebd87c707b138806f060b9ae"
const SampleSandboxName = "Integration Testing"

// Sandbox account for isolated testing.
type Sandbox struct {
	// Sandbox ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sandbox"`
	// Display name of the sandbox.
	Name string `json:"name" validate:"required"`
	// The production account that owns this sandbox.
	OwnerAccount *Account `json:"owner_account" expandable:"true"`
	// When this sandbox was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this sandbox was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleSandbox = &Sandbox{
	ID:     SampleSandboxID,
	Object: constants.ObjectTypeSandbox,
	Name:   SampleSandboxName,
	OwnerAccount: &Account{
		ID:        SampleAccountID,
		Object:    constants.ObjectTypeAccount,
		Name:      SampleAccountName,
		CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
		UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
	},
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Sandbox) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSandbox)
}
