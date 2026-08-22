package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleAccountStatusID = "acss_st5zyjmzm30k"

var SampleAccountStatusCode = constants.AccountStatusCodeNormal

const SampleAccountStatusName = "Normal"

// A lookup value describing the standing of a customer account, such as whether shipments or all activity should be held.
//
// The set of statuses is fixed by OpenMRP and cannot be added to or edited; you apply one to a customer by setting the customer's `status`.
type AccountStatus struct {
	// Account status ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_status"`
	// Machine-readable status code.
	//
	// - `normal`: standard account with no restrictions.
	// - `preferred`: account flagged for prioritized handling.
	// - `hold_shipment`: the account's shipments should be held, typically over a credit problem, while orders can still be placed.
	// - `hold_all`: all activity for the account should be held.
	//
	// The hold statuses are advisory: they are surfaced as credit-hold warnings on the customer's orders, but they do not by themselves cause order or shipment requests to be rejected.
	Code constants.AccountStatusCode `json:"code" validate:"required"`
	// Human-readable label for the status.
	Name string `json:"name" validate:"required"`
	// Owner of this resource.
	//
	// Account statuses are platform-provided and shared across all accounts, so the owner is always the OpenMRP system owner.
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
