package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleSandboxID = "sbac_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleSandboxName = "Integration Testing"

// Sandbox represents an isolated testing environment for an account.
type Sandbox struct {
	// The unique identifier for the sandbox.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sandbox"`
	// The display name of the sandbox.
	Name string `json:"name" validate:"required"`
	// The owner account of this sandbox.
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
