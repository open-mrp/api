package roleep

import (
	"context"
	"encoding/base64"
	"net/http"
	"slices"
	"strings"

	"github.com/open-mrp/apikit/object"

	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list the permission catalog.
type ListPermissionsRequest struct {
	object.ListRequest
	// Case-insensitive substring match on `code` or `name`.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Only return permissions in any of these areas.
	Areas []constants.PermissionArea `query:"areas"`
	// Only return permissions at any of these levels.
	Levels []constants.AccessLevel `query:"levels"`
}

// Returns the permission catalog.
//
// Sorted by code. Every permission belongs to one area and level; a role's `grants`, `revokes` and `permissions` use these codes. The catalog is the same for every account.
type ListPermissionsEndpoint struct{}

func (e *ListPermissionsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListPermissionsRequest, *apiresource.List[apiresource.Permission]] {
	return (&apiendpoint.APIEndpoint[*ListPermissionsRequest, *apiresource.List[apiresource.Permission]]{
		Title:             "List Permissions",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/identity/permissions",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainPermissions, Action: types.ActionRead}}},
		ObjectType:        constants.ObjectTypePermission,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListPermissionsRequest) (*apiresource.List[apiresource.Permission], *apierror.APIError) {
			return svc.(RoleSvc).ListPermissions
		},
	})
}

// Request to list role templates.
type ListRoleTemplatesRequest struct {
	object.ListRequest
}

// Returns the role templates.
//
// A template is a starting point for a role: create a role with its `template` to copy its access and grants. The templates are the same for every account.
type ListRoleTemplatesEndpoint struct{}

func (e *ListRoleTemplatesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListRoleTemplatesRequest, *apiresource.List[apiresource.RoleTemplate]] {
	return (&apiendpoint.APIEndpoint[*ListRoleTemplatesRequest, *apiresource.List[apiresource.RoleTemplate]]{
		Title:             "List Role Templates",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/identity/role-templates",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainRoles, Action: types.ActionRead}}},
		ObjectType:        constants.ObjectTypeRoleTemplate,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListRoleTemplatesRequest) (*apiresource.List[apiresource.RoleTemplate], *apierror.APIError) {
			return svc.(RoleSvc).ListRoleTemplates
		},
	})
}

func (m *roleSvcImpl) ListPermissions(ctx context.Context, req *ListPermissionsRequest) (*apiresource.List[apiresource.Permission], *apierror.APIError) {
	q := ""
	if req.Query != nil {
		q = strings.ToLower(strings.TrimSpace(*req.Query))
	}
	var all []apiresource.Permission
	keys := []string{}
	for _, p := range types.PermissionCatalog() {
		if len(req.Areas) > 0 && !slices.Contains(req.Areas, p.Area) {
			continue
		}
		if len(req.Levels) > 0 && !slices.Contains(req.Levels, p.Level) {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(p.Code), q) && !strings.Contains(strings.ToLower(p.Name), q) {
			continue
		}
		all = append(all, apiresource.PermissionOf(p))
		keys = append(keys, p.Code)
	}
	return pageOf(ctx, all, keys, req.Cursor, req.Limit)
}

func (m *roleSvcImpl) ListRoleTemplates(ctx context.Context, req *ListRoleTemplatesRequest) (*apiresource.List[apiresource.RoleTemplate], *apierror.APIError) {
	var all []apiresource.RoleTemplate
	var keys []string
	for _, t := range types.RoleTemplates() {
		all = append(all, apiresource.RoleTemplateOf(t))
		keys = append(keys, string(t.Code))
	}
	return pageOf(ctx, all, keys, req.Cursor, req.Limit)
}

// pageOf pages a fixed, ordered catalog. The cursor is the last key of the previous page.
func pageOf[T any](ctx context.Context, items []T, keys []string, cursor *string, limit int32) (*apiresource.List[T], *apierror.APIError) {
	start := 0
	if cursor != nil && *cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(*cursor)
		if err != nil {
			return nil, apierror.NewParameterInvalidError("cursor", "Invalid pagination cursor.")
		}
		i := slices.Index(keys, string(raw))
		if i < 0 {
			return nil, apierror.NewParameterInvalidError("cursor", "Invalid pagination cursor.")
		}
		start = i + 1
	}
	if limit <= 0 {
		limit = 25
	}
	end := min(start+int(limit), len(items))
	page := &pb.PageInfo{}
	if end < len(items) {
		page.NextCursor = new(base64.RawURLEncoding.EncodeToString([]byte(keys[end-1])))
	}
	return apiresource.NewList(items[start:end], grpcutil.MapProtoPageInfo(ctx, page)), nil
}
