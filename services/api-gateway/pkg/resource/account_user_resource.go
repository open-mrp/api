package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// A user's membership in an account, carrying the account-specific status, role, and department.
//
// Profile fields (name, email, username, image URL) live on the `user` sub-resource, which is shared across every account the user belongs to.
type AccountUser struct {
	// Account user ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_user"`
	// The current state of this user's membership in the account.
	//
	// - `active`: the user can sign in to the account and occupies one of the plan's seats.
	// - `disabled`: the user is locked out of the account and their sessions have been revoked, but the membership is retained.
	// - `removed`: the membership has been soft-deleted; it is hidden from listings by default and can be restored with the activate action.
	Status constants.AccountUserStatus `json:"status" validate:"required"`
	// The role that determines what this user is permitted to do in the account.
	Role *Role `json:"role" expandable:"true"`
	// The department this user belongs to within the account.
	Department *Department `json:"department" expandable:"true"`
	// The underlying user profile, shared across every account this person belongs to.
	User *User `json:"user" expandable:"true"`
	// Whether this user can be assigned as a sales representative on orders, territories, and targets.
	//
	// Independent of the `sales_rep` role type, which still scopes analytics and hides cost. Users with the `sales_rep` role are always eligible.
	IsCommissionEligible bool `json:"is_commission_eligible" validate:"required"`
	// When the user last accessed this account.
	LastUsedAt *time.Time `json:"last_used_at"`
	// When the account user was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When the account user was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

const SampleAccountUserID = "acus_e5zu8bde0z3h"

var SampleAccountUser = &AccountUser{
	ID:                   SampleAccountUserID,
	Object:               constants.ObjectTypeAccountUser,
	Status:               constants.AccountUserStatusActive,
	Role:                 SampleRole,
	Department:           SampleDepartment,
	User:                 SampleUser,
	IsCommissionEligible: false,
	LastUsedAt:           new(timeutil.TimestampToTime(sampleUpdatedAtTimestamp)),
	CreatedAt:            timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:            timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*AccountUser) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAccountUser)
}
