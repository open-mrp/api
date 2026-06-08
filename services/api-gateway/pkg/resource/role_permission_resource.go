package apiresource

import (
	"time"

	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// RolePermission is a permission entry on a role.
type RolePermission struct {
	// Role permission ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=role_permission"`
	// Permission domain code.
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

const SampleRolePermissionID = "rlpm_010497787b78b93e595cd90dab"

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
