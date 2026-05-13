package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
)

// Role represents a role record from the database.
type Role struct {
	ID        string
	Name      string `audit:"name"`
	RoleType  string `audit:"role_type_code"`
	AccountID *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// RolePermission represents a structured role_permission record.
type RolePermission struct {
	ID             string
	PermissionCode string
	Create         bool
	Read           bool
	Update         bool
	Delete         bool
	RoleID         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// RoleWithPermissions combines a role with its structured permissions.
type RoleWithPermissions struct {
	Role
	Permissions []*RolePermission
}

// ListRolesParams are the parameters for listing roles.
type ListRolesParams struct {
	AccountID string
	Cursor    *string
	Limit     int32
	Query     *string
	RoleTypes []string
	Includes  []string
}

// ListRolesPage is the paginated result from the role repository.
type ListRolesPage struct {
	Roles    []*Role
	PageInfo pagination.PageInfo
}

// ListRolesResult is the result of listing roles, including permissions.
type ListRolesResult struct {
	Roles    []*RoleWithPermissions
	PageInfo pagination.PageInfo
}

// CreateRoleParams are the parameters for creating a role.
type CreateRoleParams struct {
	AccountID   string
	Name        string
	Permissions []CreateRolePermissionInput
}

// CreateRolePermissionInput represents a single permission to attach to a role.
type CreateRolePermissionInput struct {
	PermissionCode string
	Create         bool
	Read           bool
	Update         bool
	Delete         bool
}

// UpdateRoleParams are the parameters for updating a role.
type UpdateRoleParams struct {
	RoleID      string
	AccountID   string
	Name        *string
	Permissions *[]CreateRolePermissionInput
}
