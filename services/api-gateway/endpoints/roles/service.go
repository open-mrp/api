package roleep

import (
	"context"
	"fmt"
	"strings"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type RoleSvc interface {
	ListRoles(ctx context.Context, req *ListRolesRequest) (*apiresource.List[apiresource.Role], *apierror.APIError)
	GetRole(ctx context.Context, req *GetRoleRequest) (*apiresource.Role, *apierror.APIError)
	CreateRole(ctx context.Context, req *CreateRoleRequest) (*apiresource.Role, *apierror.APIError)
	UpdateRole(ctx context.Context, req *UpdateRoleRequest) (*apiresource.Role, *apierror.APIError)
	DeleteRole(ctx context.Context, req *DeleteRoleRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type RoleSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type roleSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var roleSvcTracer = tracing.GetTracer("api-gateway.endpoints.roles.service")

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
		RoleTypeCodes: req.RoleType,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, roleSvcTracer, "service.roles.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListRolesResponse, error) {
			return m.coreClient.ListRoles(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return RoleListPresenter(resp), nil
}

func (m *roleSvcImpl) GetRole(ctx context.Context, req *GetRoleRequest) (*apiresource.Role, *apierror.APIError) {
	pbReq := &pb.GetRoleRequest{
		Id: req.RoleID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, roleSvcTracer, "service.roles.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetRoleResponse, error) {
			return m.coreClient.GetRole(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := RolePresenter(resp.Role)
	return &result, nil
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

	result := RolePresenter(resp.Role)
	return &result, nil
}

func (m *roleSvcImpl) UpdateRole(ctx context.Context, req *UpdateRoleRequest) (*apiresource.Role, *apierror.APIError) {
	var name string
	if req.Name != nil {
		name = *req.Name
	}

	pbReq := &pb.UpdateRoleRequest{
		Id:   req.RoleID,
		Name: name,
	}

	if req.Permissions != nil {
		pbReq.HasPermissions = true
		pbPerms, apiErr := parsePermissionStrings(*req.Permissions)
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

	result := RolePresenter(resp.Role)
	return &result, nil
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

// parsePermissionStrings converts "domain:action" strings (e.g. "customers:create")
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
