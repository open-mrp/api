package sandboxep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type SandboxSvc interface {
	ListSandboxes(ctx context.Context, req *apiresource.PaginationRequest) (*apiresource.List[apiresource.Sandbox], *apierror.APIError)
	GetSandbox(ctx context.Context, req *RetrieveSandboxRequest) (*apiresource.Sandbox, *apierror.APIError)
	CreateSandbox(ctx context.Context, req *CreateSandboxRequest) (*apiresource.Sandbox, *apierror.APIError)
	DeleteSandbox(ctx context.Context, req *DeleteSandboxRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type SandboxSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type sandboxSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var sandboxSvcTracer = tracing.GetTracer("api-gateway.endpoints.sandboxes.service")

func (c *SandboxSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("sandbox endpoint service: core client is required")
	}
	return nil
}

func NewSandboxSvc(config *SandboxSvcConfig) SandboxSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &sandboxSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *sandboxSvcImpl) ListSandboxes(ctx context.Context, req *apiresource.PaginationRequest) (*apiresource.List[apiresource.Sandbox], *apierror.APIError) {
	pbReq := &pb.ListSandboxAccountsRequest{
		Cursor:   req.Cursor,
		Limit:    req.Limit,
		Includes: appctx.GetRequestedIncludeKeys(ctx),
		Query:    req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, sandboxSvcTracer, "service.sandboxes.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListSandboxAccountsResponse, error) {
			return m.coreClient.ListSandboxAccounts(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return SandboxListPresenter(ctx, resp), nil
}

func (m *sandboxSvcImpl) CreateSandbox(ctx context.Context, req *CreateSandboxRequest) (*apiresource.Sandbox, *apierror.APIError) {
	var pbMode pb.SandboxMode
	if m, ok := req.Mode.Value(); ok && m == constants.SandboxModeSeeded {
		pbMode = pb.SandboxMode_SANDBOX_MODE_SEEDED
	} else {
		pbMode = pb.SandboxMode_SANDBOX_MODE_BLANK
	}

	pbReq := &pb.CreateSandboxRequest{
		Name:     req.Name,
		Mode:     pbMode,
		Includes: appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, sandboxSvcTracer, "service.sandboxes.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateSandboxResponse, error) {
			return m.coreClient.CreateSandbox(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := SandboxPresenter(resp.Sandbox)
	return &result, nil
}

func (m *sandboxSvcImpl) GetSandbox(ctx context.Context, req *RetrieveSandboxRequest) (*apiresource.Sandbox, *apierror.APIError) {
	pbReq := &pb.GetSandboxRequest{
		Id:       req.SandboxID,
		Includes: appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, sandboxSvcTracer, "service.sandboxes.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetSandboxResponse, error) {
			return m.coreClient.GetSandbox(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := SandboxPresenter(resp.Sandbox)
	return &result, nil
}

func (m *sandboxSvcImpl) DeleteSandbox(ctx context.Context, req *DeleteSandboxRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteSandboxRequest{
		Id: req.SandboxID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, sandboxSvcTracer, "service.sandboxes.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteSandbox(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}
