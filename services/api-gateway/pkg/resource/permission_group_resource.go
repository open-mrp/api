package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SamplePermissionID = "perm_01jm4r6700f8nwq3v5hx2d9ktp"
const SamplePermissionGroupID = "pg_01jm4r6700f8nwq3v5hx2d9ktq"

// Permission represents a single permission within a permission group.
type Permission struct {
	// The unique identifier for the permission.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=permission"`
	// The unique code for this permission.
	Code string `json:"code" validate:"required"`
	// The display name.
	Name string `json:"name" validate:"required"`
	// An optional description of what this permission controls.
	Description *string `json:"description"`
	// The code of the permission group this permission belongs to.
	PermissionGroupCode string `json:"group" validate:"required"`
	// When the permission was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When the permission was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SamplePermission = &Permission{
	ID:                  SamplePermissionID,
	Object:              constants.ObjectTypePermission,
	Code:                "customers:read",
	Name:                "Read Customers",
	Description:         nil,
	PermissionGroupCode: "customers",
	CreatedAt:           timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:           timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Permission) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePermission)
}

// PermissionGroup represents a grouping of related permissions.
type PermissionGroup struct {
	// The unique identifier for the permission group.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=permission_group"`
	// The unique code for this permission group.
	Code string `json:"code" validate:"required"`
	// The display name.
	Name string `json:"name" validate:"required"`
	// An optional description.
	Description *string `json:"description"`
	// The permissions belonging to this group.
	Permissions *List[Permission] `json:"permissions"`
	// The owner of this resource.
	Owner *Owner `json:"owner" expandable:"true"`
	// When the permission group was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When the permission group was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SamplePermissionGroup = &PermissionGroup{
	ID:          SamplePermissionGroupID,
	Object:      constants.ObjectTypePermissionGroup,
	Code:        "customers",
	Name:        "Customers",
	Description: nil,
	Permissions: NewList([]Permission{*SamplePermission}, PageInfo{}),
	Owner:       SampleOwnerSystem,
	CreatedAt:   timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:   timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*PermissionGroup) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePermissionGroup)
}
