package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// Role resource.
type Role struct {
	// Role ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=role"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Role type code.
	TypeCode constants.RoleType `json:"type" validate:"required"`
	// Owner of this resource.
	Owner *Owner `json:"owner" expandable:"true"`
	// Permissions in `{domain}:{action}` format.
	Permissions *[]string `json:"permissions" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at"`
}

const SampleRoleID = "rl_01gf7a8200er3ar3pkfrb6kk29"
const SampleRoleName = "Admin"

var SampleRolePermissions = []string{"customers:create", "customers:read", "customers:update", "customers:delete"}

var SampleRole = &Role{
	ID:          SampleRoleID,
	Name:        SampleRoleName,
	Object:      constants.ObjectTypeRole,
	TypeCode:    constants.RoleTypeAdmin,
	Owner:       SampleOwnerAccount,
	Permissions: &SampleRolePermissions,
	CreatedAt:   timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:   timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Role) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleRole)
}
