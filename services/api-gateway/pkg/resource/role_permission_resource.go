package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// RolePermission represents a structured permission entry on a role.
type RolePermission struct {
	// The unique identifier for this role permission.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=role_permission"`
	// The permission domain code.
	PermissionCode string `json:"permission_code" validate:"required"`
	// Whether this permission grants create access.
	Create bool `json:"create"`
	// Whether this permission grants read access.
	Read bool `json:"read"`
	// Whether this permission grants update access.
	Update bool `json:"update"`
	// Whether this permission grants delete access.
	Delete bool `json:"delete"`
	// When this role permission was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this role permission was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

const SampleRolePermissionID = "rlpm_01gf7a8200er3ar3pkfrb6kk29"

var SampleRolePermission = &RolePermission{
	ID:             SampleRolePermissionID,
	Object:         constants.ObjectTypeRolePermission,
	PermissionCode: "customers",
	Create:         true,
	Read:           true,
	Update:         true,
	Delete:         false,
	CreatedAt:      timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:      timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*RolePermission) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleRolePermission)
}
