package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

// A person's membership in an account: your own team, or a customer's or supplier's contacts.
//
// The person's name, email and photo are on `user`, which is shared across every account they belong to. Adding someone never gives them access by itself: your own team members are emailed a sign-in link when you add them, and contacts only when you invite them.
type AccountUser struct {
	// Account user ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_user"`
	// Whether the person can use the account.
	//
	// - `active`: they can sign in, and on your own team they take a seat.
	// - `disabled`: they're locked out and their sessions are ended; the membership is kept.
	// - `removed`: they're no longer a member. Records that name them, such as a customer's sales rep, keep naming them.
	//
	// Change it with the activate, disable and remove actions.
	Status constants.AccountUserStatus `json:"status" validate:"required"`
	// Whether the person can sign in yet.
	//
	// - `none`: they've never been sent a sign-in link.
	// - `invited`: they've been sent one but haven't signed in.
	// - `active`: they've signed in.
	//
	// The invite action sends a link.
	PortalAccess constants.PortalAccess `json:"portal_access" validate:"required"`
	// The role that sets what the person can do in the account.
	Role *Role `json:"role" expandable:"true"`
	// The work center the person works in.
	WorkCenter *WorkCenter `json:"work_center" expandable:"true"`
	// The person's profile, shared across every account they belong to.
	User *User `json:"user" expandable:"true"`
	// Whether the person can be named as a sales rep on customers, orders, territories and targets: `eligible` or `ineligible`.
	//
	// Members with a sales-rep role are always `eligible`. `null` for customer and supplier portal users.
	CommissionEligibility *constants.CommissionEligibility `json:"commission_eligibility" validate:"required" sensitive:"internal"`
	// The notifications you send this contact about their account.
	//
	// Set when the account you are acting in is a customer or supplier account you manage, and `null` otherwise.
	NotificationTypes *[]constants.AccountRelationNotificationType `json:"notification_types" visibility:"internal"`
	// When the person last used this account. `null` for customer and supplier portal users.
	LastUsedAt *time.Time `json:"last_used_at" sensitive:"internal"`
	// When the membership was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When the membership was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

const SampleAccountUserID = "acus_e5zu8bde0z3h"

var SampleAccountUser = &AccountUser{
	ID:                    SampleAccountUserID,
	Object:                constants.ObjectTypeAccountUser,
	Status:                constants.AccountUserStatusActive,
	PortalAccess:          constants.PortalAccessActive,
	Role:                  SampleRole,
	WorkCenter:            SampleWorkCenter,
	User:                  SampleUser,
	CommissionEligibility: new(constants.CommissionEligibilityIneligible),
	NotificationTypes: &[]constants.AccountRelationNotificationType{
		constants.AccountRelationNotificationTypeInvoice,
		constants.AccountRelationNotificationTypeOrderAcknowledgement,
	},
	LastUsedAt: new(timeutil.TimestampToTime(sampleUpdatedAtTimestamp)),
	CreatedAt:  timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:  timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*AccountUser) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAccountUser)
}
