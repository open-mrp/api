package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleChildAccountRelationID = "acre_011409c39cc78dc5655c8846bd"
const SampleChildAccountExternalNumber = "CUST-001"

// Child customer account in a parent-child relationship.
type ChildAccount struct {
	// Account relation ID.
	//
	// Identifies the relationship record, not the child account itself; use `account.id` for the account.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=child_account"`
	// The child account itself.
	Account *Account `json:"account" validate:"required"`
	// Your own identifier for this customer, such as a CRM or ERP customer number, stored on the parent-child relation rather than on the account.
	ExternalNumber *string `json:"external_number"`
	// Support email address copied from the child account's branding.
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
