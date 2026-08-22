package domain

import (
	"time"

	"github.com/open-mrp/api/shared/pagination"
)

type Permission struct {
	ID                  string
	Code                string
	Name                string
	Description         *string
	PermissionGroupCode string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type PermissionGroup struct {
	ID          string
	Code        string
	Name        string
	Description *string
	Permissions []*Permission
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ListPermissionGroupsParams struct {
	Cursor *string
	Limit  int32
	Query  *string
}

type ListPermissionGroupsResult struct {
	PermissionGroups []*PermissionGroup
	PageInfo         pagination.PageInfo
}
