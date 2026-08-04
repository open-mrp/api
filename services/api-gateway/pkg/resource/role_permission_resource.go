package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// A role's access grant for a single resource domain, with separate create, read, update, and delete flags.
type RolePermission struct {
	// Role permission ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=role_permission"`
	// Resource domain this entry grants access to, such as `customers` or `orders`.
	//
	// The `create`, `read`, `update`, and `delete` flags below apply to this domain.
	PermissionCode string `json:"domain" validate:"required"`
	// Grants create access.
	Create bool `json:"create"`
	// Grants read access.
	Read bool `json:"read"`
	// Grants update access.
	Update bool `json:"update"`
	// Grants delete access.
	Delete bool `json:"delete"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

const SampleRolePermissionID = "rlpm_5wfzi61dig0c"

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
