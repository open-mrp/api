package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleAccountStatusID = "acss_01004f532c58d60514b685cb27"

var SampleAccountStatusCode = constants.AccountStatusCodeNormal

const SampleAccountStatusName = "Normal"

// AccountStatus is an account status lookup value.
type AccountStatus struct {
	// Account status ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_status"`
	// Machine-readable status code.
	//
	// - `normal`: standard account with no restrictions.
	// - `preferred`: account flagged as preferred (e.g. for prioritized handling).
	// - `hold_shipment`: shipments to this account are held; orders may still be placed.
	// - `hold_all`: all activity for this account is held.
	Code constants.AccountStatusCode `json:"code" validate:"required"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Owner of this resource.
	Owner *Owner `json:"owner" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
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
