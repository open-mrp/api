package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// Role represents a role reference.
type Role struct {
	// The unique identifier for the role.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=role"`
	// The display name of the role.
	Name string `json:"name" validate:"required"`
	// The role type code.
	TypeCode constants.RoleTypeCode `json:"type_code" validate:"required"`
	// The owner of this resource.
	Owner *Owner `json:"owner" expandable:"true"`
	// The permissions for this role in "{domain}:{action}" format.
	Permissions *[]string `json:"permissions" expandable:"true"`
	// The timestamp when the role was created.
	CreatedAt time.Time `json:"created_at"`
	// The timestamp when the role was last updated.
	UpdatedAt time.Time `json:"updated_at"`
}

const SampleRoleID = "rl_01gf7a8200er3ar3pkfrb6kk29"
const SampleRoleName = "Admin"

var SampleRolePermissions = []string{"customers:create", "customers:read", "customers:update", "customers:delete"}

var SampleRole = &Role{
	ID:          SampleRoleID,
	Name:        SampleRoleName,
	Object:      constants.ObjectTypeRole,
	TypeCode:    constants.RoleTypeCodeAdmin,
	Owner:       SampleOwnerAccount,
	Permissions: &SampleRolePermissions,
	CreatedAt:   timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:   timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Role) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleRole)
}
