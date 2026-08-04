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
	// Display name of the role.
	//
	// Unique within the account.
	Name string `json:"name" validate:"required"`
	// The kind of role.
	//
	// The type gates behavior that individual permissions do not cover, and some actions are reserved for a single role type.
	//
	// - `admin`: full administrative access. Sensitive areas such as API keys, billing, and third-party integrations are restricted to admins no matter what permissions another role holds.
	// - `user`: a custom role tailored to a specific need, with its permissions defined explicitly. Roles created through the API always have this type.
	// - `scanner`: the role used by shop-floor scanning stations, assigned automatically when a scanning-station user is created.
	// - `sales_rep`: a role for sales representatives. Order analytics are scoped to the rep's own orders.
	// - `agent`: a role assigned to an automated agent rather than a person.
	TypeCode constants.RoleType `json:"type" validate:"required"`
	// Provenance of this role.
	//
	// System-owned roles are platform-provided defaults shared across all accounts and cannot be updated or deleted; account-owned roles are custom to your account.
	Owner *Owner `json:"owner" expandable:"true"`
	// Permissions granted by this role, in `{permission}:{action}` format, such as `customers:read`.
	Permissions *[]string `json:"permissions" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at"`
}

const SampleRoleID = "rl_3xknmfqflhvb"
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
