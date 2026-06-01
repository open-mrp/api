package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SamplePermissionID = "perm_012ef15fedb7ecb0b8fbc034c2"
const SamplePermissionGroupID = "pg_01d4698b58ad018c0c72681e46"

// Permission within a permission group.
type Permission struct {
	// Permission ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=permission"`
	// Permission code.
	Code string `json:"code" validate:"required"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Description of what this permission controls.
	Description *string `json:"description"`
	// Permission group code.
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

// Grouping of related permissions.
type PermissionGroup struct {
	// Permission group ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=permission_group"`
	// Permission group code.
	Code string `json:"code" validate:"required"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Description.
	Description *string `json:"description"`
	// Permissions in this group.
	Permissions *List[Permission] `json:"permissions"`
	// Owner of this resource.
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
