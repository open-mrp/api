package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// A user's membership in an account, carrying the account-specific status, role, and department.
//
// Profile fields (name, email, username, image URL) live on the expandable `user` sub-resource, which is shared across every account the user belongs to.
type AccountUser struct {
	// Account user ID.
	ID string `json:"id" validate:"required"`
	// Underlying user.
	User *User `json:"user" expandable:"true"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_user"`
	// Account user status.
	//
	// - `active`: the user can access the account.
	// - `disabled`: the user is locked out of the account.
	// - `removed`: the user has been removed (soft-deleted) from the account.
	Status constants.AccountUserStatus `json:"status" validate:"required"`
	// Assigned role.
	Role *Role `json:"role" expandable:"true"`
	// Assigned department.
	Department *Department `json:"department" expandable:"true"`
	// When the user last accessed this account.
	LastUsedAt *time.Time `json:"last_used_at"`
	// When the account user was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When the account user was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

const SampleAccountUserID = "acus_01ea9983ddb41dacc44ecf997c"

var SampleAccountUser = &AccountUser{
	ID:        SampleAccountUserID,
	User:      SampleUser,
	Object:    constants.ObjectTypeAccountUser,
	Status:    constants.AccountUserStatusActive,
	Role:      SampleRole,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*AccountUser) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAccountUser)
}
