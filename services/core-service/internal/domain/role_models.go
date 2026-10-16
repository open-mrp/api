package domain

import (
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/pagination"
)

// Role represents a role record from the database.
type Role struct {
	ID        string
	Name      string `audit:"name"`
	RoleType  string `audit:"role_type_code"`
	AccountID *string
	// Access is the role's level per area; nil for a role saved before access levels, which is described from what it
	// holds.
	Access    types.RoleAccess `audit:"access"`
	Template  *string          `audit:"template_code"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// RoleReferences blocks deleting a role that members (other than removed ones) or API keys still hold.
var RoleReferences = RegisterReferences(References{
	Noun: "role",
	Probes: []ReferenceProbe{
		{Label: "account users", Table: "account_user", Column: "role_id", Where: "status_code <> 'removed'"},
		{Label: "API keys", Table: "api_key", Column: "role_id", Where: "revoked_at IS NULL"},
	},
})

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

// HeldCodes is every `<domain>:<action>` code the role's permission rows grant.
func HeldCodes(perms []*RolePermission) map[string]bool {
	held := map[string]bool{}
	for _, p := range perms {
		for action, granted := range map[types.Action]bool{
			types.ActionCreate: p.Create, types.ActionRead: p.Read, types.ActionUpdate: p.Update, types.ActionDelete: p.Delete,
		} {
			if granted {
				held[types.Permission{Domain: types.PermissionDomain(p.PermissionCode), Action: action}.String()] = true
			}
		}
	}
	return held
}

// Definition describes the role as access per area plus grants and revokes. A role with no stored access is given the
// levels that describe what it holds most simply.
func (r *RoleWithPermissions) Definition() types.RoleDefinition {
	held := HeldCodes(r.Permissions)
	if r.Access == nil {
		codes := make([]string, 0, len(held))
		for c := range held {
			codes = append(codes, c)
		}
		return types.DefinitionFromEffective(codes)
	}
	return types.DefinitionWithAccess(r.Access, held)
}

// PermissionRowsFor turns held codes into one permission row input per domain.
func PermissionRowsFor(codes []string) []CreateRolePermissionInput {
	byDomain := map[string]*CreateRolePermissionInput{}
	var order []string
	for _, code := range codes {
		p, ok := types.CatalogPermissionByCode(code)
		if !ok {
			continue
		}
		domain, action, _ := cutCode(p.Code)
		in, ok := byDomain[domain]
		if !ok {
			in = &CreateRolePermissionInput{PermissionCode: domain}
			byDomain[domain] = in
			order = append(order, domain)
		}
		switch types.Action(action) {
		case types.ActionCreate:
			in.Create = true
		case types.ActionRead:
			in.Read = true
		case types.ActionUpdate:
			in.Update = true
		case types.ActionDelete:
			in.Delete = true
		}
	}
	out := make([]CreateRolePermissionInput, 0, len(order))
	for _, d := range order {
		out = append(out, *byDomain[d])
	}
	return out
}

func cutCode(code string) (string, string, bool) {
	for i := len(code) - 1; i >= 0; i-- {
		if code[i] == ':' {
			return code[:i], code[i+1:], true
		}
	}
	return code, "", false
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

// CreateRoleParams are the parameters for creating a role. Template seeds access and grants; Access overrides the
// template's level per area; Grants replaces the template's grants when set.
type CreateRoleParams struct {
	AccountID string
	Name      string
	Template  *constants.RoleTemplate
	Access    types.RoleAccess
	Grants    *[]string
	Revokes   []string
}

// CreateRolePermissionInput represents a single permission to attach to a role.
type CreateRolePermissionInput struct {
	PermissionCode string
	Create         bool
	Read           bool
	Update         bool
	Delete         bool
}

// UpdateRoleParams are the parameters for updating a role. Access changes only the areas it names; Grants and Revokes
// replace the role's when set.
type UpdateRoleParams struct {
	RoleID    string
	AccountID string
	Name      *string
	Access    types.RoleAccess
	Grants    *[]string
	Revokes   *[]string
}
