package permissiongroupep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

type PermissionGroupSvc interface {
	ListPermissionGroups(ctx context.Context, req *ListPermissionGroupsRequest) (*apiresource.List[apiresource.PermissionGroup], *apierror.APIError)
}

type PermissionGroupSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type permissionGroupSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var permissionGroupSvcTracer = tracing.GetTracer("api-gateway.endpoints.permission_groups.service")

func (c *PermissionGroupSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("permission group endpoint service: core client is required")
	}
	return nil
}

func NewPermissionGroupSvc(config *PermissionGroupSvcConfig) PermissionGroupSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &permissionGroupSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *permissionGroupSvcImpl) ListPermissionGroups(ctx context.Context, req *ListPermissionGroupsRequest) (*apiresource.List[apiresource.PermissionGroup], *apierror.APIError) {
	pbReq := &pb.ListPermissionGroupsRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, permissionGroupSvcTracer, "service.permission_groups.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListPermissionGroupsResponse, error) {
			return m.coreClient.ListPermissionGroups(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return PermissionGroupListPresenter(ctx, resp), nil
}
