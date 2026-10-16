package roleep

import (
	"context"
	"fmt"
	"strings"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type RoleSvc interface {
	ListRoles(ctx context.Context, req *ListRolesRequest) (*apiresource.List[apiresource.Role], *apierror.APIError)
	GetRole(ctx context.Context, req *RetrieveRoleRequest) (*apiresource.Role, *apierror.APIError)
	CreateRole(ctx context.Context, req *CreateRoleRequest) (*apiresource.Role, *apierror.APIError)
	UpdateRole(ctx context.Context, req *UpdateRoleRequest) (*apiresource.Role, *apierror.APIError)
	PreviewRole(ctx context.Context, req *PreviewRoleRequest) (*apiresource.List[apiresource.Permission], *apierror.APIError)
	ListPermissions(ctx context.Context, req *ListPermissionsRequest) (*apiresource.List[apiresource.Permission], *apierror.APIError)
	ListRoleTemplates(ctx context.Context, req *ListRoleTemplatesRequest) (*apiresource.List[apiresource.RoleTemplate], *apierror.APIError)
	DeleteRole(ctx context.Context, req *DeleteRoleRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type RoleSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient pb.CoreServiceClient
}

type roleSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var roleSvcTracer = tracing.GetTracer("api-gateway.endpoints.roles.service")

func roleTypeCodesToStrings(codes []constants.RoleType) []string {
	out := make([]string, len(codes))
	for i, c := range codes {
		out[i] = string(c)
	}
	return out
}

func (c *RoleSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("role endpoint service: core client is required")
	}
	return nil
}

func NewRoleSvc(config *RoleSvcConfig) RoleSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &roleSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *roleSvcImpl) ListRoles(ctx context.Context, req *ListRolesRequest) (*apiresource.List[apiresource.Role], *apierror.APIError) {
	var cursor string
	if req.Cursor != nil {
		cursor = *req.Cursor
	}
	var query string
	if req.Query != nil {
		query = *req.Query
	}

	pbReq := &pb.ListRolesRequest{
		Cursor:        cursor,
		Limit:         req.Limit,
		Query:         query,
		RoleTypeCodes: roleTypeCodesToStrings(req.RoleType),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, roleSvcTracer, "service.roles.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListRolesResponse, error) {
			return m.coreClient.ListRoles(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	ids := make([]string, len(resp.Roles))
	for i, r := range resp.Roles {
		ids[i] = r.Id
	}
	loaded, apiErr := resourceloaders.LoadRoles(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	roles := make([]apiresource.Role, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			roles = append(roles, *(v.(*apiresource.Role)))
		}
	}
	return apiresource.NewList(roles, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *roleSvcImpl) GetRole(ctx context.Context, req *RetrieveRoleRequest) (*apiresource.Role, *apierror.APIError) {
	return loadRoleByID(ctx, req.RoleID)
}

func (m *roleSvcImpl) CreateRole(ctx context.Context, req *CreateRoleRequest) (*apiresource.Role, *apierror.APIError) {
	pbReq := &pb.CreateRoleRequest{
		Name:    req.Name,
		Access:  req.Access.levels(),
		Revokes: req.Revokes,
	}
	if t, ok := req.Template.Value(); ok {
		pbReq.TemplateCode = new(string(t))
	}
	if grants, ok := req.Grants.Value(); ok {
		pbReq.HasGrants, pbReq.Grants = true, grants
	}
	if perms, ok := req.Permissions.Value(); ok {
		def, apiErr := definitionOfPermissions(perms)
		if apiErr != nil {
			return nil, apiErr
		}
		pbReq.TemplateCode, pbReq.Access, pbReq.HasGrants, pbReq.Grants, pbReq.Revokes = nil, def.access, true, def.grants, def.revokes
	}

	resp, apiErr := grpcutil.CallRPC(ctx, roleSvcTracer, "service.roles.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateRoleResponse, error) {
			return m.coreClient.CreateRole(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return loadRoleByID(ctx, resp.Role.Id)
}

func (m *roleSvcImpl) UpdateRole(ctx context.Context, req *UpdateRoleRequest) (*apiresource.Role, *apierror.APIError) {
	pbReq := &pb.UpdateRoleRequest{
		Id:     req.RoleID,
		Access: req.Access.levels(),
	}
	if v, ok := req.Name.Value(); ok {
		pbReq.Name = v
	}
	if grants, ok := req.Grants.Value(); ok {
		pbReq.HasGrants, pbReq.Grants = true, grants
	}
	if revokes, ok := req.Revokes.Value(); ok {
		pbReq.HasRevokes, pbReq.Revokes = true, revokes
	}
	if perms, ok := req.Permissions.Value(); ok {
		def, apiErr := definitionOfPermissions(perms)
		if apiErr != nil {
			return nil, apiErr
		}
		pbReq.Access, pbReq.HasGrants, pbReq.Grants, pbReq.HasRevokes, pbReq.Revokes = def.access, true, def.grants, true, def.revokes
	}

	resp, apiErr := grpcutil.CallRPC(ctx, roleSvcTracer, "service.roles.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateRoleResponse, error) {
			return m.coreClient.UpdateRole(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return loadRoleByID(ctx, resp.Role.Id)
}

// PreviewRole works out what a role would hold, without saving anything.
func (m *roleSvcImpl) PreviewRole(_ context.Context, req *PreviewRoleRequest) (*apiresource.List[apiresource.Permission], *apierror.APIError) {
	access := types.RoleAccess{}.Complete()
	var grants []string
	if t, ok := req.Template.Value(); ok {
		spec, found := types.RoleTemplateByCode(t)
		if !found {
			return nil, apierror.NewFieldError("template", apierror.CodeInvalidFormat, fmt.Sprintf("Unknown role template %q.", t))
		}
		for area, level := range spec.Access {
			access[area] = level
		}
		grants = spec.Grants
	}
	for area, level := range req.Access.roleAccess() {
		access[area] = level
	}
	if g, ok := req.Grants.Value(); ok {
		grants = g
	}
	for param, codes := range map[string][]string{"grants": grants, "revokes": req.Revokes} {
		if unknown := types.UnknownPermissionCodes(codes); len(unknown) > 0 {
			return nil, apierror.NewFieldError(param, apierror.CodeInvalidFormat, fmt.Sprintf("Unknown permission codes: %s.", strings.Join(unknown, ", ")))
		}
	}
	codes := types.EffectivePermissions(access, grants, req.Revokes)
	out := make([]apiresource.Permission, 0, len(codes))
	for _, c := range codes {
		if p, ok := types.CatalogPermissionByCode(c); ok {
			out = append(out, apiresource.PermissionOf(p))
		}
	}
	return apiresource.NewList(out, apiresource.PageInfo{}), nil
}

type permissionsDefinition struct {
	access  map[string]string
	grants  []string
	revokes []string
}

// definitionOfPermissions describes a whole permission list as access none everywhere, the list as grants, and the
// baseline permissions it leaves out as revokes, so the role holds exactly the list. Codes of a known permission with
// an action the catalog doesn't offer grant nothing and are dropped.
func definitionOfPermissions(perms []string) (permissionsDefinition, *apierror.APIError) {
	def := permissionsDefinition{access: map[string]string{}, grants: []string{}, revokes: []string{}}
	for _, area := range constants.PermissionAreas() {
		def.access[string(area)] = string(constants.AccessLevelNone)
	}
	held := map[string]bool{}
	for _, p := range perms {
		domainCode, action, ok := strings.Cut(p, ":")
		if !ok {
			return def, apierror.NewFieldError("permissions", apierror.CodeInvalidFormat, fmt.Sprintf("Invalid permission format %q: expected \"domain:action\".", p))
		}
		if !types.PermissionDomain(domainCode).IsValid() {
			return def, apierror.NewFieldError("permissions", apierror.CodeInvalidFormat, fmt.Sprintf("Unknown permission %q: %q is not a recognized permission code.", p, domainCode))
		}
		switch types.Action(action) {
		case types.ActionCreate, types.ActionRead, types.ActionUpdate, types.ActionDelete:
		default:
			return def, apierror.NewFieldError("permissions", apierror.CodeInvalidFormat, fmt.Sprintf("Invalid permission action %q: must be create, read, update, or delete.", action))
		}
		if _, known := types.CatalogPermissionByCode(p); known && !held[p] {
			held[p] = true
			def.grants = append(def.grants, p)
		}
	}
	for _, c := range types.LevelPermissions(types.RoleAccess{}) {
		if !held[c] {
			def.revokes = append(def.revokes, c)
		}
	}
	return def, nil
}

func (m *roleSvcImpl) DeleteRole(ctx context.Context, req *DeleteRoleRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteRoleRequest{
		Id: req.RoleID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, roleSvcTracer, "service.roles.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteRole(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func loadRoleByID(ctx context.Context, id string) (*apiresource.Role, *apierror.APIError) {
	loaded, apiErr := resourceloaders.LoadRoles(ctx, []string{id})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[id]
	if !ok {
		return nil, apierror.NewNotFoundError("Role not found.")
	}
	return v.(*apiresource.Role), nil
}
