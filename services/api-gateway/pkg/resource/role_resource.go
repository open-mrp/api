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
	//
	// The role's type is sometimes used to gate special behaviors in the frontend
	// and to restrict some actions to only certain types of roles. For example,
	// only roles with the type `admin` can create and manage API keys.
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

const SampleRoleID = "rl_01c16d2eb637c0d1f3a372937c"
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

// TODO: This is unacceptable. We should only send valid data back not make stuff up.
func ExpandableRoleStub(id, name string, typeCode constants.RoleType, ts time.Time) *Role {
	if id == "" {
		id = SampleRoleID
	}
	if name == "" {
		name = SampleRoleName
	}
	if typeCode == "" {
		typeCode = constants.RoleTypeAdmin
	}
	if ts.IsZero() {
		ts = time.Unix(0, 0).UTC()
	}
	return &Role{
		ID:        id,
		Object:    constants.ObjectTypeRole,
		Name:      name,
		TypeCode:  typeCode,
		Owner:     SystemOwner(),
		CreatedAt: ts,
		UpdatedAt: ts,
	}
}
