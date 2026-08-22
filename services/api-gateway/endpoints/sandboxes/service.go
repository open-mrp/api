package sandboxep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
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
	// CoreClient (required) is the core-service gRPC client.
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
	return &sandboxSvcImpl{coreClient: config.CoreClient}
}

func (m *sandboxSvcImpl) ListSandboxes(ctx context.Context, req *apiresource.PaginationRequest) (*apiresource.List[apiresource.Sandbox], *apierror.APIError) {
	pbReq := &pb.ListSandboxAccountsRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}
	resp, apiErr := grpcutil.CallRPC(ctx, sandboxSvcTracer, "service.sandboxes.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListSandboxAccountsResponse, error) {
			return m.coreClient.ListSandboxAccounts(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	ids := make([]string, len(resp.Sandboxes))
	for i, s := range resp.Sandboxes {
		ids[i] = s.Id
	}
	loaded, apiErr := resourceloaders.LoadSandboxes(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	items := make([]apiresource.Sandbox, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			items = append(items, *(v.(*apiresource.Sandbox)))
		}
	}
	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *sandboxSvcImpl) GetSandbox(ctx context.Context, req *RetrieveSandboxRequest) (*apiresource.Sandbox, *apierror.APIError) {
	return loadSandboxByID(ctx, req.SandboxID)
}

func (m *sandboxSvcImpl) CreateSandbox(ctx context.Context, req *CreateSandboxRequest) (*apiresource.Sandbox, *apierror.APIError) {
	var pbMode pb.SandboxMode
	if m, ok := req.Mode.Value(); ok && m == constants.SandboxModeSeeded {
		pbMode = pb.SandboxMode_SANDBOX_MODE_SEEDED
	} else {
		pbMode = pb.SandboxMode_SANDBOX_MODE_BLANK
	}
	pbReq := &pb.CreateSandboxRequest{
		Name: req.Name,
		Mode: pbMode,
	}
	resp, apiErr := grpcutil.CallRPC(ctx, sandboxSvcTracer, "service.sandboxes.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateSandboxResponse, error) {
			return m.coreClient.CreateSandbox(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return loadSandboxByID(ctx, resp.Sandbox.Id)
}

func (m *sandboxSvcImpl) DeleteSandbox(ctx context.Context, req *DeleteSandboxRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteSandboxRequest{Id: req.SandboxID}
	_, apiErr := grpcutil.CallRPC(ctx, sandboxSvcTracer, "service.sandboxes.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteSandbox(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return &apiresource.EmptyResource{}, nil
}

// loadSandboxByID wraps the single-ID load pattern used after every mutation
// and for the retrieve endpoint.
func loadSandboxByID(ctx context.Context, id string) (*apiresource.Sandbox, *apierror.APIError) {
	loaded, apiErr := resourceloaders.LoadSandboxes(ctx, []string{id})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[id]
	if !ok {
		return nil, apierror.NewResourceNotFoundError("Sandbox not found.")
	}
	return v.(*apiresource.Sandbox), nil
}
