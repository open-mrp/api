package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// A named set of permissions that can be assigned to users to control what they can access.
type Role struct {
	// Role ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=role"`
	// Display name, unique within the account.
	Name string `json:"name" validate:"required"`
	// The kind of role.
	//
	// The role's type is sometimes used to gate special behaviors and to restrict some actions to only certain types of roles. For example, only roles with the type `admin` can create and manage API keys.
	//
	// - `admin`: full administrative access, including managing API keys.
	// - `user`: a custom role tailored to a specific need (its permissions are defined explicitly). Roles created through the API always have this type.
	// - `scanner`: a role for scanning-station operators.
	// - `sales_rep`: a role for sales representatives.
	// - `agent`: a role assigned to an automated agent rather than a person.
	TypeCode constants.RoleType `json:"type" validate:"required"`
	// Provenance of this role.
	//
	// System-owned roles are global defaults shared across all accounts and cannot be modified or deleted; account-owned roles are custom roles created by that account.
	Owner *Owner `json:"owner" expandable:"true"`
	// Permissions granted by this role, in `{domain}:{action}` format, such as `customers:read`.
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
