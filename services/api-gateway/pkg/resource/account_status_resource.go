package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleAccountStatusID = "acss_01jm4r6700f8nwq3v5hx2d9ktp"

var SampleAccountStatusCode = constants.AccountStatusCodeNormal

const SampleAccountStatusName = "Normal"

// AccountStatus represents an account status lookup value.
type AccountStatus struct {
	// The unique identifier for the account status.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_status"`
	// The machine-readable code for this status.
	Code constants.AccountStatusCode `json:"code" validate:"required"`
	// The display name of the account status.
	Name string `json:"name" validate:"required"`
	// The owner of this resource.
	Owner *Owner `json:"owner" expandable:"true"`
	// When this account status was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this account status was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleAccountStatus = &AccountStatus{
	ID:        SampleAccountStatusID,
	Object:    constants.ObjectTypeAccountStatus,
	Code:      SampleAccountStatusCode,
	Name:      SampleAccountStatusName,
	Owner:     SampleOwnerSystem,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*AccountStatus) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAccountStatus)
}
