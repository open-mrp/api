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
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type RoleSvc interface {
	ListRoles(ctx context.Context, req *ListRolesRequest) (*apiresource.List[apiresource.Role], *apierror.APIError)
	GetRole(ctx context.Context, req *RetrieveRoleRequest) (*apiresource.Role, *apierror.APIError)
	CreateRole(ctx context.Context, req *CreateRoleRequest) (*apiresource.Role, *apierror.APIError)
	UpdateRole(ctx context.Context, req *UpdateRoleRequest) (*apiresource.Role, *apierror.APIError)
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
	pbPerms, apiErr := parsePermissionStrings(req.Permissions)
	if apiErr != nil {
		return nil, apiErr
	}

	pbReq := &pb.CreateRoleRequest{
		Name:        req.Name,
		Permissions: pbPerms,
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
	var name string
	if v, ok := req.Name.Value(); ok {
		name = v
	}

	pbReq := &pb.UpdateRoleRequest{
		Id:   req.RoleID,
		Name: name,
	}

	if perms, ok := req.Permissions.Value(); ok {
		pbReq.HasPermissions = true
		pbPerms, apiErr := parsePermissionStrings(perms)
		if apiErr != nil {
			return nil, apiErr
		}
		pbReq.Permissions = pbPerms
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
		return nil, apierror.NewResourceNotFoundError("Role not found.")
	}
	return v.(*apiresource.Role), nil
}

// parsePermissionStrings converts `<domain>:<action>` strings (e.g. "customers:create")
// into grouped proto permission inputs keyed by domain.
func parsePermissionStrings(perms []string) ([]*pb.CreateRolePermissionInput, *apierror.APIError) {
	grouped := make(map[string]*pb.CreateRolePermissionInput)
	for _, p := range perms {
		parts := strings.SplitN(p, ":", 2)
		if len(parts) != 2 {
			return nil, apierror.NewValidationErrorWithParam(
				fmt.Sprintf("Invalid permission format %q: expected \"domain:action\".", p), "permissions",
			)
		}
		domain, action := parts[0], parts[1]

		if !types.PermissionDomain(domain).IsValid() {
			return nil, apierror.NewValidationErrorWithParam(
				fmt.Sprintf("Unknown permission %q: %q is not a recognized permission code.", p, domain), "permissions",
			)
		}

		entry, ok := grouped[domain]
		if !ok {
			entry = &pb.CreateRolePermissionInput{PermissionCode: domain}
			grouped[domain] = entry
		}

		switch action {
		case "create":
			entry.Create = true
		case "read":
			entry.Read = true
		case "update":
			entry.Update = true
		case "delete":
			entry.Delete = true
		default:
			return nil, apierror.NewValidationErrorWithParam(
				fmt.Sprintf("Invalid permission action %q: must be create, read, update, or delete.", action), "permissions",
			)
		}
	}

	result := make([]*pb.CreateRolePermissionInput, 0, len(grouped))
	for _, entry := range grouped {
		result = append(result, entry)
	}
	return result, nil
}
