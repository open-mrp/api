package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleChildAccountRelationID = "acre_c76d97madwo3"
const SampleChildAccountExternalNumber = "CUST-001"

// Child customer account in a parent-child relationship.
//
// Parent-child links let you model a customer hierarchy, such as a chain's individual store locations sitting beneath its head office. Both accounts are customers of your own account, and the hierarchy is visible only to you.
type ChildAccount struct {
	// Account relation ID.
	//
	// Identifies the relationship record, not the child account itself; use `account.id` for the account.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=child_account"`
	// The child account itself.
	//
	// Only the identifying fields are populated here; fetch the account or its customer record for full detail.
	Account *Account `json:"account" validate:"required"`
	// The customer number for the child account, matching the `number` on your customer record for it.
	ExternalNumber *string `json:"external_number"`
	// Support email address published in the child account's branding.
	Email *string `json:"email"`
	// When this relation was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this relation was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleChildAccount = &ChildAccount{
	ID:     SampleChildAccountRelationID,
	Object: constants.ObjectTypeChildAccount,
	Account: &Account{
		ID:     SampleAccountID,
		Object: constants.ObjectTypeAccount,
		Name:   "Child Customer Inc",
	},
	ExternalNumber: new(SampleChildAccountExternalNumber),
	Email:          new("support@childcustomer.com"),
	CreatedAt:      timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:      timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ChildAccount) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleChildAccount)
}
