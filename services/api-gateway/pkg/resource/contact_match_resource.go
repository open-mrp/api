package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

// A contact found by email on an account you have a relationship with — one of your customers, your suppliers, or your own account.
//
// The same email can be a contact on many accounts across the platform; only accounts you relate to are returned.
//
// Only active people are matched — someone who has been disabled or removed on an account never produces a match for that account.
type ContactMatch struct {
	// Contact match ID.
	//
	// This is the matched account user's ID, so the same value also appears as `account_user.id`.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=contact_match"`
	// How you relate to the account this contact belongs to.
	//
	// - `customer`: the account is one of your customers.
	// - `supplier`: the account is one of your suppliers.
	// - `self`: the account is your own.
	Relationship constants.ContactRelationship `json:"relationship" validate:"required"`
	// The email address that was matched.
	Email string `json:"email" validate:"required"`
	// The matched account user.
	AccountUser *AccountUser `json:"account_user" expandable:"true"`
	// The account this contact belongs to.
	Account *Account `json:"account" expandable:"true"`
}

const SampleContactMatchID = SampleAccountUserID

var SampleContactMatch = &ContactMatch{
	ID:           SampleContactMatchID,
	Object:       constants.ObjectTypeContactMatch,
	Relationship: constants.ContactRelationshipCustomer,
	Email:        "buyer@acme-co.example",
	AccountUser:  SampleAccountUser,
	Account:      SampleAccount,
}

func (*ContactMatch) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleContactMatch)
}
