package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// Account user with profile, role, and department.
type AccountUser struct {
	// Account user ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_user"`
	// Display name.
	Name *string `json:"name"`
	// Email address.
	Email *string `json:"email"`
	// Username.
	Username *string `json:"username"`
	// Profile image URL.
	ImageURL *string `json:"image_url"`
	// Account user status.
	Status constants.AccountUserStatus `json:"status" validate:"required"`
	// Assigned role.
	Role *Role `json:"role" expandable:"true"`
	// Assigned department.
	Department *Department `json:"department" expandable:"true"`
	// When the user last used this account.
	LastUsedAt *time.Time `json:"last_used_at"`
	// When the account user was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When the account user was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

const SampleAccountUserID = "acus_01gf7a8200er3ar3pkfrb6kk29"

var sampleAccountUserName = "John Doe"
var sampleAccountUserEmail = "john@augno.com"

var SampleAccountUser = &AccountUser{
	ID:        SampleAccountUserID,
	Object:    constants.ObjectTypeAccountUser,
	Name:      &sampleAccountUserName,
	Email:     &sampleAccountUserEmail,
	Status:    constants.AccountUserStatusActive,
	Role:      SampleRole,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*AccountUser) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAccountUser)
}
