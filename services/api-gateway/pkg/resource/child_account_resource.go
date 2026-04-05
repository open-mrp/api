package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleChildAccountRelationID = "acre_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleChildAccountExternalNumber = "CUST-001"

// ChildAccount represents a child customer account in a parent-child relationship.
type ChildAccount struct {
	// The account relation ID.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=child_account"`
	// The counterparty account.
	Account *Account `json:"account" validate:"required"`
	// The external number for the account relation.
	ExternalNumber *string `json:"external_number"`
	// The support email from account branding.
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
